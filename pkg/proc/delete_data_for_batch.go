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

func populateUniqueKeysToDeleteMap(uniqueKeysToDeleteMap map[string][]*keyRowidPair, indexesMap sc.IdxDefMap, rowIdsToDeleteCount int, tableRecord map[string]any, rowid int64) error {
	for idxName, idxDef := range indexesMap {
		if _, ok := uniqueKeysToDeleteMap[idxName]; ok {
			var err error
			var pair keyRowidPair
			pair.rowid = rowid
			pair.key, err = sc.BuildKey(tableRecord, idxDef)
			if err != nil {
				return fmt.Errorf("while deleting previous batch attempt leftovers, cannot build a key for index %s from [%v]: %s", idxName, tableRecord, err.Error())
			}
			if len(pair.key) == 0 {
				return fmt.Errorf("invalid empty key calculated for %v", tableRecord)
			}
			uniqueKeysToDeleteMap[idxName][rowIdsToDeleteCount] = &pair
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
		// TODO: consider splitting it to avoid massive IN() on a partition key that may kick in quorum mechanism
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

type keyRowidPair struct {
	key   string
	rowid int64
}

// To test it, see comments in the end of RunCreateTableRelForBatch
func DeleteDataAndUniqueIndexesByBatchIdx(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.DeleteDataAndUniqueIndexesByBatchIdx")
	defer logger.PopF()

	if !pCtx.CurrentScriptNode.HasTableCreator() {
		logger.InfoCtx(pCtx, "no table creator, nothing to delete for %s", pCtx.Msg.FullBatchId())
		return nil
	}

	keysToDeleteMap := map[string][]*keyRowidPair{} // idx_name -> list_of_key+rowid_pair_to_delete
	for idxName := range pCtx.CurrentScriptNode.TableCreator.Indexes {
		keysToDeleteMap[idxName] = nil
	}
	logger.WarnCtx(pCtx, "deleting data and idx records for %s, %d indexes detected: [%s]", pCtx.Msg.FullBatchId(), len(keysToDeleteMap), strings.Join(slices.Collect(maps.Keys(keysToDeleteMap)), ","))

	deleteStartTime := time.Now()
	totalDataRowsDeleted := 0
	totalIdxRowsDeleted := 0

	// IMPORTANT!
	// Here, we potentially have to select ALL rows (not just rows added for this batch), which may take forever.
	// If we want to select by batch_idx only, we should make it partition key.
	// but in this case, we will not be able to use token(rowid), which we heavily rely on when going through data records (see selectBatchFromTableByToken).
	// And if we add rowid to the partition key to be able to query rows by token(batch_idx,rowid), then we lose the possibility to query just be batch_idx
	// because Cassandra cannot filter by partial partition key.
	// So, for a billion-rows scenarios, resort to the no-rerun policy, and re-run the whole node when needed.

	// retrieve all fields that are involved in building unique indexes, and batch_idx - we will manually filter by it
	idxFieldRefs := pCtx.CurrentScriptNode.GetAllIndexesFieldRefs()
	rs := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(pCtx.CurrentScriptNode.TableCreator.Name)},
		*idxFieldRefs,
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
		for uniqueIdxName := range keysToDeleteMap {
			keysToDeleteMap[uniqueIdxName] = make([]*keyRowidPair, rs.RowCount)
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
			if err := populateUniqueKeysToDeleteMap(keysToDeleteMap,
				pCtx.CurrentScriptNode.TableCreator.Indexes,
				rowIdsToDeleteCount,
				tableRecord,
				rowId); err != nil {
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
			for idxName, idxKeysToDelete := range keysToDeleteMap {
				// Trim unused empty key slots
				trimmedIdxKeysToDelete := idxKeysToDelete[:rowIdsToDeleteCount]

				logger.DebugCtx(pCtx, "deleting %d idx %s records for %s: %v", len(trimmedIdxKeysToDelete), idxName, pCtx.Msg.FullBatchId(), trimmedIdxKeysToDelete)
				if err := deleteIdxRecordByKeyAndRowid(pCtx, idxName, trimmedIdxKeysToDelete); err != nil {
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

	logger.WarnCtx(pCtx, "deleted %d data and %d idx records for %s, elapsed %v", totalDataRowsDeleted, totalIdxRowsDeleted, pCtx.Msg.FullBatchId(), time.Since(deleteStartTime))

	return nil
}
