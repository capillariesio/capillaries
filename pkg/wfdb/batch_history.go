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

func HarvestLastStatusForBatch(cqlSession gocqlshims.Session, msg *wfmodel.Message) (wfmodel.NodeBatchStatusType, time.Time, error) {
	fields := []string{"ts", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(msg.DataKeyspace).
		Cond("run_id", "=", msg.RunId).
		Cond("script_node", "=", msg.TargetNodeName).
		Cond("batch_idx", "=", msg.BatchIdx).
		Select(wfmodel.TableNameBatchHistory, fields)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.NodeBatchNone, time.Unix(0, 0), db.WrapDbErrorWithQuery(fmt.Sprintf("HarvestLastStatusForBatch: cannot get batch history for batch %s", msg.FullBatchId()), q, err)
	}

	lastStatus := wfmodel.NodeBatchNone
	lastTs := time.Unix(0, 0)
	for _, r := range rows {
		rec, err := wfmodel.NewBatchHistoryEventFromMap(r, fields)
		if err != nil {
			return wfmodel.NodeBatchNone, time.Unix(0, 0), fmt.Errorf("HarvestLastStatusForBatch: : cannot deserialize batch history row: %s, %s", err.Error(), q)
		}

		if rec.Ts.After(lastTs) {
			lastTs = rec.Ts
			lastStatus = wfmodel.NodeBatchStatusType(rec.Status)
		}
	}
	return lastStatus, lastTs, nil
}

// Used by Webapi to retrieve batch status history for a run/node pair
func GetBatchHistoryForRunAndNode(cqlSession gocqlshims.Session, keyspace string, runId int16, nodeName string) ([]map[string]any, error) {
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Cond("script_node", "=", nodeName).
		Select(wfmodel.TableNameBatchHistory, wfmodel.BatchHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("GetRunNodeBatchHistory: cannot get node batch history", q, err)
	}

	return rows, err
}

func HarvestBatchStatusesForNode(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) (wfmodel.NodeBatchStatusType, error) {
	logger.PushF("wfdb.HarvestBatchStatusesForNode")
	defer logger.PopF()

	fields := []string{"status", "batch_idx", "batches_total"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.Msg.DataKeyspace).
		Cond("run_id", "=", pCtx.Msg.RunId).
		Cond("script_node", "=", pCtx.Msg.TargetNodeName).
		Select(wfmodel.TableNameBatchHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.NodeBatchNone, db.WrapDbErrorWithQuery(fmt.Sprintf("harvestBatchStatusesForNode: cannot get node batch history for node %s", pCtx.Msg.FullBatchId()), q, err)
	}

	foundBatchesTotal := int16(-1)
	batchesInProgress := map[int16]struct{}{}

	failFound := false
	stopReceivedFound := false
	for _, r := range rows {
		rec, err := wfmodel.NewBatchHistoryEventFromMap(r, fields)
		if err != nil {
			return wfmodel.NodeBatchNone, fmt.Errorf("harvestBatchStatusesForNode: cannot deserialize batch history row %s, %s", err.Error(), q)
		}
		if foundBatchesTotal == -1 {
			foundBatchesTotal = rec.BatchesTotal
			for i := int16(0); i < rec.BatchesTotal; i++ {
				batchesInProgress[i] = struct{}{}
			}
		} else if rec.BatchesTotal != foundBatchesTotal {
			return wfmodel.NodeBatchNone, fmt.Errorf("conflicting batches total value, was %d, now %d: %s, %s", foundBatchesTotal, rec.BatchesTotal, q, pCtx.Msg.ToString())
		}

		if rec.BatchIdx >= rec.BatchesTotal || rec.BatchesTotal < 0 || rec.BatchesTotal <= 0 {
			return wfmodel.NodeBatchNone, fmt.Errorf("invalid batch idx/total(%d/%d) when processing [%v]: %s, %s", rec.BatchIdx, rec.BatchesTotal, r, q, pCtx.Msg.ToString())
		}

		if rec.Status == wfmodel.NodeBatchSuccess ||
			rec.Status == wfmodel.NodeBatchFail ||
			rec.Status == wfmodel.NodeBatchRunStopReceived {
			delete(batchesInProgress, rec.BatchIdx)
		}

		switch rec.Status {
		case wfmodel.NodeBatchFail:
			failFound = true
		case wfmodel.NodeBatchRunStopReceived:
			stopReceivedFound = true
		default:
			// Nothing interesting yet
		}
	}

	if len(batchesInProgress) == 0 {
		nodeStatus := wfmodel.NodeBatchSuccess
		if stopReceivedFound {
			nodeStatus = wfmodel.NodeBatchRunStopReceived
		}
		if failFound {
			nodeStatus = wfmodel.NodeBatchFail
		}
		logger.InfoCtx(pCtx, "node %d/%s complete, status %s", pCtx.Msg.RunId, pCtx.Msg.TargetNodeName, wfmodel.NodeBatchStatusToString(nodeStatus))
		return nodeStatus, nil
	}

	// Some batches are still not complete, and no run stop/fail/success for the whole node was signaled
	logger.DebugCtx(pCtx, "node %d/%s incomplete, still waiting for %d/%d batches", pCtx.Msg.RunId, pCtx.Msg.TargetNodeName, len(batchesInProgress), foundBatchesTotal)
	return wfmodel.NodeBatchStart, nil
}

func SetBatchStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, status wfmodel.NodeBatchStatusType, comment string) error {
	logger.PushF("wfdb.SetBatchStatus")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	qb.
		Keyspace(pCtx.Msg.DataKeyspace).
		WriteForceUnquote("ts", "toTimestamp(now())").
		Write("run_id", pCtx.Msg.RunId).
		Write("script_node", pCtx.Msg.TargetNodeName).
		Write("batch_idx", pCtx.Msg.BatchIdx).
		Write("batches_total", pCtx.Msg.BatchesTotal).
		Write("status", status).
		Write("first_token", pCtx.Msg.FirstToken).
		Write("last_token", pCtx.Msg.LastToken).
		Write("instance", logger.ZapMachine.String).
		Write("thread", logger.ZapThread.Integer)
	if len(comment) > 0 {
		qb.Write("comment", comment)
	}

	q := qb.InsertUnpreparedQuery(wfmodel.TableNameBatchHistory, cql.IfExistsOverwrite)
	err := pCtx.CqlSession.Query(q).Exec()
	if err != nil {
		err := db.WrapDbErrorWithQuery(fmt.Sprintf("cannot write batch %s status %d", pCtx.Msg.FullBatchId(), status), q, err)
		logger.ErrorCtx(pCtx, "%s", err.Error())
		return err
	}

	logger.DebugCtx(pCtx, "batch %s, set status %s", pCtx.Msg.FullBatchId(), wfmodel.NodeBatchStatusToString(status))
	return nil
}
