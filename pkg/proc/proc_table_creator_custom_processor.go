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

// See implementations in pkg/custom
type CustomProcessorRunner interface {
	Run(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, rsIn *Rowset, flushVarsArrayCallback func(varsArray []eval.VarValuesMap, varsArrayCount int) error) error
}

func runCreateTableForCustomProcessorForBatch(envConfig *env.EnvConfig,
	logger *l.CapiLogger,
	pCtx *ctx.MessageProcessingContext,
	readerNodeRunId int16,
	startLeftToken int64,
	endLeftToken int64) (BatchStats, error) {

	logger.PushF("proc.runCreateTableForCustomProcessorForBatch")
	defer logger.PopF()

	node := pCtx.CurrentScriptNode

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

	instr, err := createInserterAndStartWorkers(logger, envConfig, pCtx, &node.TableCreator, DataIdxSeqModeDataFirst, logger.ZapMachine.String)
	if err != nil {
		return bs, err
	}
	defer instr.closeInserter(logger, pCtx)

	flushVarsArrayCallback := func(varsArray []eval.VarValuesMap, varsArrayCount int) error {

		instr.startDrainer()

		// Minimize allocations to help GC in this high-traffic loop
		var tableRecord map[string]any
		indexKeyMap := map[string]string{}
		var inResult bool
		var err error
		for outRowIdx := 0; outRowIdx < varsArrayCount; outRowIdx++ {
			vars := varsArray[outRowIdx]

			tableRecord, err = node.TableCreator.CalculateTableRecordFromSrcVars(vars)
			if err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot populate table record from [%v], node %s: [%s]", vars, node.Name, err.Error()))
				return instr.waitForDrainer()
			}

			// Check table creator having
			inResult, err = node.TableCreator.CheckTableRecordHavingCondition(tableRecord)
			if err != nil {
				instr.cancelDrainer(fmt.Errorf("cannot check having condition [%s], node %s, table record [%v]: [%s]", node.TableCreator.RawHaving, node.Name, tableRecord, err.Error()))
				return instr.waitForDrainer()
			}

			if inResult {
				err = instr.buildIndexKeys(tableRecord, indexKeyMap)
				if err != nil {
					instr.cancelDrainer(fmt.Errorf("cannot build index keys for table %s: [%s]", node.TableCreator.Name, err.Error()))
					return instr.waitForDrainer()
				}

				instr.add(tableRecord, indexKeyMap)
				bs.RowsWritten++
			}
		}

		instr.doneSending()
		return instr.waitForDrainer()
	}

	var curStartLeftTokenRowIds []int64
	for {
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
			return bs, err
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
		customProcBatchStartTime := time.Now()

		if err = node.CustomProcessor.(CustomProcessorRunner).Run(logger, pCtx, rsIn, flushVarsArrayCallback); err != nil {
			return bs, err
		}

		custProcDur := time.Since(customProcBatchStartTime)
		logger.InfoCtx(pCtx, "CustomProcessor: %d items in %v (%.0f items/s)", rsIn.RowCount, custProcDur, float64(rsIn.RowCount)/custProcDur.Seconds())

		// The frequency of this hearbeat may not be enough, even for small rowset_size:
		// Python calculations even for one row may take, say, a minute
		// We cannot make heartbeats more granular here (each rowset is handled as a single Python command),
		// so our last resort is probably to increase the dead hearbeat timeout in CapiMQ
		instr.PCtx.SendHeartbeat()

		bs.RowsRead += rsIn.RowCount

		// We are tempted to "if rs.RowCount < srcBatchSize break", here but do not do that:
		// because of the rowid overlapping/epilogue logic, selectBatchFromTableByToken returns less rows than rs capacity

	} // for each source table batch

	bs.UpdateElapsedStats(time.Since(totalStartTime), instr)
	reportWriteTableComplete(logger, pCtx, bs.RowsRead, bs.RowsWritten, bs.Elapsed, len(node.TableCreator.Indexes), instr.NumWorkers)

	return bs, nil
}
