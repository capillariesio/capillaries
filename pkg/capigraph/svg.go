package capigraph

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
)

func DefaultNodeFontOptions() FontOptions {
	return FontOptions{FontTypefaceCourier, FontWeightNormal, 20, 0.3}
}

func DefaultEdgeLabelFontOptions() FontOptions {
	return FontOptions{FontTypefaceArial, FontWeightNormal, 18, 0.3}
}

func DefaultEdgeOptions() EdgeOptions {
	return EdgeOptions{2.0}
}

func intToCssColor(c int32) string {
	colorRunes := make([]rune, 6)
	colorRunes[0] = halfByteToChar(int8((c >> 20) & 0x0F))
	colorRunes[1] = halfByteToChar(int8((c >> 16) & 0x0F))
	colorRunes[2] = halfByteToChar(int8((c >> 12) & 0x0F))
	colorRunes[3] = halfByteToChar(int8((c >> 8) & 0x0F))
	colorRunes[4] = halfByteToChar(int8((c >> 4) & 0x0F))
	colorRunes[5] = halfByteToChar(int8(c & 0x0F))
	return string(colorRunes)
}

func getColorOverride(defaultColor string, rootId int16, rootColorMap []int32) string {
	if rootColorMap[rootId] == -1 {
		return defaultColor
	}
	return intToCssColor(rootColorMap[rootId])
}

func getStyleColorOverrideForRoot(attrName string, rootId int16, rootColorMap []int32) string {
	if rootColorMap[rootId] == -1 {
		return ""
	}
	return fmt.Sprintf(`style="%s:#%s;"`, attrName, intToCssColor(rootColorMap[rootId]))
}
func getStyleColorOverrideForNode(attrName string, color int32) string {
	if color == int32(0) {
		return ""
	}
	return fmt.Sprintf(`style="%s:#%s;"`, attrName, intToCssColor(color))
}

func getStyleColorOverrideForNodeWithOpacity2(attrName string, color int32) string {
	if color == int32(0) {
		return ""
	}
	return fmt.Sprintf(`style="%s:#%s;opacity:0.3;"`, attrName, intToCssColor(color))
}

func drawNodeSelections(vizNodeMap []VizNode, nodeFo FontOptions) string {
	sb := strings.Builder{}
	for i := range len(vizNodeMap) - 1 {
		curItem := vizNodeMap[i+1]
		nodeX := curItem.X + curItem.TotalW/2 - curItem.NodeW/2
		if curItem.Def.Selected {
			sb.WriteString(fmt.Sprintf(`<rect class="rect-selected-node" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
				nodeX-SelectedNodeMargin*nodeFo.SizeInPixels, curItem.Y-SelectedNodeMargin*nodeFo.SizeInPixels, curItem.NodeW+SelectedNodeMargin*nodeFo.SizeInPixels*2, curItem.NodeH+SelectedNodeMargin*nodeFo.SizeInPixels*2))
		}
	}
	return sb.String()

}
func drawEdgeLines(vizNodeMap []VizNode, curItem *VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions, rootStrokeColorMap []int32) string {
	sb := strings.Builder{}
	if curItem.Def != nil {
		for _, edge := range curItem.Def.SecIn {
			parentItem := &(vizNodeMap[edge.SrcId])

			startX := parentItem.X + parentItem.TotalW/2.0
			endX := curItem.X + curItem.TotalW/2.0
			startOffset, endOffset := getSecOffsetX(startX, endX, nodeFo.SizeInPixels/2.0)
			startX += startOffset
			endX += endOffset

			startY := parentItem.Y + parentItem.NodeH + eo.StrokeWidth*2.0
			endY := curItem.Y - eo.StrokeWidth*3

			deltaX := endX - startX
			deltaY := endY - startY

			// For nearly-vertical sec connector, add a twist - so it has a better chance not to interfere with some pri connector
			curveDeltaForSimilarX := 0.0
			if math.Abs(startY-endY)/math.Abs(startX-endX) > 8 {
				curveDeltaForSimilarX = deltaX * math.Abs(startY-endY) / math.Abs(startX-endX) / 4
			}
			sb.WriteString(fmt.Sprintf(`<path class="path-edge-sec path-edge-sec-%d" d="m%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f"/>`+"\n",
				parentItem.RootId,
				startX, startY, startX+deltaX*0.2, startY+deltaY*0.5, startX+deltaX*0.8+curveDeltaForSimilarX, startY+deltaY*0.5, endX, endY))
		}
	}

	for _, childItem := range curItem.PriChildrenAndEnclosedRoots {
		if curItem.Def != nil {
			if childItem.Def.PriIn.SrcId == curItem.Def.Id {
				startX := curItem.X + curItem.TotalW/2
				endX := childItem.X + childItem.TotalW/2
				sb.WriteString(fmt.Sprintf(`<path class="path-edge-pri path-edge-pri-%d" d="m%.2f,%.2f L%.2f,%.2f"/>`+"\n",
					curItem.RootId,
					startX,
					curItem.Y+curItem.NodeH+eo.StrokeWidth*2.0,
					endX,
					childItem.Y-eo.StrokeWidth*3))
			}
		}
		sb.WriteString(drawEdgeLines(vizNodeMap, childItem, nodeFo, edgeFo, eo, rootStrokeColorMap))
	}
	return sb.String()
}

func drawNodesAndEdgeLabels(vizNodeMap []VizNode, curItem *VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions, rootColorMap []int32) string {
	xmlReplacer := strings.NewReplacer("\"", "&quot;", "'", "&apos;", "<", "&lt;", ">", "&gt;", "&", "&amp;")
	sb := strings.Builder{}
	if curItem.Def != nil {
		title := strings.TrimSpace(xmlReplacer.Replace(fmt.Sprintf("%d %s", curItem.Def.Id, curItem.Def.IconId)))
		sb.WriteString(fmt.Sprintf(`<a xlink:title="%s">`+"\n", title))
		nodeX := curItem.X + curItem.TotalW/2 - curItem.NodeW/2
		sb.WriteString(fmt.Sprintf(`  <rect class="rect-node-background rect-node-background-%d" %s x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
			curItem.RootId,
			getStyleColorOverrideForNodeWithOpacity2("fill", curItem.Def.Color),
			nodeX, curItem.Y, curItem.NodeW, curItem.NodeH))
		sb.WriteString(fmt.Sprintf(`  <rect class="rect-node rect-node-%d" %s x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
			curItem.RootId,
			getStyleColorOverrideForNode("stroke", curItem.Def.Color),
			nodeX, curItem.Y, curItem.NodeW, curItem.NodeH))
		actualIconSize := 0.0
		if curItem.Def.IconId != "" {
			actualIconSize = curItem.NodeH - nodeFo.SizeInPixels*NodeTextDimensionMargin*2
			iconColorCssOverride := getStyleColorOverrideForRoot("fill", curItem.RootId, rootColorMap)
			if curItem.Def.Color != 0 {
				iconColorCssOverride = getStyleColorOverrideForNode("fill", curItem.Def.Color)
			}
			sb.WriteString(fmt.Sprintf(`  <g transform="translate(%.2f,%.2f)"><g transform="scale(%2f)">`+"\n    "+`<use xlink:href="#%s" %s/>`+"\n  </g></g>\n",
				nodeX+nodeFo.SizeInPixels*NodeTextDimensionMargin,
				curItem.Y+nodeFo.SizeInPixels*NodeTextDimensionMargin,
				actualIconSize/100.0,
				curItem.Def.IconId,
				iconColorCssOverride,
			))
		}
		for i, r := range strings.Split(curItem.Def.Text, "\n") {
			textX := curItem.X + curItem.TotalW/2 - curItem.NodeW/2 + nodeFo.SizeInPixels*NodeTextDimensionMargin
			if actualIconSize > 0.0 {
				textX += actualIconSize + nodeFo.SizeInPixels*NodeTextIconInterval
			}
			sb.WriteString(fmt.Sprintf(`  <text class="text-node" x="%.2f" y="%.2f">%s</text>`+"\n",
				textX,
				curItem.Y+nodeFo.SizeInPixels*NodeTextDimensionMargin+float64(i)*nodeFo.SizeInPixels*(1.0+nodeFo.Interval),
				xmlReplacer.Replace(r)))
		}
		sb.WriteString("</a>\n")

		eolReplacer := strings.NewReplacer("\r", "", "\n", " ")
		// Incoming edge labels
		for _, edgeItem := range curItem.IncomingVizEdges {
			// Do not draw zero-dimension label, it's a sec duplicate or just empty (pri or sec)
			if edgeItem.W == 0.0 {
				continue
			}
			sb.WriteString(fmt.Sprintf(`<a xlink:title="%s">`+"\n", strings.TrimSpace(xmlReplacer.Replace(eolReplacer.Replace(edgeItem.Edge.Text)))))
			parentItem := &(vizNodeMap[edgeItem.Edge.SrcId])
			if edgeItem.HierarchyType == HierarchySec {
				sb.WriteString(fmt.Sprintf(`  <rect class="rect-edge-label-background" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				sb.WriteString(fmt.Sprintf(`  <rect class="rect-edge-label-sec rect-edge-label-sec-%d" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					parentItem.RootId,
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				for i, r := range strings.Split(edgeItem.Edge.Text, "\n") {
					sb.WriteString(fmt.Sprintf(`  <text class="text-edge-label" x="%.2f" y="%.2f">%s</text>`+"\n",
						edgeItem.X+edgeFo.SizeInPixels*LabelTextDimensionMargin,
						edgeItem.Y+edgeFo.SizeInPixels*LabelTextDimensionMargin+float64(i)*edgeFo.SizeInPixels*(1+edgeFo.Interval),
						xmlReplacer.Replace(r)))
				}
			} else if edgeItem.HierarchyType == HierarchyPri {
				sb.WriteString(fmt.Sprintf(`  <rect class="rect-edge-label-background" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				sb.WriteString(fmt.Sprintf(`  <rect class="rect-edge-label-pri rect-edge-label-pri-%d" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					parentItem.RootId,
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				for i, r := range strings.Split(edgeItem.Edge.Text, "\n") {
					sb.WriteString(fmt.Sprintf(`  <text class="text-edge-label" x="%.2f" y="%.2f">%s</text>`+"\n",
						edgeItem.X+edgeFo.SizeInPixels*LabelTextDimensionMargin,
						edgeItem.Y+edgeFo.SizeInPixels*LabelTextDimensionMargin+float64(i)*edgeFo.SizeInPixels*(1+edgeFo.Interval),
						xmlReplacer.Replace(r)))
				}
			}
			sb.WriteString("</a>\n")
		}
	}

	for _, childItem := range curItem.PriChildrenAndEnclosedRoots {
		sb.WriteString(drawNodesAndEdgeLabels(vizNodeMap, childItem, nodeFo, edgeFo, eo, rootColorMap))
	}
	return sb.String()
}

type EdgeOptions struct {
	StrokeWidth float64
}

func DefaultPalette() []int32 {
	// return []int32{0x4E79A7, 0xF28E2C, 0xE15759, 0x76B7B2, 0x59A14F, 0xEDC949, 0xAF7AA1, 0xFF9DA7, 0x9C755F, 0xBAB0A} // New T10
	// return []int32{0x4C72B0, 0xDD8452, 0x55A868, 0xC44E52, 0x8172B3, 0x937860, 0xDA8BC3, 0x8C8C8C, 0xCCB974, 0x64B5CD} // deep
	// return []int32{0xA1C9F4, 0xFFB482, 0x8DE5A1, 0xFF9F9B, 0xD0BBFF, 0xDEBB9B, 0xFAB0E4, 0xCFCFCF, 0xFFFEA3, 0xB9F2F0} // pastel, too bleached
	return []int32{0x023EFF, 0xFF7C00, 0x1AC938, 0xE8000B, 0x8B2BE2, 0x9F4800, 0xF14CC1, 0xA3A3A3, 0xFFC400, 0x00D7FF} // bright, very visible
}

// Currently not used
// func getTextColorForBackground(bgColor int32) int32 {
// 	r := (bgColor >> 16) & 0xFF
// 	g := (bgColor >> 8) & 0xFF
// 	b := bgColor & 0xFF
// 	uicolors := []float64{float64(r / 255.0), float64(g / 255.0), float64(b / 255.0)}
// 	c := make([]float64, len(uicolors))
// 	for i, col := range uicolors {
// 		if col <= 0.03928 {
// 			c[i] = col / 12.92
// 		} else {
// 			c[i] = math.Pow((col+0.055)/1.055, 2.4)
// 		}
// 	}
// 	l := (0.2126 * c[0]) + (0.7152 * c[1]) + (0.0722 * c[2])
// 	if l > 0.179 {
// 		return 0
// 	} else {
// 		return 0x00FFFFFF
// 	}
// }

func buildRootColorMap(vizNodeMap []VizNode, palette []int32) []int32 {
	rootIds := make([]int16, 0, len(vizNodeMap))
	rootCountMap := make([]int, len(vizNodeMap))
	for i := range len(vizNodeMap) - 1 {
		vizNode := &vizNodeMap[i+1]
		if rootCountMap[vizNode.RootId] == 0 {
			rootIds = append(rootIds, vizNode.RootId)
		}
		rootCountMap[vizNode.RootId]++
	}
	sort.Slice(rootIds, func(i, j int) bool {
		return rootCountMap[rootIds[i]] > rootCountMap[rootIds[j]]
	})

	rootColorMap := slices.Repeat([]int32{-1}, len(vizNodeMap))
	if len(palette) > 0 {
		colorCounter := 0
		for _, rootId := range rootIds {
			rootColorMap[rootId] = palette[colorCounter%len(palette)]
			colorCounter++
		}
	}
	return rootColorMap
}

const SelectedNodeMargin float64 = 0.0

func draw(vizNodeMap []VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions, defsXml string, css string, palette []int32, totalPermutations int64, elapsed float64, bestDist float64) string {
	topCoord := math.MaxFloat64
	bottomCoord := -math.MaxFloat64
	minLeft := math.MaxFloat64
	maxRight := -math.MaxFloat64
	for i := range len(vizNodeMap) - 1 {
		hi := &vizNodeMap[i+1]
		for _, edge := range hi.IncomingVizEdges {
			if edge.X < minLeft {
				minLeft = edge.X
			}
			if edge.X+edge.W > maxRight {
				maxRight = edge.X + edge.W
			}
		}
		nodeLeft := hi.X
		nodeRight := hi.X + hi.NodeW
		nodeTop := hi.Y
		nodeBottom := hi.Y + hi.NodeH
		if hi.Def.Selected {
			nodeLeft -= SelectedNodeMargin * nodeFo.SizeInPixels
			nodeRight += SelectedNodeMargin * nodeFo.SizeInPixels
			nodeTop -= SelectedNodeMargin * nodeFo.SizeInPixels
			nodeBottom += SelectedNodeMargin * nodeFo.SizeInPixels
		}
		if nodeLeft < minLeft {
			minLeft = nodeLeft
		}
		if nodeRight > maxRight {
			maxRight = nodeRight
		}
		if nodeBottom > bottomCoord {
			bottomCoord = nodeBottom
		}
		if nodeTop < topCoord {
			topCoord = nodeTop
		}
	}

	vbLeft := int(minLeft - 10.0)
	vbRight := int(maxRight + 20.0)
	vbTop := int(topCoord - 10.0)
	vbBottom := int(bottomCoord + 20.0)

	rootColorMap := buildRootColorMap(vizNodeMap, palette)

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="%d %d %d %d">`+"\n", vbLeft, vbTop, vbRight-vbLeft, vbBottom-vbTop))
	sb.WriteString("<defs>\n")

	for rootId := range len(rootColorMap) {
		sb.WriteString(fmt.Sprintf(`<marker id="arrow-%d" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="4" markerHeight="4" orient="auto-start-reverse" fill="#%s"><path d="M 0 0 L 10 5 L 0 10 z" /></marker>`+"\n",
			rootId,
			getColorOverride("000000", int16(rootId), rootColorMap)))
	}

	// Caller-provided defs (icons etc)
	sb.WriteString(defsXml)
	sb.WriteString("</defs>\n")
	sb.WriteString("<style>\n")
	sb.WriteString(".viz-background {fill:white;opacity:1.0}\n")
	sb.WriteString(fmt.Sprintf(".rect-node-background {fill:white; rx:%d; ry:%d; stroke-width:0;opacity:0.7}\n", int(nodeFo.SizeInPixels/2), int(nodeFo.SizeInPixels/2)))
	sb.WriteString(fmt.Sprintf(".rect-node {fill:none; rx:%d; ry:%d; stroke:black; stroke-width:1;}\n", int(nodeFo.SizeInPixels/2), int(nodeFo.SizeInPixels/2)))
	sb.WriteString(fmt.Sprintf(".rect-selected-node {fill:transparent; rx:%d; ry:%d; stroke-width:%d; stroke:black; opacity:1.0}\n", int(nodeFo.SizeInPixels/2), int(nodeFo.SizeInPixels/2), int(nodeFo.SizeInPixels/4)))
	sb.WriteString(".rect-edge-label-background {fill:white; rx:10; ry:10; stroke-width:0; opacity:0.7;}\n")
	sb.WriteString(".rect-edge-label-pri {fill:none; rx:10; ry:10; stroke:#606060; stroke-width:1;}\n")
	sb.WriteString(".rect-edge-label-sec {fill:none; rx:10; ry:10; stroke:#606060; stroke-width:1; stroke-dasharray:5;}\n")
	sb.WriteString(fmt.Sprintf(".text-node {font-family:%s; font-weight:%s; font-size:%dpx; text-anchor:start; alignment-baseline:hanging; fill:black;}\n", FontTypefaceToString(nodeFo.Typeface), FontWeightToString(nodeFo.Weight), int(nodeFo.SizeInPixels)))
	sb.WriteString(fmt.Sprintf(".text-edge-label {font-family:%s; font-weight:%s; font-size:%dpx; text-anchor:start; alignment-baseline:hanging; fill:#606060;}\n", FontTypefaceToString(edgeFo.Typeface), FontWeightToString(edgeFo.Weight), int(edgeFo.SizeInPixels)))
	sb.WriteString(fmt.Sprintf(`.path-edge-pri {stroke-width:%.2f;fill:transparent;stroke:black;}`+"\n", eo.StrokeWidth))
	sb.WriteString(fmt.Sprintf(`.path-edge-sec {stroke-width:%.2f;stroke-dasharray:5;fill:transparent;stroke:black;}`+"\n", eo.StrokeWidth))

	// For each root, create a set of classes with proper color
	for rootId := range len(rootColorMap) {
		// if rootColorMap[rootId] == -1 {
		// 	continue
		// }
		// Edge coor: stroke (label border and connectors). Also, proper arrow marker color.
		sb.WriteString(fmt.Sprintf(`.path-edge-pri-%d {marker-end:url(#arrow-%d);stroke:#%s}`+"\n", rootId, rootId, getColorOverride("000000", int16(rootId), rootColorMap)))
		sb.WriteString(fmt.Sprintf(`.path-edge-sec-%d {marker-end:url(#arrow-%d);stroke:#%s}`+"\n", rootId, rootId, getColorOverride("000000", int16(rootId), rootColorMap)))
		sb.WriteString(fmt.Sprintf(`.rect-edge-label-pri-%d {stroke:#%s}`+"\n", rootId, getColorOverride("000000", int16(rootId), rootColorMap)))
		sb.WriteString(fmt.Sprintf(`.rect-edge-label-sec-%d {stroke:#%s}`+"\n", rootId, getColorOverride("000000", int16(rootId), rootColorMap)))

		// Node color: background, stroke
		sb.WriteString(fmt.Sprintf(`.rect-node-background-%d {fill:#%s;opacity:0.2}`+"\n", rootId, getColorOverride("FFFFFF", int16(rootId), rootColorMap)))
		sb.WriteString(fmt.Sprintf(`.rect-node-%d {stroke:#%s}`+"\n", rootId, getColorOverride("000000", int16(rootId), rootColorMap)))
	}
	// Renderingstats
	sb.WriteString(`.capigraph-rendering-stats {font-family:arial; font-weight:normal; font-size:10px; text-anchor:start; alignment-baseline:hanging; fill:transparent;}`)

	// Caller-provided CSS overrides
	sb.WriteString(css)
	sb.WriteString("</style>\n")
	sb.WriteString(fmt.Sprintf(`<rect class="viz-background" x="%d" y="%d" width="%d" height="%d"/>`+"\n", vbLeft, vbTop, vbRight-vbLeft, vbBottom-vbTop))

	// Node selections at the z-bottom
	sb.WriteString(drawNodeSelections(vizNodeMap, nodeFo))

	// Edge lines first, nodes and labels can overlap with them
	topItem := &vizNodeMap[0]
	sb.WriteString(drawEdgeLines(vizNodeMap, topItem, nodeFo, edgeFo, eo, rootColorMap))
	// Nodes and labels
	sb.WriteString(drawNodesAndEdgeLabels(vizNodeMap, topItem, nodeFo, edgeFo, eo, rootColorMap))

	sb.WriteString(fmt.Sprintf(`<text class="capigraph-rendering-stats" x="0" y="0">Perms %d, elapsed %.3fs, dist %.1f</text>`+"\n", totalPermutations, elapsed, bestDist))

	sb.WriteString("</svg>\n")
	return sb.String()
}

func DrawOptimized(nodeDefs []NodeDef, nodeFo FontOptions, edgeFo FontOptions, edgeOptions EdgeOptions, defsOverride string, cssOverride string, palette []int32) (string, []VizNode, int64, float64, float64, error) {
	if err := checkNodeIds(nodeDefs); err != nil {
		return "", nil, int64(0), 0.0, 0.0, err
	}

	for i := range len(nodeDefs) - 1 {
		if err := checkNodeDef(int16(i+1), nodeDefs); err != nil {
			return "", nil, int64(0), 0.0, 0.0, err
		}
	}
	vizNodeMap, totalPermutations, elapsed, bestDist, err := getBestHierarchy(nodeDefs, nodeFo, edgeFo, true)
	if err != nil {
		return "", nil, int64(0), 0.0, 0.0, err
	}
	svgString := draw(vizNodeMap, nodeFo, edgeFo, edgeOptions, defsOverride, cssOverride, palette, totalPermutations, elapsed, bestDist)
	return svgString, vizNodeMap, totalPermutations, elapsed, bestDist, nil
}

func DrawUnoptimized(nodeDefs []NodeDef, nodeFo FontOptions, edgeFo FontOptions, edgeOptions EdgeOptions, defsOverride string, cssOverride string, palette []int32) (string, []VizNode, error) {
	if err := checkNodeIds(nodeDefs); err != nil {
		return "", nil, err
	}

	for i := range len(nodeDefs) - 1 {
		if err := checkNodeDef(int16(i+1), nodeDefs); err != nil {
			return "", nil, err
		}
	}

	vizNodeMap, totalPermutations, elapsed, bestDist, err := getBestHierarchy(nodeDefs, nodeFo, edgeFo, false)
	if err != nil {
		return "", nil, err
	}
	svgString := draw(vizNodeMap, nodeFo, edgeFo, edgeOptions, defsOverride, cssOverride, palette, totalPermutations, elapsed, bestDist)
	return svgString, vizNodeMap, nil
}
