package proc

import (
	"fmt"
	"strings"
	"time"

	"github.com/kleineshertz/capillaries/pkg/cql"
	"github.com/kleineshertz/capillaries/pkg/ctx"
	"github.com/kleineshertz/capillaries/pkg/l"
	"github.com/kleineshertz/capillaries/pkg/sc"
)

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

func selectBatchFromDataTablePaged(logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	lookupNodeRunId int16,
	batchSize int,
	pageState []byte,
	rowidsToFind map[int64]struct{}) ([]byte, error) {

	logger.PushF("selectBatchFromDataTablePaged")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return nil, err
	}

	rowids := make([]int64, len(rowidsToFind))
	i := 0
	for k := range rowidsToFind {
		rowids[i] = k
		i++
	}

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		CondInInt("rowid", rowids). // This is a right-side lookup table, select by rowid
		SelectRun(tableName, lookupNodeRunId, *rs.GetFieldNames())

	iter := pCtx.CqlSession.Query(q).PageSize(batchSize).PageState(pageState).Iter()

	dbWarnings := iter.Warnings()
	if len(dbWarnings) > 0 {
		logger.WarnCtx(pCtx, strings.Join(dbWarnings, ";"))
	}

	rs.RowCount = 0

	scanner := iter.Scanner()
	for scanner.Next() {
		if rs.RowCount >= len(rs.Rows) {
			return nil, fmt.Errorf("unexpected data row retrieved, exceeding rowset size %d", len(rs.Rows))
		}
		if err := scanner.Scan(*rs.Rows[rs.RowCount]...); err != nil {
			return nil, cql.WrapDbErrorWithQuery("cannot scan paged data row", q, err)
		}
		rs.RowCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, cql.WrapDbErrorWithQuery("data paged scanner error", q, err)
	}

	return iter.PageState(), nil
}

func selectBatchPagedAllRowids(logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	lookupNodeRunId int16,
	batchSize int,
	pageState []byte) ([]byte, error) {

	logger.PushF("selectBatchPagedAllRowids")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return nil, err
	}

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		SelectRun(tableName, lookupNodeRunId, *rs.GetFieldNames())

	iter := pCtx.CqlSession.Query(q).PageSize(batchSize).PageState(pageState).Iter()

	dbWarnings := iter.Warnings()
	if len(dbWarnings) > 0 {
		logger.WarnCtx(pCtx, strings.Join(dbWarnings, ";"))
	}

	rs.RowCount = 0

	scanner := iter.Scanner()
	for scanner.Next() {
		if rs.RowCount >= len(rs.Rows) {
			return nil, fmt.Errorf("unexpected data row retrieved, exceeding rowset size %d", len(rs.Rows))
		}
		if err := scanner.Scan(*rs.Rows[rs.RowCount]...); err != nil {
			return nil, cql.WrapDbErrorWithQuery("cannot scan all rows data row", q, err)
		}
		rs.RowCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, cql.WrapDbErrorWithQuery("data all rows scanner error", q, err)
	}

	return iter.PageState(), nil
}

func selectBatchFromIdxTablePaged(logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	lookupNodeRunId int16,
	batchSize int,
	pageState []byte,
	keysToFind *[]string) ([]byte, error) {

	logger.PushF("selectBatchFromIdxTablePaged")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return nil, err
	}

	qb := cql.QueryBuilder{}
	q := qb.Keyspace(pCtx.BatchInfo.DataKeyspace).
		CondInString("key", *keysToFind). // This is an index table, select only selected keys
		SelectRun(tableName, lookupNodeRunId, *rs.GetFieldNames())

	iter := pCtx.CqlSession.Query(q).PageSize(batchSize).PageState(pageState).Iter()

	dbWarnings := iter.Warnings()
	if len(dbWarnings) > 0 {
		logger.WarnCtx(pCtx, strings.Join(dbWarnings, ";"))
	}

	rs.RowCount = 0

	scanner := iter.Scanner()
	for scanner.Next() {
		if rs.RowCount >= len(rs.Rows) {
			return nil, fmt.Errorf("unexpected idx row retrieved, exceeding rowset size %d", len(rs.Rows))
		}
		if err := scanner.Scan(*rs.Rows[rs.RowCount]...); err != nil {
			return nil, cql.WrapDbErrorWithQuery("cannot scan idx row", q, err)
		}
		rs.RowCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, cql.WrapDbErrorWithQuery("idx scanner error", q, err)
	}

	return iter.PageState(), nil
}

func selectBatchFromTableByToken(logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	rs *Rowset,
	tableName string,
	readerNodeRunId int16,
	batchSize int,
	startToken int64,
	endToken int64) (int64, error) {

	logger.PushF("selectBatchFromTableByToken")
	defer logger.PopF()

	if err := rs.InitRows(batchSize); err != nil {
		return 0, err
	}

	qb := cql.QueryBuilder{}
	q := qb.Keyspace(pCtx.BatchInfo.DataKeyspace).
		Limit(batchSize).
		Cond("token(rowid)", ">=", startToken).
		Cond("token(rowid)", "<=", endToken).
		SelectRun(tableName, readerNodeRunId, *rs.GetFieldNames())

	iter := pCtx.CqlSession.Query(q).Iter()
	dbWarnings := iter.Warnings()
	if len(dbWarnings) > 0 {
		logger.WarnCtx(pCtx, strings.Join(dbWarnings, ";"))
	}
	rs.RowCount = 0
	var lastRetrievedToken int64
	for rs.RowCount < len(rs.Rows) && iter.Scan(*rs.Rows[rs.RowCount]...) {
		lastRetrievedToken = *((*rs.Rows[rs.RowCount])[rs.FieldsByFieldName["token(rowid)"]].(*int64))
		rs.RowCount++
	}
	if err := iter.Close(); err != nil {
		return 0, cql.WrapDbErrorWithQuery("cannot close iterator", q, err)
	}

	return lastRetrievedToken, nil
}

const HarvestForDeleteRowsetSize = 1000 // Do not let users tweak it, maybe too sensitive

func DeleteDataAndUniqueIndexesByBatchIdx(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("DeleteDataAndUniqueIndexesByBatchIdx")
	defer logger.PopF()

	logger.DebugCtx(pCtx, "deleting data records for %s...", pCtx.BatchInfo.FullBatchId())
	deleteStartTime := time.Now()

	if !pCtx.CurrentScriptNode.HasTableCreator() {
		logger.InfoCtx(pCtx, "no table creator, nothing to delete for %s", pCtx.BatchInfo.FullBatchId())
		return nil
	}

	// Select from data table by rowid, retrieve all fields that are involved i building unique indexes
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

		// Harvest rowids with batchIdx we are interested in, also harvest keys

		// Prepare the storage for rowids and keys
		rowIdsToDelete := make([]int64, rs.RowCount)
		uniqueKeysToDeleteMap := map[string][]string{} // unique_idx_name -> list_of_keys_to_delete
		for idxName, idxDef := range pCtx.CurrentScriptNode.TableCreator.Indexes {
			if idxDef.Uniqueness == sc.IdxUnique {
				uniqueKeysToDeleteMap[idxName] = make([]string, rs.RowCount)
			}
		}

		rowIdsToDeleteCount := 0
		for rowIdx := 0; rowIdx < rs.RowCount; rowIdx++ {
			rowId := *((*rs.Rows[rowIdx])[rs.FieldsByFieldName["rowid"]].(*int64))
			batchIdx := int16(*((*rs.Rows[rowIdx])[rs.FieldsByFieldName["batch_idx"]].(*int64)))
			if batchIdx == pCtx.BatchInfo.BatchIdx {
				// Add this rowid to the list
				rowIdsToDelete[rowIdsToDeleteCount] = rowId
				// Build the key and add it to the list
				tableRecord, err := rs.GetTableRecord(rowIdx)
				if err != nil {
					return fmt.Errorf("while deleting previous batch attempt leftovers, cannot get table record from [%v]: %s", rs.Rows[rowIdx], err.Error())
				}
				for idxName, idxDef := range pCtx.CurrentScriptNode.TableCreator.Indexes {
					if _, ok := uniqueKeysToDeleteMap[idxName]; ok {
						uniqueKeysToDeleteMap[idxName][rowIdsToDeleteCount], err = sc.BuildKey(tableRecord, idxDef)
						if err != nil {
							return fmt.Errorf("while deleting previous batch attempt leftovers, cannot build a key for index %s from [%v]: %s", idxName, tableRecord, err.Error())
						}
						if len(uniqueKeysToDeleteMap[idxName][rowIdsToDeleteCount]) == 0 {
							logger.ErrorCtx(pCtx, "invalid empty key calculated for %v", tableRecord)
						}
					}
				}
				rowIdsToDeleteCount++
			}
		}
		if rowIdsToDeleteCount > 0 {
			rowIdsToDelete = rowIdsToDelete[:rowIdsToDeleteCount]
			// NOTE: Assuming Delete won't interfere with paging
			logger.DebugCtx(pCtx, "deleting %d data records from %s: %v", len(rowIdsToDelete), pCtx.BatchInfo.FullBatchId(), rowIdsToDelete)
			qbDel := cql.QueryBuilder{}
			qDel := qbDel.
				Keyspace(pCtx.BatchInfo.DataKeyspace).
				CondInInt("rowid", rowIdsToDelete[:rowIdsToDeleteCount]).
				DeleteRun(pCtx.CurrentScriptNode.TableCreator.Name, pCtx.BatchInfo.RunId)
			if err := pCtx.CqlSession.Query(qDel).Exec(); err != nil {
				return cql.WrapDbErrorWithQuery("cannot delete from data table", qDel, err)
			}
			logger.InfoCtx(pCtx, "deleted %d records from data table for %s, now will delete from %d indexes", len(rowIdsToDelete), pCtx.BatchInfo.FullBatchId(), len(uniqueKeysToDeleteMap))

			for idxName, idxKeysToDelete := range uniqueKeysToDeleteMap {
				logger.DebugCtx(pCtx, "deleting %d idx %s records from %d/%s idx %s for batch_idx %d: %v", len(rowIdsToDelete), idxName, pCtx.BatchInfo.RunId, pCtx.BatchInfo.TargetNodeName, idxName, pCtx.BatchInfo.BatchIdx, idxKeysToDelete)
				qbDel := cql.QueryBuilder{}
				qDel := qbDel.
					Keyspace(pCtx.BatchInfo.DataKeyspace).
					CondInString("key", idxKeysToDelete[:rowIdsToDeleteCount]).
					DeleteRun(idxName, pCtx.BatchInfo.RunId)
				if err := pCtx.CqlSession.Query(qDel).Exec(); err != nil {
					return cql.WrapDbErrorWithQuery("cannot delete from idx table", qDel, err)
				}
				logger.InfoCtx(pCtx, "deleted %d records from idx table %s for batch %d/%s/%d", len(rowIdsToDelete), idxName, pCtx.BatchInfo.RunId, pCtx.BatchInfo.TargetNodeName, pCtx.BatchInfo.BatchIdx)
			}
		}
		if rs.RowCount < pCtx.CurrentScriptNode.TableReader.RowsetSize || len(pageState) == 0 {
			break
		}
	}

	logger.DebugCtx(pCtx, "deleted data records for %s, elapsed %v", pCtx.BatchInfo.FullBatchId(), time.Since(deleteStartTime))

	return nil
}
