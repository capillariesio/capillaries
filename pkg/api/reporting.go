package api

import (
	"fmt"
	"sort"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func GetRunHistory(logger *l.Logger, cqlSession *gocql.Session, keyspace string) ([]*wfmodel.RunHistory, error) {
	logger.PushF("GetRunHistory")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(keyspace).
		Select(wfmodel.TableNameRunHistory, wfmodel.RunHistoryAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, cql.WrapDbErrorWithQuery("cannot get run history", q, err)
	}

	result := make([]*wfmodel.RunHistory, len(rows))
	for rowIdx, r := range rows {
		result[rowIdx], err = wfmodel.NewRunHistoryFromMap(r, wfmodel.RunHistoryAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize run history row: %s, %s", err.Error(), q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Ts.Before(result[j].Ts) })

	return result, nil
}

func GetNodeHistory(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runIds []int16) ([]*wfmodel.NodeHistory, error) {
	logger.PushF("GetNodeHistory")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace)
	if len(runIds) > 0 {
		qb.CondInInt16("run_id", runIds)
	}
	q := qb.Select(wfmodel.TableNameNodeHistory, wfmodel.NodeHistoryAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, cql.WrapDbErrorWithQuery("cannot get node history", q, err)
	}

	result := make([]*wfmodel.NodeHistory, len(rows))
	for rowIdx, r := range rows {
		result[rowIdx], err = wfmodel.NewNodeHistoryFromMap(r, wfmodel.NodeHistoryAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize node history row: %s, %s", err.Error(), q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Ts.Before(result[j].Ts) })
	return result, nil
}

func GetBatchHistory(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runIds []int16, scriptNodes []string) ([]*wfmodel.BatchHistory, error) {
	logger.PushF("GetBatchHistory")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace)
	if len(runIds) > 0 {
		qb.CondInInt16("run_id", runIds)
	}
	if len(scriptNodes) > 0 {
		qb.CondInString("script_node", scriptNodes)
	}
	q := qb.Select(wfmodel.TableNameBatchHistory, wfmodel.BatchHistoryAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, cql.WrapDbErrorWithQuery("cannot get batch history", q, err)
	}

	result := make([]*wfmodel.BatchHistory, len(rows))
	for rowIdx, r := range rows {
		result[rowIdx], err = wfmodel.NewBatchHistoryFromMap(r, wfmodel.BatchHistoryAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize batch history row: %s, %s", err.Error(), q)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Ts.Before(result[j].Ts) })
	return result, nil
}
