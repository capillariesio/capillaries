package proc

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
)

type TableRecord map[string]interface{}
type TableRecordPtr *map[string]interface{}
type TableRecordBatch []TableRecordPtr

const DefaultInserterBatchSize int = 5000

func reportWriteTable(logger *l.Logger, pCtx *ctx.MessageProcessingContext, recordCount int, dur time.Duration, indexCount int, workerCount int) {
	logger.InfoCtx(pCtx, "WriteTable: %d items in %.3fs (%.0f items/s, %d indexes, eff rate %.0f writes/s), %d workers",
		recordCount,
		dur.Seconds(),
		float64(recordCount)/dur.Seconds(),
		indexCount,
		float64(recordCount*(indexCount+1))/dur.Seconds(),
		workerCount)
}

func reportWriteTableLeftovers(logger *l.Logger, pCtx *ctx.MessageProcessingContext, recordCount int, dur time.Duration, indexCount int, workerCount int) {
	logger.InfoCtx(pCtx, "WriteTableLeftovers: %d items in %.3fs (%.0f items/s, %d indexes, eff rate %.0f writes/s), %d workers",
		recordCount,
		dur.Seconds(),
		float64(recordCount)/dur.Seconds(),
		indexCount,
		float64(recordCount*(indexCount+1))/dur.Seconds(),
		workerCount)
}

func reportWriteTableComplete(logger *l.Logger, pCtx *ctx.MessageProcessingContext, readCount int, recordCount int, dur time.Duration, indexCount int, workerCount int) {
	logger.InfoCtx(pCtx, "WriteTableComplete: read %d, wrote %d items in %.3fs (%.0f items/s, %d indexes, eff rate %.0f writes/s), %d workers",
		readCount,
		recordCount,
		dur.Seconds(),
		float64(recordCount)/dur.Seconds(),
		indexCount,
		float64(recordCount*(indexCount+1))/dur.Seconds(),
		workerCount)
}

func RunReadFileForBatch(envConfig *env.EnvConfig, logger *l.Logger, pCtx *ctx.MessageProcessingContext, srcFileIdx int) (BatchStats, error) {
	logger.PushF("proc.RunReadFileForBatch")
	defer logger.PopF()

	batchStartTime := time.Now()
	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0}

	node := pCtx.CurrentScriptNode

	if !node.HasFileReader() {
		return bs, fmt.Errorf("node does not have file reader")
	}
	if !node.HasTableCreator() {
		return bs, fmt.Errorf("node does not have table creator")
	}

	if srcFileIdx < 0 || srcFileIdx >= len(node.FileReader.SrcFileUrls) {
		return bs, fmt.Errorf("cannot find file to read: asked to read src file with index %d while there are only %d source files available", srcFileIdx, len(node.FileReader.SrcFileUrls))
	}
	filePath := node.FileReader.SrcFileUrls[srcFileIdx]

	u, err := url.Parse(filePath)
	if err != nil {
		return bs, fmt.Errorf("cannot parse file uri %s: %s", filePath, err.Error())
	}

	bs.Src = filePath
	bs.Dst = node.TableCreator.Name

	var csvFile *os.File
	var fileReader io.Reader
	if u.Scheme == xfer.UriSchemeFile || len(u.Scheme) == 0 {
		csvFile, err = os.Open(filePath)
		if err != nil {
			return bs, err
		}
		fileReader = bufio.NewReader(csvFile)
	} else if u.Scheme == xfer.UriSchemeHttp || u.Scheme == xfer.UriSchemeHttps {
		readCloser, err := xfer.GetHttpReadCloser(filePath, u.Scheme, envConfig.CaPath)
		if err != nil {
			return bs, err
		}
		fileReader = readCloser
		defer readCloser.Close()
	} else if u.Scheme == xfer.UriSchemeSftp {
		// When dealing with sftp, we download the *whole* file, instead of providing a reader
		dstFile, err := os.CreateTemp("", "capi")
		if err != nil {
			return bs, fmt.Errorf("cannot creeate temp file for %s: %s", filePath, err.Error())
		}

		// Download and schedule delete
		if err = xfer.DownloadSftpFile(filePath, envConfig.PrivateKeys, dstFile); err != nil {
			dstFile.Close()
			return bs, err
		}
		logger.Info("downloaded sftp file %s to %s", filePath, dstFile.Name())
		dstFile.Close()
		defer os.Remove(dstFile.Name())

		// Create a reader for the temp file
		csvFile, err = os.Open(dstFile.Name())
		if err != nil {
			return bs, fmt.Errorf("cannot read from file %s downloaded from %s: %s", dstFile.Name(), filePath, err.Error())
		}
		fileReader = bufio.NewReader(csvFile)
	} else {
		return bs, fmt.Errorf("uri scheme %s not supported: %s", u.Scheme, filePath)
	}

	r := csv.NewReader(fileReader)
	r.Comma = rune(node.FileReader.Separator[0])

	// To avoid bare \" error: https://stackoverflow.com/questions/31326659/golang-csv-error-bare-in-non-quoted-field
	r.LazyQuotes = true

	var lineIdx int64 = 0
	tableRecordBatchCount := 0

	instr := newTableInserter(envConfig, logger, pCtx, &node.TableCreator, DefaultInserterBatchSize)
	instr.verifyTablesExist()
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndClose(logger, pCtx)

	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return bs, fmt.Errorf("cannot read file [%s]: [%s]", filePath, err.Error())
		}
		if node.FileReader.ColumnIndexingMode == sc.FileColumnIndexingName && int64(node.FileReader.SrcFileHdrLineIdx) == lineIdx {
			if err := node.FileReader.ResolveColumnIndexesFromNames(line); err != nil {
				return bs, fmt.Errorf("cannot parse column headers of [%s]: [%s]", filePath, err.Error())
			}
		} else if lineIdx >= int64(node.FileReader.SrcFileFirstDataLineIdx) {

			// FileReader: read columns
			colVars := eval.VarValuesMap{}
			if err := node.FileReader.ReadLineToValuesMap(&line, colVars); err != nil {
				return bs, fmt.Errorf("cannot read values from [%s], line %d: [%s]", filePath, lineIdx, err.Error())
			}

			// TableCreator: evaluate table column expressions
			tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(false, colVars)
			if err != nil {
				return bs, fmt.Errorf("cannot populate table record from [%s], line %d: [%s]", filePath, lineIdx, err.Error())
			}

			// Check table creator having
			inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
			if err != nil {
				return bs, fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
			}

			// Write batch if needed
			if inResult {
				instr.add(tableRecord)
				tableRecordBatchCount++
				if tableRecordBatchCount == DefaultInserterBatchSize {
					if err := instr.waitForWorkers(logger, pCtx); err != nil {
						return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
					}
					reportWriteTable(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)
					batchStartTime = time.Now()
					tableRecordBatchCount = 0
					if err := instr.startWorkers(logger, pCtx); err != nil {
						return bs, err
					}

				}
				bs.RowsWritten++
			}
			bs.RowsRead++
		}
		lineIdx++
	}

	// Write leftovers regardless of tableRecordBatchCount == 0
	if err := instr.waitForWorkers(logger, pCtx); err != nil {
		return bs, fmt.Errorf("cannot save leftover record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
	}
	reportWriteTableLeftovers(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}

func RunCreateTableForCustomProcessorForBatch(envConfig *env.EnvConfig,
	logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.RunCreateTableForCustomProcessorForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName, Dst: node.TableCreator.Name}

	if readerNodeRunId == 0 {
		return bs, fmt.Errorf("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	if !node.HasTableReader() {
		return bs, fmt.Errorf("node does not have table reader")
	}
	if !node.HasTableCreator() {
		return bs, fmt.Errorf("node does not have table creator")
	}

	// Fields to read from source table
	srcLeftFieldRefs := sc.FieldRefs{}
	srcLeftFieldRefs.AppendWithFilter(*node.CustomProcessor.GetUsedInTargetExpressionsFields(), sc.ReaderAlias)
	srcLeftFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, sc.ReaderAlias)

	leftBatchSize := node.TableReader.RowsetSize
	curStartLeftToken := startLeftToken

	rsIn := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(node.TableReader.TableName)},
		sc.FieldRefs{sc.RowidTokenFieldRef()},
		srcLeftFieldRefs)

	inserterBatchSize := DefaultInserterBatchSize
	if inserterBatchSize < node.TableReader.RowsetSize {
		inserterBatchSize = node.TableReader.RowsetSize
	}
	instr := newTableInserter(envConfig, logger, pCtx, &node.TableCreator, inserterBatchSize)
	instr.verifyTablesExist()
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndClose(logger, pCtx)

	flushVarsArray := func(varsArray []*eval.VarValuesMap, varsArrayCount int) error {
		logger.PushF("proc.flushRowset")
		defer logger.PopF()

		flushStartTime := time.Now()
		rowsWritten := 0

		for outRowIdx := 0; outRowIdx < varsArrayCount; outRowIdx++ {
			vars := varsArray[outRowIdx]

			tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(false, *vars)
			if err != nil {
				return fmt.Errorf("cannot populate table record from [%v]: [%s]", vars, err.Error())
			}

			// Check table creator having
			inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
			if err != nil {
				return fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
			}

			// Write batch if needed
			if inResult {
				instr.add(tableRecord)
				rowsWritten++
				bs.RowsWritten++
			}
		}

		reportWriteTable(logger, pCtx, rowsWritten, time.Since(flushStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)
		flushStartTime = time.Now()
		rowsWritten = 0

		return nil
	}

	for {
		lastRetrievedLeftToken, err := selectBatchFromTableByToken(logger,
			pCtx,
			rsIn,
			node.TableReader.TableName,
			readerNodeRunId,
			leftBatchSize,
			curStartLeftToken,
			endLeftToken)
		if err != nil {
			return bs, err
		}
		curStartLeftToken = lastRetrievedLeftToken + 1

		if rsIn.RowCount == 0 {
			break
		}
		customProcBatchStartTime := time.Now()

		if err = node.CustomProcessor.(CustomProcessorRunner).Run(logger, pCtx, rsIn, flushVarsArray); err != nil {
			return bs, err
		}

		custProcDur := time.Since(customProcBatchStartTime)
		logger.InfoCtx(pCtx, "CustomProcessor: %d items in %v (%.0f items/s)", rsIn.RowCount, custProcDur, float64(rsIn.RowCount)/custProcDur.Seconds())

		bs.RowsRead += rsIn.RowCount
		if rsIn.RowCount < leftBatchSize {
			break
		}
	} // for each source table batch

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}

func RunCreateTableForBatch(envConfig *env.EnvConfig,
	logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.RunCreateTableForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	batchStartTime := time.Now()
	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName, Dst: node.TableCreator.Name}

	if readerNodeRunId == 0 {
		return bs, fmt.Errorf("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	if !node.HasTableReader() {
		return bs, fmt.Errorf("node does not have table reader")
	}
	if !node.HasTableCreator() {
		return bs, fmt.Errorf("node does not have table creator")
	}

	// Fields to read from source table
	srcLeftFieldRefs := sc.FieldRefs{}
	srcLeftFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, sc.ReaderAlias)

	leftBatchSize := node.TableReader.RowsetSize
	tableRecordBatchCount := 0
	curStartLeftToken := startLeftToken

	rsIn := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(node.TableReader.TableName)},
		sc.FieldRefs{sc.RowidTokenFieldRef()},
		srcLeftFieldRefs)

	inserterBatchSize := DefaultInserterBatchSize
	if inserterBatchSize < node.TableReader.RowsetSize {
		inserterBatchSize = node.TableReader.RowsetSize
	}
	instr := newTableInserter(envConfig, logger, pCtx, &node.TableCreator, inserterBatchSize)
	instr.verifyTablesExist()
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndClose(logger, pCtx)

	for {
		lastRetrievedLeftToken, err := selectBatchFromTableByToken(logger,
			pCtx,
			rsIn,
			node.TableReader.TableName,
			readerNodeRunId,
			leftBatchSize,
			curStartLeftToken,
			endLeftToken)
		if err != nil {
			return bs, err
		}
		curStartLeftToken = lastRetrievedLeftToken + 1

		if rsIn.RowCount == 0 {
			break
		}

		// Save rsIn
		for outRowIdx := 0; outRowIdx < rsIn.RowCount; outRowIdx++ {
			vars := eval.VarValuesMap{}
			if err := rsIn.ExportToVars(outRowIdx, &vars); err != nil {
				return bs, err
			}

			tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(false, vars)
			if err != nil {
				return bs, fmt.Errorf("cannot populate table record from [%v]: [%s]", vars, err.Error())
			}

			// Check table creator having
			inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
			if err != nil {
				return bs, fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
			}

			// Write batch if needed
			if inResult {
				instr.add(tableRecord)
				tableRecordBatchCount++
				if tableRecordBatchCount == DefaultInserterBatchSize {
					if err := instr.waitForWorkers(logger, pCtx); err != nil {
						return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
					}
					reportWriteTable(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)
					batchStartTime = time.Now()
					tableRecordBatchCount = 0
					if err := instr.startWorkers(logger, pCtx); err != nil {
						return bs, err
					}
				}
				bs.RowsWritten++
			}
		}

		bs.RowsRead += rsIn.RowCount
		if rsIn.RowCount < leftBatchSize {
			break
		}
	} // for each source table batch

	// Write leftovers regardless of tableRecordBatchCount == 0
	if err := instr.waitForWorkers(logger, pCtx); err != nil {
		return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
	}
	reportWriteTableLeftovers(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}

func RunCreateTableRelForBatch(envConfig *env.EnvConfig,
	logger *l.Logger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	lookupNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.RunCreateTableRelForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	batchStartTime := time.Now()
	totalStartTime := time.Now()

	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName, Dst: node.TableCreator.Name}

	if readerNodeRunId == 0 {
		return bs, fmt.Errorf("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	if lookupNodeRunId == 0 {
		return bs, fmt.Errorf("this node has a dependency node to lookup data at that was never started in this keyspace (lookupNodeRunId == 0)")
	}

	if !node.HasTableReader() {
		return bs, fmt.Errorf("node does not have table reader")
	}
	if !node.HasTableCreator() {
		return bs, fmt.Errorf("node does not have table creator")
	}
	if !node.HasLookup() {
		return bs, fmt.Errorf("node does not have lookup")
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

	// For left outer joins
	// rsEmptyRight := NewRowsetFromFieldRefs(
	// 	sc.FieldRefs{sc.RowidFieldRef(node.Lookup.TableCreator.Name)},
	// 	srcRightFieldRefs)
	// if err := rsEmptyRight.InitRowsWithTableCreatorDefaults(1, node.TableCreator.Fields); err != nil {
	// 	return err
	// }

	leftBatchSize := node.TableReader.RowsetSize
	tableRecordBatchCount := 0

	rsLeft := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(node.TableReader.TableName)},
		sc.FieldRefs{sc.RowidTokenFieldRef()},
		srcLeftFieldRefs)

	inserterBatchSize := DefaultInserterBatchSize
	if inserterBatchSize < node.TableReader.RowsetSize {
		inserterBatchSize = node.TableReader.RowsetSize
	}
	instr := newTableInserter(envConfig, logger, pCtx, &node.TableCreator, inserterBatchSize)
	instr.verifyTablesExist()
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndClose(logger, pCtx)

	curStartLeftToken := startLeftToken
	for {
		lastRetrievedLeftToken, err := selectBatchFromTableByToken(logger,
			pCtx,
			rsLeft,
			node.TableReader.TableName,
			readerNodeRunId,
			leftBatchSize,
			curStartLeftToken,
			endLeftToken)
		if err != nil {
			return bs, err
		}
		curStartLeftToken = lastRetrievedLeftToken + 1

		if rsLeft.RowCount == 0 {
			break
		}

		// Setup eval ctx for each target field if grouping is involved
		// map: rowid -> field -> ctx
		eCtxMap := map[int64]map[string]*eval.EvalCtx{}
		if node.HasLookup() && node.Lookup.IsGroup {
			for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
				rowid := *((*rsLeft.Rows[rowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
				eCtxMap[rowid] = map[string]*eval.EvalCtx{}
				for fieldName, _ := range node.TableCreator.Fields {
					newCtx := eval.NewPlainEvalCtx(eval.AggFuncEnabled)
					eCtxMap[rowid][fieldName] = &newCtx
				}
			}
		}

		// Build keys to find in the lookup index, one key may yield multiple rowids
		keyToLeftRowIdxMap := map[string][]int{}
		leftRowFoundRightLookup := make([]bool, rsLeft.RowCount)
		for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
			leftRowFoundRightLookup[rowIdx] = false
			vars := eval.VarValuesMap{}
			if err := rsLeft.ExportToVars(rowIdx, &vars); err != nil {
				return bs, err
			}
			key, err := sc.BuildKey(vars[sc.ReaderAlias], node.Lookup.TableCreator.Indexes[node.Lookup.IndexName])
			if err != nil {
				return bs, err
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

		lookupFieldRefs := sc.FieldRefs{}
		lookupFieldRefs.AppendWithFilter(node.TableCreator.UsedInHavingFields, node.Lookup.TableCreator.Name)
		lookupFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, node.Lookup.TableCreator.Name)

		rsIdx := NewRowsetFromFieldRefs(
			sc.FieldRefs{sc.RowidFieldRef(node.Lookup.IndexName)},
			sc.FieldRefs{sc.KeyTokenFieldRef()},
			sc.FieldRefs{sc.IdxKeyFieldRef()})

		var idxPageState []byte
		for {
			idxPageState, err = selectBatchFromIdxTablePaged(logger,
				pCtx,
				rsIdx,
				node.Lookup.IndexName,
				lookupNodeRunId,
				node.Lookup.IdxReadBatchSize,
				idxPageState,
				&keysToFind)
			if err != nil {
				return bs, err
			}

			if rsIdx.RowCount == 0 {
				break
			}

			// Build a map of right-row-id -> key
			rightRowIdToKeyMap := map[int64]string{}
			for rowIdx := 0; rowIdx < rsIdx.RowCount; rowIdx++ {
				rightRowId := *((*rsIdx.Rows[rowIdx])[rsIdx.FieldsByFieldName["rowid"]].(*int64))
				key := *((*rsIdx.Rows[rowIdx])[rsIdx.FieldsByFieldName["key"]].(*string))
				rightRowIdToKeyMap[rightRowId] = key
			}

			rowidsToFind := make(map[int64]struct{}, len(rightRowIdToKeyMap))
			for k := range rightRowIdToKeyMap {
				rowidsToFind[k] = struct{}{}
			}

			// Select from right table by rowid
			rsRight := NewRowsetFromFieldRefs(
				sc.FieldRefs{sc.RowidFieldRef(node.Lookup.TableCreator.Name)},
				sc.FieldRefs{sc.RowidTokenFieldRef()},
				srcRightFieldRefs)

			var rightPageState []byte
			for {
				rightPageState, err = selectBatchFromDataTablePaged(logger,
					pCtx,
					rsRight,
					node.Lookup.TableCreator.Name,
					lookupNodeRunId,
					node.Lookup.RightLookupReadBatchSize,
					rightPageState,
					rowidsToFind)
				if err != nil {
					return bs, err
				}

				if rsRight.RowCount == 0 {
					break
				}

				for rightRowIdx := 0; rightRowIdx < rsRight.RowCount; rightRowIdx++ {
					rightRowId := *((*rsRight.Rows[rightRowIdx])[rsRight.FieldsByFieldName["rowid"]].(*int64))
					rightRowKey := rightRowIdToKeyMap[rightRowId]

					// Remove this right rowid from the set, we do not need it anymore. Reset page state.
					rightPageState = nil
					delete(rowidsToFind, rightRowId)

					// Check filter condition if needed
					lookupFilterOk := true
					if node.Lookup.UsesFilter() {
						vars := eval.VarValuesMap{}
						if err := rsRight.ExportToVars(rightRowIdx, &vars); err != nil {
							return bs, err
						}
						var err error
						lookupFilterOk, err = node.Lookup.CheckFilterCondition(vars)
						if err != nil {
							return bs, fmt.Errorf("cannot check filter condition [%s] against [%v]: [%s]", node.Lookup.RawFilter, vars, err.Error())
						}
					}

					if !lookupFilterOk {
						continue
					}

					if node.Lookup.IsGroup {
						// Find correspondent row from rsLeft, merge left and right and
						// call group eval eCtxMap[leftRowid] for each output field
						for _, leftRowIdx := range keyToLeftRowIdxMap[rightRowKey] {

							leftRowFoundRightLookup[leftRowIdx] = true

							leftRowid := *((*rsLeft.Rows[leftRowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
							for fieldName, fieldDef := range node.TableCreator.Fields {
								eCtxMap[leftRowid][fieldName].Vars = &eval.VarValuesMap{}
								if err := rsLeft.ExportToVars(leftRowIdx, eCtxMap[leftRowid][fieldName].Vars); err != nil {
									return bs, err
								}
								if err := rsRight.ExportToVarsWithAlias(rightRowIdx, eCtxMap[leftRowid][fieldName].Vars, sc.LookupAlias); err != nil {
									return bs, err
								}
								_, err := eCtxMap[leftRowid][fieldName].Eval(fieldDef.ParsedExpression)
								if err != nil {
									return bs, fmt.Errorf("cannot evaluate target expression [%s]: [%s]", fieldDef.RawExpression, err.Error())
								}
							}
						}
					} else {
						// Non-group. Find correspondent row from rsLeft, merge left and right and call row-level eval
						for _, leftRowIdx := range keyToLeftRowIdxMap[rightRowKey] {

							leftRowFoundRightLookup[leftRowIdx] = true

							vars := eval.VarValuesMap{}
							if err := rsLeft.ExportToVars(leftRowIdx, &vars); err != nil {
								return bs, err
							}
							if err := rsRight.ExportToVarsWithAlias(rightRowIdx, &vars, sc.LookupAlias); err != nil {
								return bs, err
							}

							// We are ready to write this result right away, so prepare the output tableRecord
							tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(false, vars)
							if err != nil {
								return bs, fmt.Errorf("cannot populate table record from [%v]: [%s]", vars, err.Error())
							}

							// Check table creator having
							inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
							if err != nil {
								return bs, fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
							}

							// Write batch if needed
							if inResult {
								instr.add(tableRecord)
								tableRecordBatchCount++
								if tableRecordBatchCount == instr.BatchSize {
									if err := instr.waitForWorkers(logger, pCtx); err != nil {
										return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
									}
									reportWriteTable(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)
									batchStartTime = time.Now()
									tableRecordBatchCount = 0
									if err := instr.startWorkers(logger, pCtx); err != nil {
										return bs, err
									}
								}
								bs.RowsWritten++
							}
						} // non-group result row written
					} // group case handled
				} // for each found right row

				if rsRight.RowCount < node.Lookup.RightLookupReadBatchSize || len(rightPageState) == 0 {
					break
				}
			}

			if rsIdx.RowCount < node.Lookup.IdxReadBatchSize || len(idxPageState) == 0 {
				break
			}
		} // for each idx batch

		if node.Lookup.IsGroup {
			// Time to write the result of the grouped
			for leftRowIdx := 0; leftRowIdx < rsLeft.RowCount; leftRowIdx++ {
				tableRecord := map[string]interface{}{}
				if !leftRowFoundRightLookup[leftRowIdx] {
					if node.Lookup.LookupJoin == sc.LookupJoinLeft {

						// Grouped left outer join with no data on the right

						leftVars := eval.VarValuesMap{}
						if err := rsLeft.ExportToVars(leftRowIdx, &leftVars); err != nil {
							return bs, err
						}

						for fieldName, fieldDef := range node.TableCreator.Fields {
							if eval.IsRootAggFunc(fieldDef.ParsedExpression) == eval.AggFuncEnabled {
								// Aggregate func is used in field expression - ignore the expression and produce default
								tableRecord[fieldName], err = node.TableCreator.GetFieldDefaultReadyForDb(fieldName)
								if err != nil {
									return bs, fmt.Errorf("cannot initialize default field %s: [%s]", fieldName, err.Error())
								}
							} else {
								// No aggregate function used in field expression - assume it contains only left-side fields
								tableRecord[fieldName], err = sc.CalculateFieldValue(fieldName, fieldDef, leftVars, false)
								if err != nil {
									return bs, err
								}
							}
						}
					} else {

						// Grouped inner join with no data on the right
						// Do not insert this left row

						continue
					}
				} else {

					// Grouped inner or left outer with present data on the right

					leftRowid := *((*rsLeft.Rows[leftRowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
					for fieldName, fieldDef := range node.TableCreator.Fields {
						finalValue := eCtxMap[leftRowid][fieldName].Value

						if err := sc.CheckValueType(finalValue, fieldDef.Type); err != nil {
							return bs, fmt.Errorf("invalid field %s type: [%s]", fieldName, err.Error())
						}
						tableRecord[fieldName] = finalValue
					}
				}

				// Check table creator having
				inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
				if err != nil {
					return bs, fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
				}

				// Write batch if needed
				if inResult {
					instr.add(tableRecord)
					tableRecordBatchCount++
					if tableRecordBatchCount == instr.BatchSize {
						if err := instr.waitForWorkers(logger, pCtx); err != nil {
							return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
						}
						reportWriteTable(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)
						batchStartTime = time.Now()
						tableRecordBatchCount = 0
						if err := instr.startWorkers(logger, pCtx); err != nil {
							return bs, err
						}
					}
					bs.RowsWritten++
				}
			}
		} else if node.Lookup.LookupJoin == sc.LookupJoinLeft {

			// Non-grouped left outer join.
			// Handle those left rows that did not have right lookup counterpart
			// (those who had - they have been written already)

			for leftRowIdx := 0; leftRowIdx < rsLeft.RowCount; leftRowIdx++ {
				if leftRowFoundRightLookup[leftRowIdx] {
					continue
				}

				leftVars := eval.VarValuesMap{}
				if err := rsLeft.ExportToVars(leftRowIdx, &leftVars); err != nil {
					return bs, err
				}

				tableRecord := map[string]interface{}{}

				for fieldName, fieldDef := range node.TableCreator.Fields {
					if fieldDef.UsedFields.HasFieldsWithTableAlias(sc.LookupAlias) {
						// This field expression uses fields from lkp table - produce default value
						tableRecord[fieldName], err = node.TableCreator.GetFieldDefaultReadyForDb(fieldName)
						if err != nil {
							return bs, fmt.Errorf("cannot initialize non-grouped default field %s: [%s]", fieldName, err.Error())
						}
					} else {
						// This field expression does not use fields from lkp table - assume the expression contains only left-side fields
						tableRecord[fieldName], err = sc.CalculateFieldValue(fieldName, fieldDef, leftVars, false)
						if err != nil {
							return bs, err
						}
					}
				}

				// Check table creator having
				inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
				if err != nil {
					return bs, fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
				}

				// Write batch if needed
				if inResult {
					instr.add(tableRecord)
					tableRecordBatchCount++
					if tableRecordBatchCount == instr.BatchSize {
						if err := instr.waitForWorkers(logger, pCtx); err != nil {
							return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
						}
						reportWriteTable(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)
						batchStartTime = time.Now()
						tableRecordBatchCount = 0
						if err := instr.startWorkers(logger, pCtx); err != nil {
							return bs, err
						}
					}
					bs.RowsWritten++
				}
			}
		} else {
			// Non-grouped inner join, already handled above
		}

		bs.RowsRead += rsLeft.RowCount
		if rsLeft.RowCount < leftBatchSize {
			break
		}
	} // for each source table batch

	// Write leftovers regardless of tableRecordBatchCount == 0
	if err := instr.waitForWorkers(logger, pCtx); err != nil {
		return bs, fmt.Errorf("cannot save record batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
	}
	reportWriteTableLeftovers(logger, pCtx, tableRecordBatchCount, time.Since(batchStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)
	return bs, nil
}
