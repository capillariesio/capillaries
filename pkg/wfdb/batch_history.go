package wfdb

import (
	"fmt"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

// Used by daemon in the beginnig of the batch processing
func GetSingleBatchStatusRows(cqlSession gocqlshims.Session, keyspace string, runId int16, nodeName string, batchIdx int16, fields []string) ([]map[string]any, error) {
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Cond("script_node", "=", nodeName).
		Cond("batch_idx", "=", batchIdx).
		Select(wfmodel.TableNameBatchHistory, fields)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery(fmt.Sprintf("cannot get batch history for batch %s/%d/%s/%d", keyspace, runId, nodeName, batchIdx), q, err)
	}

	return rows, nil
}

// Used by Webapi to retrieve batch status history for a run/node pair
func GetAllBatchHistoryForRunAndNode(cqlSession gocqlshims.Session, keyspace string, runId int16, nodeName string, fields []string) ([]map[string]any, error) {
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Cond("script_node", "=", nodeName).
		Select(wfmodel.TableNameBatchHistory, fields)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery(fmt.Sprintf("cannot get node %s/%d/%s batch history", keyspace, runId, nodeName), q, err)
	}

	return rows, err
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
