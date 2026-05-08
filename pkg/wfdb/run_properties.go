package wfdb

import (
	"fmt"
	"strings"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func GetRunAffectedNodes(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runId int16) ([]string, error) {
	logger.PushF("wfdb.GetRunAffectedNodes")
	defer logger.PopF()

	runProps, err := GetRunProperties(logger, cqlSession, keyspace, runId)
	if err != nil {
		return []string{}, err
	}
	return strings.Split(runProps.AffectedNodes, ","), nil
}

// Used by Webapi to retrieve static run properties
func GetRunProperties(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runId int16) (*wfmodel.RunProperties, error) {
	logger.PushF("wfdb.GetRunProperties")
	defer logger.PopF()

	if runId == 0 {
		return nil, fmt.Errorf("cannot retrieve properties of run 0 for keyspace %s", keyspace)
	}

	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace)
	qb.Cond("run_id", "=", runId)
	q := qb.Select(wfmodel.TableNameRunProperties, wfmodel.RunPropertiesAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get all runs properties", q, err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("cannot retrieve properties of run %d for keyspace %s, no such run", runId, keyspace)
	}

	rec, err := wfmodel.NewRunPropertiesFromMap(rows[0], wfmodel.RunPropertiesAllFields())
	if err != nil {
		return nil, fmt.Errorf("%s, %s", err.Error(), q)
	}

	return rec, nil
}

func GetAllRunsProperties(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string) ([]map[string]any, error) {
	logger.PushF("wfdb.GetAllRunsAffectedNodes")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		Select(wfmodel.TableNameRunProperties, wfmodel.RunPropertiesAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get runs", q, err)
	}
	return rows, nil
}

/*
func harvestRunIdsByAffectedNodes(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) ([]int16, map[string][]int16, error) {
	logger.PushF("wfdb.HarvestRunIdsByAffectedNodes")
	defer logger.PopF()

	fields := []string{"run_id", "affected_nodes"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.Msg.DataKeyspace).
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
*/

/*
func harvestRunStatuses(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runIds []int16) (map[int16]wfmodel.RunStatusType, error) {
	sortedRunHistoryEvents, err := GetRunHistory(logger, cqlSession, keyspace, runIds)
	if err != nil {
		return nil, err
	}

	runStatusMap := map[int16]wfmodel.RunStatusType{}
	for _, e := range sortedRunHistoryEvents {
		curRunStatus, ok := runStatusMap[e.RunId]
		if !ok {
			curRunStatus = wfmodel.RunNone
			runStatusMap[e.RunId] = curRunStatus
		} else {
			if e.Status == wfmodel.RunStop {
				runStatusMap[e.RunId] = wfmodel.RunStop
			} else if e.Status == wfmodel.RunComplete && curRunStatus != wfmodel.RunStop {
				runStatusMap[e.RunId] = wfmodel.RunComplete
			} else if e.Status == wfmodel.RunStart && curRunStatus != wfmodel.RunStop && curRunStatus != wfmodel.RunComplete {
				runStatusMap[e.RunId] = wfmodel.RunStart
			} else {
				runStatusMap[e.RunId] = wfmodel.RunNone
			}
		}
	}
	return runStatusMap, nil
}
*/

/*
func harvestDependencyRunsAndNodes(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, depNodeNames []string) ([]int16, map[int16][]string, error) {
	logger.PushF("wfdb.harvestAffectedNodesAndRuns")
	defer logger.PopF()

	fields := []string{"run_id", "affected_nodes"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.Msg.DataKeyspace).
		Select(wfmodel.TableNameRunProperties, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, nil, db.WrapDbErrorWithQuery("cannot get runs for affected nodes", q, err)
	}

	runNodesMap := map[int16][]string{}
	runIds := make([]int16, 0)
	for _, r := range rows {
		rec, err := wfmodel.NewRunPropertiesFromMap(r, fields)
		if err != nil {
			return nil, nil, fmt.Errorf("%s, %s", err.Error(), q)
		}
		runIds = append(runIds, rec.RunId)
		// Take only dependency nodes (0, 1 or 2 - since there can be only a reader and a lookot dependency)
		runNodesMap[rec.RunId] = intersectTwoSlicesOfStrings(strings.Split(rec.AffectedNodes, ","), depNodeNames)
	}
	return runIds, runNodesMap, nil
}
*/

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
