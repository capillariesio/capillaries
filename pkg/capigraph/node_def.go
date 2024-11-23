package capigraph

import "slices"

const MissingNodeId int16 = -1
const MissingDistanceFromRootToNode int = -2
const MissingRootSubtreeHeight int = -3
const MissingLayer int = -4

type EdgeDef struct {
	SrcId int16
	Text  string
}

type NodeDef struct {
	Id     int16
	Text   string
	PriIn  EdgeDef
	SecIn  []EdgeDef
	IconId string
}

func buildPriParentMap(nodeDefs []NodeDef) []int16 {
	pMap := slices.Repeat([]int16{MissingNodeId}, len(nodeDefs))
	for i, nodeDef := range nodeDefs[1:] {
		nodeId := int16(i + 1)
		if nodeDef.PriIn.SrcId == 0 {
			pMap[nodeId] = MissingNodeId
		} else {
			pMap[nodeId] = nodeDefs[nodeDef.PriIn.SrcId].Id
		}
	}
	return pMap
}

func buildNodeToRootMap(priParentMap []int16) []int16 {
	rMap := slices.Repeat([]int16{MissingNodeId}, len(priParentMap))
	for i, parentId := range priParentMap[1:] {
		srcNodeId := int16(i + 1)
		candidateNodeId := int16(srcNodeId)
		for parentId != MissingNodeId {
			candidateNodeId = parentId
			parentId = priParentMap[parentId]
		}

		rMap[srcNodeId] = candidateNodeId
	}
	return rMap
}

func buildAllChildrenMap(nodeDefs []NodeDef) [][]int16 {
	acMap := make([][]int16, len(nodeDefs))
	for i, nodeDef := range nodeDefs[1:] {
		nodeId := int16(i + 1)
		parentId := nodeDef.PriIn.SrcId
		if parentId != 0 {
			if acMap[parentId] == nil {
				acMap[parentId] = make([]int16, 0, 32)
			}
			acMap[parentId] = append(acMap[parentId], int16(nodeId))
		}

		for _, edge := range nodeDef.SecIn {
			if acMap[edge.SrcId] == nil {
				acMap[edge.SrcId] = make([]int16, 0, 32)
			}
			acMap[edge.SrcId] = append(acMap[edge.SrcId], int16(nodeId))
		}
	}
	return acMap
}

func buildPriChildrenMap(nodeDefs []NodeDef) [][]int16 {
	pcMap := make([][]int16, len(nodeDefs))
	for i, nodeDef := range nodeDefs[1:] {
		nodeId := int16(i + 1)
		parentId := nodeDef.PriIn.SrcId
		if parentId != 0 {
			if pcMap[parentId] == nil {
				pcMap[parentId] = make([]int16, 0, 32)
			}
			pcMap[parentId] = append(pcMap[parentId], int16(nodeId))
		}
	}
	return pcMap
}

func buildRootNodeList(priParentMap []int16) []int16 {
	rootNodes := make([]int16, 0, 100)
	for i, parentId := range priParentMap[1:] {
		nodeId := int16(i + 1)
		if parentId == MissingNodeId {
			rootNodes = append(rootNodes, int16(nodeId))
		}
	}
	return rootNodes
}

func assignDistanceFromRoot(acMap [][]int16, nodeId int16, dist int, fromOneRootToNodeDistanceMap []int) {
	existingDist := fromOneRootToNodeDistanceMap[nodeId]
	if existingDist < dist {
		fromOneRootToNodeDistanceMap[nodeId] = dist
	}
	for _, childId := range acMap[nodeId] {
		assignDistanceFromRoot(acMap, int16(childId), dist+1, fromOneRootToNodeDistanceMap)
	}
}

func buildRootToNodeDistanceMap(totalNodes int, rootNodes []int16, acMap [][]int16) [][]int {
	fromRootToNodeDistanceMap := make([][]int, totalNodes)
	for _, rootNodeId := range rootNodes {
		fromRootToNodeDistanceMap[rootNodeId] = slices.Repeat([]int{MissingDistanceFromRootToNode}, len(acMap))
		assignDistanceFromRoot(acMap, rootNodeId, 0, fromRootToNodeDistanceMap[rootNodeId])
	}
	return fromRootToNodeDistanceMap
}

func getMaxSubtreeHeightByRoot(rootId int16, fromRootToNodeDistanceMap [][]int) int {
	maxHeight := 0
	for _, distanceFromRoot := range fromRootToNodeDistanceMap[rootId] {
		if distanceFromRoot > maxHeight {
			maxHeight = distanceFromRoot
		}
	}
	return maxHeight
}

func buildRootSubtreeHeightsMap(totalNodes int, rootNodes []int16, rootToNodeDistanceMap [][]int) ([]int, int) {
	rootSubtreeHeights := slices.Repeat([]int{MissingRootSubtreeHeight}, totalNodes)
	maxSubtreeHeight := 0
	for _, rootNodeId := range rootNodes {
		subtreeHeight := getMaxSubtreeHeightByRoot(rootNodeId, rootToNodeDistanceMap)
		rootSubtreeHeights[rootNodeId] = subtreeHeight
		if subtreeHeight > maxSubtreeHeight {
			maxSubtreeHeight = subtreeHeight
		}
	}
	return rootSubtreeHeights, maxSubtreeHeight
}
