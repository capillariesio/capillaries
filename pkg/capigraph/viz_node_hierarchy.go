package capigraph

import (
	"math"
	"slices"
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
	NodeFo             *FontOptions
	EdgeFo             *FontOptions
	NodeDimensionMap   []RectDimension
	PriEdgeLabelDimMap []RectDimension
	SecEdgeLabelDimMap [][]RectDimension
	TotalLayers        int
	UpperLayerGapMap   []float64
}

func NewVizNodeHierarchy(nodeDefs []NodeDef, nodeFo *FontOptions, edgeFo *FontOptions) *VizNodeHierarchy {
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
	secLabelsFromItemMap := map[int16]map[string]any{} // Fight srcid->secText duplicates
	for _, nodeDef := range nodeDefs {
		if nodeDef.PriIn.SrcId != 0 {
			w, h := getTextDimensions(nodeDef.PriIn.Text, edgeFo.Typeface, edgeFo.Weight, edgeFo.SizeInPixels)
			labelWidth, labelHeight := getLabelDimensionsFromTextDimensions(w, h, edgeFo.SizeInPixels, edgeFo.SizeInPixels)
			vnh.PriEdgeLabelDimMap[nodeDef.Id] = RectDimension{labelWidth, labelHeight}
		}
		vnh.SecEdgeLabelDimMap[nodeDef.Id] = make([]RectDimension, len(nodeDef.SecIn))
		for edgeIdx, edgeDef := range nodeDef.SecIn {
			edgeTextMap, ok := secLabelsFromItemMap[edgeDef.SrcId]
			if !ok {
				edgeTextMap = map[string]any{}
				secLabelsFromItemMap[edgeDef.SrcId] = edgeTextMap
			}
			w := 0.0
			h := 0.0
			_, ok = edgeTextMap[edgeDef.Text]
			if !ok {
				w, h = getTextDimensions(edgeDef.Text, edgeFo.Typeface, edgeFo.Weight, edgeFo.SizeInPixels)
				edgeTextMap[edgeDef.Text] = struct{}{}
			}
			vnh.SecEdgeLabelDimMap[nodeDef.Id][edgeIdx].W, vnh.SecEdgeLabelDimMap[nodeDef.Id][edgeIdx].H = getLabelDimensionsFromTextDimensions(w, h, edgeFo.SizeInPixels, edgeFo.SizeInPixels)
		}
	}
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

	vnh.TotalLayers = 0
	// Initialize static (non-hierarchy-related) properties
	for layer, row := range mx {
		if layer+1 > vnh.TotalLayers {
			vnh.TotalLayers = layer + 1
		}
		for _, nodeId := range row {
			if nodeId > FakeNodeBase {
				continue
			}
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
			if nodeId > FakeNodeBase {
				continue
			}
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
			if nodeId > FakeNodeBase {
				continue
			}
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
			if nodeId > FakeNodeBase {
				continue
			}
			vn := &(vnh.VizNodeMap[nodeId])
			vn.clean()
			vn.Layer = layer
		}
	}

	for _, row := range mx {
		for _, nodeId := range row {
			if nodeId > FakeNodeBase {
				continue
			}
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
			if nodeId > FakeNodeBase {
				continue
			}
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

func getNodeDimensions(nodeDef *NodeDef, fo *FontOptions) RectDimension {
	w, h := getTextDimensions(nodeDef.Text, fo.Typeface, fo.Weight, fo.SizeInPixels)
	w += float64(fo.SizeInPixels)
	if nodeDef.IconId != "" {
		// Add space for the HxH icon plus font-size
		w += h + fo.SizeInPixels
	}
	h += float64(fo.SizeInPixels)
	return RectDimension{w, h}
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
		r := vnh.NodeDimensionMap[vizNode.Def.Id]
		vizNode.NodeW = r.W
		vizNode.NodeH = r.H
		if vizNode.TotalW < vizNode.NodeW {
			vizNode.TotalW = vizNode.NodeW
		}
	}
}

func (vnh *VizNodeHierarchy) PopulateNodeDimensions() {
	vnh.populateNodeDimensionsRecursive(&vnh.VizNodeMap[0])
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

func (vnh *VizNodeHierarchy) PopulateUpperLayerGapMap(edgeFontSizeInPixels float64, minLayerGap float64) {
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
				panic("aaa")
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
			// Outgoing sec edges go from somewhere in the first half, so they are not confused with pri
			startX := srcHiearchyItem.X + srcHiearchyItem.TotalW/2 - srcHiearchyItem.NodeW*(0.5-SecEdgeStartXRatio)
			startY := srcHiearchyItem.Y + srcHiearchyItem.NodeH
			// Outgoing sec edges go to somewhere in the second half, so they are look secondary
			endX := dstVizNode.X + dstVizNode.TotalW/2 - srcHiearchyItem.NodeW*(0.5-SecEdgeEndXRatio)
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
