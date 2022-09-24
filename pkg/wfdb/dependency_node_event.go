package wfdb

import (
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

func BuildDependencyNodeEventLists(logger *l.Logger, pCtx *ctx.MessageProcessingContext, depNodeNames []string) (map[string][]wfmodel.DependencyNodeEvent, error) {
	logger.PushF("buildDependencyNodeEventLists")
	defer logger.PopF()

	affectingRunIds, nodeAffectingRunIdsMap, err := HarvestRunIdsByAffectedNodes(logger, pCtx, depNodeNames)
	if err != nil {
		return nil, err
	}

	runLifespanMap, err := HarvestRunLifespans(logger, pCtx, affectingRunIds)
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
				return nil, fmt.Errorf("unexpectedly, cannot find run lifespan map for run %d: %s", affectingRunId, runLifespanMap.ToString())
			}
			if runLifespan.StartTs == time.Unix(0, 0) || runLifespan.LastStatus == wfmodel.RunNone || runLifespan.LastStatusTs == time.Unix(0, 0) {
				return nil, fmt.Errorf("unexpectedly, run lifespan %d looks like the run never started: %s", affectingRunId, runLifespanMap.ToString())
			}
			e := wfmodel.DependencyNodeEvent{
				RunId:         affectingRunId,
				RunIsCurrent:  affectingRunId == pCtx.BatchInfo.RunId,
				RunStartTs:    runLifespan.StartTs,
				RunStatus:     runLifespan.LastStatus,
				RunStatusTs:   runLifespan.LastStatusTs,
				NodeIsStarted: false,
				NodeStartTs:   time.Time{},
				NodeStatus:    wfmodel.NodeBatchNone,
				NodeStatusTs:  time.Time{}}

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
