package wf

import (
	"encoding/json"
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
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const CachedNodeStateFormat string = "%s %d %d"

var (
	NodeDependencyReadynessHitCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_node_dep_ready_cache_hit_count",
		Help: "Capillaries node dependencies readiness cache hits",
	})
	NodeDependencyReadynessMissCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_node_dep_ready_cache_miss_count",
		Help: "Capillaries node dependencies readiness cache misses",
	})
	NodeDependencyReadynessGetDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "capi_node_dep_ready_get_duration",
		Help:    "Duration of checkDependencyNodesReady",
		Buckets: prometheus.ExponentialBuckets(0.001, 10.0, 4),
	})
	NodeDependencyNoneCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_node_dep_none_count",
		Help: "Capillaries node dependencies NodeNone count",
	})
	NodeDependencyWaitCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_node_dep_wait_count",
		Help: "Capillaries node dependencies NodeWait count",
	})
	NodeDependencyGoCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_node_dep_go_count",
		Help: "Capillaries node dependencies NodeGo count",
	})
	NodeDependencyNogoCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_node_dep_nogo_count",
		Help: "Capillaries node dependencies NodeNogo count",
	})
)

var NodeDependencyReadynessCache *expirable.LRU[string, string]

func checkDependencyNodesReady(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) (sc.ReadyToRunNodeCmdType, int16, int16, error) {
	logger.PushF("wf.checkDependencyNodesReady")
	defer logger.PopF()

	// Before reading state db, check our cache
	nodeDependencyReadynessCacheKey := pCtx.BatchInfo.FullNodeId()
	if NodeDependencyReadynessCache != nil {
		cachedState, ok := NodeDependencyReadynessCache.Get(nodeDependencyReadynessCacheKey)
		if ok {
			var cachedNodeCmdStr string
			var cachedRunIdReader, cachedRunIdLookup int16
			partCount, err := fmt.Sscanf(cachedState, CachedNodeStateFormat, &cachedNodeCmdStr, &cachedRunIdReader, &cachedRunIdLookup)
			if err != nil {
				logger.WarnCtx(pCtx, "cannot parse nodecmd and node ids from %s (%s), proceeding with querying state db", cachedState, err.Error())
				NodeDependencyReadynessCache.Remove(nodeDependencyReadynessCacheKey)
			} else if partCount != 3 {
				logger.WarnCtx(pCtx, "cannot parse nodecmd and node ids from %s (parsed component count %d), proceeding with querying state db", cachedState, partCount)
				NodeDependencyReadynessCache.Remove(nodeDependencyReadynessCacheKey)
			} else {
				cachedNodeCmd, err := sc.ReadyToRunNodeCmdTypeFromString(cachedNodeCmdStr)
				if err == nil {
					NodeDependencyReadynessHitCounter.Inc()
					return cachedNodeCmd, cachedRunIdReader, cachedRunIdLookup, nil
				}
				logger.WarnCtx(pCtx, "invalid cached nodecmd %s (%s), proceeding with querying state db", cachedNodeCmdStr, err.Error())
				NodeDependencyReadynessCache.Remove(nodeDependencyReadynessCacheKey)
			}
		}
		NodeDependencyReadynessMissCounter.Inc()
	}

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

	startTime := time.Now()

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
			return sc.NodeNogo, 0, 0, fmt.Errorf("target node %s, dep node %s not started yet, whoever started this run, failed to specify %s (or at least one of its dependencies) as start node", pCtx.BatchInfo.TargetNodeName, depNodeName, depNodeName)
		}
		var checkerLogMsg string
		dependencyNodeCmds[nodeIdx], dependencyRunIds[nodeIdx], checkerLogMsg, err = dpc.CheckDependencyPolicyAgainstNodeEventList(pCtx.CurrentScriptNode.DepPolDef, nodeEventListMap[depNodeName])
		if len(checkerLogMsg) > 0 {
			logger.Debug("%s", checkerLogMsg)
		}
		if err != nil {
			return sc.NodeNone, 0, 0, err
		}
		logger.DebugCtx(pCtx, "target node %s, dep node %s returned %s", pCtx.BatchInfo.TargetNodeName, depNodeName, dependencyNodeCmds[nodeIdx])
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
		logger.InfoCtx(pCtx, "checked all dependency nodes for %s, commands are %v, run ids are %v, finalCmd is %s", pCtx.BatchInfo.TargetNodeName, dependencyNodeCmds, dependencyRunIds, finalCmd)
	} else {
		logger.DebugCtx(pCtx, "checked all dependency nodes for %s, commands are %v, run ids are %v, finalCmd is wait", pCtx.BatchInfo.TargetNodeName, dependencyNodeCmds, dependencyRunIds)
	}

	NodeDependencyReadynessGetDuration.Observe(float64(time.Since(startTime).Seconds()))

	// Update cache
	if NodeDependencyReadynessCache != nil {
		NodeDependencyReadynessCache.Add(nodeDependencyReadynessCacheKey, fmt.Sprintf(CachedNodeStateFormat, finalCmd, finalRunIdReader, finalRunIdLookup))
	}

	return finalCmd, finalRunIdReader, finalRunIdLookup, nil
}

func SafeProcessBatch(envConfig *env.EnvConfig, logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, readerNodeRunId int16, lookupNodeRunId int16) (wfmodel.NodeBatchStatusType, proc.BatchStats, error) {
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

	case sc.NodeTypeDistinctTable:
		bs, err = proc.RunCreateDistinctTableForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	case sc.NodeTypeTableLookupTable:
		bs, err = proc.RunCreateTableRelForBatch(envConfig, logger, pCtx, readerNodeRunId, lookupNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	case sc.NodeTypeTableFile:
		bs, err = proc.RunCreateFile(envConfig, logger, pCtx, readerNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	case sc.NodeTypeTableCustomTfmTable:
		bs, err = proc.RunCreateTableForCustomProcessorForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.BatchInfo.FirstToken, pCtx.BatchInfo.LastToken)

	default:
		err = fmt.Errorf("unsupported node %s type %s", pCtx.BatchInfo.TargetNodeName, pCtx.CurrentScriptNode.Type)
	}

	if err != nil {
		logger.DebugCtx(pCtx, "batch processed, error: %s", err.Error())
		return wfmodel.NodeBatchFail, bs, fmt.Errorf("error running node %s of type %s in the script [%s]: [%s]", pCtx.BatchInfo.TargetNodeName, pCtx.CurrentScriptNode.Type, pCtx.BatchInfo.ScriptURL, err.Error())
	}
	logger.DebugCtx(pCtx, "batch processed ok")

	return wfmodel.NodeBatchSuccess, bs, nil
}

func UpdateNodeStatusFromBatches(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) (wfmodel.NodeBatchStatusType, bool, error) {
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
		}

		return totalNodeStatus, isApplied, nil
	}

	return totalNodeStatus, false, nil
}

func UpdateRunStatusFromNodes(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
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

func refreshNodeAndRunStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("wf.refreshNodeAndRunStatus")
	defer logger.PopF()

	_, _, err := UpdateNodeStatusFromBatches(logger, pCtx)
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot refresh run/node status: %s", err.Error())
		return err
	}

	// Ideally, we should run the code below only if isApplied signaled something changed. But, there is a possibility
	// that on the previous attempt, node status was updated and the daemon crashed right after that.
	// We need to pick it up from there and refresh run status anyways.
	err = UpdateRunStatusFromNodes(logger, pCtx)
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot refresh run status: %s", err.Error())
		return err
	}

	return nil
}

func initCtxScript(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, caPath string, privateKeys map[string]string, dataBatchInfo *wfmodel.MessagePayloadDataBatch, customProcFactory sc.CustomProcessorDefFactory, customProcSettings map[string]json.RawMessage) DaemonCmdType {
	var initProblem sc.ScriptInitProblemType
	var err error

	pCtx.Script, initProblem, err = sc.NewScriptFromFiles(caPath, privateKeys, dataBatchInfo.ScriptURL, dataBatchInfo.ScriptParamsURL, customProcFactory, customProcSettings)
	if initProblem == sc.ScriptInitNoProblem {
		return DaemonCmdNone
	}

	switch initProblem {
	case sc.ScriptInitUrlProblem:
		logger.Error("cannot init script because of URL problem, will not let other workers handle it, giving up with msg %s: %s", dataBatchInfo.ToString(), err.Error())
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

func checkRunStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, dataBatchInfo *wfmodel.MessagePayloadDataBatch, runStatus wfmodel.RunStatusType) DaemonCmdType {
	if runStatus == wfmodel.RunNone {
		comment := fmt.Sprintf("run history status for batch %s is empty, looks like this run %d was never started", dataBatchInfo.FullBatchId(), pCtx.BatchInfo.RunId)
		logger.ErrorCtx(pCtx, "%s", comment)
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchFail, comment); err != nil {
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
			logger.ErrorCtx(pCtx, "%s, cannot set batch status: %s", comment, err.Error())
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}

		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
			logger.ErrorCtx(pCtx, "%s, cannot refresh status: %s", comment, err.Error())
			if db.IsDbConnError(err) {
				return DaemonCmdReconnectDb
			}
			return DaemonCmdAckWithError
		}

		logger.DebugCtx(pCtx, "%s, status successfully refreshed", comment)
		return DaemonCmdAckSuccess
	} else if runStatus != wfmodel.RunStart {
		logger.ErrorCtx(pCtx, "cannot process batch %s, run already has unexpected status %d", dataBatchInfo.FullBatchId(), runStatus)
		return DaemonCmdAckWithError
	}

	return DaemonCmdNone
}

func checkLastBatchStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, dataBatchInfo *wfmodel.MessagePayloadDataBatch, lastBatchStatus wfmodel.NodeBatchStatusType) DaemonCmdType {
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
			if deleteErr := proc.DeleteDataAndUniqueIndexesByBatchIdx(logger, pCtx); deleteErr != nil {
				comment := fmt.Sprintf("cannot clean up leftovers of the previous processing of batch %s: %s", pCtx.BatchInfo.FullBatchId(), deleteErr.Error())
				logger.ErrorCtx(pCtx, "%s", comment)
				setBatchStatusErr := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeFail, comment)
				if setBatchStatusErr != nil {
					comment += fmt.Sprintf("; cannot set batch status: %s", setBatchStatusErr.Error())
					logger.ErrorCtx(pCtx, "%s", comment)
				}
				if db.IsDbConnError(deleteErr) {
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
	} else if lastBatchStatus == wfmodel.NodeBatchRunStopReceived {
		// Stop was signaled, do not try to handle this batch anymore, call it a success
		return DaemonCmdAckWithError
	} else if lastBatchStatus != wfmodel.NodeBatchNone {
		logger.ErrorCtx(pCtx, "unexpected batch %s status %d, expected None, looks like dev error.", pCtx.BatchInfo.FullBatchId(), lastBatchStatus)
		return DaemonCmdAckWithError
	}
	return DaemonCmdNone
}

func checkDependencyNogoOrWait(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, nodeReady sc.ReadyToRunNodeCmdType) DaemonCmdType {
	switch nodeReady {
	case sc.NodeNogo:
		comment := fmt.Sprintf("some dependency nodes for %s are in bad state, or runs executing dependency nodes were stopped/invalidated, will not run this node; for details, check rules in dependency_policies and previous runs history", pCtx.BatchInfo.FullBatchId())
		logger.InfoCtx(pCtx, "%s", comment)
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

	case sc.NodeWait:
		logger.InfoCtx(pCtx, "some dependency nodes for %s are not ready, will wait", pCtx.BatchInfo.FullBatchId())
		return DaemonCmdRejectAndRetryLater
	default:
		return DaemonCmdNone
	}
}

func ProcessDataBatchMsg(envConfig *env.EnvConfig, logger *l.CapiLogger, msgTs int64, dataBatchInfo *wfmodel.MessagePayloadDataBatch) DaemonCmdType {
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

	// Check run status first. If it's stopped, don't even bother getting the script etc. If we try to get the script first,
	// and it's not available, we may end up handling this batch forever even after the run is stopped by the operator
	if err := pCtx.DbConnect(envConfig); err != nil {
		logger.Error("cannot connect to db: %s", err.Error())
		return DaemonCmdReconnectDb
	}
	defer pCtx.DbClose()

	runStatus, err := wfdb.GetCurrentRunStatus(logger, pCtx)
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot get current run status for batch %s: %s", dataBatchInfo.FullBatchId(), err.Error())
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	// Check current run is valid
	if daemonCmd := checkRunStatus(logger, pCtx, dataBatchInfo, runStatus); daemonCmd != DaemonCmdNone {
		return daemonCmd
	}

	if daemonCmd := initCtxScript(logger, pCtx, envConfig.CaPath, envConfig.PrivateKeys, dataBatchInfo, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings); daemonCmd != DaemonCmdNone {
		return daemonCmd
	}

	var ok bool
	pCtx.CurrentScriptNode, ok = pCtx.Script.ScriptNodes[dataBatchInfo.TargetNodeName]
	if !ok {
		logger.Error("cannot find node %s in the script [%s], giving up with %s, returning DaemonCmdAckWithError, will not let other workers handle it", pCtx.BatchInfo.TargetNodeName, pCtx.BatchInfo.ScriptURL, dataBatchInfo.ToString())
		return DaemonCmdAckWithError
	}

	logger.DebugCtx(pCtx, "started processing batch %s", dataBatchInfo.FullBatchId())

	lastBatchStatus, err := wfdb.HarvestLastStatusForBatch(logger, pCtx)
	if err != nil {
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	// Check if this run/node/batch has been handled already
	if daemonCmd := checkLastBatchStatus(logger, pCtx, dataBatchInfo, lastBatchStatus); daemonCmd != DaemonCmdNone {
		return daemonCmd
	}

	// At this point, we are assuming this batch processing either never started or was started and then abandoned

	nodeReady, readerNodeRunId, lookupNodeRunId, err := checkDependencyNodesReady(logger, pCtx)
	switch nodeReady {
	case sc.NodeNone:
		NodeDependencyNoneCounter.Inc()
	case sc.NodeWait:
		NodeDependencyWaitCounter.Inc()
	case sc.NodeGo:
		NodeDependencyGoCounter.Inc()
	case sc.NodeNogo:
		NodeDependencyNogoCounter.Inc()
	}
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot verify dependency nodes status for %s: %s", pCtx.BatchInfo.FullBatchId(), err.Error())
		if db.IsDbConnError(err) {
			return DaemonCmdReconnectDb
		}
		return DaemonCmdAckWithError
	}

	if daemonCmd := checkDependencyNogoOrWait(logger, pCtx, nodeReady); daemonCmd != DaemonCmdNone {
		return daemonCmd
	}

	// At this point, we are ready to actually process the node

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
