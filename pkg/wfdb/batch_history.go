package wfdb

import (
	"fmt"
	"sort"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func HarvestLastStatusForBatch(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) (wfmodel.NodeBatchStatusType, error) {
	logger.PushF("wfdb.HarvestLastStatusForBatch")
	defer logger.PopF()

	fields := []string{"ts", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Cond("run_id", "=", pCtx.BatchInfo.RunId).
		Cond("script_node", "=", pCtx.BatchInfo.TargetNodeName).
		Cond("batch_idx", "=", pCtx.BatchInfo.BatchIdx).
		Select(wfmodel.TableNameBatchHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.NodeBatchNone, db.WrapDbErrorWithQuery(fmt.Sprintf("HarvestLastStatusForBatch: cannot get batch history for batch %s", pCtx.BatchInfo.FullBatchId()), q, err)
	}

	lastStatus := wfmodel.NodeBatchNone
	lastTs := time.Unix(0, 0)
	for _, r := range rows {
		rec, err := wfmodel.NewBatchHistoryEventFromMap(r, fields)
		if err != nil {
			return wfmodel.NodeBatchNone, fmt.Errorf("HarvestLastStatusForBatch: : cannot deserialize batch history row: %s, %s", err.Error(), q)
		}

		if rec.Ts.After(lastTs) {
			lastTs = rec.Ts
			lastStatus = wfmodel.NodeBatchStatusType(rec.Status)
		}
	}

	logger.DebugCtx(pCtx, "batch %s, status %s", pCtx.BatchInfo.FullBatchId(), lastStatus.ToString())
	return lastStatus, nil
}

func GetRunNodeBatchHistory(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runId int16, nodeName string) ([]*wfmodel.BatchHistoryEvent, error) {
	logger.PushF("wfdb.GetRunNodeBatchHistory")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Cond("script_node", "=", nodeName).
		Select(wfmodel.TableNameBatchHistory, wfmodel.BatchHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return []*wfmodel.BatchHistoryEvent{}, db.WrapDbErrorWithQuery("GetRunNodeBatchHistory: cannot get node batch history", q, err)
	}

	result := make([]*wfmodel.BatchHistoryEvent, len(rows))
	for rowIdx, row := range rows {
		rec, err := wfmodel.NewBatchHistoryEventFromMap(row, wfmodel.BatchHistoryEventAllFields())
		if err != nil {
			return []*wfmodel.BatchHistoryEvent{}, fmt.Errorf("cannot deserialize batch node history row %s, %s", err.Error(), q)
		}
		result[rowIdx] = rec
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Ts.Before(result[j].Ts) })

	return result, nil
}

func HarvestBatchStatusesForNode(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) (wfmodel.NodeBatchStatusType, error) {
	logger.PushF("wfdb.HarvestBatchStatusesForNode")
	defer logger.PopF()

	fields := []string{"status", "batch_idx", "batches_total"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Cond("run_id", "=", pCtx.BatchInfo.RunId).
		Cond("script_node", "=", pCtx.BatchInfo.TargetNodeName).
		Select(wfmodel.TableNameBatchHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.NodeBatchNone, db.WrapDbErrorWithQuery(fmt.Sprintf("harvestBatchStatusesForNode: cannot get node batch history for node %s", pCtx.BatchInfo.FullBatchId()), q, err)
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
			return wfmodel.NodeBatchNone, fmt.Errorf("conflicting batches total value, was %d, now %d: %s, %s", foundBatchesTotal, rec.BatchesTotal, q, pCtx.BatchInfo.ToString())
		}

		if rec.BatchIdx >= rec.BatchesTotal || rec.BatchesTotal < 0 || rec.BatchesTotal <= 0 {
			return wfmodel.NodeBatchNone, fmt.Errorf("invalid batch idx/total(%d/%d) when processing [%v]: %s, %s", rec.BatchIdx, rec.BatchesTotal, r, q, pCtx.BatchInfo.ToString())
		}

		if rec.Status == wfmodel.NodeBatchSuccess ||
			rec.Status == wfmodel.NodeBatchFail ||
			rec.Status == wfmodel.NodeBatchRunStopReceived {
			delete(batchesInProgress, rec.BatchIdx)
		}

		if rec.Status == wfmodel.NodeBatchFail {
			failFound = true
		} else if rec.Status == wfmodel.NodeBatchRunStopReceived {
			stopReceivedFound = true
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
		logger.InfoCtx(pCtx, "node %d/%s complete, status %s", pCtx.BatchInfo.RunId, pCtx.CurrentScriptNode.Name, nodeStatus.ToString())
		return nodeStatus, nil
	}

	// Some batches are still not complete, and no run stop/fail/success for the whole node was signaled
	logger.DebugCtx(pCtx, "node %d/%s incomplete, still waiting for %d/%d batches", pCtx.BatchInfo.RunId, pCtx.CurrentScriptNode.Name, len(batchesInProgress), foundBatchesTotal)
	return wfmodel.NodeBatchStart, nil
}

func SetBatchStatus(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, status wfmodel.NodeBatchStatusType, comment string) error {
	logger.PushF("wfdb.SetBatchStatus")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	qb.
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		WriteForceUnquote("ts", "toTimeStamp(now())").
		Write("run_id", pCtx.BatchInfo.RunId).
		Write("script_node", pCtx.CurrentScriptNode.Name).
		Write("batch_idx", pCtx.BatchInfo.BatchIdx).
		Write("batches_total", pCtx.BatchInfo.BatchesTotal).
		Write("status", status).
		Write("first_token", pCtx.BatchInfo.FirstToken).
		Write("last_token", pCtx.BatchInfo.LastToken).
		Write("instance", logger.ZapMachine.String).
		Write("thread", logger.ZapThread.Integer)
	if len(comment) > 0 {
		qb.Write("comment", comment)
	}

	q := qb.InsertUnpreparedQuery(wfmodel.TableNameBatchHistory, cql.IgnoreIfExists) // If not exists. First one wins.
	err := pCtx.CqlSession.Query(q).Exec()
	if err != nil {
		err := db.WrapDbErrorWithQuery(fmt.Sprintf("cannot write batch %s status %d", pCtx.BatchInfo.FullBatchId(), status), q, err)
		logger.ErrorCtx(pCtx, err.Error())
		return err
	}

	logger.DebugCtx(pCtx, "batch %s, set status %s", pCtx.BatchInfo.FullBatchId(), status.ToString())
	return nil
}
