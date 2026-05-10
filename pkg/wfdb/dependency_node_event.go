package wfdb

import (
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func BuildDependencyNodeRunStatusMap(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, depNodeNames []string) (map[string][]wfmodel.DependencyNodeRunStatus, error) {
	logger.PushF("wfdb.BuildDependencyNodeRunStatusMap")
	defer logger.PopF()

	// All runs in this ks with their properties
	runPropertiesFields := []string{"run_id", "affected_nodes"}
	rows, err := GetAllRunsProperties(pCtx.CqlSession, pCtx.Msg.DataKeyspace, runPropertiesFields)
	if err != nil {
		return nil, err
	}

	// Say, for current run 4 we may get [run1, run2, run3 ] and { run1: ["nodeReader"], run2: ["nodeLookup"], run3: ["nodeLookup"] }
	depRunIds, depeRunNodesMap, err := wfmodel.MultipleRunsPropertiesToDependencies(rows, depNodeNames, runPropertiesFields)
	if err != nil {
		return nil, err
	}

	// Run history only for "dependency" runs
	rows, err = GetRunHistory(logger, pCtx.CqlSession, pCtx.Msg.DataKeyspace, depRunIds)
	if err != nil {
		return nil, err
	}

	sortedRunHistoryEvents, err := wfmodel.RunHistoryRowsToEvents(rows)
	if err != nil {
		return nil, err
	}

	// Get { run1: complete, run2: stopped, run3: complete }
	runStatusMap := wfmodel.RunHistoryEventsToRunStatusMap(sortedRunHistoryEvents)

	// Get dependency node status change events
	// q := (&cql.QueryBuilder{}).
	// 	Keyspace(pCtx.Msg.DataKeyspace).
	// 	CondInInt16("run_id", depRunIds).
	// 	CondInString("script_node", depNodeNames).
	// 	Select(wfmodel.TableNameNodeHistory, wfmodel.NodeHistoryEventAllFields())
	// rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	rows, err = GetNodeHistoryForRuns(logger, pCtx.CqlSession, pCtx.Msg.DataKeyspace, depRunIds, depNodeNames)
	if err != nil {
		return nil, err
	}

	sortedNodeEvents, err := wfmodel.NodeHistoryRowsToEvents(rows)
	if err != nil {
		return nil, err
	}

	// Build { nodeReader: [{run1, RunComplete, NodeSuccess}], nodeLookup: [{run2, RunStopped, NodeSuccess}, {run3, RunComplete, NodeSuccess}]  }
	resultMap := map[string][]wfmodel.DependencyNodeRunStatus{}
	for _, runId := range depRunIds {
		_, nodeStatusMap := wfmodel.FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(sortedNodeEvents, runId, depeRunNodesMap[runId])
		for nodeName, nodeStatus := range nodeStatusMap {
			nrs := wfmodel.DependencyNodeRunStatus{
				RunId:        runId,
				RunIsCurrent: runId == pCtx.Msg.RunId,
				RunStatus:    runStatusMap[runId],
				NodeStatus:   nodeStatus,
			}
			if _, ok := resultMap[nodeName]; !ok {
				resultMap[nodeName] = make([]wfmodel.DependencyNodeRunStatus, 0)
			}
			resultMap[nodeName] = append(resultMap[nodeName], nrs)
		}
	}
	return resultMap, nil
}

// Very db-heavy
/*
func BuildDependencyNodeEventLists(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, depNodeNames []string) (map[string][]wfmodel.DependencyNodeEvent, error) {
	logger.PushF("wfdb.buildDependencyNodeEventLists")
	defer logger.PopF()

	affectingRunIds, nodeAffectingRunIdsMap, err := harvestRunIdsByAffectedNodes(logger, pCtx)
	if err != nil {
		return nil, err
	}

	runLifespanMap, err := HarvestRunLifespans(logger, pCtx.CqlSession, pCtx.Msg.DataKeyspace, affectingRunIds)
	if err != nil {
		return nil, err
	}

	runNodeLifespanMap, err := HarvestNodeLifespans(logger, pCtx, affectingRunIds, depNodeNames)
	if err != nil {
		return nil, err
	}

	nodeEventListMap := map[string][]wfmodel.DependencyNodeEvent{}
	for _, nodeName := range depNodeNames {
		nodeEventListMap[nodeName] = []wfmodel.DependencyNodeEvent{}
		// Walk through only runs that affect this specific node. Do not use all affectingRunIds here.
		for _, affectingRunId := range nodeAffectingRunIdsMap[nodeName] {
			runLifespan, ok := runLifespanMap[affectingRunId]
			if !ok {
				return nil, fmt.Errorf("unexpectedly, cannot find run lifespan map for run %d, was it ever started?", affectingRunId)
			}
			if runLifespan.StartTs.Equal(time.Unix(0, 0)) || runLifespan.FinalStatus == wfmodel.RunNone {
				return nil, fmt.Errorf("unexpectedly, run lifespan %d looks like the run never started: %s", affectingRunId, runLifespanMap.ToString())
			}
			e := wfmodel.DependencyNodeEvent{
				RunId:          affectingRunId,
				RunIsCurrent:   affectingRunId == pCtx.Msg.RunId,
				RunStartTs:     runLifespan.StartTs,
				RunFinalStatus: runLifespan.FinalStatus,
				RunCompletedTs: runLifespan.CompletedTs,
				RunStoppedTs:   runLifespan.StoppedTs,
				NodeIsStarted:  false,
				NodeStartTs:    time.Time{},
				NodeStatus:     wfmodel.NodeBatchNone,
				NodeStatusTs:   time.Time{}}

			nodeLifespanMap, ok := runNodeLifespanMap[affectingRunId]
			if !ok {
				return nil, fmt.Errorf("unexpectedly, cannot find node lifespan map for run %d: %s", affectingRunId, runNodeLifespanMap.ToString())
			}

			if nodeLifespan, ok := nodeLifespanMap[nodeName]; ok {
				// This run already started this node, so it has some status. Update last few attributes.
				e.NodeIsStarted = true
				e.NodeStartTs = nodeLifespan.StartTs
				e.NodeStatus = nodeLifespan.LastStatus
				e.NodeStatusTs = nodeLifespan.LastStatusTs
			}

			nodeEventListMap[nodeName] = append(nodeEventListMap[nodeName], e)
		}
	}
	return nodeEventListMap, nil
}
*/
