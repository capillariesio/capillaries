package proc

import (
	"fmt"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/gocql/gocql"
)

const MaxAmazonKeyspacesBatchLen int = 30

// func ClearNodeOutputs(logger *l.Logger, script *sc.ScriptDef, session *gocql.Session, keyspace string, nodeName string, runId int16) error {
// 	node, ok := script.ScriptNodes[nodeName]
// 	if !ok {
// 		return fmt.Errorf("cannot find node %s", nodeName)
// 	}
// 	if node.HasTableCreator() {
// 		qb := cql.QueryBuilder{}
// 		logger.Info("deleting data table %s.%s...", keyspace, node.TableCreator.Name)
// 		query := qb.Keyspace(keyspace).DropRun(node.TableCreator.Name, runId)
// 		if err := session.Query(query).Exec(); err != nil {
// 			return fmt.Errorf("cannot drop data table [%s]: [%s]", query, err.Error())
// 		}

// 		for idxName, _ := range node.TableCreator.Indexes {
// 			qb := cql.QueryBuilder{}
// 			logger.Info("deleting index table %s.%s...", keyspace, idxName)
// 			query := qb.Keyspace(keyspace).DropRun(idxName, runId)
// 			if err := session.Query(query).Exec(); err != nil {
// 				return fmt.Errorf("cannot drop idx table [%s]: [%s]", query, err.Error())
// 			}
// 		}
// 	} else if node.HasFileCreator() {
// 		if _, err := os.Stat(node.FileCreator.Url); err == nil {
// 			logger.Info("deleting output file %s...", node.FileCreator.Url)
// 			if err := os.Remove(node.FileCreator.Url); err != nil {
// 				return fmt.Errorf("cannot delete file [%s]: [%s]", node.FileCreator.Url, err.Error())
// 			}
// 		}
// 	}
// 	return nil
// }

func getFirstIntsFromSet(intSet map[int64]struct{}, cnt int) []int64 {
	intSliceLen := cnt
	if intSliceLen > len(intSet) {
		intSliceLen = len(intSet)
	}
	intSlice := make([]int64, intSliceLen)
	i := 0
	for v := range intSet {
		intSlice[i] = v
		i++
		if i == intSliceLen {
			break
		}
	}
	return intSlice
}

func selectBatchFromDataTablePaged(logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	lookupNodeRunId int16,
	batchSize int,
	pageState []byte,
	rowids []int64) ([]byte, error) {

	logger.PushF("proc.selectBatchFromDataTablePaged")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return nil, err
	}

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		CondInPrepared("rowid"). // This is a right-side lookup table, select by rowid
		SelectRun(tableName, lookupNodeRunId, *rs.GetFieldNames())

	var iter *gocql.Iter
	selectRetryIdx := 0
	curSelectExpBackoffFactor := 1
	var nextPageState []byte
	for {
		iter = pCtx.CqlSession.Query(q, rowids).PageSize(batchSize).PageState(pageState).Iter()
		nextPageState = iter.PageState()

		dbWarnings := iter.Warnings()
		if len(dbWarnings) > 0 {
			// TODO: figure out what those warnigs can be, never saw one
			logger.WarnCtx(pCtx, "got warnigs while selecting %d rows from %s%s: %s", batchSize, tableName, cql.RunIdSuffix(lookupNodeRunId), strings.Join(dbWarnings, ";"))
		}

		rs.RowCount = 0

		scanner := iter.Scanner()
		for scanner.Next() {
			if rs.RowCount >= len(rs.Rows) {
				return nil, fmt.Errorf("unexpected data row retrieved, exceeding rowset size %d", len(rs.Rows))
			}
			if err := scanner.Scan(*rs.Rows[rs.RowCount]...); err != nil {
				return nil, db.WrapDbErrorWithQuery("cannot scan paged data row", q, err)
			}
			// We assume gocql creates only UTC timestamps, so this is not needed.
			// If we ever catch a ts stored in our tables with a non-UTC tz, or gocql returning a non-UTC tz - investigate it. Sanitizing is the last resort and should be avoided.
			// if err := rs.SanitizeScannedDatetimesToUtc(rs.RowCount); err != nil {
			// 	return nil, db.WrapDbErrorWithQuery("cannot sanitize datetimes", q, err)
			// }
			rs.RowCount++
		}

		err := scanner.Err()
		if err == nil {
			break
		}
		if !(strings.Contains(err.Error(), "Operation timed out") || strings.Contains(err.Error(), "Cannot achieve consistency level") && selectRetryIdx < 3) {
			return nil, db.WrapDbErrorWithQuery(fmt.Sprintf("paged data scanner cannot select %d rows from %s%s after %d attempts; another worker may retry this batch later, but, if some unique idx records has been written already by current worker, the next worker handling this batch will throw an error on them and there is nothing we can do about it;", batchSize, tableName, cql.RunIdSuffix(lookupNodeRunId), selectRetryIdx+1), q, err)
		}
		logger.WarnCtx(pCtx, "cannot select %d rows from %s%s on retry %d, getting timeout/consistency error (%s), will wait for %dms and retry", batchSize, tableName, cql.RunIdSuffix(lookupNodeRunId), selectRetryIdx, err.Error(), 10*curSelectExpBackoffFactor)
		time.Sleep(time.Duration(10*curSelectExpBackoffFactor) * time.Millisecond)
		curSelectExpBackoffFactor *= 2
		selectRetryIdx++
	}

	return nextPageState, nil
}

func selectBatchPagedAllRowids(logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	lookupNodeRunId int16,
	batchSize int,
	pageState []byte) ([]byte, error) {

	logger.PushF("proc.selectBatchPagedAllRowids")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return nil, err
	}

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		SelectRun(tableName, lookupNodeRunId, *rs.GetFieldNames())

	iter := pCtx.CqlSession.Query(q).PageSize(batchSize).PageState(pageState).Iter()
	nextPageState := iter.PageState()

	dbWarnings := iter.Warnings()
	if len(dbWarnings) > 0 {
		logger.WarnCtx(pCtx, "%s", strings.Join(dbWarnings, ";"))
	}

	rs.RowCount = 0

	scanner := iter.Scanner()
	for scanner.Next() {
		if rs.RowCount >= len(rs.Rows) {
			return nil, fmt.Errorf("unexpected data row retrieved, exceeding rowset size %d", len(rs.Rows))
		}
		if err := scanner.Scan(*rs.Rows[rs.RowCount]...); err != nil {
			return nil, db.WrapDbErrorWithQuery("cannot scan all rows data row", q, err)
		}
		// We assume gocql creates only UTC timestamps, so this is not needed
		// If we ever catch a ts stored in our tables with a non-UTC tz, or gocql returning a non-UTC tz - investigate it. Sanitizing is the last resort and should be avoided.
		// if err := rs.SanitizeScannedDatetimesToUtc(rs.RowCount); err != nil {
		// 	return nil, db.WrapDbErrorWithQuery("cannot sanitize datetimes", q, err)
		// }
		rs.RowCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, db.WrapDbErrorWithQuery("data all rows scanner error", q, err)
	}

	return nextPageState, nil
}

func selectBatchFromIdxTablePaged(logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	lookupNodeRunId int16,
	batchSize int,
	pageState []byte,
	keysToFind *[]string) ([]byte, error) {

	logger.PushF("proc.selectBatchFromIdxTablePaged")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return nil, err
	}

	qb := cql.QueryBuilder{}
	q := qb.Keyspace(pCtx.BatchInfo.DataKeyspace).
		CondInPrepared("key"). // This is an index table, select only selected keys
		SelectRun(tableName, lookupNodeRunId, *rs.GetFieldNames())

	iter := pCtx.CqlSession.Query(q, *keysToFind).PageSize(batchSize).PageState(pageState).Iter()
	nextPageState := iter.PageState()

	dbWarnings := iter.Warnings()
	if len(dbWarnings) > 0 {
		logger.WarnCtx(pCtx, "%s", strings.Join(dbWarnings, ";"))
	}

	rs.RowCount = 0

	scanner := iter.Scanner()
	for scanner.Next() {
		if rs.RowCount >= len(rs.Rows) {
			return nil, fmt.Errorf("unexpected idx row retrieved, exceeding rowset size %d", len(rs.Rows))
		}
		if err := scanner.Scan(*rs.Rows[rs.RowCount]...); err != nil {
			return nil, db.WrapDbErrorWithQuery("cannot scan idx row", q, err)
		}
		rs.RowCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, db.WrapDbErrorWithQuery("idx scanner error", q, err)
	}

	return nextPageState, nil
}

func selectBatchFromTableByToken(logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	readerNodeRunId int16,
	batchSize int,
	startToken int64,
	endToken int64) (int64, error) {

	logger.PushF("proc.selectBatchFromTableByToken")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return 0, err
	}

	qb := cql.QueryBuilder{}
	q := qb.Keyspace(pCtx.BatchInfo.DataKeyspace).
		Limit(batchSize).
		CondPrepared("token(rowid)", ">=").
		CondPrepared("token(rowid)", "<=").
		SelectRun(tableName, readerNodeRunId, *rs.GetFieldNames())

	// TODO: consider retries as we do in selectBatchFromDataTablePaged(); although no timeouts were detected so far here

	iter := pCtx.CqlSession.Query(q, startToken, endToken).Iter()

	dbWarnings := iter.Warnings()
	if len(dbWarnings) > 0 {
		logger.WarnCtx(pCtx, "%s", strings.Join(dbWarnings, ";"))
	}
	rs.RowCount = 0
	var lastRetrievedToken int64
	for rs.RowCount < len(rs.Rows) && iter.Scan(*rs.Rows[rs.RowCount]...) {
		lastRetrievedToken = *((*rs.Rows[rs.RowCount])[rs.FieldsByFieldName["token(rowid)"]].(*int64))
		rs.RowCount++
	}
	if err := iter.Close(); err != nil {
		return 0, db.WrapDbErrorWithQuery("cannot close iterator", q, err)
	}

	return lastRetrievedToken, nil
}

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
					Keyspace(pCtx.BatchInfo.DataKeyspace).
					Cond("rowid", "=", rowid).
					DeleteRun(pCtx.CurrentScriptNode.TableCreator.Name, pCtx.BatchInfo.RunId))
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
			Keyspace(pCtx.BatchInfo.DataKeyspace).
			CondInInt("rowid", rowids).
			DeleteRun(pCtx.CurrentScriptNode.TableCreator.Name, pCtx.BatchInfo.RunId)
		if err := pCtx.CqlSession.Query(q).Exec(); err != nil {
			return db.WrapDbErrorWithQuery("cannot delete from data table", q, err)
		}
	}
	return nil
}

func deleteIdxRecordByKey(pCtx *ctx.MessageProcessingContext, idxName string, keys []string) error {
	if pCtx.CassandraEngine == db.CassandraEngineAmazonKeyspaces {
		// Amazon Keyspaces supports unlogged batch commands with up to 30 commands in the batch
		sb := strings.Builder{}
		for i, key := range keys {
			sb.WriteString(
				(&cql.QueryBuilder{}).
					Keyspace(pCtx.BatchInfo.DataKeyspace).
					Cond("key", "=", key).
					DeleteRun(pCtx.CurrentScriptNode.TableCreator.Name, pCtx.BatchInfo.RunId))
			sb.WriteString(";")
			if (i+1)%MaxAmazonKeyspacesBatchLen == 0 || i == len(key)-1 {
				batchStmt := "BEGIN UNLOGGED BATCH " + sb.String() + " APPLY BATCH"
				if err := pCtx.CqlSession.Query(batchStmt).Exec(); err != nil {
					return db.WrapDbErrorWithQuery("cannot delete from idx table", batchStmt, err)
				}
				sb.Reset()
			}
		}
	} else {
		q := (&cql.QueryBuilder{}).
			Keyspace(pCtx.BatchInfo.DataKeyspace).
			CondInString("key", keys).
			DeleteRun(idxName, pCtx.BatchInfo.RunId)
		if err := pCtx.CqlSession.Query(q).Exec(); err != nil {
			return db.WrapDbErrorWithQuery("cannot delete from idx table", q, err)
		}
	}
	return nil
}

const HarvestForDeleteRowsetSize = 1000 // Do not let users tweak it, maybe too sensitive

// To test it, see comments in the end of RunCreateTableRelForBatch
func DeleteDataAndUniqueIndexesByBatchIdx(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.DeleteDataAndUniqueIndexesByBatchIdx")
	defer logger.PopF()

	if !pCtx.CurrentScriptNode.HasTableCreator() {
		logger.InfoCtx(pCtx, "no table creator, nothing to delete for %s", pCtx.BatchInfo.FullBatchId())
		return nil
	}

	logger.DebugCtx(pCtx, "deleting data records for %s...", pCtx.BatchInfo.FullBatchId())

	deleteStartTime := time.Now()

	// Retrieve ALL records from data table (we cannot filter by batch_idx, this is Cassandra),
	// retrieve all fields that are involved in building unique indexes.
	// It may take a while, but there is no other way.
	uniqueIdxFieldRefs := pCtx.CurrentScriptNode.GetUniqueIndexesFieldRefs()
	rs := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(pCtx.CurrentScriptNode.TableCreator.Name)},
		*uniqueIdxFieldRefs,
		sc.FieldRefs{sc.FieldRef{TableName: pCtx.CurrentScriptNode.TableCreator.Name, FieldName: "batch_idx", FieldType: sc.FieldTypeInt}})

	var pageState []byte
	var err error
	for {
		pageState, err = selectBatchPagedAllRowids(logger,
			pCtx,
			rs,
			pCtx.CurrentScriptNode.TableCreator.Name,
			pCtx.BatchInfo.RunId,
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
			if batchIdx != pCtx.BatchInfo.BatchIdx {
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

			// Delete data records by rowid
			logger.DebugCtx(pCtx, "deleting %d data records from %s: %v", len(rowIdsToDelete), pCtx.BatchInfo.FullBatchId(), rowIdsToDelete)
			if err := deleteDataRecordByRowid(pCtx, rowIdsToDelete); err != nil {
				return err
			}

			// Delete index records by key
			logger.InfoCtx(pCtx, "deleted %d records from data table for %s, now will delete from %d indexes", len(rowIdsToDelete), pCtx.BatchInfo.FullBatchId(), len(uniqueKeysToDeleteMap))
			for idxName, idxKeysToDelete := range uniqueKeysToDeleteMap {
				// Trim unused empty key slots
				trimmedIdxKeysToDelete := idxKeysToDelete[:rowIdsToDeleteCount]
				logger.DebugCtx(pCtx, "deleting %d idx %s records from %d/%s idx %s for batch_idx %d: '%s'", len(rowIdsToDelete), idxName, pCtx.BatchInfo.RunId, pCtx.BatchInfo.TargetNodeName, idxName, pCtx.BatchInfo.BatchIdx, strings.Join(trimmedIdxKeysToDelete, `','`))
				if err := deleteIdxRecordByKey(pCtx, idxName, trimmedIdxKeysToDelete); err != nil {
					return err
				}
			}

			// TODO: assuming Delete won't interfere with paging used above;
			// do we need to reset the pageState? After all, we have deleted some records from that table.
			// On the other hand, if we reset it, we will have to walk through thousands of rows that do not belong to this batch, again.
		}

		// Amazon Keyspaces: do not rely on the retrieved row count, use pagestate
		// if rs.RowCount < pCtx.CurrentScriptNode.TableReader.RowsetSize || len(pageState) == 0 {
		if len(pageState) == 0 {
			break
		}
	}

	logger.DebugCtx(pCtx, "deleted data records for %s, elapsed %v", pCtx.BatchInfo.FullBatchId(), time.Since(deleteStartTime))

	return nil
}
