package wfdb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func GetRunAffectedNodes(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runId int16) ([]string, error) {
	logger.PushF("wfdb.GetRunAffectedNodes")
	defer logger.PopF()

	runPropsList, err := GetRunProperties(logger, cqlSession, keyspace, runId)
	if err != nil {
		return []string{}, err
	}
	if len(runPropsList) != 1 {
		return []string{}, fmt.Errorf("run affected nodes for ks %s, run id %d returned wrong number of rows (%d), expected 1", keyspace, runId, len(runPropsList))
	}
	return strings.Split(runPropsList[0].AffectedNodes, ","), nil
}

// func GetAllRunsProperties(cqlSession *gocql.Session, keyspace string) ([]*wfmodel.RunAffectedNodes, error) {
// 	return getRunProperties(cqlSession, keyspace, 0)
// }

func GetRunProperties(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runId int16) ([]*wfmodel.RunProperties, error) {
	logger.PushF("wfdb.GetRunProperties")
	defer logger.PopF()

	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace)
	if runId > 0 {
		qb.Cond("run_id", "=", runId)
	}
	q := qb.Select(wfmodel.TableNameRunAffectedNodes, wfmodel.RunPropertiesAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return []*wfmodel.RunProperties{}, db.WrapDbErrorWithQuery("cannot get all runs properties", q, err)
	}

	runs := make([]*wfmodel.RunProperties, len(rows))
	for rowIdx, row := range rows {
		rec, err := wfmodel.NewRunPropertiesFromMap(row, wfmodel.RunPropertiesAllFields())
		if err != nil {
			return []*wfmodel.RunProperties{}, fmt.Errorf("%s, %s", err.Error(), q)
		}
		runs[rowIdx] = rec
	}

	sort.Slice(runs, func(i, j int) bool { return runs[i].RunId < runs[j].RunId })

	return runs, nil
}

func HarvestRunIdsByAffectedNodes(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) ([]int16, map[string][]int16, error) {
	logger.PushF("wfdb.HarvestRunIdsByAffectedNodes")
	defer logger.PopF()

	fields := []string{"run_id", "affected_nodes"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Select(wfmodel.TableNameRunAffectedNodes, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, nil, db.WrapDbErrorWithQuery("cannot get runs for affected nodes", q, err)
	}

	runIds := make([]int16, len(rows))
	nodeAffectingRunIdsMap := map[string][]int16{}
	for runIdx, r := range rows {

		rec, err := wfmodel.NewRunPropertiesFromMap(r, fields)
		if err != nil {
			return nil, nil, fmt.Errorf("%s, %s", err.Error(), q)
		}

		runIds[runIdx] = rec.RunId

		affectedNodes := strings.Split(rec.AffectedNodes, ",")
		for _, affectedNodeName := range affectedNodes {
			_, ok := nodeAffectingRunIdsMap[affectedNodeName]
			if !ok {
				nodeAffectingRunIdsMap[affectedNodeName] = make([]int16, 1)
				nodeAffectingRunIdsMap[affectedNodeName][0] = rec.RunId
			} else {
				nodeAffectingRunIdsMap[affectedNodeName] = append(nodeAffectingRunIdsMap[affectedNodeName], rec.RunId)

			}
		}
	}

	return runIds, nodeAffectingRunIdsMap, nil
}

func WriteRunProperties(cqlSession *gocql.Session, keyspace string, runId int16, startNodes []string, affectedNodes []string, scriptUrl string, scriptParamsUrl string, runDescription string) error {
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Write("run_id", runId).
		Write("start_nodes", strings.Join(startNodes, ",")).
		Write("affected_nodes", strings.Join(affectedNodes, ",")).
		Write("script_url", scriptUrl).
		Write("script_params_url", scriptParamsUrl).
		Write("run_description", runDescription).
		InsertUnpreparedQuery(wfmodel.TableNameRunAffectedNodes, cql.IgnoreIfExists) // If not exists. First one wins.
	err := cqlSession.Query(q).Exec()
	if err != nil {
		return db.WrapDbErrorWithQuery("cannot write affected nodes", q, err)
	}

	return nil
}
