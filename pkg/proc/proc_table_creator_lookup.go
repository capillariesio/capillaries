package proc

import (
	"errors"
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/evalcapi"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

const MaxAmazonKeyspacesInElements int = 100

func buildKeysToFindInTheLookupIndex(rsLeft *Rowset, scriptNodeLookup sc.LookupDef) ([]string, map[string][]int, error) {
	// Build keys to find in the lookup index, one key may yield multiple rowids
	keyToLeftRowIdxMap := map[string][]int{}
	for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
		vars := eval.VarValuesMap{}
		if err := rsLeft.ExportToVars(rowIdx, vars); err != nil {
			return nil, nil, err
		}
		key, err := sc.BuildKey(vars[sc.ReaderAlias], scriptNodeLookup.TableCreator.Indexes[scriptNodeLookup.IndexName])
		if err != nil {
			return nil, nil, err
		}

		_, ok := keyToLeftRowIdxMap[key]
		if !ok {
			keyToLeftRowIdxMap[key] = make([]int, 0)
		}
		keyToLeftRowIdxMap[key] = append(keyToLeftRowIdxMap[key], rowIdx)
	}

	keysToFind := make([]string, len(keyToLeftRowIdxMap))
	i := 0
	for k := range keyToLeftRowIdxMap {
		keysToFind[i] = k
		i++
	}

	return keysToFind, keyToLeftRowIdxMap, nil
}

func getRightRowidsToFind(rsIdx *Rowset) (map[int64]struct{}, map[int64]string) {
	// Build a map of right-row-id -> key
	rightRowIdToKeyMap := map[int64]string{}
	rowidsToFind := map[int64]struct{}{}
	for rowIdx := 0; rowIdx < rsIdx.RowCount; rowIdx++ {
		rightRowId := *((*rsIdx.Rows[rowIdx])[rsIdx.FieldsByFieldName["rowid"]].(*int64))
		key := *((*rsIdx.Rows[rowIdx])[rsIdx.FieldsByFieldName["key"]].(*string))
		rightRowIdToKeyMap[rightRowId] = key
		rowidsToFind[rightRowId] = struct{}{}
	}
	return rowidsToFind, rightRowIdToKeyMap
}

func setupEvalCtxForGroup(node *sc.ScriptNodeDef, rsLeft *Rowset) (map[int64]map[string]*eval.EvalCtx, error) {
	eCtxMap := map[int64]map[string]*eval.EvalCtx{}
	if node.Lookup.IsGroup {
		for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
			rowid := *((*rsLeft.Rows[rowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
			eCtxMap[rowid] = map[string]*eval.EvalCtx{}
			for fieldName, fieldDef := range node.TableCreator.Fields {
				// Expression may contain an agg function and may not. Handle both. No var values available yet.
				aggFuncEnabled, aggFuncType, aggFuncArgs := eval.DetectRootAggFunc(fieldDef.ParsedExpression)
				var newCtx *eval.EvalCtx
				var newCtxErr error
				if aggFuncEnabled == eval.AggFuncEnabled {
					newCtx, newCtxErr = eval.NewAggEvalCtx(aggFuncType, aggFuncArgs, evalcapi.CapillariesEvalFunctions, evalcapi.CapillariesEvalConstants, nil)
					if newCtxErr != nil {
						return nil, fmt.Errorf("cannot initialize ctx for group calc: %s", newCtxErr.Error())
					}
					newCtx.SetRoundDec(2) // decimal2
				} else {
					newCtx = eval.NewPlainEvalCtx(evalcapi.CapillariesEvalFunctions, evalcapi.CapillariesEvalConstants, nil)
				}
				eCtxMap[rowid][fieldName] = newCtx
			}
		}
	}
	return eCtxMap, nil
}

func evalRowGroupedFields(writerFieldDefs map[string]*sc.WriteTableFieldDef, rsLeft *Rowset, leftRowIdx int, rsRight *Rowset, rightRowIdx int, eCtxMap map[int64]map[string]*eval.EvalCtx) error {
	leftRowid := *((*rsLeft.Rows[leftRowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
	for fieldName, fieldDef := range writerFieldDefs {
		vars := eval.VarValuesMap{}
		if err := rsLeft.ExportToVars(leftRowIdx, vars); err != nil {
			return err
		}
		if err := rsRight.ExportToVarsWithAlias(rightRowIdx, vars, sc.LookupAlias); err != nil {
			return err
		}
		eCtxMap[leftRowid][fieldName].SetVars(vars)
		_, err := eCtxMap[leftRowid][fieldName].Eval(fieldDef.ParsedExpression)
		if err != nil {
			return fmt.Errorf("cannot evaluate target expression [%s]: [%s]", fieldDef.RawExpression, err.Error())
		}
	}
	return nil
}

func checkLookupFilter(lookupDef *sc.LookupDef, rsRight *Rowset, rightRowIdx int) (bool, error) {
	lookupFilterOk := true
	if lookupDef.UsesFilter() {
		vars := eval.VarValuesMap{}
		if err := rsRight.ExportToVars(rightRowIdx, vars); err != nil {
			return false, err
		}
		var err error
		lookupFilterOk, err = lookupDef.CheckFilterCondition(vars)
		if err != nil {
			return false, fmt.Errorf("cannot check filter condition [%s] against [%v]: [%s]", lookupDef.RawFilter, vars, err.Error())
		}
	}
	return lookupFilterOk, nil
}

/*
func saveCompletedBatch(pCtx *ctx.MessageProcessingContext, logger *l.CapiLogger, tableCreator *sc.TableCreatorDef, tableRecordBatchCount int, batchStartTime time.Time, instr *TableInserter) (int, time.Time, error) {
	if err := instr.waitForWorkers(logger, pCtx); err != nil {
		return 0, time.Now(), fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, tableCreator.Name, err.Error())
	}
	reportWriteTable(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(tableCreator.Indexes), instr.NumWorkers)
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return 0, time.Now(), err
	}
	return 0, time.Now(), nil
}
*/

func produceGroupedTableRecord(node *sc.ScriptNodeDef, rsLeft *Rowset, leftRowIdx int, leftRowFoundRightLookup []bool, eCtxMap map[int64]map[string]*eval.EvalCtx) (map[string]any, error) {

	tableRecord := map[string]any{}

	if !leftRowFoundRightLookup[leftRowIdx] {
		if node.Lookup.LookupJoin == sc.LookupJoinInner {
			// Grouped inner join with no data on the right
			// Do not insert this left row
			return nil, nil
		}
		// Grouped left outer join with no data on the right
		leftVars := eval.VarValuesMap{}
		if err := rsLeft.ExportToVars(leftRowIdx, leftVars); err != nil {
			return nil, err
		}

		var err error
		for fieldName, fieldDef := range node.TableCreator.Fields {
			isAggEnabled, _, _ := eval.DetectRootAggFunc(fieldDef.ParsedExpression)
			if isAggEnabled == eval.AggFuncEnabled {
				// Aggregate func is used in field expression - ignore the expression and produce default
				tableRecord[fieldName], err = node.TableCreator.GetFieldDefaultReadyForDb(fieldName)
				if err != nil {
					return nil, fmt.Errorf("cannot initialize default field %s: [%s]", fieldName, err.Error())
				}
			} else {
				// No aggregate function used in field expression - assume it contains only left-side fields
				tableRecord[fieldName], err = sc.CalculateFieldValue(fieldName, fieldDef, leftVars)
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		// Grouped inner or left outer with present data on the right
		leftRowid := *((*rsLeft.Rows[leftRowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
		for fieldName, fieldDef := range node.TableCreator.Fields {
			// WARNING: this can be considered a Capillaries shortcoming:
			// what if there are no rows to aggregate? SQL/CQL would return nil, but Capillaries cannot.
			// So we have to use default value. Or should we make it configurable?
			finalValue := eCtxMap[leftRowid][fieldName].GetSafeValue(sc.GetDefaultFieldTypeValue(fieldDef.Type))

			if err := sc.CheckValueType(finalValue, fieldDef.Type); err != nil {
				return nil, fmt.Errorf("invalid field %s type: [%s]", fieldName, err.Error())
			}
			tableRecord[fieldName] = finalValue
		}
	}
	return tableRecord, nil
}

func produceNonGroupedTableRecordForLeftWithChildren(node *sc.ScriptNodeDef, rsLeft *Rowset, leftRowIdx int, rsRight *Rowset, rightRowIdx int) (map[string]any, error) {
	vars := eval.VarValuesMap{}
	if err := rsLeft.ExportToVars(leftRowIdx, vars); err != nil {
		return nil, err
	}
	if err := rsRight.ExportToVarsWithAlias(rightRowIdx, vars, sc.LookupAlias); err != nil {
		return nil, err
	}

	// We are ready to write this result right away, so prepare the output tableRecord
	tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(vars)
	if err != nil {
		return nil, fmt.Errorf("cannot populate table record from [%v]: [%s]", vars, err.Error())
	}
	return tableRecord, nil
}

func produceNonGroupedTableRecordForCheldlessLeft(node *sc.ScriptNodeDef, rsLeft *Rowset, leftRowIdx int) (map[string]any, error) {
	tableRecord := map[string]any{}

	leftVars := eval.VarValuesMap{}
	if err := rsLeft.ExportToVars(leftRowIdx, leftVars); err != nil {
		return nil, err
	}

	var err error
	for fieldName, fieldDef := range node.TableCreator.Fields {
		if fieldDef.UsedFields.HasFieldsWithTableAlias(sc.LookupAlias) {
			// This field expression uses fields from lkp table - produce default value
			tableRecord[fieldName], err = node.TableCreator.GetFieldDefaultReadyForDb(fieldName)
			if err != nil {
				return nil, fmt.Errorf("cannot initialize non-grouped default field %s: [%s]", fieldName, err.Error())
			}
		} else {
			// This field expression does not use fields from lkp table - assume the expression contains only left-side fields
			tableRecord[fieldName], err = sc.CalculateFieldValue(fieldName, fieldDef, leftVars)
			if err != nil {
				return nil, err
			}
		}
	}
	return tableRecord, nil
}

func checkHavingAddRecordAndSaveBatchIfNeeded(logger *l.CapiLogger, node *sc.ScriptNodeDef, tableRecord map[string]any, indexKeyMap map[string]string, instr *TableInserter) error {
	logger.PushF("proc.checkHavingAddRecordAndSaveBatchIfNeeded")
	defer logger.PopF()

	rowsWritten := 0
	// Check table creator having
	inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
	if err != nil {
		return fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
	}

	if inResult {
		err = instr.buildIndexKeys(tableRecord, indexKeyMap)
		if err != nil {
			return fmt.Errorf("cannot build index keys for %s: [%s]", node.TableCreator.Name, err.Error())
		}
		instr.add(tableRecord, indexKeyMap)
		rowsWritten++
	}

	return nil
}

func checkRunCreateTableRelForBatchSanity(node *sc.ScriptNodeDef, readerNodeRunId int16, lookupNodeRunId int16) error {
	if readerNodeRunId == 0 {
		return errors.New("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	if lookupNodeRunId == 0 {
		return errors.New("this node has a dependency node to lookup data at that was never started in this keyspace (lookupNodeRunId == 0)")
	}

	if !node.HasTableReader() {
		return errors.New("node does not have table reader")
	}
	if !node.HasTableCreator() {
		return errors.New("node does not have table creator")
	}
	if !node.HasLookup() {
		return errors.New("node does not have lookup")
	}
	return nil
}

func splitKeysIntoChunks(allKeys []string, chunkSize int) [][]string {
	chunkCount := len(allKeys) / chunkSize
	if len(allKeys)%chunkSize > 0 {
		chunkCount++
	}

	keysChunks := make([][]string, chunkCount)
	keyIdx := 0
	for chunkIdx := range keysChunks {
		keysChunks[chunkIdx] = make([]string, 0)
		for {
			keysChunks[chunkIdx] = append(keysChunks[chunkIdx], allKeys[keyIdx])
			keyIdx++
			if keyIdx == len(allKeys) || keyIdx%MaxAmazonKeyspacesBatchLen == 0 {
				break
			}
		}
	}
	return keysChunks
}

func runCreateTableRelForBatch(envConfig *env.EnvConfig,
	logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	lookupNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.runCreateTableRelForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	totalStartTime := time.Now()

	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName + cql.RunIdSuffix(readerNodeRunId), Dst: node.TableCreator.Name + cql.RunIdSuffix(readerNodeRunId)}

	if err := checkRunCreateTableRelForBatchSanity(node, readerNodeRunId, lookupNodeRunId); err != nil {
		return bs, err
	}

	// Fields to read from source table
	srcLeftFieldRefs := sc.FieldRefs{}
	srcLeftFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, sc.ReaderAlias)
	srcLeftFieldRefs.Append(node.Lookup.LeftTableFields)

	srcRightFieldRefs := sc.FieldRefs{}
	srcRightFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, sc.LookupAlias)
	if node.Lookup.UsesFilter() {
		srcRightFieldRefs.AppendWithFilter(node.Lookup.UsedInFilterFields, sc.LookupAlias)
	}

	leftBatchSize := node.TableReader.RowsetSize

	rsLeft := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(node.TableReader.TableName)},
		sc.FieldRefs{sc.RowidTokenFieldRef()},
		srcLeftFieldRefs)

	instr, err := createInserterAndStartWorkers(logger, envConfig, pCtx, &node.TableCreator, DataIdxSeqModeDataFirst, logger.ZapMachine.String)
	if err != nil {
		return bs, err
	}
	instr.startDrainer()
	defer instr.closeInserter(logger, pCtx)

	curStartLeftToken := startLeftToken
	leftPageIdx := 0
	var curStartLeftTokenRowIds []int64
	for {
		selectLeftBatchByTokenStartTime := time.Now()
		lastRetrievedLeftToken, endTokenRowIds, err := selectBatchFromTableByToken(logger,
			pCtx,
			rsLeft,
			node.TableReader.TableName,
			readerNodeRunId,
			leftBatchSize,
			curStartLeftToken,
			endLeftToken,
			curStartLeftTokenRowIds)
		if err != nil {
			instr.cancelDrainer(fmt.Errorf("cannot select batch from source table, node %s: %s", node.Name, err.Error()))
			return bs, instr.waitForDrainer()
		}

		logger.DebugCtx(pCtx, "selectBatchFromTableByToken: leftPageIdx %d, queried tokens from %d to %d in %.3fs, retrieved %d rows", leftPageIdx, curStartLeftToken, endLeftToken, time.Since(selectLeftBatchByTokenStartTime).Seconds(), rsLeft.RowCount)

		// If token(rowid) guaranteed uniqueness, we would just "curStartLeftToken = lastRetrievedLeftToken + 1"
		// But duplicates are possible, so we have to be prepared to handle token overlaps
		// (rows with same token but different rowids returned in separate selectBatchFromTableByToken calls)
		// See overlap/epilogue logic in selectBatchFromTableByToken.
		curStartLeftToken = lastRetrievedLeftToken
		curStartLeftTokenRowIds = endTokenRowIds

		if rsLeft.RowCount == 0 {
			break
		}

		// Setup eval ctx for each target field if grouping is involved
		// map: rowid -> field -> ctx
		eCtxMap, err := setupEvalCtxForGroup(node, rsLeft)
		if err != nil {
			instr.cancelDrainer(fmt.Errorf("cannot setup eval ctx, node %s: %s", node.Name, err.Error()))
			return bs, instr.waitForDrainer()
		}

		// Array that says if a left row has any right counterparts
		leftRowFoundRightLookup := make([]bool, rsLeft.RowCount)
		for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
			leftRowFoundRightLookup[rowIdx] = false
		}

		// Build keys to find in the lookup index, one key may yield multiple rowids
		allKeysToFind, keyToLeftRowIdxMap, err := buildKeysToFindInTheLookupIndex(rsLeft, node.Lookup)
		if err != nil {
			instr.cancelDrainer(fmt.Errorf("cannot build keys for the left-side rowset, node %s: %s", node.Name, err.Error()))
			return bs, instr.waitForDrainer()
		}

		lookupFieldRefs := sc.FieldRefs{}
		lookupFieldRefs.AppendWithFilter(node.TableCreator.UsedInHavingFields, node.Lookup.TableCreator.Name)
		lookupFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, node.Lookup.TableCreator.Name)

		// Select from idx table by keys
		rsIdx := NewRowsetFromFieldRefs(
			sc.FieldRefs{sc.RowidFieldRef(node.Lookup.IndexName)},
			sc.FieldRefs{sc.KeyTokenFieldRef()},
			sc.FieldRefs{sc.IdxKeyFieldRef()})

		keysToFindChunks := splitKeysIntoChunks(allKeysToFind, MaxAmazonKeyspacesBatchLen)
		for _, keysToFind := range keysToFindChunks {
			var idxPageState []byte
			rightIdxPageIdx := 0
			for {
				selectIdxBatchStartTime := time.Now()
				idxPageState, err = selectBatchFromIdxTablePaged(logger,
					pCtx,
					rsIdx,
					node.Lookup.IndexName,
					lookupNodeRunId,
					node.Lookup.IdxReadBatchSize,
					idxPageState,
					&keysToFind)
				if err != nil {
					instr.cancelDrainer(fmt.Errorf("cannot select batch from idx table, node %s: %s", node.Name, err.Error()))
					return bs, instr.waitForDrainer()
				}

				if rsIdx.RowCount == 0 {
					break
				}

				// Build a map of right-row-id -> key
				rightRowidsToFind, rightRowIdToKeyMap := getRightRowidsToFind(rsIdx)

				logger.DebugCtx(pCtx, "selectBatchFromIdxTablePaged: leftPageIdx %d, rightIdxPageIdx %d, queried %d keys in %.3fs, retrieved %d right rowids", leftPageIdx, rightIdxPageIdx, len(allKeysToFind), time.Since(selectIdxBatchStartTime).Seconds(), len(rightRowidsToFind))

				keyToFindRowIdsMap := map[int64]struct{}{}

				// Select from right table by rowid
				rsRight := NewRowsetFromFieldRefs(
					sc.FieldRefs{sc.RowidFieldRef(node.Lookup.TableCreator.Name)},
					sc.FieldRefs{sc.RowidTokenFieldRef()},
					srcRightFieldRefs)

				rightDataAttemptIdx := 0
				for {
					// We will keep resetting page state because we will keep shrinking rightRowidsToFind
					// Let's keep uisng paging in case there are too many ids to retireve
					var rightPageState []byte
					selectBatchStartTime := time.Now()
					_, err = selectBatchFromDataTablePaged(logger,
						pCtx,
						rsRight,
						node.Lookup.TableCreator.Name,
						lookupNodeRunId,
						node.Lookup.RightLookupReadBatchSize,
						rightPageState,
						getFirstIntsFromSet(rightRowidsToFind, MaxAmazonKeyspacesInElements)) // Amazon Keyspaces allows max 100 IN elements
					if err != nil {
						instr.cancelDrainer(fmt.Errorf("cannot select batch from right-side table, node %s: %s", node.Name, err.Error()))
						return bs, instr.waitForDrainer()
					}

					logger.DebugCtx(pCtx, "selectBatchFromDataTablePaged: leftPageIdx %d, rightIdxPageIdx %d, rightDataAttemptIdx %d, queried %d rowids in %.3fs, retrieved %d rowids", leftPageIdx, rightIdxPageIdx, rightDataAttemptIdx, len(rightRowidsToFind), time.Since(selectBatchStartTime).Seconds(), rsRight.RowCount)

					if rsRight.RowCount == 0 {
						break
					}

					// Help GC
					var indexKeyMap = map[string]string{}
					var tableRecord map[string]any
					for rightRowIdx := 0; rightRowIdx < rsRight.RowCount; rightRowIdx++ {
						rightRowId := *((*rsRight.Rows[rightRowIdx])[rsRight.FieldsByFieldName["rowid"]].(*int64))
						rightRowKey := rightRowIdToKeyMap[rightRowId]

						if _, ok := keyToFindRowIdsMap[rightRowId]; ok {
							logger.DebugCtx(pCtx, "selectBatchFromDataTablePaged: isKeyInQuestionInKeysToFind got rowid in data %d", rightRowId)
						}

						// Remove this right rowid from the set, we do not need it anymore.
						delete(rightRowidsToFind, rightRowId)

						// Check filter condition if needed
						lookupFilterOk, err := checkLookupFilter(&node.Lookup, rsRight, rightRowIdx)
						if err != nil {
							instr.cancelDrainer(fmt.Errorf("cannot check lookup filter, node %s: %s", node.Name, err.Error()))
							return bs, instr.waitForDrainer()
						}

						if !lookupFilterOk {
							// Skip this right row
							continue
						}

						if node.Lookup.IsGroup {
							// Find correspondent row from rsLeft, merge left and right and
							// call group eval eCtxMap[leftRowid] for each output field,
							// but do not write them yet - there may be more
							for _, leftRowIdx := range keyToLeftRowIdxMap[rightRowKey] {

								leftRowFoundRightLookup[leftRowIdx] = true
								if err := evalRowGroupedFields(node.TableCreator.Fields, rsLeft, leftRowIdx, rsRight, rightRowIdx, eCtxMap); err != nil {
									instr.cancelDrainer(fmt.Errorf("cannot eval grouped fields, node %s: %s", node.Name, err.Error()))
									return bs, instr.waitForDrainer()
								}
							}
						} else {
							// Non-group, and the right row was found for the parent left row.
							// Find correspondent row from rsLeft, merge left and right and call row-level eval
							for _, leftRowIdx := range keyToLeftRowIdxMap[rightRowKey] {

								leftRowFoundRightLookup[leftRowIdx] = true

								tableRecord, err = produceNonGroupedTableRecordForLeftWithChildren(node, rsLeft, leftRowIdx, rsRight, rightRowIdx)
								if err != nil {
									instr.cancelDrainer(fmt.Errorf("cannot produceNonGroupedTableRecordForLeftWithChildren, node %s: %s", node.Name, err.Error()))
									return bs, instr.waitForDrainer()
								}

								if err = checkHavingAddRecordAndSaveBatchIfNeeded(logger, node, tableRecord, indexKeyMap, instr); err != nil {
									instr.cancelDrainer(fmt.Errorf("cannot checkHavingAddRecordAndSaveBatchIfNeeded, node %s: %s", node.Name, err.Error()))
									return bs, instr.waitForDrainer()
								}
								bs.RowsWritten++

							} // non-group result row written
						} // group case handled
					} // for each found right row

					// No more ids in the IN condition, we are done retrieving right-side rowids
					if len(rightRowidsToFind) == 0 {
						break
					}

					rightDataAttemptIdx++
					instr.PCtx.SendHeartbeat() // Hopefully, calling heartbeat this often is enough
				} // for each data page

				// For Cassandra, we can rely on rsIdx.RowCount. But for Amazon Keyspaces, gocql returns only a fraction of records page after page, until page state is empty
				// if rsIdx.RowCount < node.Lookup.IdxReadBatchSize || len(idxPageState) == 0 {
				if len(idxPageState) == 0 {
					break
				}
				rightIdxPageIdx++
			} // for each idx page
		} // for each 100-key chunk

		// For grouped - group
		// For non-grouped left join - add empty left-side (those who have right counterpart were alredy hendled above)
		// Non-grouped inner join - already handled above
		if node.Lookup.IsGroup {
			// Help GC
			var indexKeyMap = map[string]string{}
			var tableRecord map[string]any
			// Time to write the result of the grouped we evaluated above using eCtxMap
			for leftRowIdx := 0; leftRowIdx < rsLeft.RowCount; leftRowIdx++ {
				tableRecord, err = produceGroupedTableRecord(node, rsLeft, leftRowIdx, leftRowFoundRightLookup, eCtxMap)
				if err != nil {
					instr.cancelDrainer(fmt.Errorf("cannot produceGroupedTableRecord, node %s: %s", node.Name, err.Error()))
					return bs, instr.waitForDrainer()
				}
				if tableRecord == nil {
					// No record generated, it's ok (inner join and no right rows)
					continue
				}

				if err = checkHavingAddRecordAndSaveBatchIfNeeded(logger, node, tableRecord, indexKeyMap, instr); err != nil {
					instr.cancelDrainer(fmt.Errorf("cannot Group checkHavingAddRecordAndSaveBatchIfNeeded, node %s: %s", node.Name, err.Error()))
					return bs, instr.waitForDrainer()
				}
				bs.RowsWritten++
			}
		} else if node.Lookup.LookupJoin == sc.LookupJoinLeft {

			// Non-grouped left outer join.
			// Handle those left rows that did not have right lookup counterpart
			// (those who had - they have been written already)

			// Help GC
			var indexKeyMap = map[string]string{}
			var tableRecord map[string]any
			for leftRowIdx := 0; leftRowIdx < rsLeft.RowCount; leftRowIdx++ {
				if leftRowFoundRightLookup[leftRowIdx] {
					// This left row had right counterparts and grouped result was already written
					continue
				}

				tableRecord, err = produceNonGroupedTableRecordForCheldlessLeft(node, rsLeft, leftRowIdx)
				if err != nil {
					instr.cancelDrainer(fmt.Errorf("cannot JoinLeft produceNonGroupedTableRecordForCheldlessLeft, node %s: %s", node.Name, err.Error()))
					return bs, instr.waitForDrainer()
				}

				if err = checkHavingAddRecordAndSaveBatchIfNeeded(logger, node, tableRecord, indexKeyMap, instr); err != nil {
					instr.cancelDrainer(fmt.Errorf("cannot JoinLeft checkHavingAddRecordAndSaveBatchIfNeeded, node %s: %s", node.Name, err.Error()))
					return bs, instr.waitForDrainer()
				}
				bs.RowsWritten++
			}
		}

		bs.RowsRead += rsLeft.RowCount

		// We are tempted to "if rs.RowCount < srcBatchSize break", here but do not do that:
		// because of the rowid overlapping/epilogue logic, selectBatchFromTableByToken returns less rows than rs capacity

		leftPageIdx++
		// instr.PCtx.SendHeartbeat() - this may be not enough, processing may take longer, send heartbeats inside
	} // for each source table batch

	instr.doneSending()
	if err := instr.waitForDrainer(); err != nil {
		return bs, err
	}

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	// TEST ONLY
	// To test DeleteDataAndUniqueIndexesByBatchIdx:
	// uncomment the exit()
	// start the daemon
	// run lookup_quicktest
	// wait for the daemon to finish
	// comment the exit()
	// start the daemon
	// in the log, watch for DeleteDataAndUniqueIndexesByBatchIdx messagesbatchStartTime
	// make sure lookup_quicktest completed successfully and result data is good
	// os.Exit(0)

	return bs, nil
}
