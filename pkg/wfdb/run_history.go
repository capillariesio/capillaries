package wfdb

import (
	"fmt"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

// Used by daemon ProcessDataBatchMsg just to get run status
func GetRunStatusRows(cqlSession gocqlshims.Session, keyspace string, runId int16) ([]map[string]any, error) {
	if runId <= 0 {
		return nil, fmt.Errorf("cannot retrieve status of run 0 for keyspace %s", keyspace)
	}

	fields := []string{"ts", "status"}
	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Select(wfmodel.TableNameRunHistory, fields)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery(fmt.Sprintf("cannot query run status for %s/%d", keyspace, runId), q, err)
	}
	return rows, nil
}

// Used by daemon ProcessDataBatchMsg to update run status, by webapi/toolbelt to stop_run, webapi/toolbelt to start_run
func SetRunStatus(cqlSession gocqlshims.Session, keyspace string, runId int16, status wfmodel.RunStatusType, comment string) error {
	if runId <= 0 {
		return fmt.Errorf("cannot set status of run 0 for keyspace %s", keyspace)
	}

	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		WriteForceUnquote("ts", "toTimestamp(now())").
		Write("run_id", runId).
		Write("status", status).
		Write("comment", comment).
		InsertUnpreparedQuery(wfmodel.TableNameRunHistory, cql.IfNotExistsLwt) // Potential contention
	err := cqlSession.Query(q).Exec()
	if err != nil {
		return db.WrapDbErrorWithQuery("cannot write run status", q, err)
	}
	return nil
}

// Used by Toolbelt (get_run_history command)
// Used by Webapi to retrieve all runs that happened in this keyspace and their current status, and by checkDependencyNodesReady
// Used by daemon when checking dependencies
func GetRunHistory(cqlSession gocqlshims.Session, keyspace string, runIds []int16) ([]map[string]any, error) {
	qb := (&cql.QueryBuilder{}).Keyspace(keyspace)
	if len(runIds) > 0 {
		qb.CondInInt16("run_id", runIds)
	}
	q := qb.Select(wfmodel.TableNameRunHistory, wfmodel.RunHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get run history", q, err)
	}

	return rows, nil
}
