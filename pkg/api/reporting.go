package api

import (
	"fmt"
	"sort"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func GetRunHistory(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string) ([]*wfmodel.RunHistoryEvent, error) {
	logger.PushF("api.GetRunHistory")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(keyspace).
		Select(wfmodel.TableNameRunHistory, wfmodel.RunHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get run history", q, err)
	}

	result := make([]*wfmodel.RunHistoryEvent, len(rows))
	for rowIdx, r := range rows {
		result[rowIdx], err = wfmodel.NewRunHistoryEventFromMap(r, wfmodel.RunHistoryEventAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize run history row: %s, %s", err.Error(), q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Ts.Before(result[j].Ts) })

	return result, nil
}

func GetRunsNodeHistory(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runIds []int16) ([]*wfmodel.NodeHistoryEvent, error) {
	logger.PushF("api.GetNodeHistory")
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

	result := make([]*wfmodel.NodeHistoryEvent, len(rows))
	for rowIdx, r := range rows {
		result[rowIdx], err = wfmodel.NewNodeHistoryEventFromMap(r, wfmodel.NodeHistoryEventAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize node history row: %s, %s", err.Error(), q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Ts.Before(result[j].Ts) })
	return result, nil
}

func GetBatchHistory(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runIds []int16, scriptNodes []string) ([]*wfmodel.BatchHistoryEvent, error) {
	logger.PushF("api.GetBatchHistory")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace)
	if len(runIds) > 0 {
		qb.CondInInt16("run_id", runIds)
	}
	if len(scriptNodes) > 0 {
		qb.CondInString("script_node", scriptNodes)
	}
	q := qb.Select(wfmodel.TableNameBatchHistory, wfmodel.BatchHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get batch history", q, err)
	}

	result := make([]*wfmodel.BatchHistoryEvent, len(rows))
	for rowIdx, r := range rows {
		result[rowIdx], err = wfmodel.NewBatchHistoryEventFromMap(r, wfmodel.BatchHistoryEventAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize batch history row: %s, %s", err.Error(), q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Ts.Before(result[j].Ts) })
	return result, nil
}
