package capigraph

import (
	"fmt"
	"slices"
)

// We want to pull down subtrees as far as possible, so their roots may become enclosed under bigger subtrees.
func pullDownRootSubtreeIfNeeded(rootNodeId int16, tallestSubtreeRootIds *Int16Set, rootToNodeDistanceMap [][]int) bool {
	delta := 0
	for tallRootId := range *tallestSubtreeRootIds {
		if rootNodeId == tallRootId {
			panic(fmt.Sprintf("pullDownRootSubtreeIfNeeded(): comparing root %d against the %d subtree, and they have the same root", rootNodeId, tallRootId))
		}
		for i, thisNodeDist := range rootToNodeDistanceMap[rootNodeId][1:] {
			thisNodeId := int16(i + 1)
			for j, tallNodeDist := range rootToNodeDistanceMap[tallRootId][1:] {
				tallNodeId := int16(j + 1)
				if tallNodeDist == MissingDistanceFromRootToNode {
					// Tall node is not a part of thisRootId subtree (value not in the map)
					continue
				}
				if thisNodeDist == MissingDistanceFromRootToNode {
					// This node is not a part of tallRootId subtree (value not in the map)
					continue
				}
				if thisNodeId != tallNodeId {
					// We have not found the common node yet, keep iterating
					continue
				}
				// This node belongs to both trees, check distances
				if tallNodeDist > thisNodeDist {
					// delta > tallNodeDist-thisNodeDist is important:
					// we are not looking for the maximum delta, we are looking for min shift,
					// otherwise we are guaranteed to shift down too much
					if delta == 0 || delta > tallNodeDist-thisNodeDist {
						delta = tallNodeDist - thisNodeDist
					}
				}
			}
		}
	}

	if delta == 0 {
		// No need to pull this subtree down
		return false
	}

	// Let's pull down this subtree
	for nodeId := range rootToNodeDistanceMap[rootNodeId] {
		rootToNodeDistanceMap[rootNodeId][nodeId] += delta
	}
	return true
}

func buildLayerMap(nodeDefs []NodeDef, priParentMap []int16) []int {
	nodeLayerMap := slices.Repeat([]int{MissingLayer}, len(nodeDefs))
	allChildrenMap := buildAllChildrenMap(nodeDefs)
	rootNodes := buildRootNodeList(priParentMap)
	nodeToRootMap := buildNodeToRootMap(priParentMap)
	rootToNodeDistanceMap := buildRootToNodeDistanceMap(len(nodeDefs), rootNodes, allChildrenMap)
	rootSubtreeHeightMap, maxSubtreeHeight := buildRootSubtreeHeightsMap(len(nodeDefs), rootNodes, rootToNodeDistanceMap)

	// These subtrees are already settled
	layeredSubtreesSet := Int16Set{}
	for rootNodeId, rootSubtreeHeight := range rootSubtreeHeightMap {
		if rootSubtreeHeight == maxSubtreeHeight {
			layeredSubtreesSet.add(int16(rootNodeId))
		}
	}

	candidateSubtreesSet := stringSliceToSet(rootNodes)
	candidateSubtreesSet.subtract(&layeredSubtreesSet)
	for {
		var rootIdPulledDown int16
		for candidateRootId := range *candidateSubtreesSet {
			if pullDownRootSubtreeIfNeeded(candidateRootId, &layeredSubtreesSet, rootToNodeDistanceMap) {
				rootIdPulledDown = candidateRootId
				layeredSubtreesSet.add(candidateRootId)
				break
			}
		}
		if rootIdPulledDown != 0 {
			// This subtree is now properly layered. We will not pull it down anymore.
			// But other candidate subtrees that share nodes with it, may need to be pulled down.
			candidateSubtreesSet.del(rootIdPulledDown)
		} else {
			// No candidates were pulled down, that means all subtrees are layered, so quit
			break
		}
	}

	// At this point, rootToNodeDistanceMap contains layer numbers shifted when needed:
	// after pullDownRootSubtreeIfNeeded, distance magically becomes layer number
	for i := range nodeDefs[1:] {
		nodeId := int16(i + 1)
		rootId := nodeToRootMap[nodeId]
		nodeLayerMap[nodeId] = rootToNodeDistanceMap[rootId][nodeId]
	}

	return nodeLayerMap
}

// func nodeIteratorForLayer(nodeLayerMap []int, layerIdx int) func(yield func(int, int16) bool) {
// 	cnt := 0
// 	return func(yield func(int, int16) bool) {
// 		for i, nodeLayerIdx := range nodeLayerMap[1:] {
// 			nodeId := int16(i + 1)
// 			if nodeLayerIdx == layerIdx {
// 				if !yield(cnt, nodeId) {
// 					return
// 				}
// 				cnt++
// 			}
// 		}
// 	}
// }
