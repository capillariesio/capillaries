package proc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/evalcapi"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

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

// processLookupsForKeys performs the lookup for a single left-side page (rsLeft).
//
// It replaces the former "split keys into chunks + IN(...) paged selects" approach with two nested,
// size-limited goroutine pools:
//
//   - An outer pool (envConfig.Daemon.LookupKeyWorkers, default 20) whose workers each take one key
//     from allKeysToFind and page through ALL key/rowid records for that key in the idx table
//     (partition key "key" - potentially millions of rowids).
//   - For every rowid found, an inner pool per key worker (envConfig.Daemon.LookupRowidWorkers,
//     default 10) retrieves the single data-table record by rowid (rowid is unique -> zero or one
//     row) and handles it exactly like the sequential implementation did: lookup filter, then the
//     grouped (IsGroup) vs non-grouped logic.
//
// Shared state is protected as follows:
//   - eCtxMap and leftRowFoundRightLookup for a given key are only touched under that key's keyMu.
//     Since every left row maps to exactly one key (see buildKeysToFindInTheLookupIndex), different
//     keys operate on disjoint left rows, so a per-key mutex fully isolates them.
//   - instr.add (and the rowsWritten counter) are serialized by inserterMu, because TableInserter's
//     RecordsSent bookkeeping and RecordsIn channel send are not safe for concurrent producers.
//
// It returns the number of non-grouped rows written during the lookup (grouped and childless
// left-join rows are written by the caller afterwards, single-threaded).
func processLookupsForKeys(
	envConfig *env.EnvConfig,
	logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	node *sc.ScriptNodeDef,
	lookupNodeRunId int16,
	srcRightFieldRefs sc.FieldRefs,
	allKeysToFind []string,
	keyToLeftRowIdxMap map[string][]int,
	rsLeft *Rowset,
	leftRowFoundRightLookup []bool,
	eCtxMap map[int64]map[string]*eval.EvalCtx,
	instr *TableInserter) (int64, error) {

	logger.PushF("proc.processLookupsForKeys")
	defer logger.PopF()

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var rowsWritten int64

	// processRightDataRow handles one data-table row (0 or 1 per rowid), reusing the exact same
	// grouped vs non-grouped logic as the sequential implementation.
	processRightDataRow := func(workerLogger *l.CapiLogger, rsRight *Rowset, leftRowIdxs []int) error {
		workerLogger.PushF("proc.runRelLookupForLeftPageParallel.processRightDataRow")
		defer workerLogger.PopF()

		// if pCtx.Msg.BatchIdx == 0 {
		// 	return errors.New("processRightDataRow fake error 2")
		// }

		rightRowIdx := 0

		lookupFilterOk, err := checkLookupFilter(&node.Lookup, rsRight, rightRowIdx)
		if err != nil {
			return fmt.Errorf("cannot check lookup filter, node %s: %s", node.Name, err.Error())
		}
		if !lookupFilterOk {
			// Skip this right row
			return nil
		}

		if node.Lookup.IsGroup {
			// Find correspondent rows from rsLeft, merge left and right and call group eval
			// eCtxMap[leftRowid] for each output field, but do not write them yet - there may be more.
			for _, leftRowIdx := range leftRowIdxs {
				leftRowFoundRightLookup[leftRowIdx] = true
				if err := evalRowGroupedFields(node.TableCreator.Fields, rsLeft, leftRowIdx, rsRight, rightRowIdx, eCtxMap); err != nil {
					return fmt.Errorf("cannot eval grouped fields, node %s: %s", node.Name, err.Error())
				}
			}
			return nil
		}

		// Non-group, and the right row was found for the parent left row(s).
		// Find correspondent rows from rsLeft, merge left and right and call row-level eval.
		for _, leftRowIdx := range leftRowIdxs {
			leftRowFoundRightLookup[leftRowIdx] = true

			tableRecord, err := produceNonGroupedTableRecordForLeftWithChildren(node, rsLeft, leftRowIdx, rsRight, rightRowIdx)
			if err != nil {
				return fmt.Errorf("cannot produceNonGroupedTableRecordForLeftWithChildren, node %s: %s", node.Name, err.Error())
			}

			// Help GC
			indexKeyMap := map[string]string{}
			err = checkHavingAddRecordAndSaveBatchIfNeeded(workerLogger, node, tableRecord, indexKeyMap, instr)
			if err == nil {
				rowsWritten++
			}
			if err != nil {
				return fmt.Errorf("cannot checkHavingAddRecordAndSaveBatchIfNeeded, node %s: %s", node.Name, err.Error())
			}
		}
		return nil
	}

	// processKey pages through all rowids of a single key and dispatches them to a per-key inner pool.
	processKey := func(workerLogger *l.CapiLogger, rsIdx *Rowset, key string) error {
		workerLogger.PushF("proc.runRelLookupForLeftPageParallel.processKey")
		defer workerLogger.PopF()

		leftRowIdxs := keyToLeftRowIdxMap[key]
		var processRightDataRowMux sync.Mutex // Serializes this key's eCtxMap/leftRowFoundRightLookup access

		var firstSelectRightRowErr error
		var firstSelectRightRowErrMux sync.Mutex

		// Inner pool: retrieve data rows by rowid in parallel
		rowidCh := make(chan int64)
		var rowidWg sync.WaitGroup
		for r := 0; r < envConfig.Daemon.LookupRowidWorkers; r++ {
			rowidWg.Add(1)

			// Own data rowset, statically sized to a single row (rowid is unique), reused across rowids
			rsSingleRight := NewRowsetFromFieldRefs(
				sc.FieldRefs{sc.RowidFieldRef(node.Lookup.TableCreator.Name)},
				srcRightFieldRefs)

			if err := rsSingleRight.InitRows(1); err != nil {
				firstSelectRightRowErr = err
				break
			}

			go func(logger *l.CapiLogger, rs *Rowset) {
				defer rowidWg.Done()
				logger.PushF(fmt.Sprintf("proc.runRelLookupForLeftPageParallel.rowidWorker_%0d", r))
				defer logger.Close()

				for rowid := range rowidCh {
					if firstSelectRightRowErr != nil {
						break
					}
					if err := selectDataRowByRowid(logger, pCtx, rs, node.Lookup.TableCreator.Name, lookupNodeRunId, rowid); err != nil {
						firstSelectRightRowErrMux.Lock()
						if firstSelectRightRowErr == nil {
							firstSelectRightRowErr = fmt.Errorf("cannot select data row by key/rowid %s/%d, node %s: %s", key, rowid, node.Name, err.Error())
						}
						firstSelectRightRowErrMux.Unlock()
					} else {
						if rs.RowCount > 0 {
							processRightDataRowMux.Lock()
							err = processRightDataRow(logger, rs, leftRowIdxs)
							processRightDataRowMux.Unlock()
							if err != nil {
								firstSelectRightRowErrMux.Lock()
								if firstSelectRightRowErr == nil {
									firstSelectRightRowErr = fmt.Errorf("cannot process right data row %s/%d, node %s: %s", key, rowid, node.Name, err.Error())
								}
								firstSelectRightRowErrMux.Unlock()
							}
						}
					}
				}
			}(l.NewLoggerFromLogger(workerLogger), rsSingleRight)
		}

		var pageState []byte
		var selectRowidsByKeyError error
		for {
			var err error
			pageState, err = selectRowidsFromIdxTablePagedByKey(workerLogger, pCtx, rsIdx, node.Lookup.IndexName, lookupNodeRunId, node.Lookup.IdxReadBatchSize, pageState, key)
			if err != nil {
				selectRowidsByKeyError = fmt.Errorf("cannot select idx rowids by key, node %s: %s", node.Name, err.Error())
				break
			}

			for i := 0; i < rsIdx.RowCount; i++ {
				rowid := *((*rsIdx.Rows[i])[rsIdx.FieldsByFieldName["rowid"]].(*int64))
				rowidCh <- rowid
			}

			// This function is thread-safe
			pCtx.SendHeartbeat()

			// For Cassandra we could rely on rsIdx.RowCount, but for Amazon Keyspaces gocql returns
			// only a fraction of records page after page until page state is empty.
			if len(pageState) == 0 {
				break
			}
		}

		// Page through all key/rowid records for this key and feed the inner pool.
		close(rowidCh)
		rowidWg.Wait()

		if selectRowidsByKeyError != nil {
			return selectRowidsByKeyError
		}

		if firstSelectRightRowErr != nil {
			return firstSelectRightRowErr
		}

		return nil
	}

	// Feed keys to the outer key-worker pool.
	keysCh := make(chan string)
	go func() {
		defer close(keysCh)
		for _, key := range allKeysToFind {
			select {
			case keysCh <- key:
			case <-cancelCtx.Done():
				return
			}
		}
	}()

	var firstHandleKeyErr error
	var firstHandleKeyErrMux sync.Mutex
	var keyWorkersWg sync.WaitGroup
	for w := 0; w < envConfig.Daemon.LookupKeyWorkers; w++ {
		keyWorkersWg.Add(1)
		go func(logger *l.CapiLogger) {
			defer keyWorkersWg.Done()

			logger.PushF(fmt.Sprintf("proc.runRelLookupForLeftPageParallel.keyWorker_%0d", w))
			defer logger.Close()

			// Own idx rowset (rowid only), reused across the keys this worker handles and their pages
			rsIdx := NewRowsetFromFieldRefs(sc.FieldRefs{sc.RowidFieldRef(node.Lookup.IndexName)})

			for key := range keysCh {
				if err := processKey(logger, rsIdx, key); err != nil {
					firstHandleKeyErrMux.Lock()
					if firstHandleKeyErr == nil {
						firstHandleKeyErr = err
					}
					firstHandleKeyErrMux.Unlock()
					break
				}
			}
		}(l.NewLoggerFromLogger(logger))
	}

	keyWorkersWg.Wait()

	return rowsWritten, firstHandleKeyErr
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

		// Process all lookup keys for this left page using nested, size-limited goroutine pools:
		// an outer pool per key (paging all rowids from the idx table by partition key "key"), and
		// an inner pool per key retrieving each data-table record by rowid. See the function doc for
		// the concurrency model. Non-grouped rows are written inside; grouped and childless left-join
		// rows are written by the single-threaded epilogue below.
		lookupStartTime := time.Now()
		rowsWrittenInLookup, err := processLookupsForKeys(envConfig,
			logger,
			pCtx,
			node,
			lookupNodeRunId,
			srcRightFieldRefs,
			allKeysToFind,
			keyToLeftRowIdxMap,
			rsLeft,
			leftRowFoundRightLookup,
			eCtxMap,
			instr)
		if err != nil {
			instr.cancelDrainer(fmt.Errorf("cannot run parallel rel lookup, node %s: %s", node.Name, err.Error()))
			return bs, instr.waitForDrainer()
		}
		bs.RowsWritten += int(rowsWrittenInLookup)

		logger.DebugCtx(pCtx, "runRelLookupForLeftPageParallel: leftPageIdx %d, processed %d keys in %.3fs, wrote %d non-grouped rows", leftPageIdx, len(allKeysToFind), time.Since(lookupStartTime).Seconds(), rowsWrittenInLookup)

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

	bs.UpdateElapsedStats(time.Since(totalStartTime), instr)
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
