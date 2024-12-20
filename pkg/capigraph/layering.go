package capigraph

import (
	"math"
	"slices"
)

func maxDistToPullSubtreeDownRecursive(subtreeRoot int16, nodeLayerMap []int, nodeToRootMap []int16, priChildrenMap [][]int16, secChildrenMap [][]int16) int {
	allowedDist := math.MaxInt
	for _, secChild := range secChildrenMap[subtreeRoot] {
		// If this sec child has the same root as subtreeRoot - ignore it,
		// we can safely push it down later, so no distance restrictions added.
		// If we don't do this, we may end up in an infinite pull/push loop, see testNodeDefsPriAndSecInfinitePulldown
		if nodeToRootMap[subtreeRoot] == nodeToRootMap[secChild] {
			continue
		}
		thisLayer := nodeLayerMap[subtreeRoot]
		secDependantLayer := nodeLayerMap[secChild]
		maxAllowedDistCandidate := secDependantLayer - thisLayer - 1
		if maxAllowedDistCandidate < 0 {
			// secChild is already at the level of subtreeRoot or even higher.
			// It will be probably pushed down on the next iteration.
			// Anyways, no go.
			return 0
		}
		if maxAllowedDistCandidate < allowedDist {
			allowedDist = maxAllowedDistCandidate
		}
	}

	// Use primary children for recursion - we are checking a subtree of this root.
	for _, priChild := range priChildrenMap[subtreeRoot] {
		priChildAllowedDist := maxDistToPullSubtreeDownRecursive(priChild, nodeLayerMap, nodeToRootMap, priChildrenMap, secChildrenMap)
		// Do not allow pulling down more than this child allows
		if priChildAllowedDist < allowedDist {
			allowedDist = priChildAllowedDist
		}
	}
	return allowedDist
}

func buildLayerMap(nodeDefs []NodeDef) []int {
	allChildrenMap := buildAllChildrenMap(nodeDefs)
	secChildrenMap := buildSecChildrenMap(nodeDefs)
	priChildrenMap := buildPriChildrenMap(nodeDefs)
	nodeToRootMap := buildNodeToRootMap(buildPriParentMap(nodeDefs))
	nodeLayerMap := slices.Repeat([]int{MissingLayer}, len(nodeDefs))
	// Initialize layers for root nodes
	for i := range len(nodeDefs) - 1 {
		nodeIdx := i + 1
		if nodeDefs[nodeIdx].PriIn.SrcId == 0 {
			nodeLayerMap[nodeIdx] = 0
		}
	}

	for {
		reScan := false
		// Push down stressed by pri or sec parent
		for i := range len(nodeDefs) - 1 {
			nodeIdx := int16(i + 1)
			parentNodeIdx := nodeDefs[nodeIdx].PriIn.SrcId
			if parentNodeIdx != 0 && nodeLayerMap[parentNodeIdx] != MissingLayer {
				if nodeLayerMap[nodeIdx] == MissingLayer || nodeLayerMap[nodeIdx] <= nodeLayerMap[parentNodeIdx] {
					nodeLayerMap[nodeIdx] = nodeLayerMap[parentNodeIdx] + 1
					reScan = true
				}
			}
			for _, secParent := range nodeDefs[nodeIdx].SecIn {
				parentNodeIdx := secParent.SrcId
				if parentNodeIdx != 0 && nodeLayerMap[parentNodeIdx] != MissingLayer {
					if nodeLayerMap[nodeIdx] == MissingLayer || nodeLayerMap[nodeIdx] <= nodeLayerMap[parentNodeIdx] {
						nodeLayerMap[nodeIdx] = nodeLayerMap[parentNodeIdx] + 1
						reScan = true
					}
				}
			}
		}

		// Pull down if there is room
		for i := range len(nodeDefs) - 1 {
			nodeIdx := int16(i + 1)
			minChildLayer := math.MaxInt
			for _, childIdx := range allChildrenMap[nodeIdx] {
				if nodeLayerMap[childIdx] < minChildLayer {
					minChildLayer = nodeLayerMap[childIdx]
				}
			}
			if minChildLayer < math.MaxInt {
				allowedDist := minChildLayer - 1 - nodeLayerMap[nodeIdx]
				if allowedDist > 0 {
					nodeLayerMap[nodeIdx] += allowedDist
					reScan = true
				} else {
					// Ok, no room between this nodeIdx and its immediate children (pri or sec).
					// Can we pull the whole nodeIdx subtree down?
					subtreeAllowedDist := maxDistToPullSubtreeDownRecursive(nodeIdx, nodeLayerMap, nodeToRootMap, priChildrenMap, secChildrenMap)
					// It can return math.MaxInt which means we can pull down this subtree to infinity. But it does not make sense, so leave it where it is.
					if subtreeAllowedDist < math.MaxInt && subtreeAllowedDist > 0 {
						nodeLayerMap[nodeIdx] += subtreeAllowedDist
						reScan = true
					}
				}
			}
		}

		if !reScan {
			break
		}
	}

	return nodeLayerMap
}
