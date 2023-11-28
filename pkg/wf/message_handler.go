package wf

import (
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/dpc"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"go.uber.org/zap"
)

func checkDependencyNodesReady(logger *l.Logger, pCtx *ctx.MessageProcessingContext) (sc.ReadyToRunNodeCmdType, int16, int16, error) {
	logger.PushF("wf.checkDependencyNodesReady")
	defer logger.PopF()

	depNodeNames := make([]string, 2)
	depNodeCount := 0
	if pCtx.CurrentScriptNode.HasTableReader() {
		tableToReadFrom := pCtx.CurrentScriptNode.TableReader.TableName
		nodeToReadFrom, ok := pCtx.Script.TableCreatorNodeMap[tableToReadFrom]
		if !ok {
			return sc.NodeNone, 0, 0, fmt.Errorf("cannot find the node that creates reader table [%s]", tableToReadFrom)
		}
		depNodeNames[depNodeCount] = nodeToReadFrom.Name
		depNodeCount++
	}
	if pCtx.CurrentScriptNode.HasLookup() {
		tableToReadFrom := pCtx.CurrentScriptNode.Lookup.TableCreator.Name
		nodeToReadFrom, ok := pCtx.Script.TableCreatorNodeMap[tableToReadFrom]
		if !ok {
			return sc.NodeNone, 0, 0, fmt.Errorf("cannot find the node that creates lookup table [%s]", tableToReadFrom)
		}
		depNodeNames[depNodeCount] = nodeToReadFrom.Name
		depNodeCount++
	}

	if depNodeCount == 0 {
		return sc.NodeGo, 0, 0, nil
	}

	depNodeNames = depNodeNames[:depNodeCount]

	nodeEventListMap, err := wfdb.BuildDependencyNodeEventLists(logger, pCtx, depNodeNames)
	if err != nil {
		return sc.NodeNone, 0, 0, err
	}

	logger.DebugCtx(pCtx, "nodeEventListMap %v", nodeEventListMap)

	dependencyNodeCmds := make([]sc.ReadyToRunNodeCmdType, len(depNodeNames))
	dependencyRunIds := make([]int16, len(depNodeNames))
	for nodeIdx, depNodeName := range depNodeNames {
		if len(nodeEventListMap[depNodeName]) == 0 {
			return sc.NodeNogo, 0, 0, fmt.Errorf("target node %s, dep node %s not started yet, whoever started this run, failed to specify %s (or at least one of its dependencies) as start node", pCtx.CurrentScriptNode.Name, depNodeName, depNodeName)
		}
		var checkerLogMsg string
		dependencyNodeCmds[nodeIdx], dependencyRunIds[nodeIdx], checkerLogMsg, err = dpc.CheckDependencyPolicyAgainstNodeEventList(pCtx.CurrentScriptNode.DepPolDef, nodeEventListMap[depNodeName])
		if len(checkerLogMsg) > 0 {
			logger.Debug(checkerLogMsg)
		}
		if err != nil {
			return sc.NodeNone, 0, 0, err
		}
		logger.DebugCtx(pCtx, "target node %s, dep node %s returned %s", pCtx.CurrentScriptNode.Name, depNodeName, dependencyNodeCmds[nodeIdx])
	}

	finalCmd := dependencyNodeCmds[0]
	finalRunIdReader := dependencyRunIds[0]
	finalRunIdLookup := int16(0)
	if len(dependencyNodeCmds) == 2 {
		finalRunIdLookup = dependencyRunIds[1]
		if dependencyNodeCmds[0] == sc.NodeNogo || dependencyNodeCmds[1] == sc.NodeNogo {
			finalCmd = sc.NodeNogo
		} else if dependencyNodeCmds[0] == sc.NodeWait || dependencyNodeCmds[1] == sc.NodeWait {
			finalCmd = sc.NodeWait
		} else {
			finalCmd = sc.NodeGo
		}
	}

	if finalCmd == sc.NodeNogo || finalCmd == sc.NodeGo {
		logger.InfoCtx(pCtx, "checked all dependency nodes for %s, commands are %v, run ids are %v, finalCmd is %s", pCtx.CurrentScriptNode.Name, dependencyNodeCmds, dependencyRunIds, finalCmd)
	} else {
		logger.DebugCtx(pCtx, "checked all dependency nodes for %s, commands are %v, run ids are %v, finalCmd is wait", pCtx.CurrentScriptNode.Name, dependencyNodeCmds, dependencyRunIds)
	}

	return finalCmd, finalRunIdReader, finalRunIdLookup, nil
}

func SafeProcessBatch(envConfig *env.EnvConfig, logger *l.Logger, pCtx *ctx.MessageProcessingContext, readerNodeRunId int16, lookupNodeRunId int16) (wfmodel.NodeBatchStatusType, proc.BatchStats, error) {
	logger.PushF("wf.SafeProcessBatch")
	defer logger.PopF()

	var bs proc.BatchStats
	var err error

	switch pCtx.CurrentScriptNode.Type {
	case sc.NodeTypeFileTable:
		if pCtx.BatchInfo.FirstToken != pCtx.BatchInfo.LastToken || pCtx.BatchInfo.FirstToken < 0 || pCtx.BatchInfo.FirstToken >= int64(len(pCtx.CurrentScriptNode.FileReader.SrcFileUrls)) {
			err = fmt.Errorf(
				"startToken %d must equal endToken %d must be smaller than the number of files specified by file reader %d",
				pCtx.BatchInfo.FirstToken,
				pCtx.BatchInfo.LastToken,
				len(pCtx.CurrentScriptNode.FileReader.SrcFileUrls))
		} else {
			bs, err = proc.RunReadFileForBatch(envConfig, logger, pCtx, int(pCtx.BatchInfo.FirstToken))
		}

	case sc.NodeTypeTableTable:
		bs, err = proc.RunCreateTableForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	case sc.NodeTypeTableLookupTable:
		bs, err = proc.RunCreateTableRelForBatch(envConfig, logger, pCtx, readerNodeRunId, lookupNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	case sc.NodeTypeTableFile:
		bs, err = proc.RunCreateFile(envConfig, logger, pCtx, readerNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	case sc.NodeTypeTableCustomTfmTable:
		bs, err = proc.RunCreateTableForCustomProcessorForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	default:
		err = fmt.Errorf("unsupported node %s type %s", pCtx.CurrentScriptNode.Name, pCtx.CurrentScriptNode.Type)
	}

	if err != nil {
		logger.DebugCtx(pCtx, "batch processed, error: %s", err.Error())
		return wfmodel.NodeBatchFail, bs, fmt.Errorf("error running node %s of type %s in the script [%s]: [%s]", pCtx.CurrentScriptNode.Name, pCtx.CurrentScriptNode.Type, pCtx.BatchInfo.ScriptURI, err.Error())
	} else {
		logger.DebugCtx(pCtx, "batch processed ok")
	}

	return wfmodel.NodeBatchSuccess, bs, nil
}

func UpdateNodeStatusFromBatches(logger *l.Logger, pCtx *ctx.MessageProcessingContext) (wfmodel.NodeBatchStatusType, bool, error) {
	logger.PushF("wf.UpdateNodeStatusFromBatches")
	defer logger.PopF()

	// Check all batches for this run/node, mark node complete if needed
	totalNodeStatus, err := wfdb.HarvestBatchStatusesForNode(logger, pCtx)
	if err != nil {
		return wfmodel.NodeBatchNone, false, err
	}

	if totalNodeStatus == wfmodel.NodeBatchFail || totalNodeStatus == wfmodel.NodeBatchSuccess || totalNodeStatus == wfmodel.NodeBatchRunStopReceived {
		// Node processing completed, mark whole node as complete
		var comment string
		switch totalNodeStatus {
		case wfmodel.NodeBatchSuccess:
			comment = "completed - all batches ok"
		case wfmodel.NodeBatchFail:
			comment = "completed with some failed batches - check batch history"
		case wfmodel.NodeBatchRunStopReceived:
			comment = "run was stopped,check run and batch history"
		}

		isApplied, err := wfdb.SetNodeStatus(logger, pCtx, totalNodeStatus, comment)
		if err != nil {
			return wfmodel.NodeBatchNone, false, err
		} else {
			return totalNodeStatus, isApplied, nil
		}
	}

	return totalNodeStatus, false, nil
}

func UpdateRunStatusFromNodes(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("wf.UpdateRunStatusFromNodes")
	defer logger.PopF()

	// Let's see if this run is complete
	affectedNodes, err := wfdb.GetRunAffectedNodes(logger, pCtx.CqlSession, pCtx.BatchInfo.DataKeyspace, pCtx.BatchInfo.RunId)
	if err != nil {
		return err
	}
	combinedNodeStatus, nodeStatusString, err := wfdb.HarvestNodeStatusesForRun(logger, pCtx, affectedNodes)
	if err != nil {
		return err
	}

	if combinedNodeStatus == wfmodel.NodeBatchSuccess || combinedNodeStatus == wfmodel.NodeBatchFail {
		// Mark run as complete
		if err := wfdb.SetRunStatus(logger, pCtx.CqlSession, pCtx.BatchInfo.DataKeyspace, pCtx.BatchInfo.RunId, wfmodel.RunComplete, nodeStatusString, cql.IgnoreIfExists); err != nil {
			return err
		}
	}

	return nil
}

func refreshNodeAndRunStatus(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("wf.refreshNodeAndRunStatus")
	defer logger.PopF()

	_, _, err := UpdateNodeStatusFromBatches(logger, pCtx)
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot refresh run/node status: %s", err.Error())
		return err
	} else {
		// Ideally, we should run the code below only if isApplied signaled something changed. But, there is a possibility
		// that on the previous attempt, node status was updated and the daemon crashed right after that.
		// We need to pick it up from there and refresh run status anyways.
		err := UpdateRunStatusFromNodes(logger, pCtx)
		if err != nil {
			logger.ErrorCtx(pCtx, "cannot refresh run status: %s", err.Error())
			return err
		}
	}
	return nil
}

func ProcessDataBatchMsg(envConfig *env.EnvConfig, logger *l.Logger, msgTs int64, dataBatchInfo *wfmodel.MessagePayloadDataBatch) DaemonCmdType {
	logger.PushF("wf.ProcessDataBatchMsg")
	defer logger.PopF()

	pCtx := &ctx.MessageProcessingContext{
		MsgTs:           msgTs,
		BatchInfo:       *dataBatchInfo,
		ZapDataKeyspace: zap.String("ks", dataBatchInfo.DataKeyspace),
		ZapRun:          zap.Int16("run", dataBatchInfo.RunId),
		ZapNode:         zap.String("node", dataBatchInfo.TargetNodeName),
		ZapBatchIdx:     zap.Int16("bi", dataBatchInfo.BatchIdx),
		ZapMsgAgeMillis: zap.Int64("age", time.Now().UnixMilli()-msgTs)}

	var err error
	var initProblem sc.ScriptInitProblemType
	pCtx.Script, err, initProblem = sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, dataBatchInfo.ScriptURI, dataBatchInfo.ScriptParamsURI, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if initProblem != sc.ScriptInitNoProblem {
		switch initProblem {
		case sc.ScriptInitUrlProblem:
			logger.Error("cannot init script because of URI problem, will not let other workers handle it, giving up with msg %s: %s", dataBatchInfo.ToString(), err.Error())
			return DaemonCmdAckWithError
		case sc.ScriptInitContentProblem:
			logger.Error("cannot init script because of content problem, will not let other workers handle it, giving up with msg %s: %s", dataBatchInfo.ToString(), err.Error())
			return DaemonCmdAckWithError
		case sc.ScriptInitConnectivityProblem:
			logger.Error("cannot init script because of connectivity problem, will let other workers handle it, giving up with msg %s: %s", dataBatchInfo.ToString(), err.Error())
			return DaemonCmdRejectAndRetryLater
		default:
			logger.Error("unexpected: cannot init script for unknown reason %d, will let other workers handle it, giving up with msg %s: %s", initProblem, dataBatchInfo.ToString(), err.Error())
			return DaemonCmdRejectAndRetryLater
		}
	}

	var ok bool
	pCtx.CurrentScriptNode, ok = pCtx.Script.ScriptNodes[dataBatchInfo.TargetNodeName]
	if !ok {
		logger.Error("cannot find node %s in the script [%s], giving up with %s, returning DaemonCmdAckWithError, will not let other workers handle it", pCtx.BatchInfo.TargetNodeName, pCtx.BatchInfo.ScriptURI, dataBatchInfo.ToString())
		return DaemonCmdAckWithError
	}

	if err := pCtx.DbConnect(envConfig); err != nil {
		logger.Error("cannot connect to db: %s", err.Error())
		return DaemonCmdReconnectDb
	}
	defer pCtx.DbClose()

	logger.DebugCtx(pCtx, "started processing batch %s", dataBatchInfo.FullBatchId())

	runStatus, err := wfdb.GetCurrentRunStatus(logger, pCtx)
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot get current run status for batch %s: %s", dataBatchInfo.FullBatchId(), err.Error())
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	if runStatus == wfmodel.RunNone {
		comment := fmt.Sprintf("run history status for batch %s is empty, looks like this run %d was never started", dataBatchInfo.FullBatchId(), pCtx.BatchInfo.RunId)
		logger.ErrorCtx(pCtx, comment)
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchRunStopReceived, comment); err != nil {
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}
		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}
		return DaemonCmdAckWithError
	}

	// If the user signaled stop to this proc, all results of the run are invalidated
	if runStatus == wfmodel.RunStop {
		comment := fmt.Sprintf("run stopped, batch %s marked %s", dataBatchInfo.FullBatchId(), wfmodel.NodeBatchRunStopReceived.ToString())
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchRunStopReceived, comment); err != nil {
			logger.ErrorCtx(pCtx, fmt.Sprintf("%s, cannot set batch status: %s", comment, err.Error()))
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}

		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
			logger.ErrorCtx(pCtx, fmt.Sprintf("%s, cannot refresh status: %s", comment, err.Error()))
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}

		logger.DebugCtx(pCtx, fmt.Sprintf("%s, status successfully refreshed", comment))
		return DaemonCmdAckSuccess
	} else if runStatus != wfmodel.RunStart {
		logger.ErrorCtx(pCtx, "cannot process batch %s, run already has unexpected status %d", dataBatchInfo.FullBatchId(), runStatus)
		return DaemonCmdAckWithError
	}

	// Check if this run/node/batch has been handled already
	lastBatchStatus, err := wfdb.HarvestLastStatusForBatch(logger, pCtx)
	if err != nil {
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	if lastBatchStatus == wfmodel.NodeBatchFail || lastBatchStatus == wfmodel.NodeBatchSuccess {
		logger.InfoCtx(pCtx, "will not process batch %s, it has been already processed (processor crashed after processing it and before marking as success/fail?) with status %d", dataBatchInfo.FullBatchId(), lastBatchStatus)
		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}
		return DaemonCmdAckSuccess
	} else if lastBatchStatus == wfmodel.NodeBatchStart {
		// This run/node/batch has been picked up by another crashed processor (processor crashed before marking success/fail)
		if pCtx.CurrentScriptNode.RerunPolicy == sc.NodeRerun {
			if err := proc.DeleteDataAndUniqueIndexesByBatchIdx(logger, pCtx); err != nil {
				comment := fmt.Sprintf("cannot clean up leftovers of the previous processing of batch %s: %s", pCtx.BatchInfo.FullBatchId(), err.Error())
				logger.ErrorCtx(pCtx, comment)
				wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeFail, comment)
				if db.IsDbConnError(err) {
					return DaemonCmdReconnectDb
				}
				return DaemonCmdAckWithError
			}
			// Clean up successful, process this node
		} else if pCtx.CurrentScriptNode.RerunPolicy == sc.NodeFail {
			logger.ErrorCtx(pCtx, "will not rerun %s, rerun policy says we have to fail", pCtx.BatchInfo.FullBatchId())
			return DaemonCmdAckWithError
		} else {
			logger.ErrorCtx(pCtx, "unexpected rerun policy %s, looks like dev error", pCtx.CurrentScriptNode.RerunPolicy)
			return DaemonCmdAckWithError
		}
	} else if lastBatchStatus != wfmodel.NodeBatchNone {
		logger.ErrorCtx(pCtx, "unexpected batch %s status %d, expected None, looks like dev error.", pCtx.BatchInfo.FullBatchId(), lastBatchStatus)
		return DaemonCmdAckWithError
	}

	// Here, we are assuming this batch processing either never started or was started and then abandoned

	// Check if we have dependency nodes ready
	nodeReady, readerNodeRunId, lookupNodeRunId, err := checkDependencyNodesReady(logger, pCtx)
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot verify dependency nodes status for %s: %s", pCtx.BatchInfo.FullBatchId(), err.Error())
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	if nodeReady == sc.NodeNogo {
		comment := fmt.Sprintf("some dependency nodes for %s are in bad state, or runs executing dependency nodes were stopped/invalidated, will not run this node; for details, check rules in dependency_policies and previous runs history", pCtx.BatchInfo.FullBatchId())
		logger.InfoCtx(pCtx, comment)
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeFail, comment); err != nil {
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}

		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
		}
		return DaemonCmdAckWithError

	} else if nodeReady == sc.NodeWait {
		logger.InfoCtx(pCtx, "some dependency nodes for %s are not ready, will wait", pCtx.BatchInfo.FullBatchId())
		return DaemonCmdRejectAndRetryLater
	}

	// Here, we are ready to actually process the node

	if _, err := wfdb.SetNodeStatus(logger, pCtx, wfmodel.NodeStart, "started"); err != nil {
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeStart, ""); err != nil {
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	batchStatus, batchStats, batchErr := SafeProcessBatch(envConfig, logger, pCtx, readerNodeRunId, lookupNodeRunId)

	// TODO: test only!!!
	// if pCtx.BatchInfo.TargetNodeName == "order_item_date_inner" && pCtx.BatchInfo.BatchIdx == 3 {
	// 	rnd := rand.New(rand.NewSource(time.Now().UnixMilli()))
	// 	if rnd.Float32() < .5 {
	// 		logger.InfoCtx(pCtx, "ProcessBatchWithStatus: test error")
	// 		return DaemonCmdRejectAndRetryLater
	// 	}
	// }

	if batchErr != nil {
		logger.ErrorCtx(pCtx, "ProcessBatchWithStatus: %s", batchErr.Error())
		if db.IsDbConnError(batchErr) {
			return DaemonCmdReconnectDb
		}
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchFail, batchErr.Error()); err != nil {
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}
	} else {
		logger.InfoCtx(pCtx, "ProcessBatchWithStatus: success")
		if err := wfdb.SetBatchStatus(logger, pCtx, batchStatus, batchStats.ToString()); err != nil {
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}
	}

	if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	return DaemonCmdAckSuccess
}
