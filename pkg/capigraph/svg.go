package capigraph

import (
	"fmt"
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

func drawEdgeLines(vizNodeMap []VizNode, curItem *VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions) string {
	sb := strings.Builder{}
	if curItem.Def != nil {
		for _, edge := range curItem.Def.SecIn {
			parentItem := &(vizNodeMap[edge.SrcId])
			// Outgoing sec edges go from somewhere in the first half, so they are not confused with pri
			startX := parentItem.X + parentItem.TotalW/2 - parentItem.NodeW*(0.5-SecEdgeStartXRatio)
			startY := parentItem.Y + parentItem.NodeH + eo.StrokeWidth*2.0
			// Outgoing sec edges go to somewhere close to center, but not exactly, so they look secondary
			secDelta := curItem.NodeW * (0.5 - SecEdgeEndXRatio) // Enter child on the right (60%)
			if startX < curItem.X+curItem.TotalW/2+0.1 {         // This stands for startX <= curItem.X+curItem.TotalW/2, but for float
				secDelta = -secDelta // Enter child on the left (40%)
			}
			endX := curItem.X + curItem.TotalW/2 - secDelta
			// endX += arrowEndDeltaX(startX, endX, eo.StrokeWidth)
			endY := curItem.Y - eo.StrokeWidth*3
			deltaX := endX - startX
			deltaY := endY - startY
			sb.WriteString(fmt.Sprintf(`<path class="path-edge-sec" d="m%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f"/>`+"\n",
				startX, startY, startX+deltaX*0.2, startY+deltaY*0.5, startX+deltaX*0.8, startY+deltaY*0.5, endX, endY))
		}
	}

	for _, childItem := range curItem.PriChildrenAndEnclosedRoots {
		if curItem.Def != nil {
			if childItem.Def.PriIn.SrcId == curItem.Def.Id {
				startX := curItem.X + curItem.TotalW/2
				endX := childItem.X + childItem.TotalW/2
				sb.WriteString(fmt.Sprintf(`<path class="path-edge-pri" d="m%.2f,%.2f L%.2f,%.2f"/>`+"\n",
					startX,
					curItem.Y+curItem.NodeH+eo.StrokeWidth*2.0,
					endX, //+arrowEndDeltaX(startX, endX, eo.StrokeWidth),
					childItem.Y-eo.StrokeWidth*3))
			}
		}
		sb.WriteString(drawEdgeLines(vizNodeMap, childItem, nodeFo, edgeFo, eo))
	}
	return sb.String()
}

func drawNodesAndEdgeLabels(curItem *VizNode, nodeFo FontOptions, edgeFo FontOptions, eo EdgeOptions) string {
	sb := strings.Builder{}
	if curItem.Def != nil {
		sb.WriteString(fmt.Sprintf(`<a xlink:title="%d">`+"\n", curItem.Def.Id))
		sb.WriteString(fmt.Sprintf(`<rect class="rect-node-background" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n", curItem.X+curItem.TotalW/2-curItem.NodeW/2, curItem.Y, curItem.NodeW, curItem.NodeH))
		sb.WriteString(fmt.Sprintf(`<rect class="rect-node" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n", curItem.X+curItem.TotalW/2-curItem.NodeW/2, curItem.Y, curItem.NodeW, curItem.NodeH))
		actualIconSize := 0.0
		if curItem.Def.IconId != "" {
			actualIconSize = curItem.NodeH - nodeFo.SizeInPixels
			sb.WriteString(fmt.Sprintf(`<g transform="translate(%.2f,%.2f)"><g transform="scale(%2f)"><use xlink:href="#%s"/></g></g>`+"\n",
				curItem.X+curItem.TotalW/2-curItem.NodeW/2+nodeFo.SizeInPixels/2,
				curItem.Y+nodeFo.SizeInPixels/2,
				actualIconSize/100.0,
				curItem.Def.IconId))
		}
		for i, r := range strings.Split(curItem.Def.Text, "\n") {
			textX := curItem.X + curItem.TotalW/2 - curItem.NodeW/2 + nodeFo.SizeInPixels/2
			if actualIconSize > 0.0 {
				textX += actualIconSize + nodeFo.SizeInPixels
			}
			sb.WriteString(fmt.Sprintf(`<text class="text-node" x="%.2f" y="%.2f"><![CDATA[%s]]></text>`+"\n",
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
			if edgeItem.HierarchyType == HierarchySec {
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-background" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-sec" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				for i, r := range strings.Split(edgeItem.Edge.Text, "\n") {
					sb.WriteString(fmt.Sprintf(`<text class="text-edge-label" x="%.2f" y="%.2f"><![CDATA[%s]]></text>`+"\n",
						edgeItem.X+edgeFo.SizeInPixels/2, edgeItem.Y+edgeFo.SizeInPixels/2+float64(i)*edgeFo.SizeInPixels, r))
				}
			} else if edgeItem.HierarchyType == HierarchyPri {
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-background" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				sb.WriteString(fmt.Sprintf(`<rect class="rect-edge-label-pri" x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`+"\n",
					edgeItem.X, edgeItem.Y, edgeItem.W, edgeItem.H))
				for i, r := range strings.Split(edgeItem.Edge.Text, "\n") {
					sb.WriteString(fmt.Sprintf(`<text class="text-edge-label" x="%.2f" y="%.2f"><![CDATA[%s]]></text>`+"\n",
						edgeItem.X+edgeFo.SizeInPixels/2, edgeItem.Y+edgeFo.SizeInPixels/2+float64(i)*edgeFo.SizeInPixels, r))
				}
			}
			sb.WriteString("</a>\n")
		}
	}

	for _, childItem := range curItem.PriChildrenAndEnclosedRoots {
		sb.WriteString(drawNodesAndEdgeLabels(childItem, nodeFo, edgeFo, eo))
	}
	return sb.String()
}

type EdgeOptions struct {
	StrokeWidth float64
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
	vbRight := int(maxRight + 5.0)
	vbTop := -5
	vbBottom := int(bottomCoord + 10.0)

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="%d %d %d %d">`+"\n", vbLeft, vbTop, vbRight, vbBottom))
	sb.WriteString("<defs>\n")
	sb.WriteString(`<marker id="arrow" viewBox="0 0 10 10" refX="5" refY="5" markerWidth="4" markerHeight="4" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" /></marker>` + "\n")
	// Caller-provided defs (icons etc)
	sb.WriteString(defsXml)
	sb.WriteString("</defs>\n")
	sb.WriteString("<style>\n")
	sb.WriteString(".rect-node-background {fill:white; rx:5; ry:5; stroke-width:0;opacity:0.8}\n")
	sb.WriteString(".rect-node {fill:none; rx:5; ry:5; stroke:black; stroke-width:1;}\n")
	sb.WriteString(".rect-edge-label-background {fill:white; rx:10; ry:10; stroke-width:0; opacity:0.8;}\n")
	sb.WriteString(".rect-edge-label-pri {fill:none; rx:10; ry:10; stroke:#606060; stroke-width:1;}\n")
	sb.WriteString(".rect-edge-label-sec {fill:none; rx:10; ry:10; stroke:#606060; stroke-width:1; stroke-dasharray:5;}\n")
	sb.WriteString(fmt.Sprintf(".text-node {font-family:%s; font-weight:%s; font-size:%dpx; text-anchor:start; alignment-baseline:hanging; fill:black;}\n", FontTypefaceToString(nodeFo.Typeface), FontWeightToString(nodeFo.Weight), int(nodeFo.SizeInPixels)))
	sb.WriteString(fmt.Sprintf(".text-edge-label {font-family:%s; font-weight:%s; font-size:%dpx; text-anchor:start; alignment-baseline:hanging; fill:#606060;}\n", FontTypefaceToString(edgeFo.Typeface), FontWeightToString(edgeFo.Weight), int(edgeFo.SizeInPixels)))
	sb.WriteString(fmt.Sprintf(`.path-edge-pri {marker-end:url(#arrow); stroke:black; stroke-width:%.2f;}`+"\n", eo.StrokeWidth))
	sb.WriteString(fmt.Sprintf(`.path-edge-sec {marker-end:url(#arrow); stroke:black; stroke-width:%.2f;stroke-dasharray:5;fill:none;}`+"\n", eo.StrokeWidth))
	// Caller-provided CSS overrides
	sb.WriteString(css)
	sb.WriteString("</style>\n")
	sb.WriteString(fmt.Sprintf(`<rect fill="white" x="%d" y="%d" width="%d" height="%d"/>`+"\n", vbLeft, vbTop, vbRight-vbLeft, vbBottom-vbTop))

	// Edge lines first, nodes and labels can overlap with them
	sb.WriteString(drawEdgeLines(vizNodeMap, topItem, nodeFo, edgeFo, eo))
	// Nodes and labels
	sb.WriteString(drawNodesAndEdgeLabels(topItem, nodeFo, edgeFo, eo))

	sb.WriteString("</svg>\n")
	return sb.String()
}

func drawStatistics(totalPermutations int64, elapsedSeconds float64, bestDist float64) string {
	return fmt.Sprintf(`<text style="font-family:arial; font-weight:normal; font-size:10px; text-anchor:start; alignment-baseline:hanging; fill:black;" x="0" y="0">Perms %d, elapsed %.3fs, dist %.1f</text>`, totalPermutations, elapsedSeconds, bestDist)
}
