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
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type ProcessDeliveryResultType int8

const (
	ProcessDeliveryOK                    ProcessDeliveryResultType = 0 // OK
	ProcessDeliveryAckSuccess            ProcessDeliveryResultType = 2 // Best case
	ProcessDeliveryRejectAndRetryLater   ProcessDeliveryResultType = 3 // Node dependencies are not ready, wait with processing this node
	ProcessDeliveryReconnectDb           ProcessDeliveryResultType = 4 // Db workflow error, try to reconnect
	ProcessDeliveryAckWithError          ProcessDeliveryResultType = 6 // There was a processing error: either some serious biz logic re-trying will not help, or it was a data table error (we consider it persistent), so ack it
	ProcessDeliveryReconnectQueueNotUsed ProcessDeliveryResultType = 7 // Queue error, try to reconnect
)

func (result ProcessDeliveryResultType) ToString() string {
	switch result {
	case ProcessDeliveryOK:
		return "ok"
	case ProcessDeliveryAckSuccess:
		return "success"
	case ProcessDeliveryRejectAndRetryLater:
		return "reject_and_retry"
	case ProcessDeliveryReconnectDb:
		return "reconnect_db"
	case ProcessDeliveryAckWithError:
		return "ack_with_error"
	case ProcessDeliveryReconnectQueueNotUsed:
		return "reconnect_queue"
	default:
		return "unknown"
	}
}

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
	ReceivedMsgCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_received_msg_count",
		Help: "Capillaries RabbitMQ received msg count",
	})
)

var NodeDependencyReadynessCache *expirable.LRU[string, string]

func checkDependencyNodesReady(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) (sc.ReadyToRunNodeCmdType, int16, int16, error) {
	logger.PushF("wf.checkDependencyNodesReady")
	defer logger.PopF()

	// Before reading state db, check our cache
	nodeDependencyReadynessCacheKey := pCtx.Msg.FullNodeId()
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
			return sc.NodeNogo, 0, 0, fmt.Errorf("target node %s, dep node %s not started yet, whoever started this run, failed to specify %s (or at least one of its dependencies) as start node", pCtx.Msg.TargetNodeName, depNodeName, depNodeName)
		}
		var checkerLogMsg string
		dependencyNodeCmds[nodeIdx], dependencyRunIds[nodeIdx], checkerLogMsg, err = dpc.CheckDependencyPolicyAgainstNodeEventList(pCtx.CurrentScriptNode.DepPolDef, nodeEventListMap[depNodeName])
		if len(checkerLogMsg) > 0 {
			logger.Debug("%s", checkerLogMsg)
		}
		if err != nil {
			return sc.NodeNone, 0, 0, err
		}
		logger.DebugCtx(pCtx, "target node %s, dep node %s returned %s", pCtx.Msg.TargetNodeName, depNodeName, dependencyNodeCmds[nodeIdx])
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
		logger.InfoCtx(pCtx, "checked all dependency nodes for %s, commands are %v, run ids are %v, finalCmd is %s", pCtx.Msg.TargetNodeName, dependencyNodeCmds, dependencyRunIds, finalCmd)
	} else {
		logger.DebugCtx(pCtx, "checked all dependency nodes for %s, commands are %v, run ids are %v, finalCmd is wait", pCtx.Msg.TargetNodeName, dependencyNodeCmds, dependencyRunIds)
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
		if pCtx.Msg.FirstToken != pCtx.Msg.LastToken || pCtx.Msg.FirstToken < 0 || pCtx.Msg.FirstToken >= int64(len(pCtx.CurrentScriptNode.FileReader.SrcFileUrls)) {
			err = fmt.Errorf(
				"startToken %d must equal endToken %d must be smaller than the number of files specified by file reader %d",
				pCtx.Msg.FirstToken,
				pCtx.Msg.LastToken,
				len(pCtx.CurrentScriptNode.FileReader.SrcFileUrls))
		} else {
			bs, err = proc.RunReadFileForBatch(envConfig, logger, pCtx, int(pCtx.Msg.FirstToken))
		}

	case sc.NodeTypeTableTable:
		bs, err = proc.RunCreateTableForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeDistinctTable:
		bs, err = proc.RunCreateDistinctTableForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeTableLookupTable:
		bs, err = proc.RunCreateTableRelForBatch(envConfig, logger, pCtx, readerNodeRunId, lookupNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeTableFile:
		bs, err = proc.RunCreateFile(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

	case sc.NodeTypeTableCustomTfmTable:
		bs, err = proc.RunCreateTableForCustomProcessorForBatch(envConfig, logger, pCtx, readerNodeRunId, pCtx.Msg.FirstToken, pCtx.Msg.LastToken)

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
		default:
			return wfmodel.NodeBatchNone, false, fmt.Errorf("unexpected totalNodeStatus %v", totalNodeStatus)
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
	affectedNodes, err := wfdb.GetRunAffectedNodes(logger, pCtx.CqlSession, pCtx.Msg.DataKeyspace, pCtx.Msg.RunId)
	if err != nil {
		return err
	}
	combinedNodeStatus, nodeStatusString, err := wfdb.HarvestNodeStatusesForRun(logger, pCtx, affectedNodes)
	if err != nil {
		return err
	}

	if combinedNodeStatus == wfmodel.NodeBatchSuccess || combinedNodeStatus == wfmodel.NodeBatchFail {
		// Mark run as complete
		if err := wfdb.SetRunStatus(logger, pCtx.CqlSession, pCtx.Msg.DataKeyspace, pCtx.Msg.RunId, wfmodel.RunComplete, nodeStatusString, cql.IgnoreIfExists); err != nil {
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

func initCtxScript(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, caPath string, privateKeys map[string]string, msg *wfmodel.Message, customProcFactory sc.CustomProcessorDefFactory, customProcSettings map[string]json.RawMessage) FurtherProcessingCmd {
	var initProblem sc.ScriptInitProblemType
	var err error
	pCtx.Script, initProblem, err = sc.NewScriptFromFiles(caPath, privateKeys, msg.ScriptURL, msg.ScriptParamsURL, customProcFactory, customProcSettings)
	if initProblem == sc.ScriptInitNoProblem {
		return FurtherProcessingProceed
	}
	switch initProblem {
	case sc.ScriptInitUrlProblem:
		logger.Error("cannot init script because of URL problem, will not let other workers handle it, giving up with msg %s: %s", msg.ToString(), err.Error())
		return FurtherProcessingAck
	case sc.ScriptInitContentProblem:
		logger.Error("cannot init script because of content problem, will not let other workers handle it, giving up with msg %s: %s", msg.ToString(), err.Error())
		return FurtherProcessingAck
	case sc.ScriptInitConnectivityProblem:
		logger.Error("cannot init script because of connectivity problem, will let other workers handle it, giving up with msg %s: %s", msg.ToString(), err.Error())
		return FurtherProcessingRetry
	default:
		logger.Error("unexpected: cannot init script for unknown reason %d, will let other workers handle it, giving up with msg %s: %s", initProblem, msg.ToString(), err.Error())
		return FurtherProcessingRetry
	}
}

type FurtherProcessingCmd int

const (
	FurtherProcessingProceed FurtherProcessingCmd = iota
	FurtherProcessingAck
	FurtherProcessingRetry
)

func checkRunStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, msg *wfmodel.Message, runStatus wfmodel.RunStatusType) FurtherProcessingCmd {
	switch runStatus {
	case wfmodel.RunNone:
		comment := fmt.Sprintf("run history status for batch %s is empty, looks like this run %d was never started, will ack with error", msg.FullBatchId(), pCtx.Msg.RunId)
		logger.ErrorCtx(pCtx, "%s", comment)
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchFail, comment); err != nil && db.IsDbConnError(err) {
			return FurtherProcessingRetry
		}
		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil && db.IsDbConnError(err) {
			return FurtherProcessingRetry
		}
		// Unexpected non-db error, embrace the problem, do not retry
		return FurtherProcessingAck

	case wfmodel.RunStop:
		// If the user signaled stop to this proc, all results of the run are invalidated
		comment := fmt.Sprintf("run stopped, batch %s marked %s", msg.FullBatchId(), wfmodel.NodeBatchRunStopReceived.ToString())
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchRunStopReceived, comment); err != nil {
			if db.IsDbConnError(err) {
				return FurtherProcessingRetry
			}
			// Unexpected non-db error, embrace the problem, do not retry
			return FurtherProcessingAck
		}
		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
			logger.ErrorCtx(pCtx, "%s, cannot refresh status: %s", comment, err.Error())
			if db.IsDbConnError(err) {
				return FurtherProcessingRetry
			}
			// Unexpected non-db error, embrace the problem, do not retry
			return FurtherProcessingAck
		}
		logger.DebugCtx(pCtx, "%s, stop status successfully refreshed, no further processing needed", comment)
		return FurtherProcessingAck

	case wfmodel.RunStart:
		// Happy path
		return FurtherProcessingProceed

	default:
		logger.ErrorCtx(pCtx, "cannot process batch %s, run already has unexpected status %d", msg.FullBatchId(), runStatus)
		// Unexpected non-db error, embrace the problem, do not retry
		return FurtherProcessingAck
	}
}

func checkLastBatchStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, msg *wfmodel.Message, lastBatchStatus wfmodel.NodeBatchStatusType) FurtherProcessingCmd {
	switch lastBatchStatus {
	case wfmodel.NodeBatchFail, wfmodel.NodeBatchSuccess:
		logger.InfoCtx(pCtx, "will not process batch %s, it has been already processed (processor crashed after processing it and before marking as success/fail?) with status %d", msg.FullBatchId(), lastBatchStatus)
		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil && db.IsDbConnError(err) {
			return FurtherProcessingRetry
		}
		return FurtherProcessingAck

	case wfmodel.NodeBatchStart:
		// This run/node/batch has been picked up by another processor that crashed before marking success/fail
		switch pCtx.CurrentScriptNode.RerunPolicy {
		case sc.NodeRerun:
			if deleteErr := proc.DeleteDataAndUniqueIndexesByBatchIdx(logger, pCtx); deleteErr != nil {
				if db.IsDbConnError(deleteErr) {
					return FurtherProcessingRetry
				}
				comment := fmt.Sprintf("cannot clean up leftovers of the previous processing of batch %s, giving up, will try to set batch status to failed: %s", pCtx.Msg.FullBatchId(), deleteErr.Error())
				logger.ErrorCtx(pCtx, "%s", comment)
				if setBatchStatusErr := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeFail, comment); setBatchStatusErr != nil {
					logger.ErrorCtx(pCtx, "cannot set batch status: %s", setBatchStatusErr.Error())
				}
				return FurtherProcessingAck
			}
			// Clean up successful, process this node further
			return FurtherProcessingProceed

		case sc.NodeFail:
			logger.ErrorCtx(pCtx, "will not rerun %s, rerun policy says we have to fail", pCtx.Msg.FullBatchId())
			return FurtherProcessingAck

		default:
			logger.ErrorCtx(pCtx, "unexpected rerun policy %s, looks like dev error", pCtx.CurrentScriptNode.RerunPolicy)
			return FurtherProcessingAck
		}

	case wfmodel.NodeBatchRunStopReceived:
		// Stop was signaled, do not try to handle this batch anymore, call it a success
		return FurtherProcessingAck
	case wfmodel.NodeBatchNone:
		// Happy path
		return FurtherProcessingProceed
	default:
		logger.ErrorCtx(pCtx, "unexpected batch %s status %d", pCtx.Msg.FullBatchId(), lastBatchStatus)
		return FurtherProcessingAck
	}
}

func checkDependencyNogoOrWait(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, nodeReady sc.ReadyToRunNodeCmdType) FurtherProcessingCmd {
	switch nodeReady {
	case sc.NodeNogo:
		comment := fmt.Sprintf("some dependency nodes for %s are in bad state, or runs executing dependency nodes were stopped/invalidated, will not run this node; for details, check rules in dependency_policies and previous runs history", pCtx.Msg.FullBatchId())
		logger.InfoCtx(pCtx, "%s", comment)
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeFail, comment); err != nil {
			if db.IsDbConnError(err) {
				return FurtherProcessingRetry
			}
			return FurtherProcessingAck
		}
		if err := refreshNodeAndRunStatus(logger, pCtx); err != nil && db.IsDbConnError(err) {
			return FurtherProcessingRetry
		}
		return FurtherProcessingAck

	case sc.NodeWait:
		logger.InfoCtx(pCtx, "some dependency nodes for %s are not ready, will wait", pCtx.Msg.FullBatchId())
		return FurtherProcessingRetry

	default:
		return FurtherProcessingProceed
	}
}

func ProcessDataBatchMsgNew(envConfig *env.EnvConfig, logger *l.CapiLogger, msg *wfmodel.Message, heartbeatInterval int64, heartbeatCallback ctx.HeartbeatCallbackFunc) mq.AcknowledgerCmd {
	logger.PushF("wf.ProcessDataBatchMsg")
	defer logger.PopF()

	ReceivedMsgCounter.Inc()

	pCtx := &ctx.MessageProcessingContext{
		Msg:                     *msg,
		ZapMsgId:                zap.String("id", msg.Id),
		ZapDataKeyspace:         zap.String("ks", msg.DataKeyspace),
		ZapRun:                  zap.Int16("run", msg.RunId),
		ZapNode:                 zap.String("node", msg.TargetNodeName),
		ZapBatchIdx:             zap.Int16("bi", msg.BatchIdx),
		ZapMsgAgeMillis:         zap.Int64("age", time.Now().UnixMilli()-msg.Ts),
		LastHeartbeatTs:         time.Now().UnixMilli(),
		HeartbeatIntervalMillis: heartbeatInterval,
		HeartbeatCallback:       heartbeatCallback}

	// Check run status first. If it's stopped, don't even bother getting the script etc. If we try to get the script first,
	// and it's not available, we may end up handling this batch forever even after the run is stopped by the operator
	if err := pCtx.DbConnect(envConfig); err != nil {
		logger.Error("cannot connect to db: %s", err.Error())
		return mq.AcknowledgerCmdRetry
	}
	defer pCtx.DbClose()

	runStatus, err := wfdb.GetCurrentRunStatus(logger, pCtx)
	if err != nil {
		if db.IsDbConnError(err) {
			logger.ErrorCtx(pCtx, "cannot get current run status for batch %s, will let other instance to retry: %s", msg.FullBatchId(), err.Error())
			return mq.AcknowledgerCmdRetry
		}
		logger.ErrorCtx(pCtx, "cannot get current run status for batch %s, will ack with error: %s", msg.FullBatchId(), err.Error())
		return mq.AcknowledgerCmdAck
	}

	// Check current run is valid
	furtherProcCmd := checkRunStatus(logger, pCtx, msg, runStatus)
	switch furtherProcCmd {
	case FurtherProcessingRetry:
		return mq.AcknowledgerCmdRetry
	case FurtherProcessingAck:
		return mq.AcknowledgerCmdAck
	}

	// Script/params must be valid
	furtherProcCmd = initCtxScript(logger, pCtx, envConfig.CaPath, envConfig.PrivateKeys, msg, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	switch furtherProcCmd {
	case FurtherProcessingRetry:
		return mq.AcknowledgerCmdRetry
	case FurtherProcessingAck:
		return mq.AcknowledgerCmdAck
	}

	var ok bool
	pCtx.CurrentScriptNode, ok = pCtx.Script.ScriptNodes[msg.TargetNodeName]
	if !ok {
		logger.Error("cannot find node %s in the script [%s], giving up with %s, returning ProcessDeliveryAckWithError, will not let other workers handle it", pCtx.Msg.TargetNodeName, pCtx.Msg.ScriptURL, msg.ToString())
		return mq.AcknowledgerCmdAck
	}

	logger.DebugCtx(pCtx, "started processing batch %s", msg.FullBatchId())

	lastBatchStatus, err := wfdb.HarvestLastStatusForBatch(logger, pCtx)
	if err != nil {
		if db.IsDbConnError(err) {
			return mq.AcknowledgerCmdRetry
		}
		return mq.AcknowledgerCmdAck
	}

	// Check if this run/node/batch has been handled already
	furtherProcCmd = checkLastBatchStatus(logger, pCtx, msg, lastBatchStatus)
	switch furtherProcCmd {
	case FurtherProcessingRetry:
		return mq.AcknowledgerCmdRetry
	case FurtherProcessingAck:
		return mq.AcknowledgerCmdAck
	}

	// At this point, we are assuming this batch processing either never started or was started and then abandoned

	nodeReady, readerNodeRunId, lookupNodeRunId, err := checkDependencyNodesReady(logger, pCtx)
	if err != nil {
		logger.ErrorCtx(pCtx, "cannot verify dependency nodes status for %s: %s", pCtx.Msg.FullBatchId(), err.Error())
		if db.IsDbConnError(err) {
			return mq.AcknowledgerCmdRetry
		}
		return mq.AcknowledgerCmdAck
	}

	switch nodeReady {
	case sc.NodeNone:
		NodeDependencyNoneCounter.Inc()
	case sc.NodeWait:
		NodeDependencyWaitCounter.Inc()
	case sc.NodeGo:
		NodeDependencyGoCounter.Inc()
	case sc.NodeNogo:
		NodeDependencyNogoCounter.Inc()
	default:
		logger.ErrorCtx(pCtx, "unexpected nodeReady %v", nodeReady)
		return mq.AcknowledgerCmdAck
	}

	furtherProcCmd = checkDependencyNogoOrWait(logger, pCtx, nodeReady)
	switch furtherProcCmd {
	case FurtherProcessingRetry:
		return mq.AcknowledgerCmdRetry
	case FurtherProcessingAck:
		return mq.AcknowledgerCmdAck
	}

	// At this point, we are ready to actually process the node

	if _, err := wfdb.SetNodeStatus(logger, pCtx, wfmodel.NodeStart, "started"); err != nil {
		if db.IsDbConnError(err) {
			return mq.AcknowledgerCmdRetry
		}
		return mq.AcknowledgerCmdAck
	}

	if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeStart, ""); err != nil {
		if db.IsDbConnError(err) {
			return mq.AcknowledgerCmdRetry
		}
		return mq.AcknowledgerCmdAck
	}

	batchStatus, batchStats, batchErr := SafeProcessBatch(envConfig, logger, pCtx, readerNodeRunId, lookupNodeRunId)

	// TODO: test only!!!
	// if pCtx.BatchInfo.TargetNodeName == "order_item_date_inner" && pCtx.BatchInfo.BatchIdx == 3 {
	// 	rnd := rand.New(rand.NewSource(time.Now().UnixMilli()))
	// 	if rnd.Float32() < .5 {
	// 		logger.InfoCtx(pCtx, "ProcessBatchWithStatus: test error")
	// 		return mq.AcknowledgerCmdRetry
	// 	}
	// }

	if batchErr != nil {
		logger.ErrorCtx(pCtx, "ProcessBatchWithStatus: %s", batchErr.Error())
		if db.IsDbConnError(batchErr) {
			return mq.AcknowledgerCmdRetry
		}
		// There was some non-db error, report it in the failed batch status
		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchFail, batchErr.Error()); err != nil {
			if db.IsDbConnError(err) {
				return mq.AcknowledgerCmdRetry
			}
			return mq.AcknowledgerCmdAck
		}
		// Here: batch was processed with some non-db error
	} else {
		logger.InfoCtx(pCtx, "ProcessBatchWithStatus: success")
		if err := wfdb.SetBatchStatus(logger, pCtx, batchStatus, batchStats.ToString()); err != nil {
			if db.IsDbConnError(err) {
				return mq.AcknowledgerCmdRetry
			}
			return mq.AcknowledgerCmdAck
		}
	}

	if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
		if db.IsDbConnError(err) {
			return mq.AcknowledgerCmdRetry
		}
		return mq.AcknowledgerCmdAck
	}

	// Here: batch was processed with some non-db error, or successfully. Either way - ack
	return mq.AcknowledgerCmdAck
}

// func ProcessDataBatchMsg(envConfig *env.EnvConfig, logger *l.CapiLogger, msg *wfmodel.Message) ProcessDeliveryResultType {
// 	logger.PushF("wf.ProcessDataBatchMsg")
// 	defer logger.PopF()

// 	ReceivedMsgCounter.Inc()

// 	pCtx := &ctx.MessageProcessingContext{
// 		Msg:             *msg,
// 		ZapMsgId:        zap.String("id", msg.Id),
// 		ZapDataKeyspace: zap.String("ks", msg.DataKeyspace),
// 		ZapRun:          zap.Int16("run", msg.RunId),
// 		ZapNode:         zap.String("node", msg.TargetNodeName),
// 		ZapBatchIdx:     zap.Int16("bi", msg.BatchIdx),
// 		ZapMsgAgeMillis: zap.Int64("age", time.Now().UnixMilli()-msg.Ts)}

// 	// Check run status first. If it's stopped, don't even bother getting the script etc. If we try to get the script first,
// 	// and it's not available, we may end up handling this batch forever even after the run is stopped by the operator
// 	if err := pCtx.DbConnect(envConfig); err != nil {
// 		logger.Error("cannot connect to db: %s", err.Error())
// 		return ProcessDeliveryReconnectDb
// 	}
// 	defer pCtx.DbClose()

// 	runStatus, err := wfdb.GetCurrentRunStatus(logger, pCtx)
// 	if err != nil {
// 		logger.ErrorCtx(pCtx, "cannot get current run status for batch %s: %s", msg.FullBatchId(), err.Error())
// 		if db.IsDbConnError(err) {
// 			return ProcessDeliveryReconnectDb
// 		}
// 		return ProcessDeliveryAckWithError
// 	}

// 	// Check current run is valid
// 	if processDeliveryResult := checkRunStatus(logger, pCtx, msg, runStatus); processDeliveryResult != ProcessDeliveryOK {
// 		return processDeliveryResult
// 	}

// 	if processDeliveryResult := initCtxScript(logger, pCtx, envConfig.CaPath, envConfig.PrivateKeys, msg, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings); processDeliveryResult != ProcessDeliveryOK {
// 		return processDeliveryResult
// 	}

// 	var ok bool
// 	pCtx.CurrentScriptNode, ok = pCtx.Script.ScriptNodes[msg.TargetNodeName]
// 	if !ok {
// 		logger.Error("cannot find node %s in the script [%s], giving up with %s, returning ProcessDeliveryAckWithError, will not let other workers handle it", pCtx.Msg.TargetNodeName, pCtx.Msg.ScriptURL, msg.ToString())
// 		return ProcessDeliveryAckWithError
// 	}

// 	logger.DebugCtx(pCtx, "started processing batch %s", msg.FullBatchId())

// 	lastBatchStatus, err := wfdb.HarvestLastStatusForBatch(logger, pCtx)
// 	if err != nil {
// 		if db.IsDbConnError(err) {
// 			return ProcessDeliveryReconnectDb
// 		}
// 		return ProcessDeliveryAckWithError
// 	}

// 	// Check if this run/node/batch has been handled already
// 	if processDeliveryResult := checkLastBatchStatus(logger, pCtx, msg, lastBatchStatus); processDeliveryResult != ProcessDeliveryOK {
// 		return processDeliveryResult
// 	}

// 	// At this point, we are assuming this batch processing either never started or was started and then abandoned

// 	nodeReady, readerNodeRunId, lookupNodeRunId, err := checkDependencyNodesReady(logger, pCtx)
// 	if err != nil {
// 		logger.ErrorCtx(pCtx, "cannot verify dependency nodes status for %s: %s", pCtx.Msg.FullBatchId(), err.Error())
// 		if db.IsDbConnError(err) {
// 			return ProcessDeliveryReconnectDb
// 		}
// 		return ProcessDeliveryAckWithError
// 	}

// 	switch nodeReady {
// 	case sc.NodeNone:
// 		NodeDependencyNoneCounter.Inc()
// 	case sc.NodeWait:
// 		NodeDependencyWaitCounter.Inc()
// 	case sc.NodeGo:
// 		NodeDependencyGoCounter.Inc()
// 	case sc.NodeNogo:
// 		NodeDependencyNogoCounter.Inc()
// 	default:
// 		logger.ErrorCtx(pCtx, "unexpected nodeReady %v", nodeReady)
// 		return ProcessDeliveryAckWithError
// 	}

// 	if processDeliveryResult := checkDependencyNogoOrWait(logger, pCtx, nodeReady); processDeliveryResult != ProcessDeliveryOK {
// 		return processDeliveryResult
// 	}

// 	// At this point, we are ready to actually process the node

// 	if _, err := wfdb.SetNodeStatus(logger, pCtx, wfmodel.NodeStart, "started"); err != nil {
// 		if db.IsDbConnError(err) {
// 			return ProcessDeliveryReconnectDb
// 		}
// 		return ProcessDeliveryAckWithError
// 	}

// 	if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeStart, ""); err != nil {
// 		if db.IsDbConnError(err) {
// 			return ProcessDeliveryReconnectDb
// 		}
// 		return ProcessDeliveryAckWithError
// 	}

// 	batchStatus, batchStats, batchErr := SafeProcessBatch(envConfig, logger, pCtx, readerNodeRunId, lookupNodeRunId)

// 	// TODO: test only!!!
// 	// if pCtx.BatchInfo.TargetNodeName == "order_item_date_inner" && pCtx.BatchInfo.BatchIdx == 3 {
// 	// 	rnd := rand.New(rand.NewSource(time.Now().UnixMilli()))
// 	// 	if rnd.Float32() < .5 {
// 	// 		logger.InfoCtx(pCtx, "ProcessBatchWithStatus: test error")
// 	// 		return ProcessDeliveryRejectAndRetryLater
// 	// 	}
// 	// }

// 	if batchErr != nil {
// 		logger.ErrorCtx(pCtx, "ProcessBatchWithStatus: %s", batchErr.Error())
// 		if db.IsDbConnError(batchErr) {
// 			return ProcessDeliveryReconnectDb
// 		}
// 		if err := wfdb.SetBatchStatus(logger, pCtx, wfmodel.NodeBatchFail, batchErr.Error()); err != nil {
// 			if db.IsDbConnError(err) {
// 				return ProcessDeliveryReconnectDb
// 			}
// 			return ProcessDeliveryAckWithError
// 		}
// 	} else {
// 		logger.InfoCtx(pCtx, "ProcessBatchWithStatus: success")
// 		if err := wfdb.SetBatchStatus(logger, pCtx, batchStatus, batchStats.ToString()); err != nil {
// 			if db.IsDbConnError(err) {
// 				return ProcessDeliveryReconnectDb
// 			}
// 			return ProcessDeliveryAckWithError
// 		}
// 	}

// 	if err := refreshNodeAndRunStatus(logger, pCtx); err != nil {
// 		if db.IsDbConnError(err) {
// 			return ProcessDeliveryReconnectDb
// 		}
// 		return ProcessDeliveryAckWithError
// 	}

// 	return ProcessDeliveryAckSuccess
// }
