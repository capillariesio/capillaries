package proc

import (
	"errors"
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

func runCreateDistinctTableForBatch(envConfig *env.EnvConfig,
	logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.runCreateDistinctTableForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

	// batchStartTime := time.Now()
	totalStartTime := time.Now()
	bs := BatchStats{RowsRead: 0, RowsWritten: 0, Src: node.TableReader.TableName + cql.RunIdSuffix(readerNodeRunId), Dst: node.TableCreator.Name + cql.RunIdSuffix(readerNodeRunId)}

	if readerNodeRunId == 0 {
		return bs, errors.New("this node has a dependency node to read data from that was never started in this keyspace (readerNodeRunId == 0)")
	}

	if !node.HasTableReader() {
		return bs, errors.New("node does not have table reader")
	}
	if !node.HasTableCreator() {
		return bs, errors.New("node does not have table creator")
	}

	if node.TableCreator.RawHaving != "" {
		return bs, errors.New("distinct_table node is not allowed to have Having condition")
	}

	distinctIdxName, _, err := node.TableCreator.GetSingleUniqueIndexDef()
	if err != nil {
		return bs, err
	}

	// Fields to read from source table
	srcLeftFieldRefs := sc.FieldRefs{}
	srcLeftFieldRefs.AppendWithFilter(node.TableCreator.UsedInTargetExpressionsFields, sc.ReaderAlias)

	leftBatchSize := node.TableReader.RowsetSize
	// tableRecordBatchCount := 0
	curStartLeftToken := startLeftToken

	rsIn := NewRowsetFromFieldRefs(
		sc.FieldRefs{sc.RowidFieldRef(node.TableReader.TableName)},
		sc.FieldRefs{sc.RowidTokenFieldRef()},
		srcLeftFieldRefs)

	instr, err := createInserterAndStartWorkers(logger, envConfig, pCtx, &node.TableCreator, DataIdxSeqModeDistinctIdxFirst, logger.ZapMachine.String)
	if err != nil {
		return bs, err
	}
	instr.startDrainer()
	defer instr.closeInserter(logger, pCtx)

	var curStartLeftTokenRowIds []int64
	for {
		// Poor man's cache that spans across rsIn retrieved, works well for low-cardinality datasets
		usedDistinctKeysMap := map[string]struct{}{}
		distinctCacheHits := 0

		lastRetrievedLeftToken, endTokenRowIds, err := selectBatchFromTableByToken(logger,
			pCtx,
			rsIn,
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

		// If token(rowid) guaranteed uniqueness, we would just "curStartLeftToken = lastRetrievedLeftToken + 1"
		// But duplicates are possible, so we have to be prepared to handle token overlaps
		// (rows with same token but different rowids returned in separate selectBatchFromTableByToken calls)
		// See overlap/epilogue logic in selectBatchFromTableByToken.
		curStartLeftToken = lastRetrievedLeftToken
		curStartLeftTokenRowIds = endTokenRowIds

		if rsIn.RowCount == 0 {
			break
		}

		// Minimize allocations to help GC in this high-traffic loop
		var tableRecord map[string]any
		indexKeyMap := map[string]string{}
		vars := eval.VarValuesMap{}

		// Save rsIn
		for outRowIdx := 0; outRowIdx < rsIn.RowCount; outRowIdx++ {
			clear(vars)
			if err = rsIn.ExportToVars(outRowIdx, vars); err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot export to vars from source table, node %s: %s", node.Name, err.Error()))
				return bs, instr.waitForDrainer()
			}

			tableRecord, err = node.TableCreator.CalculateTableRecordFromSrcVars(vars)
			if err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot populate table record from [%v], node %s: [%s]", vars, node.Name, err.Error()))
				return bs, instr.waitForDrainer()
			}

			err = instr.buildIndexKeys(tableRecord, indexKeyMap)
			if err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot build index keys for table %s: [%s]", node.TableCreator.Name, err.Error()))
				return bs, instr.waitForDrainer()
			}

			uniqueDistinctKey := indexKeyMap[distinctIdxName]

			// Poor man's cache
			if _, ok := usedDistinctKeysMap[uniqueDistinctKey]; !ok {
				usedDistinctKeysMap[uniqueDistinctKey] = struct{}{}
				instr.add(tableRecord, indexKeyMap)

				// TODO: this is incorrect, find a way to return proper RowsWritten or abandon them for distinct node altogether
				bs.RowsWritten++
			} else {
				distinctCacheHits++
				logger.DebugCtx(pCtx, "skipped writing unique key %s to %s, already written", uniqueDistinctKey, node.TableCreator.Name)
			}
		}

		logger.DebugCtx(pCtx, "distinct cache hits %d/%d=%d percent %s", distinctCacheHits, rsIn.RowCount, distinctCacheHits*100/rsIn.RowCount, node.TableCreator.Name)

		bs.RowsRead += rsIn.RowCount

		// We are tempted to "if rs.RowCount < srcBatchSize break", here but do not do that:
		// because of the rowid overlapping/epilogue logic, selectBatchFromTableByToken returns less rows than rs capacity

		instr.PCtx.SendHeartbeat()
	} // for each source table batch

	instr.doneSending()
	if err := instr.waitForDrainer(); err != nil {
		return bs, err
	}

	bs.Elapsed = time.Since(totalStartTime)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}
