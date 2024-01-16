package proc

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/xfer"
)

const DistinctIdxName string = "idx_distinct"

type TableRecord map[string]any
type TableRecordPtr *map[string]any
type TableRecordBatch []TableRecordPtr

const DefaultInserterBatchSize int = 5000

func reportWriteTable(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, recordCount int, dur time.Duration, indexCount int, workerCount int) {
	logger.InfoCtx(pCtx, "WriteTable: %d items in %.3fs (%.0f items/s, %d indexes, eff rate %.0f writes/s), %d workers",
		recordCount,
		dur.Seconds(),
		float64(recordCount)/dur.Seconds(),
		indexCount,
		float64(recordCount*(indexCount+1))/dur.Seconds(),
		workerCount)
}

func reportWriteTableLeftovers(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, recordCount int, dur time.Duration, indexCount int, workerCount int) {
	logger.InfoCtx(pCtx, "WriteTableLeftovers: %d items in %.3fs (%.0f items/s, %d indexes, eff rate %.0f writes/s), %d workers",
		recordCount,
		dur.Seconds(),
		float64(recordCount)/dur.Seconds(),
		indexCount,
		float64(recordCount*(indexCount+1))/dur.Seconds(),
		workerCount)
}

func reportWriteTableComplete(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, readCount int, recordCount int, dur time.Duration, indexCount int, workerCount int) {
	logger.InfoCtx(pCtx, "WriteTableComplete: read %d, wrote %d items in %.3fs (%.0f items/s, %d indexes, eff rate %.0f writes/s), %d workers",
		readCount,
		recordCount,
		dur.Seconds(),
		float64(recordCount)/dur.Seconds(),
		indexCount,
		float64(recordCount*(indexCount+1))/dur.Seconds(),
		workerCount)
}

func checkRunReadFileForBatchSanity(node *sc.ScriptNodeDef, srcFileIdx int) error {
	if !node.HasFileReader() {
		return fmt.Errorf("node does not have file reader")
	}
	if !node.HasTableCreator() {
		return fmt.Errorf("node does not have table creator")
	}

	if srcFileIdx < 0 || srcFileIdx >= len(node.FileReader.SrcFileUrls) {
		return fmt.Errorf("cannot find file to read: asked to read src file with index %d while there are only %d source files available", srcFileIdx, len(node.FileReader.SrcFileUrls))
	}
	return nil
}

func RunReadFileForBatch(envConfig *env.EnvConfig, logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, srcFileIdx int) (BatchStats, error) {
	logger.PushF("proc.RunReadFileForBatch")
	defer logger.PopF()

	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0}

	node := pCtx.CurrentScriptNode

	if err := checkRunReadFileForBatchSanity(node, srcFileIdx); err != nil {
		return bs, err
	}

	filePath := node.FileReader.SrcFileUrls[srcFileIdx]

	u, err := url.Parse(filePath)
	if err != nil {
		return bs, fmt.Errorf("cannot parse file uri %s: %s", filePath, err.Error())
	}

	bs.Src = filePath
	bs.Dst = node.TableCreator.Name + cql.RunIdSuffix(pCtx.BatchInfo.RunId)

	var localSrcFile *os.File
	var fileReader io.Reader
	var fileReadSeeker io.ReadSeeker
	if u.Scheme == xfer.UriSchemeFile || len(u.Scheme) == 0 {
		localSrcFile, err = os.Open(filePath)
		if err != nil {
			return bs, err
		}
		defer localSrcFile.Close()
		fileReader = bufio.NewReader(localSrcFile)
		fileReadSeeker = localSrcFile
	} else if u.Scheme == xfer.UriSchemeHttp || u.Scheme == xfer.UriSchemeHttps {
		// If this is a parquet file, download it and then open so we have fileReadSeeker
		if node.FileReader.ReaderFileType == sc.ReaderFileTypeParquet {
			dstFile, err := os.CreateTemp("", "capi")
			if err != nil {
				return bs, fmt.Errorf("cannot create temp file for %s: %s", filePath, err.Error())
			}

			readCloser, err := xfer.GetHttpReadCloser(filePath, u.Scheme, envConfig.CaPath)
			if err != nil {
				dstFile.Close()
				return bs, fmt.Errorf("cannot open http file %s: %s", filePath, err.Error())
			}
			defer readCloser.Close()

			if _, err := io.Copy(dstFile, readCloser); err != nil {
				dstFile.Close()
				return bs, fmt.Errorf("cannot save http file %s to temp file %s: %s", filePath, dstFile.Name(), err.Error())
			}

			logger.Info("downloaded http file %s to %s", filePath, dstFile.Name())
			dstFile.Close()
			defer os.Remove(dstFile.Name())

			localSrcFile, err = os.Open(dstFile.Name())
			if err != nil {
				return bs, fmt.Errorf("cannot read from file %s downloaded from %s: %s", dstFile.Name(), filePath, err.Error())
			}
			defer localSrcFile.Close()
			fileReadSeeker = localSrcFile
		} else {
			// Just read from the net
			readCloser, err := xfer.GetHttpReadCloser(filePath, u.Scheme, envConfig.CaPath)
			if err != nil {
				return bs, err
			}
			fileReader = readCloser
			defer readCloser.Close()
		}
	} else if u.Scheme == xfer.UriSchemeSftp {
		// When dealing with sftp, we download the *whole* file, instead of providing a reader
		dstFile, err := os.CreateTemp("", "capi")
		if err != nil {
			return bs, fmt.Errorf("cannot create temp file for %s: %s", filePath, err.Error())
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
		localSrcFile, err = os.Open(dstFile.Name())
		if err != nil {
			return bs, fmt.Errorf("cannot read from file %s downloaded from %s: %s", dstFile.Name(), filePath, err.Error())
		}
		defer localSrcFile.Close()
		fileReader = bufio.NewReader(localSrcFile)
		fileReadSeeker = localSrcFile
	} else {
		return bs, fmt.Errorf("uri scheme %s not supported: %s", u.Scheme, filePath)
	}

	if node.FileReader.ReaderFileType == sc.ReaderFileTypeCsv {
		return readCsv(envConfig, logger, pCtx, totalStartTime, filePath, fileReader)
	} else if node.FileReader.ReaderFileType == sc.ReaderFileTypeParquet {
		return readParquet(envConfig, logger, pCtx, totalStartTime, filePath, fileReadSeeker)
	}

	return BatchStats{RowsRead: 0, RowsWritten: 0}, fmt.Errorf("unknown reader file type: %d", node.FileReader.ReaderFileType)
}

func RunCreateTableForCustomProcessorForBatch(envConfig *env.EnvConfig,
	logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.RunCreateTableForCustomProcessorForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName + cql.RunIdSuffix(readerNodeRunId), Dst: node.TableCreator.Name + cql.RunIdSuffix(readerNodeRunId)}

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
	instr := newTableInserter(envConfig, pCtx, &node.TableCreator, inserterBatchSize, DataIdxSeqModeDataFirst)
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndCloseErrorsOut(logger, pCtx)

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
				if err = instr.add(tableRecord); err != nil {
					return fmt.Errorf("cannot add record to %s: [%s]", node.TableCreator.Name, err.Error())
				}
				rowsWritten++
				bs.RowsWritten++
			}
		}

		reportWriteTable(logger, pCtx, rowsWritten, time.Since(flushStartTime), len(node.TableCreator.Indexes), instr.NumWorkers)

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
	logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.RunCreateTableForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	batchStartTime := time.Now()
	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName + cql.RunIdSuffix(readerNodeRunId), Dst: node.TableCreator.Name + cql.RunIdSuffix(readerNodeRunId)}

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
	instr := newTableInserter(envConfig, pCtx, &node.TableCreator, inserterBatchSize, DataIdxSeqModeDataFirst)
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndCloseErrorsOut(logger, pCtx)

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
				tableRecordBatchCount, batchStartTime, err = addRecordAndSaveBatchIfNeeded(pCtx, logger, node, tableRecord, tableRecordBatchCount, batchStartTime, instr)
				if err != nil {
					return bs, err
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

func RunCreateDistinctTableForBatch(envConfig *env.EnvConfig,
	logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.RunCreateDistinctTableForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	batchStartTime := time.Now()
	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName + cql.RunIdSuffix(readerNodeRunId), Dst: node.TableCreator.Name + cql.RunIdSuffix(readerNodeRunId)}

	if readerNodeRunId == 0 {
		return bs, fmt.Errorf("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	if !node.HasTableReader() {
		return bs, fmt.Errorf("node does not have table reader")
	}
	if !node.HasTableCreator() {
		return bs, fmt.Errorf("node does not have table creator")
	}

	if node.TableCreator.RawHaving != "" {
		return bs, fmt.Errorf("distinct_table node is not allowed to have Having condition")
	}

	if node.RerunPolicy == sc.NodeRerun {
		return bs, fmt.Errorf("distinct_table node is not allowed to have rerun policy, no reruns possible")
	}

	if _, _, err := node.TableCreator.GetSingleUniqueIndexDef(); err != nil {
		return bs, err
	}

	/*
		// Create our own unique index with the name DistinctIdxName
		idxComponents := make([]sc.IdxComponentDef, len(node.TableCreator.Fields))
		componentCount := 0
		for fName, fDef := range node.TableCreator.Fields {
			idxComponents[componentCount] = sc.IdxComponentDef{
				FieldName:       fName,
				CaseSensitivity: sc.IdxCaseSensitive,
				SortOrder:       sc.IdxSortAsc,
				StringLen:       sc.DefaultStringComponentLen,
				FieldType:       fDef.Type}
			componentCount++
		}

		// Very important: TableCreator.Fields map is unordered, so order idx components
		sort.Slice(idxComponents, func(i, j int) bool {
			return idxComponents[i].FieldName < idxComponents[j].FieldName
		})

		node.TableCreator.Indexes = sc.IdxDefMap{
			DistinctIdxName: &sc.IdxDef{
				Uniqueness: sc.IdxUnique,
				Components: idxComponents}}
	*/

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
	instr := newTableInserter(envConfig, pCtx, &node.TableCreator, inserterBatchSize, DataIdxSeqModeDistinctIdxFirst)
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndCloseErrorsOut(logger, pCtx)

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

			tableRecordBatchCount, batchStartTime, err = addRecordAndSaveBatchIfNeeded(pCtx, logger, node, tableRecord, tableRecordBatchCount, batchStartTime, instr)
			if err != nil {
				return bs, err
			}

			// TODO: this is incorrect, find a way to return proper RowsWritten or abandon them for distinct node
			bs.RowsWritten++
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

func buildKeysToFindInTheLookupIndex(rsLeft *Rowset, scriptNodeLookup sc.LookupDef) ([]string, map[string][]int, error) {
	// Build keys to find in the lookup index, one key may yield multiple rowids
	keyToLeftRowIdxMap := map[string][]int{}
	for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
		vars := eval.VarValuesMap{}
		if err := rsLeft.ExportToVars(rowIdx, &vars); err != nil {
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
	for rowIdx := 0; rowIdx < rsIdx.RowCount; rowIdx++ {
		rightRowId := *((*rsIdx.Rows[rowIdx])[rsIdx.FieldsByFieldName["rowid"]].(*int64))
		key := *((*rsIdx.Rows[rowIdx])[rsIdx.FieldsByFieldName["key"]].(*string))
		rightRowIdToKeyMap[rightRowId] = key
	}

	rowidsToFind := make(map[int64]struct{}, len(rightRowIdToKeyMap))
	for k := range rightRowIdToKeyMap {
		rowidsToFind[k] = struct{}{}
	}

	return rowidsToFind, rightRowIdToKeyMap
}

func setupEvalCtxForGroup(node *sc.ScriptNodeDef, rsLeft *Rowset) (map[int64]map[string]*eval.EvalCtx, error) {
	eCtxMap := map[int64]map[string]*eval.EvalCtx{}
	if !node.Lookup.IsGroup {
		return eCtxMap, nil
	}
	if node.Lookup.IsGroup {
		for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
			rowid := *((*rsLeft.Rows[rowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
			eCtxMap[rowid] = map[string]*eval.EvalCtx{}
			for fieldName, fieldDef := range node.TableCreator.Fields {
				aggFuncEnabled, aggFuncType, aggFuncArgs := eval.DetectRootAggFunc(fieldDef.ParsedExpression)
				newCtx, newCtxErr := eval.NewPlainEvalCtxAndInitializedAgg(aggFuncEnabled, aggFuncType, aggFuncArgs)
				if newCtxErr != nil {
					return nil, newCtxErr
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
		eCtxMap[leftRowid][fieldName].Vars = &eval.VarValuesMap{}
		if err := rsLeft.ExportToVars(leftRowIdx, eCtxMap[leftRowid][fieldName].Vars); err != nil {
			return err
		}
		if err := rsRight.ExportToVarsWithAlias(rightRowIdx, eCtxMap[leftRowid][fieldName].Vars, sc.LookupAlias); err != nil {
			return err
		}
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
		if err := rsRight.ExportToVars(rightRowIdx, &vars); err != nil {
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
		if err := rsLeft.ExportToVars(leftRowIdx, &leftVars); err != nil {
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
				tableRecord[fieldName], err = sc.CalculateFieldValue(fieldName, fieldDef, leftVars, false)
				if err != nil {
					return nil, err
				}
			}
		}
	} else {
		// Grouped inner or left outer with present data on the right
		leftRowid := *((*rsLeft.Rows[leftRowIdx])[rsLeft.FieldsByFieldName["rowid"]].(*int64))
		for fieldName, fieldDef := range node.TableCreator.Fields {
			finalValue := eCtxMap[leftRowid][fieldName].Value

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
	if err := rsLeft.ExportToVars(leftRowIdx, &vars); err != nil {
		return nil, err
	}
	if err := rsRight.ExportToVarsWithAlias(rightRowIdx, &vars, sc.LookupAlias); err != nil {
		return nil, err
	}

	// We are ready to write this result right away, so prepare the output tableRecord
	tableRecord, err := node.TableCreator.CalculateTableRecordFromSrcVars(false, vars)
	if err != nil {
		return nil, fmt.Errorf("cannot populate table record from [%v]: [%s]", vars, err.Error())
	}
	return tableRecord, nil
}

func produceNonGroupedTableRecordForCheldlessLeft(node *sc.ScriptNodeDef, rsLeft *Rowset, leftRowIdx int) (map[string]any, error) {
	tableRecord := map[string]any{}

	leftVars := eval.VarValuesMap{}
	if err := rsLeft.ExportToVars(leftRowIdx, &leftVars); err != nil {
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
			tableRecord[fieldName], err = sc.CalculateFieldValue(fieldName, fieldDef, leftVars, false)
			if err != nil {
				return nil, err
			}
		}
	}
	return tableRecord, nil
}

func addRecordAndSaveBatchIfNeeded(pCtx *ctx.MessageProcessingContext, logger *l.CapiLogger, node *sc.ScriptNodeDef, tableRecord map[string]any, tableRecordBatchCount int, batchStartTime time.Time, instr *TableInserter) (int, time.Time, error) {
	if err := instr.add(tableRecord); err != nil {
		return tableRecordBatchCount, batchStartTime, fmt.Errorf("cannot add record to batch of size %d to %s: [%s]", tableRecordBatchCount, node.TableCreator.Name, err.Error())
	}
	tableRecordBatchCount++
	if tableRecordBatchCount == instr.BatchSize {
		tableRecordBatchCount, batchStartTime, err := saveCompletedBatch(pCtx, logger, &node.TableCreator, tableRecordBatchCount, batchStartTime, instr)
		if err != nil {
			return tableRecordBatchCount, batchStartTime, err
		}
	}
	return tableRecordBatchCount, batchStartTime, nil
}

func checkHavingAddRecordAndSaveBatchIfNeeded(pCtx *ctx.MessageProcessingContext, logger *l.CapiLogger, node *sc.ScriptNodeDef, tableRecord map[string]any, tableRecordBatchCount int, batchStartTime time.Time, instr *TableInserter) (int, time.Time, int, error) {
	rowsWritten := 0
	// Check table creator having
	inResult, err := node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
	if err != nil {
		return tableRecordBatchCount, batchStartTime, 0, fmt.Errorf("cannot check having condition [%s], table record [%v]: [%s]", node.TableCreator.RawHaving, tableRecord, err.Error())
	}

	// Write batch if needed
	if inResult {
		tableRecordBatchCount, batchStartTime, err = addRecordAndSaveBatchIfNeeded(pCtx, logger, node, tableRecord, tableRecordBatchCount, batchStartTime, instr)
		if err != nil {
			return tableRecordBatchCount, batchStartTime, 0, err
		}
		rowsWritten++
	}

	return tableRecordBatchCount, batchStartTime, rowsWritten, nil
}

func checkRunCreateTableRelForBatchSanity(node *sc.ScriptNodeDef, readerNodeRunId int16, lookupNodeRunId int16) error {
	if readerNodeRunId == 0 {
		return fmt.Errorf("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	if lookupNodeRunId == 0 {
		return fmt.Errorf("this node has a dependency node to lookup data at that was never started in this keyspace (lookupNodeRunId == 0)")
	}

	if !node.HasTableReader() {
		return fmt.Errorf("node does not have table reader")
	}
	if !node.HasTableCreator() {
		return fmt.Errorf("node does not have table creator")
	}
	if !node.HasLookup() {
		return fmt.Errorf("node does not have lookup")
	}
	return nil
}

func RunCreateTableRelForBatch(envConfig *env.EnvConfig,
	logger *l.CapiLogger,
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
	tableRecordBatchCount := 0

	rsLeft := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(node.TableReader.TableName)},
		sc.FieldRefs{sc.RowidTokenFieldRef()},
		srcLeftFieldRefs)

	inserterBatchSize := DefaultInserterBatchSize
	if inserterBatchSize < node.TableReader.RowsetSize {
		inserterBatchSize = node.TableReader.RowsetSize
	}
	instr := newTableInserter(envConfig, pCtx, &node.TableCreator, inserterBatchSize, DataIdxSeqModeDataFirst)
	if err := instr.startWorkers(logger, pCtx); err != nil {
		return bs, err
	}
	defer instr.waitForWorkersAndCloseErrorsOut(logger, pCtx)

	curStartLeftToken := startLeftToken
	leftPageIdx := 0
	for {
		selectLeftBatchByTokenStartTime := time.Now()
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

		logger.DebugCtx(pCtx, "selectBatchFromTableByToken: leftPageIdx %d, queried tokens from %d to %d in %.3fs, retrieved %d rows", leftPageIdx, curStartLeftToken, endLeftToken, time.Since(selectLeftBatchByTokenStartTime).Seconds(), rsLeft.RowCount)

		curStartLeftToken = lastRetrievedLeftToken + 1

		if rsLeft.RowCount == 0 {
			break
		}

		// Setup eval ctx for each target field if grouping is involved
		// map: rowid -> field -> ctx
		eCtxMap, err := setupEvalCtxForGroup(node, rsLeft)
		if err != nil {
			return bs, err
		}

		// Array that says if a left row has any right counterparts
		leftRowFoundRightLookup := make([]bool, rsLeft.RowCount)
		for rowIdx := 0; rowIdx < rsLeft.RowCount; rowIdx++ {
			leftRowFoundRightLookup[rowIdx] = false
		}

		// Build keys to find in the lookup index, one key may yield multiple rowids
		keysToFind, keyToLeftRowIdxMap, err := buildKeysToFindInTheLookupIndex(rsLeft, node.Lookup)
		if err != nil {
			return bs, err
		}

		lookupFieldRefs := sc.FieldRefs{}
		lookupFieldRefs.AppendWithFilter(node.TableCreator.UsedInHavingFields, node.Lookup.TableCreator.Name)
		lookupFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, node.Lookup.TableCreator.Name)

		// Select from idx table by keys
		rsIdx := NewRowsetFromFieldRefs(
			sc.FieldRefs{sc.RowidFieldRef(node.Lookup.IndexName)},
			sc.FieldRefs{sc.KeyTokenFieldRef()},
			sc.FieldRefs{sc.IdxKeyFieldRef()})

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
				return bs, err
			}

			if rsIdx.RowCount == 0 {
				break
			}

			// Build a map of right-row-id -> key
			rightRowidsToFind, rightRowIdToKeyMap := getRightRowidsToFind(rsIdx)

			logger.DebugCtx(pCtx, "selectBatchFromIdxTablePaged: leftPageIdx %d, rightIdxPageIdx %d, queried %d keys in %.3fs, retrieved %d right rowids", leftPageIdx, rightIdxPageIdx, len(keysToFind), time.Since(selectIdxBatchStartTime).Seconds(), len(rightRowidsToFind))

			// Select from right table by rowid
			rsRight := NewRowsetFromFieldRefs(
				sc.FieldRefs{sc.RowidFieldRef(node.Lookup.TableCreator.Name)},
				sc.FieldRefs{sc.RowidTokenFieldRef()},
				srcRightFieldRefs)

			var rightPageState []byte
			rightDataPageIdx := 0
			for {
				selectBatchStartTime := time.Now()
				rightPageState, err = selectBatchFromDataTablePaged(logger,
					pCtx,
					rsRight,
					node.Lookup.TableCreator.Name,
					lookupNodeRunId,
					node.Lookup.RightLookupReadBatchSize,
					rightPageState,
					rightRowidsToFind)
				if err != nil {
					return bs, err
				}

				logger.DebugCtx(pCtx, "selectBatchFromDataTablePaged: leftPageIdx %d, rightIdxPageIdx %d, rightDataPageIdx %d, queried %d rowids in %.3fs, retrieved %d rowids", leftPageIdx, rightIdxPageIdx, rightDataPageIdx, len(rightRowidsToFind), time.Since(selectBatchStartTime).Seconds(), rsRight.RowCount)

				if rsRight.RowCount == 0 {
					break
				}

				for rightRowIdx := 0; rightRowIdx < rsRight.RowCount; rightRowIdx++ {
					rightRowId := *((*rsRight.Rows[rightRowIdx])[rsRight.FieldsByFieldName["rowid"]].(*int64))
					rightRowKey := rightRowIdToKeyMap[rightRowId]

					// Remove this right rowid from the set, we do not need it anymore.
					// Reset page state, we will have to start over
					rightPageState = nil
					delete(rightRowidsToFind, rightRowId)

					// Check filter condition if needed
					lookupFilterOk, err := checkLookupFilter(&node.Lookup, rsRight, rightRowIdx)
					if err != nil {
						return bs, err
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
								return bs, err
							}
						}
					} else {
						// Non-group, and the right row was found for the parent left row.
						// Find correspondent row from rsLeft, merge left and right and call row-level eval
						for _, leftRowIdx := range keyToLeftRowIdxMap[rightRowKey] {

							leftRowFoundRightLookup[leftRowIdx] = true

							tableRecord, err := produceNonGroupedTableRecordForLeftWithChildren(node, rsLeft, leftRowIdx, rsRight, rightRowIdx)
							if err != nil {
								return bs, err
							}

							var rowsWritten int
							tableRecordBatchCount, batchStartTime, rowsWritten, err = checkHavingAddRecordAndSaveBatchIfNeeded(pCtx, logger, node, tableRecord, tableRecordBatchCount, batchStartTime, instr)
							if err != nil {
								return bs, err
							}
							bs.RowsWritten += rowsWritten

						} // non-group result row written
					} // group case handled
				} // for each found right row

				if rsRight.RowCount < node.Lookup.RightLookupReadBatchSize || len(rightPageState) == 0 {
					break
				}
				rightDataPageIdx++
			} // for each data page

			if rsIdx.RowCount < node.Lookup.IdxReadBatchSize || len(idxPageState) == 0 {
				break
			}
			rightIdxPageIdx++
		} // for each idx page

		// For grouped - group
		// For non-grouped left join - add empty left-side (those who have right counterpart were alredy hendled above)
		// Non-grouped inner join - already handled above
		if node.Lookup.IsGroup {
			// Time to write the result of the grouped we evaluated above using eCtxMap
			for leftRowIdx := 0; leftRowIdx < rsLeft.RowCount; leftRowIdx++ {
				tableRecord, err := produceGroupedTableRecord(node, rsLeft, leftRowIdx, leftRowFoundRightLookup, eCtxMap)
				if err != nil {
					return bs, err
				}
				if tableRecord == nil {
					// No record generated, it's ok (inner join and no right rows)
					continue
				}

				var rowsWritten int
				tableRecordBatchCount, batchStartTime, rowsWritten, err = checkHavingAddRecordAndSaveBatchIfNeeded(pCtx, logger, node, tableRecord, tableRecordBatchCount, batchStartTime, instr)
				if err != nil {
					return bs, err
				}
				bs.RowsWritten += rowsWritten
			}
		} else if node.Lookup.LookupJoin == sc.LookupJoinLeft {

			// Non-grouped left outer join.
			// Handle those left rows that did not have right lookup counterpart
			// (those who had - they have been written already)

			for leftRowIdx := 0; leftRowIdx < rsLeft.RowCount; leftRowIdx++ {
				if leftRowFoundRightLookup[leftRowIdx] {
					// This left row had right counterparts and grouped result was already written
					continue
				}

				tableRecord, err := produceNonGroupedTableRecordForCheldlessLeft(node, rsLeft, leftRowIdx)
				if err != nil {
					return bs, nil
				}

				var rowsWritten int
				tableRecordBatchCount, batchStartTime, rowsWritten, err = checkHavingAddRecordAndSaveBatchIfNeeded(pCtx, logger, node, tableRecord, tableRecordBatchCount, batchStartTime, instr)
				if err != nil {
					return bs, err
				}
				bs.RowsWritten += rowsWritten
			}
		}

		bs.RowsRead += rsLeft.RowCount
		if rsLeft.RowCount < leftBatchSize {
			break
		}
		leftPageIdx++
	} // for each source table batch

	// Write leftovers regardless of tableRecordBatchCount == 0
	if _, _, err := saveCompletedBatch(pCtx, logger, &node.TableCreator, tableRecordBatchCount, batchStartTime, instr); err != nil {
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
