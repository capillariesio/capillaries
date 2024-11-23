package capigraph

import (
	"math"
	"slices"
)

type VizNodeHierarchy struct {
	VizNodeMap   []VizNode
	NodeDefs     []NodeDef
	PriParentMap []int16
	RootMap      []int16
	NodeFo       *FontOptions
}

func NewVizNodeHierarchy(nodeDefs []NodeDef, nodeFo *FontOptions) *VizNodeHierarchy {
	vnh := VizNodeHierarchy{}
	vnh.NodeDefs = nodeDefs
	vnh.PriParentMap = buildPriParentMap(nodeDefs)
	vnh.RootMap = buildNodeToRootMap(vnh.PriParentMap)
	vnh.NodeFo = nodeFo
	return &vnh
}

func (vnh *VizNodeHierarchy) insertRootToNearestParent(rootVizNode *VizNode, leftId int16, rightId int16) {
	leftParentVisitedMap := make([]*VizNode, len(vnh.VizNodeMap))
	leftChildId := leftId
	for {
		leftParentId := vnh.PriParentMap[leftChildId]
		if leftParentId == MissingNodeId {
			break
		}
		leftParentVisitedMap[leftParentId] = &(vnh.VizNodeMap[leftParentId])
		leftChildId = leftParentId
	}

	var rightParentId int16
	rightChildId := rightId
	for {
		rightParentId = vnh.PriParentMap[rightChildId]
		commonParentVizNode := leftParentVisitedMap[rightParentId]
		if commonParentVizNode != nil {
			// Now rightChildId and commonParentItem contain the place to insert
			for childIdx, childVizNode := range commonParentVizNode.PriChildrenAndEnclosedRoots {
				if childVizNode.Def.Id == rightChildId {
					commonParentVizNode.PriChildrenAndEnclosedRoots = slices.Insert(commonParentVizNode.PriChildrenAndEnclosedRoots, childIdx, rootVizNode)
					return
				}
			}
			panic("ddd")
		}
		rightChildId = rightParentId
	}
}

// func insertRoot(perm []int16, idx int, rootMap []int16, priParentMap []int16, hierarchyItemLookup map[string]*PriHierarchyItem, newHierarchyItem *PriHierarchyItem) {
func (vnh *VizNodeHierarchy) insertRoot(rootVizNode *VizNode, perm []int16, idx int) {
	thisRootId := perm[idx]
	i := idx - 1
	for i >= 0 {
		leftRootId := vnh.RootMap[perm[i]]
		if leftRootId != thisRootId {
			j := idx + 1
			for j < len(perm) {
				if leftRootId == vnh.RootMap[perm[j]] {
					vnh.insertRootToNearestParent(rootVizNode, perm[i], perm[j])
					return
				}
				j += 1
			}
		}
		i -= 1
	}

	// No enclose, resort to adding it to the top ubernode
	topItem := &(vnh.VizNodeMap[0])
	topItem.PriChildrenAndEnclosedRoots = append(topItem.PriChildrenAndEnclosedRoots, rootVizNode)
}

func (vnh *VizNodeHierarchy) buildNewRootSubtreeHierarchy(mx LayerMx) {
	vnh.VizNodeMap = make([]VizNode, len(vnh.NodeDefs))
	topItem := &(vnh.VizNodeMap[0])
	topItem.Layer = -1
	topItem.PriChildrenAndEnclosedRoots = make([]*VizNode, 0, MaxLayerLen)

	// Initialize static (non-hierarchy-related) properties
	for layer, row := range mx {
		for _, nodeId := range row {
			// Static properties
			vizNode := &(vnh.VizNodeMap[nodeId])
			vizNode.Def = &(vnh.NodeDefs[nodeId])
			vizNode.RootId = vnh.RootMap[nodeId]
			vizNode.Layer = layer
			incomingEdgesLen := len(vizNode.Def.SecIn)
			if vizNode.Def.PriIn.SrcId != 0 {
				incomingEdgesLen++
			}
			vizNode.IncomingVizEdges = make([]VizEdge, incomingEdgesLen)
			// Properties to change from mx to mx
			vizNode.PriChildrenAndEnclosedRoots = make([]*VizNode, 0, MaxLayerLen)
		}
	}

	// Add pri children to PriChildrenAndEnclosedRoots
	for _, row := range mx {
		for _, nodeId := range row {
			vizNode := &(vnh.VizNodeMap[nodeId])
			if vizNode.RootId != nodeId {
				// This is a non-root node, just append it
				parentNodeId := vnh.PriParentMap[nodeId]
				parentVizNode := &(vnh.VizNodeMap[parentNodeId])
				parentVizNode.PriChildrenAndEnclosedRoots = append(parentVizNode.PriChildrenAndEnclosedRoots, vizNode)
			}
		}
	}

	// Add roots to PriChildrenAndEnclosedRoots
	for _, row := range mx {
		for j, nodeId := range row {
			rootVizNode := &(vnh.VizNodeMap[nodeId])
			if rootVizNode.RootId == nodeId {
				vnh.insertRoot(rootVizNode, row, j)
			}
		}
	}
}

func (vnh *VizNodeHierarchy) reuseRootSubtreeHierarchy(mx LayerMx) {
	vnh.VizNodeMap[0].clean()
	vnh.VizNodeMap[0].Layer = -1

	for layer, row := range mx {
		for _, nodeId := range row {
			vn := &(vnh.VizNodeMap[nodeId])
			vn.clean()
			vn.Layer = layer
		}
	}

	for _, row := range mx {
		for _, nodeId := range row {
			rootId := vnh.RootMap[nodeId]
			if rootId != nodeId {
				// This is a non-root node, just append it
				vn := &(vnh.VizNodeMap[nodeId])
				parentNodeId := vnh.PriParentMap[nodeId]
				parentVizNode := &(vnh.VizNodeMap[parentNodeId])
				parentVizNode.PriChildrenAndEnclosedRoots = append(parentVizNode.PriChildrenAndEnclosedRoots, vn)
			}
		}
	}

	// Same for root nodes
	for _, row := range mx {
		for j, nodeId := range row {
			rootId := vnh.RootMap[nodeId]
			if rootId == nodeId {
				vn := &(vnh.VizNodeMap[nodeId])
				vnh.insertRoot(vn, row, j)
			}
		}
	}
}

func (vnh *VizNodeHierarchy) PopulateSubtreeHierarchy(mx LayerMx) {
	if vnh.VizNodeMap == nil {
		vnh.buildNewRootSubtreeHierarchy(mx)
	} else {
		vnh.reuseRootSubtreeHierarchy(mx)
	}
}

type RectDimension struct {
	W float64
	H float64
}

func getNodeDimensions(nodeDef *NodeDef, fo *FontOptions) (float64, float64) {
	w, h := getTextDimensions(nodeDef.Text, fo.Typeface, fo.Weight, fo.SizeInPixels)
	w += float64(fo.SizeInPixels)
	if nodeDef.IconId != "" {
		// Add space for the HxH icon plus font-size
		w += h + fo.SizeInPixels
	}
	h += float64(fo.SizeInPixels)
	return w, h
}

const (
	SecEdgeLabelGapFromSourceInLines        float64 = 5.0
	PriEdgeLabelGapFromDestinatioInLines    float64 = 2.0
	gapBetweenSecAndPrimeEdgeLabelsInPixels float64 = 10.0
	NodeHorizontalGapInPixels               float64 = 20.0
	SecEdgeStartXRatio                      float64 = 0.45
	SecEdgeEndXRatio                        float64 = 0.55
)

func (vnh *VizNodeHierarchy) populateNodeDimensionsRecursive(vizNode *VizNode) {
	for i, childItem := range vizNode.PriChildrenAndEnclosedRoots {
		vnh.populateNodeDimensionsRecursive(childItem)
		if i != 0 {
			vizNode.TotalW += NodeHorizontalGapInPixels
		}
		vizNode.TotalW += childItem.TotalW
	}
	// Check it's not the top ubernode and add NodeDef width
	if vizNode.Def != nil {
		vizNode.NodeW, vizNode.NodeH = getNodeDimensions(vizNode.Def, vnh.NodeFo)
		if vizNode.TotalW < vizNode.NodeW {
			vizNode.TotalW = vizNode.NodeW
		}
	}
}

func (vnh *VizNodeHierarchy) PopulateNodeDimensions() {
	vnh.populateNodeDimensionsRecursive(&vnh.VizNodeMap[0])
}
func populateHierarchyItemXCoordRecursive(vizNode *VizNode) {
	// Decide where to start drawing child items: their cumulative width may be well smaller than parent's
	cumulativeChildrenAndEnclosedRootsWidth := 0.0
	for j, childItem := range vizNode.PriChildrenAndEnclosedRoots {
		cumulativeChildrenAndEnclosedRootsWidth += childItem.TotalW
		if j != len(vizNode.PriChildrenAndEnclosedRoots)-1 {
			cumulativeChildrenAndEnclosedRootsWidth += NodeHorizontalGapInPixels
		}
	}
	curX := vizNode.X + (vizNode.TotalW-cumulativeChildrenAndEnclosedRootsWidth)/2
	for _, childItem := range vizNode.PriChildrenAndEnclosedRoots {
		childItem.X = curX
		populateHierarchyItemXCoordRecursive(childItem)
		curX += childItem.TotalW + NodeHorizontalGapInPixels
	}
}

func (vnh *VizNodeHierarchy) PopulateHierarchyItemsXCoords() {
	vnh.VizNodeMap[0].X = 0
	populateHierarchyItemXCoordRecursive(&vnh.VizNodeMap[0])
}

func calculateSecShiftRecursive(srcVizNode *VizNode, tgtVizNode *VizNode) float64 {
	startX := srcVizNode.X + srcVizNode.TotalW/2 - srcVizNode.NodeW*(0.5-SecEdgeStartXRatio)
	// Outgoing sec edges go to somewhere in the second half, so they are look secondary
	endX := tgtVizNode.X + tgtVizNode.TotalW/2 - tgtVizNode.NodeW*(0.5-SecEdgeEndXRatio)
	return math.Abs(endX - startX)
}

func (vnh *VizNodeHierarchy) CalculateTotalHorizontalShift() float64 {
	sum := 0.0
	for i := range len(vnh.VizNodeMap) {
		tgtVizNode := &vnh.VizNodeMap[i]
		if tgtVizNode.Def != nil && len(tgtVizNode.Def.SecIn) > 0 {
			for _, edge := range tgtVizNode.Def.SecIn {
				sum += calculateSecShiftRecursive(&vnh.VizNodeMap[edge.SrcId], tgtVizNode)
			}
		}
	}
	return sum
}
