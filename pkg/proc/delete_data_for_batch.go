package proc

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

const HarvestForDeleteRowsetSize = 1000 // Do not let users tweak it, maybe too sensitive

func populateUniqueKeysToDeleteMap(uniqueKeysToDeleteMap map[string][]string, indexesMap sc.IdxDefMap, rowIdsToDeleteCount int, tableRecord map[string]any) error {
	for idxName, idxDef := range indexesMap {
		if _, ok := uniqueKeysToDeleteMap[idxName]; ok {
			var err error
			uniqueKeysToDeleteMap[idxName][rowIdsToDeleteCount], err = sc.BuildKey(tableRecord, idxDef)
			if err != nil {
				return fmt.Errorf("while deleting previous batch attempt leftovers, cannot build a key for index %s from [%v]: %s", idxName, tableRecord, err.Error())
			}
			if len(uniqueKeysToDeleteMap[idxName][rowIdsToDeleteCount]) == 0 {
				return fmt.Errorf("invalid empty key calculated for %v", tableRecord)
			}
		}
	}
	return nil
}

func deleteDataRecordByRowid(pCtx *ctx.MessageProcessingContext, rowids []int64) error {
	if pCtx.CassandraEngine == db.CassandraEngineAmazonKeyspaces {
		// Amazon Keyspaces supports unlogged batch commands with up to 30 commands in the batch
		sb := strings.Builder{}
		for i, rowid := range rowids {
			sb.WriteString(
				(&cql.QueryBuilder{}).
					Keyspace(pCtx.Msg.DataKeyspace).
					Cond("rowid", "=", rowid).
					DeleteRun(pCtx.CurrentScriptNode.TableCreator.Name, pCtx.Msg.RunId))
			sb.WriteString(";")
			if (i+1)%MaxAmazonKeyspacesBatchLen == 0 || i == len(rowids)-1 {
				batchStmt := "BEGIN UNLOGGED BATCH " + sb.String() + " APPLY BATCH"
				if err := pCtx.CqlSession.Query(batchStmt).Exec(); err != nil {
					return db.WrapDbErrorWithQuery("cannot delete from data table", batchStmt, err)
				}
				sb.Reset()
			}
		}
	} else {
		q := (&cql.QueryBuilder{}).
			Keyspace(pCtx.Msg.DataKeyspace).
			CondInInt("rowid", rowids).
			DeleteRun(pCtx.CurrentScriptNode.TableCreator.Name, pCtx.Msg.RunId)
		if err := pCtx.CqlSession.Query(q).Exec(); err != nil {
			return db.WrapDbErrorWithQuery("cannot delete from data table", q, err)
		}
	}
	return nil
}

// To test it, see comments in the end of RunCreateTableRelForBatch
func DeleteDataAndUniqueIndexesByBatchIdx(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.DeleteDataAndUniqueIndexesByBatchIdx")
	defer logger.PopF()

	if !pCtx.CurrentScriptNode.HasTableCreator() {
		logger.InfoCtx(pCtx, "no table creator, nothing to delete for %s", pCtx.Msg.FullBatchId())
		return nil
	}

	uniqueKeysToDeleteMap := map[string][]string{} // unique_idx_name -> list_of_keys_to_delete
	for idxName, idxDef := range pCtx.CurrentScriptNode.TableCreator.Indexes {
		// We are not going to delete non-unique index entries because they may point to data rows from other batches.
		// Even if those non-unique idx records for current abandoned remain, they will not hurt on lookups:
		// selectBatchFromIdxTablePaged() will return a key/rowid pair with a non-existing rowid, and
		// subsequent selectBatchFromDataTablePaged(rightRowidsToFind) will simply NOT return anything for that specific missing rowid.
		if idxDef.Uniqueness == sc.IdxUnique {
			uniqueKeysToDeleteMap[idxName] = nil
		}
	}
	logger.WarnCtx(pCtx, "deleting data and unique idx records for %s, %d unique indexes detected: [%s]", pCtx.Msg.FullBatchId(), len(uniqueKeysToDeleteMap), strings.Join(slices.Collect(maps.Keys(uniqueKeysToDeleteMap)), ","))

	deleteStartTime := time.Now()
	totalDataRowsDeleted := 0
	totalIdxRowsDeleted := 0

	// IMPORTANT!
	// Here, we potentially have to select ALL rows (not just rows added for this batch), which may take forever.
	// If we want to select by batch_idx only, we should make it partitioning key.
	// but in this case, we will not be able to use token(rowid), which we heavily rely on when going through data records (see selectBatchFromTableByToken).
	// And if we add rowid to the partition key to be able to query rows by token(batch_idx,rowid), then we lose the possibility to query just be batch_idx
	// because Cassandra cannot filter by partial partitioning key.
	// So, for a billion-rows scenarios, resort to the no-rerun policy, and re-run the whole node when needed.

	// retrieve all fields that are involved in building unique indexes, and batch_idx - we will manually filter by it
	uniqueIdxFieldRefs := pCtx.CurrentScriptNode.GetUniqueIndexesFieldRefs()
	rs := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(pCtx.CurrentScriptNode.TableCreator.Name)},
		*uniqueIdxFieldRefs,
		sc.FieldRefs{sc.BatchIdxFieldRef(pCtx.CurrentScriptNode.TableCreator.Name)})

	var pageState []byte
	var err error
	for {
		pageState, err = selectBatchPagedAllRowids(logger,
			pCtx,
			rs,
			pCtx.CurrentScriptNode.TableCreator.Name,
			pCtx.Msg.RunId,
			HarvestForDeleteRowsetSize,
			pageState)
		if err != nil {
			return err
		}

		if rs.RowCount == 0 {
			break
		}

		// Prepare the storage for rowids and keys
		rowIdsToDelete := make([]int64, rs.RowCount)
		for uniqueIdxName := range uniqueKeysToDeleteMap {
			uniqueKeysToDeleteMap[uniqueIdxName] = make([]string, rs.RowCount)
		}

		rowIdsToDeleteCount := 0
		for rowIdx := 0; rowIdx < rs.RowCount; rowIdx++ {
			rowId := *((*rs.Rows[rowIdx])[rs.FieldsByFieldName["rowid"]].(*int64))
			batchIdx := int16(*((*rs.Rows[rowIdx])[rs.FieldsByFieldName["batch_idx"]].(*int64)))

			// Harvest only rowids with batchIdx we are interested in (specific batch_idx), also harvest keys
			if batchIdx != pCtx.Msg.BatchIdx {
				continue
			}

			// Add this rowid to the list of rowids to delete
			rowIdsToDelete[rowIdsToDeleteCount] = rowId

			// Get full table record, not just rowid+batch_idx
			tableRecord, err := rs.GetTableRecord(rowIdx)
			if err != nil {
				return fmt.Errorf("while deleting previous batch attempt leftovers, cannot get table record from [%v]: %s", rs.Rows[rowIdx], err.Error())
			}

			// For each idx, build the key and add it to the uniqueKeysToDeleteMap
			if err := populateUniqueKeysToDeleteMap(uniqueKeysToDeleteMap,
				pCtx.CurrentScriptNode.TableCreator.Indexes,
				rowIdsToDeleteCount,
				tableRecord); err != nil {
				return err
			}
			rowIdsToDeleteCount++
		}

		if rowIdsToDeleteCount > 0 {
			// Trim unused empty rowid slots
			rowIdsToDelete = rowIdsToDelete[:rowIdsToDeleteCount]

			// Ordering matters for crash recovery: delete INDEX records by key FIRST, then DATA records by rowid.
			// If we crash between the two, we are left with orphan DATA rows (which the full data-table scan above
			// will find and clean up on the next rerun) rather than orphan INDEX rows (which point to data rows that
			// no longer exist and cannot be discovered by scanning the data table, permanently breaking uniqueness).
			for idxName, idxKeysToDelete := range uniqueKeysToDeleteMap {
				// Trim unused empty key slots
				trimmedIdxKeysToDelete := idxKeysToDelete[:rowIdsToDeleteCount]

				logger.DebugCtx(pCtx, "deleting %d idx %s records for %s: [%s]", len(trimmedIdxKeysToDelete), idxName, pCtx.Msg.FullBatchId(), strings.Join(trimmedIdxKeysToDelete, `','`))
				if err := deleteIdxRecordByKey(pCtx, idxName, trimmedIdxKeysToDelete); err != nil {
					return err
				}
				totalIdxRowsDeleted += len(trimmedIdxKeysToDelete)
			}

			// Delete data records by rowid
			logger.DebugCtx(pCtx, "deleting %d data records for %s: %v", len(rowIdsToDelete), pCtx.Msg.FullBatchId(), rowIdsToDelete)
			if err := deleteDataRecordByRowid(pCtx, rowIdsToDelete); err != nil {
				return err
			}
			totalDataRowsDeleted += len(rowIdsToDelete)
		}

		// Amazon Keyspaces: do not rely on the retrieved row count, use pagestate
		if pCtx.CassandraEngine == db.CassandraEngineAmazonKeyspaces && len(pageState) == 0 {
			break
		}

		// Reset pageState, DELETE above messed with it. Yes, we will have to walk through many rows from other batches AGAIN, but this is the price we have to pay
		if rowIdsToDeleteCount > 0 {
			pageState = []byte{}
		}

	}

	logger.WarnCtx(pCtx, "deleted %d data and %d unique idx records for %s, elapsed %v", totalDataRowsDeleted, totalIdxRowsDeleted, pCtx.Msg.FullBatchId(), time.Since(deleteStartTime))

	return nil
}
