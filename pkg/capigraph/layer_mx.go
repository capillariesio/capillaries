package capigraph

import (
	"fmt"
	"strings"
)

const FakeNodeBase int16 = 10000

func sanitizeFakeNodeId(nodeId int16) int16 {
	sanitizedNodeId := nodeId
	if nodeId > FakeNodeBase {
		sanitizedNodeId -= FakeNodeBase
	}
	return sanitizedNodeId
}

type LayerMx [][]int16

func NewLayerMx(nodeDefs []NodeDef, nodeLayerMap []int, rootNodes []int16) (LayerMx, error) {
	maxLayer := 0
	for i := range nodeDefs[1:] {
		nodeId := int16(i + 1)
		nodeLayer := nodeLayerMap[nodeId]
		if nodeLayer > maxLayer {
			maxLayer = nodeLayer
		}
	}

	layerMx := make(LayerMx, maxLayer+1)

	for layerIdx := range maxLayer + 1 {
		layerMx[layerIdx] = make([]int16, 0, MaxLayerLen)
	}

	// This will populate mx with a sample valid ids. As we overwrite top layers, bottom layers will become invalid.
	// This is why we have a similar piece of code in mxIterRecursive. Let's keep this code though:
	// it inserts fake nodes and calculates each layer size nicely
	priChildrenMap := buildPriChildrenMap(nodeDefs)
	for _, rootNodeId := range rootNodes {
		layerMx.addNodeRecursively(rootNodeId, nodeLayerMap, priChildrenMap)
	}

	for layerIdx := range len(layerMx) {
		if len(layerMx[layerIdx]) > MaxLayerLen {
			return nil, fmt.Errorf("cannot create NewLayerMx, too many nodes in row %d: %d; max allowed is %d", layerIdx, len(layerMx[layerIdx]), MaxLayerLen)
		}
	}

	return layerMx, nil
}

func (mx LayerMx) String() string {
	sb := strings.Builder{}
	for i, row := range mx {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%d %v", i, row))
	}
	return sb.String()
}

func (mx LayerMx) clone() LayerMx {
	newMx := make(LayerMx, len(mx))
	for i, row := range mx {
		newMx[i] = make([]int16, len(row))
		copy(newMx[i], row)
	}
	return newMx
}

func (mx LayerMx) addNodeRecursively(curNodeId int16, nodeLayerMap []int, priChildrenMap [][]int16) {
	curLayerIdx := nodeLayerMap[curNodeId]
	mx[curLayerIdx] = append(mx[curLayerIdx], curNodeId)

	for _, childId := range priChildrenMap[curNodeId] {
		childLayerIdx := nodeLayerMap[childId]
		if childLayerIdx > curLayerIdx+1 {
			// Insert fake nodes when pri edge len > 1
			interLayerIdx := curLayerIdx + 1
			for interLayerIdx < childLayerIdx {
				mx[interLayerIdx] = append(mx[interLayerIdx], childId+FakeNodeBase)
				interLayerIdx++
			}
		}
		mx.addNodeRecursively(childId, nodeLayerMap, priChildrenMap)
	}
}

// TODO: this function is unused, although works nicely. Remove later.
func (mx LayerMx) isMonotonous(priParentMap []int16, totalNodes int) bool {
	edgeStarts := make([]int16, MaxLayerLen)
	edgeStartIndexes := make([]int, MaxLayerLen)
	prevLayerNodeIdxMap := make([]int, totalNodes)

	for layerIdx := range len(mx) {
		if layerIdx > 0 {
			edgeStarts := edgeStarts[:len(mx[layerIdx])]
			for j, edgeEndNodeId := range mx[layerIdx] {
				edgeStarts[j] = priParentMap[sanitizeFakeNodeId(edgeEndNodeId)]
			}

			edgeStartIndexes := edgeStartIndexes[:len(mx[layerIdx])]
			for j, edgeStartIdx := range edgeStarts {
				if edgeStartIdx == -1 {
					// Simulate monotonous
					if j == 0 {
						edgeStartIndexes[j] = 0
					} else {
						edgeStartIndexes[j] = edgeStartIndexes[j-1]
					}
				} else {
					edgeStartIndexes[j] = prevLayerNodeIdxMap[sanitizeFakeNodeId(edgeStartIdx)]
				}
			}

			// Verify it's growing monotonously
			for j := range len(edgeStartIndexes) - 1 {
				if edgeStartIndexes[j] > edgeStartIndexes[j+1] {
					return false
				}
			}
		}
		for idx, nodeId := range mx[layerIdx] {
			prevLayerNodeIdxMap[sanitizeFakeNodeId(nodeId)] = idx
		}
	}
	return true
}
