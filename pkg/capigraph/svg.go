package capigraph

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
)

func DefaultNodeFontOptions() FontOptions {
	return FontOptions{FontTypefaceCourier, FontWeightNormal, 20}
}

func DefaultEdgeLabelFontOptions() FontOptions {
	return FontOptions{FontTypefaceArial, FontWeightNormal, 18}
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

func getColorOverride(rootId int16, rootColorMap []int32) string {
	if rootColorMap[rootId] == -1 {
		return ""
	}
	return intToCssColor(rootColorMap[rootId])
}

func getStyleColorOverride(attrName string, rootId int16, rootColorMap []int32) string {
	if rootColorMap[rootId] == -1 {
		return ""
	}
	return fmt.Sprintf(`style="%s:#%s;"`, attrName, intToCssColor(rootColorMap[rootId]))
}

func drawEdgeLines(vizNodeMap []VizNode, curItem *VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions, rootStrokeColorMap []int32) string {
	sb := strings.Builder{}
	if curItem.Def != nil {
		for _, edge := range curItem.Def.SecIn {
			parentItem := &(vizNodeMap[edge.SrcId])
			secDeltaSrc := parentItem.NodeW * (0.5 - SecEdgeStartXRatio) // Exit parent on the left (45%)
			startX := parentItem.X + parentItem.TotalW/2 - secDeltaSrc
			// Outgoing sec edges go from somewhere in the first half, so they are not confused with pri
			startY := parentItem.Y + parentItem.NodeH + eo.StrokeWidth*2.0
			// Outgoing sec edges go to somewhere close to center, but not exactly, so they look secondary

			secDeltaDst := curItem.NodeW * (0.5 - SecEdgeEndXRatio) // Enter child on the right (60%)
			if startX < curItem.X+curItem.TotalW/2+0.1 {            // This stands for startX <= curItem.X+curItem.TotalW/2, but for float
				secDeltaDst = -secDeltaDst // Enter child on the left (40%)
			}
			endX := curItem.X + curItem.TotalW/2 - secDeltaDst
			// endX += arrowEndDeltaX(startX, endX, eo.StrokeWidth)
			endY := curItem.Y - eo.StrokeWidth*3
			deltaX := endX - startX
			deltaY := endY - startY
			sb.WriteString(fmt.Sprintf(`<path class="path-edge-sec path-edge-sec-%d" d="m%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f"/>`+"\n",
				parentItem.RootId,
				//getStyleColorOverride("stroke", parentItem.RootId, rootStrokeColorMap),
				startX, startY, startX+deltaX*0.2, startY+deltaY*0.5, startX+deltaX*0.8, startY+deltaY*0.5, endX, endY))
		}
	}

	for _, childItem := range curItem.PriChildrenAndEnclosedRoots {
		if curItem.Def != nil {
			if childItem.Def.PriIn.SrcId == curItem.Def.Id {
				startX := curItem.X + curItem.TotalW/2
				endX := childItem.X + childItem.TotalW/2
				sb.WriteString(fmt.Sprintf(`<path class="path-edge-pri path-edge-pri-%d" d="m%.2f,%.2f L%.2f,%.2f"/>`+"\n",
					curItem.RootId,
					//getStyleColorOverride("stroke", curItem.RootId, rootStrokeColorMap),
					startX,
					curItem.Y+curItem.NodeH+eo.StrokeWidth*2.0,
					endX, //+arrowEndDeltaX(startX, endX, eo.StrokeWidth),
					childItem.Y-eo.StrokeWidth*3))
			}
		}
		sb.WriteString(drawEdgeLines(vizNodeMap, childItem, nodeFo, edgeFo, eo, rootStrokeColorMap))
	}
	return sb.String()
}

func drawNodesAndEdgeLabels(vizNodeMap []VizNode, curItem *VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions, rootBackgroundColorMap []int32, rootStrokeColorMap []int32, rootTextColorMap []int32) string {
	sb := strings.Builder{}
	if curItem.Def != nil {
		sb.WriteString(fmt.Sprintf(`<a xlink:title="%d %s">`+"\n", curItem.Def.Id, curItem.Def.IconId))
		sb.WriteString(fmt.Sprintf(`<rect class="rect-node-background" %s x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
			getStyleColorOverride("fill", curItem.RootId, rootBackgroundColorMap),
			curItem.X+curItem.TotalW/2-curItem.NodeW/2, curItem.Y, curItem.NodeW, curItem.NodeH))
		sb.WriteString(fmt.Sprintf(`<rect class="rect-node" %s x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
			getStyleColorOverride("stroke", curItem.RootId, rootStrokeColorMap),
			curItem.X+curItem.TotalW/2-curItem.NodeW/2, curItem.Y, curItem.NodeW, curItem.NodeH))
		actualIconSize := 0.0
		if curItem.Def.IconId != "" {
			actualIconSize = curItem.NodeH - nodeFo.SizeInPixels
			sb.WriteString(fmt.Sprintf(`<g transform="translate(%.2f,%.2f)"><g transform="scale(%2f)"><use xlink:href="#%s" %s/></g></g>`+"\n",
				curItem.X+curItem.TotalW/2-curItem.NodeW/2+nodeFo.SizeInPixels/2,
				curItem.Y+nodeFo.SizeInPixels/2,
				actualIconSize/100.0,
				curItem.Def.IconId,
				getStyleColorOverride("fill", curItem.RootId, rootTextColorMap),
			))
		}
		for i, r := range strings.Split(curItem.Def.Text, "\n") {
			textX := curItem.X + curItem.TotalW/2 - curItem.NodeW/2 + nodeFo.SizeInPixels/2
			if actualIconSize > 0.0 {
				textX += actualIconSize + nodeFo.SizeInPixels
			}
			sb.WriteString(fmt.Sprintf(`<text class="text-node" %s x="%.2f" y="%.2f"><![CDATA[%s]]></text>`+"\n",
				getStyleColorOverride("fill", curItem.RootId, rootTextColorMap),
				textX, curItem.Y+nodeFo.SizeInPixels/2+float64(i)*nodeFo.SizeInPixels, r))
		}
		sb.WriteString("</a>\n")

		eolReplacer := strings.NewReplacer("\r", "", "\n", " ")
		// Incoming edge labels
		for _, edgeItem := range curItem.IncomingVizEdges {
			// Do not draw zero-dimension label, it's a duplicate
			if edgeItem.HierarchyType == HierarchySec && edgeItem.W == 0.0 {
				continue
			}
			sb.WriteString(fmt.Sprintf(`<a xlink:title="%s">`+"\n", eolReplacer.Replace(edgeItem.Edge.Text)))
			parentItem := &(vizNodeMap[edgeItem.Edge.SrcId])
			if edgeItem.HierarchyType == HierarchySec {
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-background" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-sec" %s x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					getStyleColorOverride("stroke", parentItem.RootId, rootStrokeColorMap),
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				for i, r := range strings.Split(edgeItem.Edge.Text, "\n") {
					sb.WriteString(fmt.Sprintf(`<text class="text-edge-label" %s x="%.2f" y="%.2f"><![CDATA[%s]]></text>`+"\n",
						getStyleColorOverride("fill", parentItem.RootId, rootBackgroundColorMap),
						edgeItem.X+edgeFo.SizeInPixels/2, edgeItem.Y+edgeFo.SizeInPixels/2+float64(i)*edgeFo.SizeInPixels, r))
				}
			} else if edgeItem.HierarchyType == HierarchyPri {
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-background" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-pri" %s x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					getStyleColorOverride("stroke", parentItem.RootId, rootStrokeColorMap),
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				for i, r := range strings.Split(edgeItem.Edge.Text, "\n") {
					sb.WriteString(fmt.Sprintf(`<text class="text-edge-label" %s x="%.2f" y="%.2f"><![CDATA[%s]]></text>`+"\n",
						getStyleColorOverride("fill", parentItem.RootId, rootBackgroundColorMap),
						edgeItem.X+edgeFo.SizeInPixels/2, edgeItem.Y+edgeFo.SizeInPixels/2+float64(i)*edgeFo.SizeInPixels, r))
				}
			}
			sb.WriteString("</a>\n")
		}
	}

	for _, childItem := range curItem.PriChildrenAndEnclosedRoots {
		sb.WriteString(drawNodesAndEdgeLabels(vizNodeMap, childItem, nodeFo, edgeFo, eo, rootBackgroundColorMap, rootStrokeColorMap, rootTextColorMap))
	}
	return sb.String()
}

type EdgeOptions struct {
	StrokeWidth float64
}

func DefaultPalette() []int32 {
	//return []int32{0x4E79A7, 0xF28E2C, 0xE15759, 0x76B7B2, 0x59A14F, 0xEDC949, 0xAF7AA1, 0xFF9DA7, 0x9C755F, 0xBAB0A} // New T10
	return []int32{0x4C72B0, 0xDD8452, 0x55A868, 0xC44E52, 0x8172B3, 0x937860, 0xDA8BC3, 0x8C8C8C, 0xCCB974, 0x64B5CD} // deep
	//return []int32{0xA1C9F4, 0xFFB482, 0x8DE5A1, 0xFF9F9B, 0xD0BBFF, 0xDEBB9B, 0xFAB0E4, 0xCFCFCF, 0xFFFEA3, 0xB9F2F0} // pastel, too bleached
	//return []int32{0x023EFF, 0xFF7C00, 0x1AC938, 0xE8000B, 0x8B2BE2, 0x9F4800, 0xF14CC1, 0xA3A3A3, 0xFFC400, 0x00D7FF} // bright, very visible
}

func getTextColorForBackground(bgColor int32) int32 {
	r := (bgColor >> 16) & 0xFF
	g := (bgColor >> 8) & 0xFF
	b := bgColor & 0xFF
	uicolors := []float64{float64(r / 255.0), float64(g / 255.0), float64(b / 255.0)}
	c := make([]float64, len(uicolors))
	for i, col := range uicolors {
		if col <= 0.03928 {
			c[i] = col / 12.92
		} else {
			c[i] = math.Pow((col+0.055)/1.055, 2.4)
		}
	}
	l := (0.2126 * c[0]) + (0.7152 * c[1]) + (0.0722 * c[2])
	if l > 0.179 {
		return 0
	} else {
		return 0x00FFFFFF
	}
}

func buildRootColorMaps(vizNodeMap []VizNode, palette []int32) ([]int32, []int32, []int32) {
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

	rootBackgroundColorMap := slices.Repeat([]int32{-1}, len(vizNodeMap))
	rootStrokeColorMap := slices.Repeat([]int32{-1}, len(vizNodeMap))
	rootTextColorMap := slices.Repeat([]int32{-1}, len(vizNodeMap))
	if len(palette) > 0 {
		colorCounter := 0
		for _, rootId := range rootIds {
			paletteColor := palette[colorCounter%len(palette)]
			rootBackgroundColorMap[rootId] = paletteColor
			rootStrokeColorMap[rootId] = paletteColor
			rootTextColorMap[rootId] = getTextColorForBackground(paletteColor)
			colorCounter++
		}
	}
	return rootBackgroundColorMap, rootStrokeColorMap, rootTextColorMap
}

func draw(vizNodeMap []VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions, defsXml string, css string) string {
	topItem := &vizNodeMap[0]
	minLeft := 0.0
	maxRight := topItem.TotalW
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
	}

	bottomCoord := 0.0
	for i := range len(vizNodeMap) - 1 {
		hi := &vizNodeMap[i+1]
		if bottomCoord < hi.Y+hi.NodeH {
			bottomCoord = hi.Y + hi.NodeH
		}
	}
	vbLeft := int(minLeft - 5.0)
	vbRight := int(maxRight + 10.0)
	vbTop := -5
	vbBottom := int(bottomCoord + 10.0)

	palette := DefaultPalette()
	rootBackgroundColorMap, rootStrokeColorMap, rootTextColorMap := buildRootColorMaps(vizNodeMap, palette)

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="%d %d %d %d">`+"\n", vbLeft, vbTop, vbRight, vbBottom))
	sb.WriteString("<defs>\n")

	// For each root, create a separate marker to honor edge color
	for rootId := range len(rootStrokeColorMap) {
		if rootStrokeColorMap[rootId] != -1 {
			sb.WriteString(fmt.Sprintf(`<marker id="arrow-%d" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="4" markerHeight="4" orient="auto-start-reverse" fill="#%s"><path d="M 0 0 L 10 5 L 0 10 z" /></marker>`+"\n",
				rootId,
				getColorOverride(int16(rootId), rootStrokeColorMap)))
		}
	}

	// Caller-provided defs (icons etc)
	sb.WriteString(defsXml)
	sb.WriteString("</defs>\n")
	sb.WriteString("<style>\n")
	sb.WriteString(".viz-background {fill:white;opacity:1.0}\n")
	sb.WriteString(".rect-node-background {fill:white; rx:5; ry:5; stroke-width:0;opacity:0.85}\n")
	sb.WriteString(".rect-node {fill:none; rx:5; ry:5; stroke:black; stroke-width:1;}\n")
	sb.WriteString(".rect-edge-label-background {fill:white; rx:10; ry:10; stroke-width:0; opacity:0.85;}\n")
	sb.WriteString(".rect-edge-label-pri {fill:none; rx:10; ry:10; stroke:#606060; stroke-width:1;}\n")
	sb.WriteString(".rect-edge-label-sec {fill:none; rx:10; ry:10; stroke:#606060; stroke-width:1; stroke-dasharray:5;}\n")
	sb.WriteString(fmt.Sprintf(".text-node {font-family:%s; font-weight:%s; font-size:%dpx; text-anchor:start; alignment-baseline:hanging; fill:black;}\n", FontTypefaceToString(nodeFo.Typeface), FontWeightToString(nodeFo.Weight), int(nodeFo.SizeInPixels)))
	sb.WriteString(fmt.Sprintf(".text-edge-label {font-family:%s; font-weight:%s; font-size:%dpx; text-anchor:start; alignment-baseline:hanging; fill:#606060;}\n", FontTypefaceToString(edgeFo.Typeface), FontWeightToString(edgeFo.Weight), int(edgeFo.SizeInPixels)))
	sb.WriteString(fmt.Sprintf(`.path-edge-pri {stroke-width:%.2f;}`+"\n", eo.StrokeWidth))
	sb.WriteString(fmt.Sprintf(`.path-edge-sec {stroke-width:%.2f;stroke-dasharray:5;fill:transparent;}`+"\n", eo.StrokeWidth))

	// For each root, create a separate edge class to honor edge marker color
	for i := range len(rootStrokeColorMap) - 1 {
		rootId := int16(i + 1)
		if rootStrokeColorMap[rootId] != -1 {
			sb.WriteString(fmt.Sprintf(`.path-edge-pri-%d {marker-end:url(#arrow-%d);stroke:#%s}`+"\n", rootId, rootId, getColorOverride(rootId, rootStrokeColorMap)))
			sb.WriteString(fmt.Sprintf(`.path-edge-sec-%d {marker-end:url(#arrow-%d);stroke:#%s}`+"\n", rootId, rootId, getColorOverride(rootId, rootStrokeColorMap)))
		}
	}
	// Caller-provided CSS overrides
	sb.WriteString(css)
	sb.WriteString("</style>\n")
	sb.WriteString(fmt.Sprintf(`<rect class="viz-background" x="%d" y="%d" width="%d" height="%d"/>`+"\n", vbLeft, vbTop, vbRight-vbLeft, vbBottom-vbTop))

	// Edge lines first, nodes and labels can overlap with them
	sb.WriteString(drawEdgeLines(vizNodeMap, topItem, nodeFo, edgeFo, eo, rootStrokeColorMap))
	// Nodes and labels
	sb.WriteString(drawNodesAndEdgeLabels(vizNodeMap, topItem, nodeFo, edgeFo, eo, rootBackgroundColorMap, rootStrokeColorMap, rootTextColorMap))

	sb.WriteString("</svg>\n")
	return sb.String()
}

func drawStatistics(totalPermutations int64, elapsedSeconds float64, bestDist float64) string {
	return fmt.Sprintf(`<text style="font-family:arial; font-weight:normal; font-size:10px; text-anchor:start; alignment-baseline:hanging; fill:black;" x="0" y="0">Perms %d, elapsed %.3fs, dist %.1f</text>`, totalPermutations, elapsedSeconds, bestDist)
}
