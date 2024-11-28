package capigraph

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicMx(t *testing.T) {
	mx := LayerMx{{1, 3}, {2}}

	vnh := NewVizNodeHierarchy(testNodeDefsBasic, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())

	vnh.buildNewRootSubtreeHierarchy(mx)
	vnh.reuseRootSubtreeHierarchy(mx)
	vnh.PopulateNodeTotalWidth()

	assert.Equal(t, 84.0, vnh.VizNodeMap[0].TotalW)
	assert.Equal(t, 0.0, vnh.VizNodeMap[0].NodeW)
	assert.Equal(t, 0.0, vnh.VizNodeMap[0].NodeH)

	assert.Equal(t, 32.0, vnh.VizNodeMap[1].TotalW)
	assert.Equal(t, 32.0, vnh.VizNodeMap[1].NodeW)
	assert.Equal(t, 40.0, vnh.VizNodeMap[1].NodeH)

	assert.Equal(t, 32.0, vnh.VizNodeMap[2].TotalW)
	assert.Equal(t, 32.0, vnh.VizNodeMap[2].NodeW)
	assert.Equal(t, 40.0, vnh.VizNodeMap[2].NodeH)

	assert.Equal(t, 32.0, vnh.VizNodeMap[3].TotalW)
	assert.Equal(t, 32.0, vnh.VizNodeMap[3].NodeW)
	assert.Equal(t, 40.0, vnh.VizNodeMap[3].NodeH)

	vnh.PopulateNodesXCoords()

	assert.Equal(t, 0.0, vnh.VizNodeMap[0].X)
	assert.Equal(t, 0.0, vnh.VizNodeMap[1].X)
	assert.Equal(t, 0.0, vnh.VizNodeMap[2].X)
	assert.Equal(t, 52.0, vnh.VizNodeMap[3].X)

	horShift := vnh.CalculateTotalHorizontalShift()
	assert.Equal(t, 52.0, math.Round(horShift*100)/100.0)

	vnh.PopulateEdgeLabelDimensions()

	assert.Equal(t, int16(1), vnh.VizNodeMap[2].IncomingVizEdges[0].Edge.SrcId)
	assert.Equal(t, HierarchyPri, vnh.VizNodeMap[2].IncomingVizEdges[0].HierarchyType)
	assert.Equal(t, 0.0, vnh.VizNodeMap[2].IncomingVizEdges[0].W) // No label text
	assert.Equal(t, 0.0, vnh.VizNodeMap[2].IncomingVizEdges[0].H) // No label text

	assert.Equal(t, int16(3), vnh.VizNodeMap[2].IncomingVizEdges[1].Edge.SrcId)
	assert.Equal(t, HierarchySec, vnh.VizNodeMap[2].IncomingVizEdges[1].HierarchyType)
	assert.Equal(t, 69.12, vnh.VizNodeMap[2].IncomingVizEdges[1].W)
	assert.Equal(t, 36.0, vnh.VizNodeMap[2].IncomingVizEdges[1].H)

	vnh.PopulateUpperLayerGapMap(DefaultEdgeLabelFontOptions().SizeInPixels)

	assert.Equal(t, 60.0, vnh.UpperLayerGapMap[0]) // Changed
	assert.Equal(t, 216.0, vnh.UpperLayerGapMap[1])

	vnh.PopulateNodesYCoords()

	assert.Equal(t, 0.0, vnh.VizNodeMap[0].Y)
	assert.Equal(t, 0.0, vnh.VizNodeMap[1].Y)
	assert.Equal(t, 256.0, vnh.VizNodeMap[2].Y)
	assert.Equal(t, 0.0, vnh.VizNodeMap[3].Y)

	vnh.PopulateEdgeLabelCoords()

	assert.Equal(t, 16.0, vnh.VizNodeMap[2].IncomingVizEdges[0].X)
	assert.Equal(t, 220.0, vnh.VizNodeMap[2].IncomingVizEdges[0].Y)

	assert.Equal(t, 7.44, math.Round(vnh.VizNodeMap[2].IncomingVizEdges[1].X*100.0)/100.0)
	assert.Equal(t, 130.0, vnh.VizNodeMap[2].IncomingVizEdges[1].Y)
}
