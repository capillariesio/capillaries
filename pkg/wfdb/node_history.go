package wfdb

import (
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func HarvestNodeStatusesForRun(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, affectedNodes []string) (wfmodel.NodeBatchStatusType, wfmodel.NodeStatusMap, error) {
	logger.PushF("wfdb.HarvestNodeStatusesForRun")
	defer logger.PopF()

	fields := []string{"ts", "run_id", "script_node", "written_by_batch_idx", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.Msg.DataKeyspace).
		Cond("run_id", "=", pCtx.Msg.RunId).
		CondInString("script_node", affectedNodes). // TODO: Is this really necessary? Shouldn't run id be enough? Of course, it's safer to be extra cautious, but...?
		Select(wfmodel.TableNameNodeHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.NodeBatchNone, nil, db.WrapDbErrorWithQuery(fmt.Sprintf("cannot get node history for %s", pCtx.Msg.FullBatchId()), q, err)
	}

	sortedNodeEvents, err := wfmodel.NodeHistoryRowsToNodeHistoryEvents(rows, fields)
	if err != nil {
		return wfmodel.NodeBatchNone, nil, err
	}

	runStatus, affectedNodesStatusMap := wfmodel.FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(sortedNodeEvents, pCtx.Msg.RunId, affectedNodes)
	return runStatus, affectedNodesStatusMap, nil
}

func HarvestNodeLifespans(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, affectingRuns []int16, affectedNodes []string) (wfmodel.RunNodeLifespanMap, error) {
	logger.PushF("wfdb.HarvestNodeLifespans")
	defer logger.PopF()

	fields := []string{"ts", "run_id", "script_node", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.Msg.DataKeyspace).
		CondInInt16("run_id", affectingRuns).
		CondInString("script_node", affectedNodes).
		Select(wfmodel.TableNameNodeHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get node history", q, err)
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

func SetNodeStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, status wfmodel.NodeBatchStatusType, comment string) (bool, error) {
	logger.PushF("wfdb.SetNodeStatus")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.Msg.DataKeyspace).
		WriteForceUnquote("ts", "toTimestamp(now())").
		Write("run_id", pCtx.Msg.RunId).
		Write("script_node", pCtx.Msg.TargetNodeName).
		Write("written_by_batch_idx", pCtx.Msg.BatchIdx).
		Write("status", status).
		Write("comment", comment).
		InsertUnpreparedQuery(wfmodel.TableNameNodeHistory, cql.IgnoreIfExists) // If not exists. First one wins.

	existingDataRow := map[string]any{}
	isApplied, err := pCtx.CqlSession.Query(q).MapScanCAS(existingDataRow)

	if err != nil {
		err = db.WrapDbErrorWithQuery(fmt.Sprintf("cannot update node %d/%s status to %d", pCtx.Msg.RunId, pCtx.Msg.TargetNodeName, status), q, err)
		logger.ErrorCtx(pCtx, "%s", err.Error())
		return false, err
	}
	logger.DebugCtx(pCtx, "%d/%s, %s, isApplied=%t", pCtx.Msg.RunId, pCtx.Msg.TargetNodeName, status.ToString(), isApplied)
	return isApplied, nil
}

// Used by Webapi to retrieve each node status history for a run
/*
func GetNodeHistoryForRun(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runId int16) ([]*wfmodel.NodeHistoryEvent, error) {
	logger.PushF("wfdb.GetNodeHistoryForRun")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Select(wfmodel.TableNameNodeHistory, wfmodel.NodeHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return []*wfmodel.NodeHistoryEvent{}, db.WrapDbErrorWithQuery(fmt.Sprintf("cannot get node history for run %d", runId), q, err)
	}

	sortedNodeEvents, err := wfmodel.NodeHistoryRowsToNodeHistoryEvents(rows, wfmodel.NodeHistoryEventAllFields())
	if err != nil {
		return []*wfmodel.NodeHistoryEvent{}, err
	}

	return sortedNodeEvents, nil
}
*/

func GetNodeHistoryForRuns(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runIds []int16) ([]*wfmodel.NodeHistoryEvent, error) {
	logger.PushF("wfdb.GetNodeHistoryForRuns")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace)
	if len(runIds) > 0 {
		qb.CondInInt16("run_id", runIds)
	}

	q := qb.Select(wfmodel.TableNameNodeHistory, wfmodel.NodeHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get node history", q, err)
	}

	return wfmodel.NodeHistoryRowsToNodeHistoryEvents(rows, wfmodel.NodeHistoryEventAllFields())
}
