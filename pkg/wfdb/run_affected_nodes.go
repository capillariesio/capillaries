package wfdb

import (
	"fmt"
	"sort"
	"strings"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func GetRunAffectedNodes(cqlSession *gocql.Session, keyspace string, runId int16) ([]string, error) {
	runPropsList, err := getRunsProperties(cqlSession, keyspace, runId)
	if err != nil {
		return []string{}, err
	}
	if len(runPropsList) != 1 {
		return []string{}, fmt.Errorf("run affected nodes for ks %s, run id %d returned wrong number of rows (%d), expected 1", keyspace, runId, len(runPropsList))
	}
	return strings.Split(runPropsList[0].AffectedNodes, ","), nil
}

func GetAllRunsProperties(cqlSession *gocql.Session, keyspace string) ([]*wfmodel.RunAffectedNodes, error) {
	return getRunsProperties(cqlSession, keyspace, 0)
}

func getRunsProperties(cqlSession *gocql.Session, keyspace string, runId int16) ([]*wfmodel.RunAffectedNodes, error) {
	fields := []string{"run_id", "start_nodes", "affected_nodes", "script_uri", "script_params_uri"}
	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace)
	if runId > 0 {
		qb.Cond("run_id", "=", runId)
	}
	q := qb.Select(wfmodel.TableNameRunAffectedNodes, fields)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return []*wfmodel.RunAffectedNodes{}, cql.WrapDbErrorWithQuery("cannot get all runs properties", q, err)
	}

	runs := make([]*wfmodel.RunAffectedNodes, len(rows))
	for rowIdx, row := range rows {
		rec, err := wfmodel.NewRunAffectedNodesFromMap(row, fields)
		if err != nil {
			return []*wfmodel.RunAffectedNodes{}, fmt.Errorf("%s, %s", err.Error(), q)
		}
		runs[rowIdx] = rec
	}

	sort.Slice(runs, func(i, j int) bool { return runs[i].RunId < runs[j].RunId })

	return runs, nil
}

func HarvestRunIdsByAffectedNodes(logger *l.Logger, pCtx *ctx.MessageProcessingContext, nodeNames []string) ([]int16, map[string][]int16, error) {
	logger.PushF("HarvestRunIdsByAffectedNodes")
	defer logger.PopF()

	fields := []string{"run_id", "affected_nodes"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Select(wfmodel.TableNameRunAffectedNodes, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, nil, cql.WrapDbErrorWithQuery("cannot get runs for affected nodes", q, err)
	}

	runIds := make([]int16, len(rows))
	nodeAffectingRunIdsMap := map[string][]int16{}
	for runIdx, r := range rows {

		rec, err := wfmodel.NewRunAffectedNodesFromMap(r, fields)
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

func WriteAffectedNodes(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16, startNodes []string, affectedNodes []string, scriptUri string, scriptParamsUri string) error {
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Write("run_id", runId).
		Write("start_nodes", strings.Join(startNodes, ",")).
		Write("affected_nodes", strings.Join(affectedNodes, ",")).
		Write("script_uri", scriptUri).
		Write("script_params_uri", scriptParamsUri).
		Insert(wfmodel.TableNameRunAffectedNodes, cql.IgnoreIfExists) // If not exists. First one wins.
	err := cqlSession.Query(q).Exec()
	if err != nil {
		return cql.WrapDbErrorWithQuery("cannot write affected nodes", q, err)
	}

	return nil
}
