package wfdb

import (
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func HarvestNodeStatusesForRun(logger *l.Logger, pCtx *ctx.MessageProcessingContext, affectedNodes []string) (wfmodel.NodeBatchStatusType, string, error) {
	logger.PushF("HarvestNodeStatusesForRun")
	defer logger.PopF()

	fields := []string{"script_node", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Cond("run_id", "=", pCtx.BatchInfo.RunId).
		CondInString("script_node", affectedNodes).
		Select(wfmodel.TableNameNodeHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.NodeBatchNone, "", cql.WrapDbErrorWithQuery(fmt.Sprintf("cannot get node history for %s", pCtx.BatchInfo.FullBatchId()), q, err)
	}

	nodeStatusMap := wfmodel.NodeStatusMap{}
	for nodeIdx := 0; nodeIdx < len(affectedNodes); nodeIdx++ {
		nodeStatusMap[affectedNodes[nodeIdx]] = wfmodel.NodeBatchNone
	}

	for _, r := range rows {
		rec, err := wfmodel.NewNodeHistoryFromMap(r, fields)
		if err != nil {
			return wfmodel.NodeBatchNone, "", fmt.Errorf("cannot deserialize node history row %s, %s", err.Error(), q)
		}

		// Use status priority
		if rec.Status > nodeStatusMap[rec.ScriptNode] {
			nodeStatusMap[rec.ScriptNode] = rec.Status
		}
	}

	highestStatus := wfmodel.NodeBatchNone
	lowestStatus := wfmodel.NodeBatchRunStopReceived
	for _, status := range nodeStatusMap {
		if status > highestStatus {
			highestStatus = status
		}
		if status < lowestStatus {
			lowestStatus = status
		}
	}

	if lowestStatus > wfmodel.NodeBatchStart {
		logger.InfoCtx(pCtx, "run %d complete, status map %s", pCtx.BatchInfo.RunId, nodeStatusMap.ToString())
		return highestStatus, nodeStatusMap.ToString(), nil
	} else {
		logger.DebugCtx(pCtx, "run %d incomplete, lowest status %s, status map %s", pCtx.BatchInfo.RunId, lowestStatus.ToString(), nodeStatusMap.ToString())
		return lowestStatus, nodeStatusMap.ToString(), nil
	}
}

func HarvestNodeLifespans(logger *l.Logger, pCtx *ctx.MessageProcessingContext, affectingRuns []int16, affectedNodes []string) (wfmodel.RunNodeLifespanMap, error) {
	logger.PushF("HarvestLastNodeStatuses")
	defer logger.PopF()

	fields := []string{"ts", "run_id", "script_node", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		CondInInt16("run_id", affectingRuns).
		CondInString("script_node", affectedNodes).
		Select(wfmodel.TableNameNodeHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, cql.WrapDbErrorWithQuery("cannot get node history", q, err)
	}

	runNodeLifespanMap := wfmodel.RunNodeLifespanMap{}
	for _, runId := range affectingRuns {
		runNodeLifespanMap[runId] = wfmodel.NodeLifespanMap{}
		for _, nodeName := range affectedNodes {
			runNodeLifespanMap[runId][nodeName] = &wfmodel.NodeLifespan{
				StartTs:      time.Time{},
				LastStatus:   wfmodel.NodeBatchNone,
				LastStatusTs: time.Time{}}
		}
	}

	for _, r := range rows {
		rec, err := wfmodel.NewNodeHistoryFromMap(r, fields)
		if err != nil {
			return nil, fmt.Errorf("%s, %s", err.Error(), q)
		}

		nodeLifespanMap, ok := runNodeLifespanMap[rec.RunId]
		if !ok {
			return nil, fmt.Errorf("unexpected run_id %d in the result %s", rec.RunId, q)
		}

		if rec.Status == wfmodel.NodeStart {
			nodeLifespanMap[rec.ScriptNode].StartTs = rec.Ts
		}

		// Later status wins, Stop always wins
		if rec.Ts.After(nodeLifespanMap[rec.ScriptNode].LastStatusTs) || rec.Status == wfmodel.NodeBatchRunStopReceived {
			nodeLifespanMap[rec.ScriptNode].LastStatus = rec.Status
			nodeLifespanMap[rec.ScriptNode].LastStatusTs = rec.Ts
		}
	}
	return runNodeLifespanMap, nil
}

func SetNodeStatus(logger *l.Logger, pCtx *ctx.MessageProcessingContext, status wfmodel.NodeBatchStatusType, comment string) (bool, error) {
	logger.PushF("SetNodeStatus")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		WriteForceUnquote("ts", "toTimeStamp(now())").
		Write("run_id", pCtx.BatchInfo.RunId).
		Write("script_node", pCtx.CurrentScriptNode.Name).
		Write("status", status).
		Write("comment", comment).
		Insert(wfmodel.TableNameNodeHistory, cql.IgnoreIfExists) // If not exists. First one wins.

	existingDataRow := map[string]interface{}{}
	isApplied, err := pCtx.CqlSession.Query(q).MapScanCAS(existingDataRow)

	if err != nil {
		err = cql.WrapDbErrorWithQuery(fmt.Sprintf("cannot update node %d/%s status to %d", pCtx.BatchInfo.RunId, pCtx.BatchInfo.TargetNodeName, status), q, err)
		logger.ErrorCtx(pCtx, err.Error())
		return false, err
	}
	logger.DebugCtx(pCtx, "%d/%s, %s, isApplied=%t", pCtx.BatchInfo.RunId, pCtx.CurrentScriptNode.Name, status.ToString(), isApplied)
	return isApplied, nil
}