package capigraph

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

type RectDimension struct {
	W float64
	H float64
}

type VizNodeHierarchy struct {
	VizNodeMap         []VizNode
	NodeDefs           []NodeDef
	PriParentMap       []int16
	RootMap            []int16
	NodeFo             FontOptions
	EdgeFo             FontOptions
	NodeDimensionMap   []RectDimension
	PriEdgeLabelDimMap []RectDimension
	SecEdgeLabelDimMap [][]RectDimension
	TotalLayers        int
	UpperLayerGapMap   []float64
}

func (vnh *VizNodeHierarchy) String() string {
	sb := strings.Builder{}
	for i, vn := range vnh.VizNodeMap {
		if i == 0 {
			continue
		}
		if i > 1 {
			sb.WriteString(", ")
		}
		sb.WriteString(vn.String())
	}
	return sb.String()
}

func NewVizNodeHierarchy(nodeDefs []NodeDef, nodeFo FontOptions, edgeFo FontOptions) *VizNodeHierarchy {
	vnh := VizNodeHierarchy{}
	vnh.NodeDefs = nodeDefs
	vnh.PriParentMap = buildPriParentMap(nodeDefs)
	vnh.RootMap = buildNodeToRootMap(vnh.PriParentMap)
	vnh.NodeFo = nodeFo
	vnh.EdgeFo = edgeFo
	vnh.NodeDimensionMap = make([]RectDimension, len(nodeDefs))
	for i := range len(nodeDefs) - 1 {
		nodeDef := nodeDefs[i+1]
		vnh.NodeDimensionMap[nodeDef.Id] = getNodeDimensions(&nodeDef, vnh.NodeFo)
	}

	vnh.PriEdgeLabelDimMap = make([]RectDimension, len(nodeDefs))
	vnh.SecEdgeLabelDimMap = make([][]RectDimension, len(nodeDefs))
	for _, nodeDef := range nodeDefs {
		if nodeDef.PriIn.SrcId != 0 {
			w, h := getTextDimensions(nodeDef.PriIn.Text, edgeFo.Typeface, edgeFo.Weight, edgeFo.SizeInPixels, edgeFo.Interval)
			vnh.PriEdgeLabelDimMap[nodeDef.Id].W, vnh.PriEdgeLabelDimMap[nodeDef.Id].H = getLabelDimensionsFromTextDimensions(w, h, edgeFo.SizeInPixels*LabelTextDimensionMargin)
		}
		vnh.SecEdgeLabelDimMap[nodeDef.Id] = make([]RectDimension, len(nodeDef.SecIn))
		for edgeIdx, edgeDef := range nodeDef.SecIn {
			w, h := getTextDimensions(edgeDef.Text, edgeFo.Typeface, edgeFo.Weight, edgeFo.SizeInPixels, edgeFo.Interval)
			vnh.SecEdgeLabelDimMap[nodeDef.Id][edgeIdx].W, vnh.SecEdgeLabelDimMap[nodeDef.Id][edgeIdx].H = getLabelDimensionsFromTextDimensions(w, h, edgeFo.SizeInPixels*LabelTextDimensionMargin)
		}
	}
	return &vnh
}

func (vnh *VizNodeHierarchy) insertRootToNearestParent(rootVizNode *VizNode, leftId int16, rightId int16) {
	leftParentVisitedMap := make([]*VizNode, len(vnh.VizNodeMap))

	// Left branch: walk up and harvest ids
	leftChildId := leftId
	for {
		leftParentId := vnh.PriParentMap[sanitizeFakeNodeId(leftChildId)]
		if leftParentId == MissingNodeId {
			break
		}
		leftParentVisitedMap[leftParentId] = &(vnh.VizNodeMap[leftParentId])
		leftChildId = leftParentId
	}

	// Right branch: go up, identify first common parent, insert
	var rightParentId int16
	rightChildId := sanitizeFakeNodeId(rightId)
	for {
		rightParentId = vnh.PriParentMap[rightChildId]
		commonParentVizNode := leftParentVisitedMap[rightParentId]
		if commonParentVizNode != nil {
			// Now rightChildId and commonParentVizNode contain the place to insert
			for childIdx, childVizNode := range commonParentVizNode.PriChildrenAndEnclosedRoots {
				if childVizNode.Def.Id == rightChildId {
					commonParentVizNode.PriChildrenAndEnclosedRoots = slices.Insert(commonParentVizNode.PriChildrenAndEnclosedRoots, childIdx, rootVizNode)
					return
				}
			}
			panic(fmt.Sprintf("insertRootToNearestParent for root id %d cannot find parent for left %d and right %d", rootVizNode.Def.Id, leftId, rightId))
		}
		rightChildId = rightParentId
	}
}

func (vnh *VizNodeHierarchy) insertRoot(rootVizNode *VizNode, perm []int16, idx int) {
	thisRootId := perm[idx]
	// Start at the left neighbour, move left
	i := idx - 1
	for i >= 0 {
		leftRootId := vnh.RootMap[sanitizeFakeNodeId(perm[i])]
		if leftRootId != thisRootId {
			// Left neighbour belongs to another AND the closest root subtree, now find the nearest common parent for enclosure.
			// Start with the nearest right neighbour and move right.
			j := idx + 1
			for j < len(perm) {
				if leftRootId == vnh.RootMap[sanitizeFakeNodeId(perm[j])] {
					// Same root as the left neighbour. Now:
					// - go up and find the nearst common parent.
					// - insert rootVizNode between first-generation children of the nearst common parent
					vnh.insertRootToNearestParent(rootVizNode, perm[i], perm[j])
					return
				}
				j++
			}
		}
		i--
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

	vnh.TotalLayers = 0
	// Initialize static (non-hierarchy-related) properties
	for layer, row := range mx {
		if layer+1 > vnh.TotalLayers {
			vnh.TotalLayers = layer + 1
		}
		for _, nodeId := range row {
			// Static properties: remain the same regardless of the mx
			vizNode := &(vnh.VizNodeMap[sanitizeFakeNodeId(nodeId)])
			vizNode.Def = &(vnh.NodeDefs[sanitizeFakeNodeId(nodeId)])
			vizNode.RootId = vnh.RootMap[sanitizeFakeNodeId(nodeId)]
			if nodeId > FakeNodeBase {
				// For now, set it to -1, later iteratin will eventually set it to proper layer (where the actual child node resides)
				// This is not crucial, but may help when troubleshooting
				vizNode.Layer = -1
			} else {
				// This is the real node - the end of fake seq
				vizNode.Layer = layer
			}
			r := vnh.NodeDimensionMap[vizNode.Def.Id]
			vizNode.NodeW = r.W
			vizNode.NodeH = r.H
			incomingEdgesLen := len(vizNode.Def.SecIn)
			if vizNode.Def.PriIn.SrcId != 0 {
				incomingEdgesLen++
			}
			vizNode.IncomingVizEdges = make([]VizEdge, incomingEdgesLen) // Just pre-allocate, it will be populated by PopulateEdgeLabelDimensions

			// Properties to change from mx to mx
			vizNode.PriChildrenAndEnclosedRoots = make([]*VizNode, 0, MaxLayerLen) // No need to fill at the moment, just pre-allocate
		}
	}
}

func (vnh *VizNodeHierarchy) reuseRootSubtreeHierarchy(mx LayerMx) {
	vnh.VizNodeMap[0].cleanPropertiesSubjectToPermutation()
	vnh.VizNodeMap[0].Layer = -1

	// Re-init non-static properties and add pri children to PriChildrenAndEnclosedRoots
	for layer, row := range mx {
		for _, nodeId := range row {
			vizNode := &vnh.VizNodeMap[sanitizeFakeNodeId(nodeId)]
			vizNode.cleanPropertiesSubjectToPermutation()
			// Do not handle root items yet - handling them requires enclosing tree in place.
			// For non-roots - add it to some node on some previous layer in the order they appear on this layer, honoring the perm
			rootId := vnh.RootMap[sanitizeFakeNodeId(nodeId)]
			if rootId != nodeId {
				// Add only the last node of the fake sequence, so there is only one copy of it among the children
				if vizNode.Layer == layer {
					parentVizNode := &vnh.VizNodeMap[vnh.PriParentMap[sanitizeFakeNodeId(nodeId)]]
					parentVizNode.PriChildrenAndEnclosedRoots = append(parentVizNode.PriChildrenAndEnclosedRoots, vizNode)
				}
			}
		}
	}

	// Add roots to PriChildrenAndEnclosedRoots (now i's ok to do that)
	for _, row := range mx {
		for j, nodeId := range row {
			if nodeId < FakeNodeBase {
				rootId := vnh.RootMap[nodeId]
				if rootId == nodeId {
					vn := &(vnh.VizNodeMap[nodeId])
					vnh.insertRoot(vn, row, j)
				}
			}
		}
	}
}

const NodeTextDimensionMargin float64 = 1.0
const NodeTextIconInterval float64 = 1.0
const LabelTextDimensionMargin float64 = 0.5

func getNodeDimensions(nodeDef *NodeDef, fo FontOptions) RectDimension {
	w, h := getTextDimensions(nodeDef.Text, fo.Typeface, fo.Weight, fo.SizeInPixels, fo.Interval)
	w += float64(fo.SizeInPixels) * NodeTextDimensionMargin * 2 // left+right
	if nodeDef.IconId != "" {
		// Add space for the HxH icon plus font-size*some coefficient
		w += h + fo.SizeInPixels*NodeTextIconInterval
	}
	h += float64(fo.SizeInPixels) * NodeTextDimensionMargin * 2 // top+bottom
	return RectDimension{w, h}
}

const (
	SecEdgeLabelGapFromSourceInLines        float64 = 5.0
	PriEdgeLabelGapFromDestinatioInLines    float64 = 2.0
	gapBetweenSecAndPrimeEdgeLabelsInPixels float64 = 10.0
	NodeHorizontalGapInPixels               float64 = 20.0
	SecEdgeOffsetX                          float64 = 10
)

func (vnh *VizNodeHierarchy) populateNodeTotalWidthRecursive(vizNode *VizNode) {
	// Recursively visit children and add their TotalW to this TotalW
	for i, childItem := range vizNode.PriChildrenAndEnclosedRoots {
		vnh.populateNodeTotalWidthRecursive(childItem)
		if i != 0 {
			vizNode.TotalW += NodeHorizontalGapInPixels
		}
		vizNode.TotalW += childItem.TotalW
	}

	// If this node has really wide text, it may be even wider than
	// children subtree. Pay attention to this case.
	if vizNode.Def != nil {
		if vizNode.TotalW < vizNode.NodeW {
			vizNode.TotalW = vizNode.NodeW
		}
	}
}

func (vnh *VizNodeHierarchy) PopulateNodeTotalWidth() {
	vnh.populateNodeTotalWidthRecursive(&vnh.VizNodeMap[0])
}
func populateNodeXCoordRecursive(vizNode *VizNode) {
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
		populateNodeXCoordRecursive(childItem)
		curX += childItem.TotalW + NodeHorizontalGapInPixels
	}
}

func (vnh *VizNodeHierarchy) PopulateNodesXCoords() {
	vnh.VizNodeMap[0].X = 0
	populateNodeXCoordRecursive(&vnh.VizNodeMap[0])
}

func (vnh *VizNodeHierarchy) CalculateTotalHorizontalShift() float64 {
	sum := 0.0
	for i := range len(vnh.VizNodeMap) {
		tgtVizNode := &vnh.VizNodeMap[i]
		if tgtVizNode.Def != nil && len(tgtVizNode.Def.SecIn) > 0 {
			for _, edge := range tgtVizNode.Def.SecIn {
				srcVizNode := vnh.VizNodeMap[edge.SrcId]
				startX := srcVizNode.X + srcVizNode.TotalW/2.0 - srcVizNode.NodeW/2.0
				endX := tgtVizNode.X + tgtVizNode.TotalW/2.0 - tgtVizNode.NodeW/2.0
				sum += math.Abs(endX - startX)
			}
		}
	}
	return sum
}

// Merely copies pre-calculated edge label dimensions to the hierarchy vizitems
func (vnh *VizNodeHierarchy) PopulateEdgeLabelDimensions() {
	for i := range len(vnh.VizNodeMap) - 1 {
		dstVizNode := &(vnh.VizNodeMap[i+1])

		// Pri edge
		incomingEdgeIdx := 0
		if dstVizNode.Def.PriIn.SrcId != 0 {
			labelRectDim := vnh.PriEdgeLabelDimMap[dstVizNode.Def.Id]
			dstVizNode.IncomingVizEdges[incomingEdgeIdx] = VizEdge{dstVizNode.Def.PriIn, HierarchyPri, 0.0, 0.0, labelRectDim.W, labelRectDim.H}
			incomingEdgeIdx++
		}

		// Sec edges
		secLabelRectDims := vnh.SecEdgeLabelDimMap[dstVizNode.Def.Id]
		for edgeIdx, edge := range dstVizNode.Def.SecIn {
			dstVizNode.IncomingVizEdges[incomingEdgeIdx] = VizEdge{edge, HierarchySec, 0.0, 0.0, 0.0, 0.0}
			if secLabelRectDims[edgeIdx].W > 0.0 {
				dstVizNode.IncomingVizEdges[incomingEdgeIdx].W = secLabelRectDims[edgeIdx].W
				dstVizNode.IncomingVizEdges[incomingEdgeIdx].H = secLabelRectDims[edgeIdx].H
			}
			incomingEdgeIdx++
		}
	}
}

func (vnh *VizNodeHierarchy) PopulateUpperLayerGapMap(edgeFontSizeInPixels float64) {
	minLayerGap := math.Max(vnh.VizNodeMap[0].TotalW/20.0, vnh.NodeFo.SizeInPixels*3.0) // Purely empiric
	maxPriEdgeLabelHightMap := slices.Repeat([]float64{-1.0}, vnh.TotalLayers)
	maxSecEdgeLabelHightMap := slices.Repeat([]float64{-1.0}, vnh.TotalLayers)
	for i := range len(vnh.VizNodeMap) - 1 {
		hi := &vnh.VizNodeMap[i+1]
		for _, edge := range hi.IncomingVizEdges {
			if edge.HierarchyType == HierarchyPri {
				prevMaxEdgeLabelHeight := maxPriEdgeLabelHightMap[hi.Layer]
				if prevMaxEdgeLabelHeight == -1 || prevMaxEdgeLabelHeight < edge.H {
					maxPriEdgeLabelHightMap[hi.Layer] = edge.H
				}
			} else if edge.HierarchyType == HierarchySec {
				// Make sure it's for the correspondent layer,
				// otherwise it's not gonna work for cases when an edge goes up more than one level
				layer := vnh.VizNodeMap[edge.Edge.SrcId].Layer + 1
				prevMaxEdgeLabelHeight := maxSecEdgeLabelHightMap[layer]
				if prevMaxEdgeLabelHeight == -1 || prevMaxEdgeLabelHeight < edge.H {
					maxSecEdgeLabelHightMap[layer] = edge.H
				}
			} else {
				panic(fmt.Sprintf("PopulateUpperLayerGapMap: unknown hierarchy type %d", edge.HierarchyType))
			}
		}

		// Make sure there are no empty map elements for each layer
		if maxPriEdgeLabelHightMap[hi.Layer] == -1 {
			maxPriEdgeLabelHightMap[hi.Layer] = 0
		}
		if maxSecEdgeLabelHightMap[hi.Layer] == -1 {
			maxSecEdgeLabelHightMap[hi.Layer] = 0
		}
	}

	vnh.UpperLayerGapMap = make([]float64, vnh.TotalLayers)
	for layer, maxPriEdgeLabelHeight := range maxPriEdgeLabelHightMap {
		maxSecEdgeLabelHeight := maxSecEdgeLabelHightMap[layer]
		if maxSecEdgeLabelHeight > 0 && maxPriEdgeLabelHeight > 0 {
			vnh.UpperLayerGapMap[layer] = edgeFontSizeInPixels*SecEdgeLabelGapFromSourceInLines + maxSecEdgeLabelHeight + gapBetweenSecAndPrimeEdgeLabelsInPixels + maxPriEdgeLabelHeight + edgeFontSizeInPixels*PriEdgeLabelGapFromDestinatioInLines
		} else if maxSecEdgeLabelHeight > 0 {
			// Only sec labels
			vnh.UpperLayerGapMap[layer] = edgeFontSizeInPixels*SecEdgeLabelGapFromSourceInLines + maxSecEdgeLabelHeight + edgeFontSizeInPixels*SecEdgeLabelGapFromSourceInLines
		} else if maxPriEdgeLabelHeight > 0 {
			// Only pri labels here
			vnh.UpperLayerGapMap[layer] = edgeFontSizeInPixels*PriEdgeLabelGapFromDestinatioInLines + maxPriEdgeLabelHeight + edgeFontSizeInPixels*PriEdgeLabelGapFromDestinatioInLines
		} else {
			vnh.UpperLayerGapMap[layer] = 0
		}
		if vnh.UpperLayerGapMap[layer] < minLayerGap {
			vnh.UpperLayerGapMap[layer] = minLayerGap
		}
	}
}

func populateLayerHeightsRecursive(vizNode *VizNode, layerHeightMap []float64) {
	if vizNode.Def != nil {
		prevCollectedMaxHeight := layerHeightMap[vizNode.Layer]
		if prevCollectedMaxHeight == -1.0 || prevCollectedMaxHeight < vizNode.NodeH {
			layerHeightMap[vizNode.Layer] = vizNode.NodeH
		}
	}

	for _, childItem := range vizNode.PriChildrenAndEnclosedRoots {
		populateLayerHeightsRecursive(childItem, layerHeightMap)
	}
}

func populateNodeYCoordRecursive(vizNode *VizNode, layerYCoords []float64) {
	if vizNode.Def != nil {
		vizNode.Y = layerYCoords[vizNode.Layer]
	}
	for _, childItem := range vizNode.PriChildrenAndEnclosedRoots {
		populateNodeYCoordRecursive(childItem, layerYCoords)
	}
}

func (vnh *VizNodeHierarchy) PopulateNodesYCoords() {
	// First, assign all level heights recursively
	layerHeightMap := slices.Repeat([]float64{-1.0}, vnh.TotalLayers)
	populateLayerHeightsRecursive(&vnh.VizNodeMap[0], layerHeightMap)

	// Second, figure out the Y coord for each layer
	layerYCoords := make([]float64, len(layerHeightMap))
	curY := 0.0
	for i := range len(layerHeightMap) {
		layerYCoords[i] = curY
		if i < len(layerHeightMap)-1 {
			curY += layerHeightMap[i] + vnh.UpperLayerGapMap[i+1]
		}
	}

	// Third, assign those Y coords to nodes recursively
	populateNodeYCoordRecursive(&vnh.VizNodeMap[0], layerYCoords)
}

func getSecOffsetX(startX float64, endX float64, offset float64) (float64, float64) {
	if startX-offset >= endX+offset {
		// Offset regions do not intersect, upper node is way right
		return -offset, offset
	} else if startX+offset <= endX-offset {
		// Offset regions do not intersect, upper node is way left
		return offset, -offset
	} else if startX > endX {
		// Intersect: left to left
		return -offset, -offset
	}
	// Intersect: right to right
	return SecEdgeOffsetX, SecEdgeOffsetX
}

func (vnh *VizNodeHierarchy) PopulateEdgeLabelCoords() {
	for i := range len(vnh.VizNodeMap) - 1 {
		dstVizNode := &vnh.VizNodeMap[i+1]

		// Pri edge
		if dstVizNode.Def.PriIn.SrcId != 0 {
			srcHiearchyItem := &vnh.VizNodeMap[dstVizNode.Def.PriIn.SrcId]
			startX := srcHiearchyItem.X + srcHiearchyItem.TotalW/2
			startY := srcHiearchyItem.Y + srcHiearchyItem.NodeH
			endX := dstVizNode.X + dstVizNode.TotalW/2
			endY := dstVizNode.Y
			deltaX := endX - startX
			deltaY := endY - startY
			for i := range len(dstVizNode.IncomingVizEdges) {
				if dstVizNode.IncomingVizEdges[i].Edge.SrcId == dstVizNode.Def.PriIn.SrcId {
					labelCenterY := endY - vnh.EdgeFo.SizeInPixels*2 - dstVizNode.IncomingVizEdges[i].H/2
					labelCenterX := endX - (endY-labelCenterY)*deltaX/deltaY
					dstVizNode.IncomingVizEdges[i].Y = labelCenterY - dstVizNode.IncomingVizEdges[i].H/2
					dstVizNode.IncomingVizEdges[i].X = labelCenterX - dstVizNode.IncomingVizEdges[i].W/2
				}
			}
		}

		// Sec edges
		for _, edge := range dstVizNode.Def.SecIn {
			srcHiearchyItem := &vnh.VizNodeMap[edge.SrcId]

			startX := srcHiearchyItem.X + srcHiearchyItem.TotalW/2.0
			endX := dstVizNode.X + dstVizNode.TotalW/2.0
			startOffset, endOffset := getSecOffsetX(startX, endX, vnh.NodeFo.SizeInPixels/2.0)
			startX += startOffset
			endX += endOffset

			startY := srcHiearchyItem.Y + srcHiearchyItem.NodeH
			endY := dstVizNode.Y
			if dstVizNode.Layer > srcHiearchyItem.Layer+1 {
				// Adjust position: we want to put the label between the srcLayer and srcLayer+1
				trueEndY := startY + vnh.UpperLayerGapMap[srcHiearchyItem.Layer+1]
				endX = startX + (endX-startX)*(trueEndY-startY)/(endY-startY)
				endY = trueEndY
			}
			deltaX := endX - startX
			deltaY := endY - startY
			for i := range len(dstVizNode.IncomingVizEdges) {
				if dstVizNode.IncomingVizEdges[i].Edge.SrcId == edge.SrcId {
					labelCenterY := startY + vnh.EdgeFo.SizeInPixels*5 + dstVizNode.IncomingVizEdges[i].H/2
					labelCenterX := startX + (labelCenterY-startY)*deltaX/deltaY
					dstVizNode.IncomingVizEdges[i].Y = labelCenterY - dstVizNode.IncomingVizEdges[i].H/2
					dstVizNode.IncomingVizEdges[i].X = labelCenterX - dstVizNode.IncomingVizEdges[i].W/2
				}
			}
		}
	}
}

func (vnh *VizNodeHierarchy) RemoveDuplicateSecEdgeLabels() {
	secLabelsFromItemMap := map[int16]map[string][]*VizEdge{} // Fight srcid->secText duplicates
	for i := range len(vnh.VizNodeMap) - 1 {
		vizNode := vnh.VizNodeMap[i+1]
		for j := range vizNode.IncomingVizEdges {
			e := &vizNode.IncomingVizEdges[j]
			if e.HierarchyType != HierarchySec {
				continue
			}
			edgeTextMap, ok := secLabelsFromItemMap[e.Edge.SrcId]
			if !ok {
				edgeTextMap = map[string][]*VizEdge{}
				secLabelsFromItemMap[e.Edge.SrcId] = edgeTextMap
			}
			_, ok = edgeTextMap[e.Edge.Text]
			if !ok {
				edgeTextMap[e.Edge.Text] = make([]*VizEdge, 0, 10)
			}
			edgeTextMap[e.Edge.Text] = append(edgeTextMap[e.Edge.Text], e)
		}
	}

	for _, edgeTextMap := range secLabelsFromItemMap {
		for _, edges := range edgeTextMap {
			sort.Slice(edges, func(i int, j int) bool {
				return edges[i].X < edges[j].X
			})
			i := 0
			j := 1
			for i < len(edges) {
				for i < j && j < len(edges) {
					if edges[i].X+edges[i].W > edges[j].X {
						// We have an intersection, remove j
						edges[j].W = 0.0
						edges[j].H = 0.0
						edges = slices.Delete(edges, j, j+1)
					} else {
						j++
					}
				}
				i++
				j = i + 1
			}
		}
	}

}

func getBestHierarchy(nodeDefs []NodeDef, nodeFo FontOptions, edgeFo FontOptions, optimize bool) ([]VizNode, int64, float64, float64, error) {
	priParentMap := buildPriParentMap(nodeDefs)
	layerMap := buildLayerMap(nodeDefs)
	rootNodes := buildRootNodeList(priParentMap)
	mx, err := NewLayerMx(nodeDefs, layerMap, rootNodes)
	if err != nil {
		return nil, int64(0), 0.0, 0.0, err
	}

	vnh := NewVizNodeHierarchy(nodeDefs, nodeFo, edgeFo)

	vnh.buildNewRootSubtreeHierarchy(mx)

	var bestMx LayerMx
	mxPermCnt := 0
	bestDistSec := math.MaxFloat64
	var tElapsed float64
	if optimize {
		bestSignature := "z"
		tStart := time.Now()
		mxi, err := NewLayerMxPermIterator(nodeDefs, mx)
		if err != nil {
			return nil, int64(0), 0.0, 0.0, err
		}
		mxi.MxIterator(func(_ int, mxPerm LayerMx) {

			// Hierarchy
			vnh.reuseRootSubtreeHierarchy(mxPerm)

			// X coord
			vnh.PopulateNodeTotalWidth()
			vnh.PopulateNodesXCoords()

			distSec := vnh.CalculateTotalHorizontalShift()
			if distSec < bestDistSec {
				// This: 1. Adds determinism 2. helps user choose ids that go first (to some extent)
				signature := mxPerm.signature()
				if distSec < bestDistSec-0.1 || signature < bestSignature {
					bestDistSec = distSec
					bestMx = mxPerm.clone()
					bestSignature = signature
				}
			}
			mxPermCnt++
		})
		tElapsed = time.Since(tStart).Seconds()

		if bestMx == nil {
			return nil, int64(mxPermCnt), tElapsed, 0.0, errors.New("no best")
		}
	} else {
		bestMx = mx
		bestDistSec = 0.0
	}

	// Hierarchy
	vnh.reuseRootSubtreeHierarchy(bestMx)

	// X coord
	vnh.PopulateNodeTotalWidth()
	vnh.PopulateNodesXCoords()

	// Y coord
	vnh.PopulateEdgeLabelDimensions()
	vnh.PopulateUpperLayerGapMap(edgeFo.SizeInPixels)
	vnh.PopulateNodesYCoords()

	// Edge label X and Y
	vnh.PopulateEdgeLabelCoords()
	vnh.RemoveDuplicateSecEdgeLabels()

	return vnh.VizNodeMap, int64(mxPermCnt), tElapsed, bestDistSec, nil
}
