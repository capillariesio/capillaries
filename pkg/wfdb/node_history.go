package wfdb

import (
	"fmt"
	"sort"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func HarvestNodeStatusesForRun(logger *l.Logger, pCtx *ctx.MessageProcessingContext, affectedNodes []string) (wfmodel.NodeBatchStatusType, string, error) {
	logger.PushF("wfdb.HarvestNodeStatusesForRun")
	defer logger.PopF()

	fields := []string{"script_node", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Cond("run_id", "=", pCtx.BatchInfo.RunId).
		CondInString("script_node", affectedNodes). // TODO: Is this really necessary? Shouldn't run id be enough? Of course, it's safer to be extra cautious, but...?
		Select(wfmodel.TableNameNodeHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.NodeBatchNone, "", cql.WrapDbErrorWithQuery(fmt.Sprintf("cannot get node history for %s", pCtx.BatchInfo.FullBatchId()), q, err)
	}

	nodeStatusMap := wfmodel.NodeStatusMap{}
	for _, affectedNodeName := range affectedNodes {
		nodeStatusMap[affectedNodeName] = wfmodel.NodeBatchNone
	}

	nodeEvents := make([]*wfmodel.NodeHistoryEvent, len(rows))

	for idx, r := range rows {
		rec, err := wfmodel.NewNodeHistoryEventFromMap(r, fields)
		if err != nil {
			return wfmodel.NodeBatchNone, "", fmt.Errorf("cannot deserialize node history row %s, %s", err.Error(), q)
		}
		nodeEvents[idx] = rec
	}

	sort.Slice(nodeEvents, func(i, j int) bool { return nodeEvents[i].Ts.Before(nodeEvents[j].Ts) })

	for _, e := range nodeEvents {
		lastStatus, ok := nodeStatusMap[e.ScriptNode]
		if !ok {
			nodeStatusMap[e.ScriptNode] = e.Status
		} else {
			// Stopreceived is higher priority than anything else
			if lastStatus != wfmodel.NodeBatchRunStopReceived {
				nodeStatusMap[e.ScriptNode] = e.Status
			}
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
	logger.PushF("wfdb.HarvestLastNodeStatuses")
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
		rec, err := wfmodel.NewNodeHistoryEventFromMap(r, fields)
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
	logger.PushF("wfdb.SetNodeStatus")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		WriteForceUnquote("ts", "toTimeStamp(now())").
		Write("run_id", pCtx.BatchInfo.RunId).
		Write("script_node", pCtx.CurrentScriptNode.Name).
		Write("status", status).
		Write("comment", comment).
		InsertUnpreparedQuery(wfmodel.TableNameNodeHistory, cql.IgnoreIfExists) // If not exists. First one wins.

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

func GetNodeHistoryForRun(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16) ([]*wfmodel.NodeHistoryEvent, error) {
	logger.PushF("wfdb.GetNodeHistoryForRun")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Select(wfmodel.TableNameNodeHistory, wfmodel.NodeHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return []*wfmodel.NodeHistoryEvent{}, cql.WrapDbErrorWithQuery(fmt.Sprintf("cannot get node history for run %d", runId), q, err)
	}

	result := make([]*wfmodel.NodeHistoryEvent, len(rows))

	for idx, r := range rows {
		rec, err := wfmodel.NewNodeHistoryEventFromMap(r, wfmodel.NodeHistoryEventAllFields())
		if err != nil {
			return []*wfmodel.NodeHistoryEvent{}, fmt.Errorf("cannot deserialize node history row %s, %s", err.Error(), q)
		}
		result[idx] = rec
	}

	return result, nil
}
