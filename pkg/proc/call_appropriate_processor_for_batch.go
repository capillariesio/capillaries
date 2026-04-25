package proc

import (
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type TableRecordPtr *map[string]any
type TableRecordBatch []TableRecordPtr

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

func CallAppropriateProcessorForBatch(envConfig *env.EnvConfig, logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, readerNodeRunId int16, lookupNodeRunId int16) (wfmodel.NodeBatchStatusType, BatchStats, error) {
	logger.PushF("proc.CallAppropriateProcessorForBatch")
	defer logger.PopF()

	var bs BatchStats
	var err error

	switch pCtx.CurrentScriptNode.Type {
	case sc.NodeTypeFileTable:
		if pCtx.Msg.FirstToken != pCtx.Msg.LastToken || pCtx.Msg.FirstToken < 0 || pCtx.Msg.FirstToken >= int64(len(pCtx.CurrentScriptNode.FileReader.SrcFileUrls)) {
			err = fmt.Errorf(
				"startToken %d must equal endToken %d and must be smaller than the number of files specified by file reader %d",
				pCtx.Msg.FirstToken,
				pCtx.Msg.LastToken,
				len(pCtx.CurrentScriptNode.FileReader.SrcFileUrls))
		} else {
			bs, err = runReadFileForBatch(envConfig, logger, pCtx, int(pCtx.Msg.FirstToken))
		}

	case sc.NodeTypeTableTable:
		bs, err = runCreateTableForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeDistinctTable:
		bs, err = runCreateDistinctTableForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeTableLookupTable:
		bs, err = runCreateTableRelForBatch(envConfig, logger, pCtx, readerNodeRunId, lookupNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeTableFile:
		bs, err = runCreateFile(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeTableCustomTfmTable:
		bs, err = runCreateTableForCustomProcessorForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	default:
		err = fmt.Errorf("unsupported node %s type %s", pCtx.Msg.TargetNodeName, pCtx.CurrentScriptNode.Type)
	}

	if err != nil {
		logger.DebugCtx(pCtx, "batch processed, error: %s", err.Error())
		return wfmodel.NodeBatchFail, bs, fmt.Errorf("error running node %s of type %s in the script [%s]: [%s]", pCtx.Msg.TargetNodeName, pCtx.CurrentScriptNode.Type, pCtx.Msg.ScriptURL, err.Error())
	}
	logger.DebugCtx(pCtx, "batch processed ok")

	return wfmodel.NodeBatchSuccess, bs, nil
}
