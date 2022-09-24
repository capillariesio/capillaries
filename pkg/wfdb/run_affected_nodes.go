package wfdb

import (
	"fmt"
	"strings"

	"github.com/gocql/gocql"
	"github.com/kleineshertz/capillaries/pkg/cql"
	"github.com/kleineshertz/capillaries/pkg/ctx"
	"github.com/kleineshertz/capillaries/pkg/l"
	"github.com/kleineshertz/capillaries/pkg/wfmodel"
)

func GetRunAffectedNodes(cqlSession *gocql.Session, keyspace string, runId int16) ([]string, error) {
	fields := []string{"affected_nodes"}
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Cond("run_id", "=", runId).
		Select(wfmodel.TableNameRunAffectedNodes, fields)
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return []string{}, cql.WrapDbErrorWithQuery("cannot get run affected nodes", q, err)
	}

	if len(rows) != 1 {
		return []string{}, fmt.Errorf("run affected nodes returned wrong number of rows (%d): %s", len(rows), q)
	}

	rec, err := wfmodel.NewRunAffectedNodesFromMap(rows[0], fields)
	if err != nil {
		return []string{}, fmt.Errorf("%s, %s", err.Error(), q)
	}

	return strings.Split(rec.AffectedNodes, ","), nil
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

func WriteAffectedNodes(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16, affectedNodes []string) error {
	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Write("run_id", runId).
		Write("affected_nodes", strings.Join(affectedNodes, ",")).
		Insert(wfmodel.TableNameRunAffectedNodes, cql.IgnoreIfExists) // If not exists. First one wins.
	err := cqlSession.Query(q).Exec()
	if err != nil {
		return cql.WrapDbErrorWithQuery("cannot write affected nodes", q, err)
	}

	return nil
}
