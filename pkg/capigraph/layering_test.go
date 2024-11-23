package capigraph

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 0:    1  2
// -     | /|
// 1:    3  |
// -     | /
// 2:    4
func TestTwoLevelsFromOneParent(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{1, ""}, []EdgeDef{{2, ""}}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{{2, ""}}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 0 1 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1 2], 1 [3], 2 [4]", mx.String())
}

// 0:    1
// -     |  \
// 1:    2     3
// -     |     |
// 2:    4  8  5
// -     | / \ |
// 3:    6     7
func TestOneEnclosingTwoLevelsDown(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{4, ""}, []EdgeDef{{8, ""}}, ""},
		{7, "7", EdgeDef{5, ""}, []EdgeDef{{8, ""}}, ""},
		{8, "8", EdgeDef{}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 1 2 2 3 3 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3], 2 [4 5 8], 3 [6 7]", mx.String())
}

// 0:    1
// - 	 |
// 1:    2
// -     |
// 2:    3     5
// -     |  /  |
// 3:    4     6
// -           |
// 4:          7
func TestShortSubtreeBelowLong(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{{5, ""}}, ""},
		{5, "5", EdgeDef{}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{5, ""}, []EdgeDef{}, ""},
		{7, "7", EdgeDef{6, ""}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 2 3 2 3 4]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2], 2 [3 5], 3 [4 6], 4 [7]", mx.String())
}

// 0:    1
// - 	 |
// 1:    2      6
// -     |    / |
// 2:    3   /  7
// -     |  /   |
// 3:    4      8
// -     |   /
// 4:    5
func TestOneNotTwoTwoLevelsDown(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{{6, ""}}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{{8, ""}}, ""},
		{6, "6", EdgeDef{}, []EdgeDef{}, ""},
		{7, "7", EdgeDef{6, ""}, []EdgeDef{}, ""},
		{8, "8", EdgeDef{7, ""}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 2 3 4 1 2 3]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 6], 2 [3 7], 3 [4 8], 4 [5]", mx.String())
}

// 0:    1
// - 	 |
// 1:    2    5
// -     |   /|
// 2:    3  / 6
// -     | //
// 3:    4
func TestMultiSecParentPullDown(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{{5, ""}, {6, ""}}, ""},
		{5, "5", EdgeDef{}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{5, ""}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 2 3 1 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 5], 2 [3 6], 3 [4]", mx.String())

}

// 0:    1   4
// - 	 |  /|
// 1:    2 / 5
// -     |//
// 2:    3
func TestMultiSecParentNoPullDown(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{{4, ""}, {5, ""}}, ""},
		{4, "4", EdgeDef{}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 2 0 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1 4], 1 [2 5], 2 [3]", mx.String())
}

// 0:    1
// - 	 | \
// 1:    2   4
// -     |   |
// 2:    F3  5
// -     | /
// 3:    3
func TestFakeNodes(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{{5, ""}}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 3 1 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 4], 2 [10003 5], 3 [3]", mx.String())
}

// 0:    1
// - 	 | \
// 1:    2   4
// -     |   |
// 2:    F3  5
// -     |   |
// 3:    F3  6
// -     | /
// 4:    3
func TestTwoFakeNodes(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{{6, ""}}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{5, ""}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 4 1 2 3]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 4], 2 [10003 5], 3 [10003 6], 4 [3]", mx.String())
}
