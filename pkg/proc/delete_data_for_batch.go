package proc

import (
	"fmt"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/evalcapi"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

const HarvestForDeleteRowsetSize = 1000 // Do not let users tweak it, maybe too sensitive

func initRowidsAndKeysToDelete(rowCount int, indexesMap sc.IdxDefMap) ([]int64, map[string][]string) {
	rowIdsToDelete := make([]int64, rowCount)
	uniqueKeysToDeleteMap := map[string][]string{} // unique_idx_name -> list_of_keys_to_delete
	for idxName, idxDef := range indexesMap {
		if idxDef.Uniqueness == sc.IdxUnique {
			uniqueKeysToDeleteMap[idxName] = make([]string, rowCount)
		}
	}
	return rowIdsToDelete, uniqueKeysToDeleteMap
}

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

	logger.WarnCtx(pCtx, "deleting data records for %s...", pCtx.Msg.FullBatchId())

	deleteStartTime := time.Now()

	// Retrieve ALL records from data table (we cannot filter by batch_idx, this is Cassandra),
	// retrieve all fields that are involved in building unique indexes.
	// It may take a while, but there is no other way.
	uniqueIdxFieldRefs := pCtx.CurrentScriptNode.GetUniqueIndexesFieldRefs()
	rs := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(pCtx.CurrentScriptNode.TableCreator.Name)},
		*uniqueIdxFieldRefs,
		sc.FieldRefs{sc.FieldRef{TableName: pCtx.CurrentScriptNode.TableCreator.Name, FieldName: "batch_idx", FieldType: evalcapi.FieldTypeInt}})

	var pageState []byte
	var err error
	// TODO: here, we potentially have to select ALL rows (not just rows added for this batch), which may take forever.
	// If we could attach batch_idx to every data row and filter by it in selectBatchPagedAllRowids, that would help.
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
		rowIdsToDelete, uniqueKeysToDeleteMap := initRowidsAndKeysToDelete(rs.RowCount, pCtx.CurrentScriptNode.TableCreator.Indexes)

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
			logger.InfoCtx(pCtx, "deleting index records for %s from %d indexes before deleting %d data records", pCtx.Msg.FullBatchId(), len(uniqueKeysToDeleteMap), len(rowIdsToDelete))
			for idxName, idxKeysToDelete := range uniqueKeysToDeleteMap {
				// Trim unused empty key slots
				trimmedIdxKeysToDelete := idxKeysToDelete[:rowIdsToDeleteCount]

				// // Show all idx records here
				// qb := cql.QueryBuilder{}
				// q := qb.Keyspace(pCtx.Msg.DataKeyspace).SelectRun(idxName, pCtx.Msg.RunId, []string{"key", "rowid"})
				// it := pCtx.CqlSession.Query(q).Iter()
				// rows, _ := it.SliceMap()
				// logger.Error("found %d in %s: %v", len(rows), idxName, rows)
				// logger.Error("deleting %d from %s: %s", len(trimmedIdxKeysToDelete), idxName, trimmedIdxKeysToDelete)

				logger.DebugCtx(pCtx, "deleting %d idx %s records from %d/%s idx %s for batch_idx %d: '%s'", len(rowIdsToDelete), idxName, pCtx.Msg.RunId, pCtx.Msg.TargetNodeName, idxName, pCtx.Msg.BatchIdx, strings.Join(trimmedIdxKeysToDelete, `','`))
				if err := deleteIdxRecordByKey(pCtx, idxName, trimmedIdxKeysToDelete); err != nil {
					return err
				}
			}

			// Delete data records by rowid
			logger.DebugCtx(pCtx, "deleting %d data records from %s: %v", len(rowIdsToDelete), pCtx.Msg.FullBatchId(), rowIdsToDelete)
			if err := deleteDataRecordByRowid(pCtx, rowIdsToDelete); err != nil {
				return err
			}
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

	logger.DebugCtx(pCtx, "deleted data records for %s, elapsed %v", pCtx.Msg.FullBatchId(), time.Since(deleteStartTime))

	return nil
}
