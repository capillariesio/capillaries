package capigraph

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 0:    1  3
// -     |/
// 1:    2
func TestBasic(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{{3, "from 3"}}, ""},
		{3, "3", EdgeDef{}, []EdgeDef{}, ""},
	}
	mx := [][]int16{{1, 3}, {2}}

	nodeFo := FontOptions{FontTypefaceVerdana, FontWeightNormal, 20}
	vnh := NewVizNodeHierarchy(nodeDefs, &nodeFo)

	vnh.PopulateSubtreeHierarchy(mx)
	vnh.PopulateNodeDimensions()
	vnh.PopulateHierarchyItemsXCoords()
	horShift := vnh.CalculateTotalHorizontalShift()
	assert.Equal(t, 49.52, math.Round(horShift*100)/100.0)

	vnh.NodeFo.Weight = FontWeightBold
	vnh.PopulateSubtreeHierarchy(mx)
	vnh.PopulateNodeDimensions()
	vnh.PopulateHierarchyItemsXCoords()
	horShift = vnh.CalculateTotalHorizontalShift()
	assert.Equal(t, 50.78, math.Round(horShift*100)/100.0)
}
