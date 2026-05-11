package wfdb

import (
	"fmt"
	"strings"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

// Used by Webapi to retrieve static run properties, used by daemon in updateRunStatusFromNodes
func GetRunProperties(cqlSession gocqlshims.Session, keyspace string, runId int16, fieldNames []string) (map[string]any, error) {
	if runId <= 0 {
		return nil, fmt.Errorf("cannot retrieve properties of run 0 for keyspace %s", keyspace)
	}

	qb := (&cql.QueryBuilder{}).Keyspace(keyspace).Cond("run_id", "=", runId)
	q := qb.Select(wfmodel.TableNameRunProperties, fieldNames)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get all runs properties", q, err)
	}

	if len(rows) != 1 {
		return nil, fmt.Errorf("cannot retrieve properties of run %d for keyspace %s, exactly one row expected, got %d", runId, keyspace, len(rows))
	}

	return rows[0], nil
}

// Used by daemon for dependency checking
func GetAllRunsProperties(cqlSession gocqlshims.Session, keyspace string, runPropertiesFields []string) ([]map[string]any, error) {
	q := (&cql.QueryBuilder{}).Keyspace(keyspace).Select(wfmodel.TableNameRunProperties, runPropertiesFields)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get runs", q, err)
	}
	return rows, nil
}

func WriteRunProperties(cqlSession gocqlshims.Session, keyspace string, runId int16, startNodes []string, affectedNodes []string, scriptUrl string, scriptParamsUrl string, runDescription string) error {
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Write("run_id", runId).
		Write("start_nodes", strings.Join(startNodes, ",")).
		Write("affected_nodes", strings.Join(affectedNodes, ",")).
		Write("script_url", scriptUrl).
		Write("script_params_url", scriptParamsUrl).
		Write("run_description", runDescription).
		InsertUnpreparedQuery(wfmodel.TableNameRunProperties, cql.IfNotExistsLwt) // If not exists. First one wins. Potential contention
	err := cqlSession.Query(q).Exec()
	if err != nil {
		return db.WrapDbErrorWithQuery("cannot write affected nodes", q, err)
	}

	return nil
}
