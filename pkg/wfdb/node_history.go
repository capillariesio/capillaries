package wfdb

import (
	"fmt"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func GetNodeHistoryForRuns(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runIds []int16, nodeNames []string) ([]map[string]any, error) {
	logger.PushF("wfdb.GetNodeHistoryForRuns")
	defer logger.PopF()

	qb := (&cql.QueryBuilder{}).Keyspace(keyspace)
	if len(runIds) > 0 {
		qb.CondInInt16("run_id", runIds)
	}
	if len(nodeNames) > 0 {
		qb.CondInString("script_node", nodeNames)
	}
	q := qb.Select(wfmodel.TableNameNodeHistory, wfmodel.NodeHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery(fmt.Sprintf("cannot get node history for %s, %v, %v", keyspace, runIds, nodeNames), q, err)
	}

	return rows, nil
}

func SetNodeStatus(cqlSession gocqlshims.Session, msg *wfmodel.Message, status wfmodel.NodeBatchStatusType, comment string) error {
	q := (&cql.QueryBuilder{}).
		Keyspace(msg.DataKeyspace).
		WriteForceUnquote("ts", "toTimestamp(now())").
		Write("run_id", msg.RunId).
		Write("script_node", msg.TargetNodeName).
		Write("written_by_batch_idx", msg.BatchIdx).
		Write("status", status).
		Write("comment", comment).
		InsertUnpreparedQuery(wfmodel.TableNameNodeHistory, cql.IfExistsOverwrite) // To avoid contention, overwrite
	err := cqlSession.Query(q).Exec()

	if err != nil {
		err = db.WrapDbErrorWithQuery(fmt.Sprintf("cannot update node %d/%s status to %d", msg.RunId, msg.TargetNodeName, status), q, err)
		return err
	}
	return nil
}
