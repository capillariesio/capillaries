package capigraph

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Common MxPermutator and SVG tests

func helperAll(t *testing.T,
	nodeDefs []NodeDef,
	expectedLayerMap string,
	expectedStartMx string,
	expectedIterCount int64,
	expectedPermMxs string,
	expectedHierarchies string) {
	priParentMap := buildPriParentMap(nodeDefs)
	layerMap := buildLayerMap(nodeDefs, priParentMap)
	assert.Equal(t, expectedLayerMap, fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(priParentMap)
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, expectedStartMx, mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
	vnh := NewVizNodeHierarchy(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())

	vnh.buildNewRootSubtreeHierarchy(mx)

	sbPerms := strings.Builder{}
	sbHierarchies := strings.Builder{}
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sbPerms.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
		vnh.reuseRootSubtreeHierarchy(mxPerm)
		vnh.PopulateNodeTotalWidth()
		vnh.PopulateNodesXCoords()
		vnh.PopulateEdgeLabelDimensions()
		vnh.PopulateUpperLayerGapMap(DefaultEdgeLabelFontOptions().SizeInPixels)
		vnh.PopulateNodesYCoords()
		sbHierarchies.WriteString(fmt.Sprintf("Hierarchy %d\n%s", i, vnh.String()))
	})

	assert.Equal(t, expectedIterCount, mxi.MxIteratorCount())
	assert.Equal(t, expectedPermMxs, sbPerms.String())
	assert.Equal(t, expectedHierarchies, sbHierarchies.String())
}

func helperIteratorAndIncrementalCount(t *testing.T,
	nodeDefs []NodeDef,
	expectedLayerMap string,
	expectedStartMx string,
	expectedIterCount int64,
	expectedIncrementalCount int64) {
	priParentMap := buildPriParentMap(nodeDefs)
	layerMap := buildLayerMap(nodeDefs, priParentMap)
	assert.Equal(t, expectedLayerMap, fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(priParentMap)
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, expectedStartMx, mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)

	cnt := int64(0)
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		cnt++
	})

	assert.Equal(t, expectedIterCount, mxi.MxIteratorCount())
	assert.Equal(t, expectedIncrementalCount, cnt)
}

func TestBasicMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsBasic,
		"[-4 0 1 0]",
		"0 [1 3], 1 [2]",
		int64(2),
		`0: 0 [3 1], 1 [2]
1: 0 [1 3], 1 [2]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [2] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:256.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [2] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:256.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestTrivialParallelMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsTrivialParallel,
		"[-4 0 1 0 1]",
		"0 [1 3], 1 [2 4]",
		int64(2),
		`0: 0 [3 1], 1 [4 2]
1: 0 [1 3], 1 [2 4]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [2] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [2] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestOneEnclosingOneLevelMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsOneEnclosedOneLevel,
		"[-4 0 1 1 2 2 1]",
		"0 [1], 1 [2 3 6], 2 [4 5]",
		int64(6),
		`0: 0 [1], 1 [6 3 2], 2 [5 4]
1: 0 [1], 1 [3 6 2], 2 [5 4]
2: 0 [1], 1 [3 2 6], 2 [5 4]
3: 0 [1], 1 [6 2 3], 2 [4 5]
4: 0 [1], 1 [2 6 3], 2 [4 5]
5: 0 [1], 1 [2 3 6], 2 [4 5]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [3 6 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 2
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 3
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 4
Id:1 RootId:1 Layer:0 [2 6 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 5
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestOneEnclosedTwoLevelsMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsOneEnclosedTwoLevels,
		"[-4 0 1 2 3 1 2 3 2]",
		"0 [1], 1 [2 5], 2 [3 6 8], 3 [4 7]",
		int64(6),
		`0: 0 [1], 1 [5 2], 2 [8 6 3], 3 [7 4]
1: 0 [1], 1 [5 2], 2 [6 8 3], 3 [7 4]
2: 0 [1], 1 [5 2], 2 [6 3 8], 3 [7 4]
3: 0 [1], 1 [2 5], 2 [8 3 6], 3 [4 7]
4: 0 [1], 1 [2 5], 2 [3 8 6], 3 [4 7]
5: 0 [1], 1 [2 5], 2 [3 6 8], 3 [4 7]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [5 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [5 8 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 2
Id:1 RootId:1 Layer:0 [5 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 3
Id:1 RootId:1 Layer:0 [2 5] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 4
Id:1 RootId:1 Layer:0 [2 8 5] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:2 [7] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 5
Id:1 RootId:1 Layer:0 [2 5] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestNoIntervalsMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsNoIntervals,
		"[-4 0 1 2 1 2]",
		"0 [1], 1 [2 4], 2 [3 5]",
		int64(2),
		`0: 0 [1], 1 [4 2], 2 [5 3]
1: 0 [1], 1 [2 4], 2 [3 5]
`, `Hierarchy 0
Id:1 RootId:1 Layer:0 [4 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [2 4] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestFlat10MxPermutator(t *testing.T) {
	helperIteratorAndIncrementalCount(t,
		testNodeDefsFlat10,
		"[-4 0 0 0 0 0 0 0 0 0 0]",
		"0 [1 2 3 4 5 6 7 8 9 10]",
		int64(3628800),
		int64(3628800))
}

func TestTwoEnclosingTwoLevelsNodeSizeMattersMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsTwoEnclosedNodeSizeMatters,
		"[-4 0 1 1 2 2 3 3 2 2]",
		"0 [1], 1 [2 3], 2 [4 5 8 9], 3 [6 7]",
		int64(24),
		`0: 0 [1], 1 [3 2], 2 [9 8 5 4], 3 [7 6]
1: 0 [1], 1 [3 2], 2 [8 9 5 4], 3 [7 6]
2: 0 [1], 1 [3 2], 2 [8 5 9 4], 3 [7 6]
3: 0 [1], 1 [3 2], 2 [8 5 4 9], 3 [7 6]
4: 0 [1], 1 [3 2], 2 [9 5 8 4], 3 [7 6]
5: 0 [1], 1 [3 2], 2 [5 9 8 4], 3 [7 6]
6: 0 [1], 1 [3 2], 2 [5 8 9 4], 3 [7 6]
7: 0 [1], 1 [3 2], 2 [5 8 4 9], 3 [7 6]
8: 0 [1], 1 [3 2], 2 [9 5 4 8], 3 [7 6]
9: 0 [1], 1 [3 2], 2 [5 9 4 8], 3 [7 6]
10: 0 [1], 1 [3 2], 2 [5 4 9 8], 3 [7 6]
11: 0 [1], 1 [3 2], 2 [5 4 8 9], 3 [7 6]
12: 0 [1], 1 [2 3], 2 [9 8 4 5], 3 [6 7]
13: 0 [1], 1 [2 3], 2 [8 9 4 5], 3 [6 7]
14: 0 [1], 1 [2 3], 2 [8 4 9 5], 3 [6 7]
15: 0 [1], 1 [2 3], 2 [8 4 5 9], 3 [6 7]
16: 0 [1], 1 [2 3], 2 [9 4 8 5], 3 [6 7]
17: 0 [1], 1 [2 3], 2 [4 9 8 5], 3 [6 7]
18: 0 [1], 1 [2 3], 2 [4 8 9 5], 3 [6 7]
19: 0 [1], 1 [2 3], 2 [4 8 5 9], 3 [6 7]
20: 0 [1], 1 [2 3], 2 [9 4 5 8], 3 [6 7]
21: 0 [1], 1 [2 3], 2 [4 9 5 8], 3 [6 7]
22: 0 [1], 1 [2 3], 2 [4 5 9 8], 3 [6 7]
23: 0 [1], 1 [2 3], 2 [4 5 8 9], 3 [6 7]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 2
Id:1 RootId:1 Layer:0 [3 9 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 3
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 4
Id:1 RootId:1 Layer:0 [3 8 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 5
Id:1 RootId:1 Layer:0 [3 9 8 2] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:156.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 6
Id:1 RootId:1 Layer:0 [3 8 9 2] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:156.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 7
Id:1 RootId:1 Layer:0 [3 8 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 8
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 9
Id:1 RootId:1 Layer:0 [3 9 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 10
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 11
Id:1 RootId:1 Layer:0 [3 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 12
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 13
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 14
Id:1 RootId:1 Layer:0 [2 9 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 15
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 16
Id:1 RootId:1 Layer:0 [2 8 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 17
Id:1 RootId:1 Layer:0 [2 9 8 3] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:156.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 18
Id:1 RootId:1 Layer:0 [2 8 9 3] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:156.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 19
Id:1 RootId:1 Layer:0 [2 8 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 20
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 21
Id:1 RootId:1 Layer:0 [2 9 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:104.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 22
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 23
Id:1 RootId:1 Layer:0 [2 3] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [4] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:2 [6] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [7] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:8 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:9 RootId:9 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestOneSecondaryMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsOneSecondary,
		"[-4 0 1 0 1 2 1]",
		"0 [1 3], 1 [2 4 6], 2 [5]",
		int64(6),
		`0: 0 [3 1], 1 [6 4 2], 2 [5]
1: 0 [3 1], 1 [4 6 2], 2 [5]
2: 0 [3 1], 1 [4 2 6], 2 [5]
3: 0 [1 3], 1 [6 2 4], 2 [5]
4: 0 [1 3], 1 [2 6 4], 2 [5]
5: 0 [1 3], 1 [2 4 6], 2 [5]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [2] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:3 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [2] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:3 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 2
Id:1 RootId:1 Layer:0 [2] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:3 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 3
Id:1 RootId:1 Layer:0 [2] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:3 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 4
Id:1 RootId:1 Layer:0 [2] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:3 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 5
Id:1 RootId:1 Layer:0 [2] X:0.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:3 Layer:0 [4] X:52.00 Y:0.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:3 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:3 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestDiamonMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsDiamond,
		"[-4 0 1 1 1 2 1]",
		"0 [1], 1 [2 3 4 6], 2 [5]",
		int64(24),
		`0: 0 [1], 1 [6 4 2 3], 2 [5]
1: 0 [1], 1 [4 6 2 3], 2 [5]
2: 0 [1], 1 [4 2 6 3], 2 [5]
3: 0 [1], 1 [4 2 3 6], 2 [5]
4: 0 [1], 1 [6 3 4 2], 2 [5]
5: 0 [1], 1 [3 6 4 2], 2 [5]
6: 0 [1], 1 [3 4 6 2], 2 [5]
7: 0 [1], 1 [3 4 2 6], 2 [5]
8: 0 [1], 1 [6 3 2 4], 2 [5]
9: 0 [1], 1 [3 6 2 4], 2 [5]
10: 0 [1], 1 [3 2 6 4], 2 [5]
11: 0 [1], 1 [3 2 4 6], 2 [5]
12: 0 [1], 1 [6 4 3 2], 2 [5]
13: 0 [1], 1 [4 6 3 2], 2 [5]
14: 0 [1], 1 [4 3 6 2], 2 [5]
15: 0 [1], 1 [4 3 2 6], 2 [5]
16: 0 [1], 1 [6 2 4 3], 2 [5]
17: 0 [1], 1 [2 6 4 3], 2 [5]
18: 0 [1], 1 [2 4 6 3], 2 [5]
19: 0 [1], 1 [2 4 3 6], 2 [5]
20: 0 [1], 1 [6 2 3 4], 2 [5]
21: 0 [1], 1 [2 6 3 4], 2 [5]
22: 0 [1], 1 [2 3 6 4], 2 [5]
23: 0 [1], 1 [2 3 4 6], 2 [5]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [4 2 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [4 6 2 3] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 2
Id:1 RootId:1 Layer:0 [4 2 6 3] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 3
Id:1 RootId:1 Layer:0 [4 2 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 4
Id:1 RootId:1 Layer:0 [3 4 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 5
Id:1 RootId:1 Layer:0 [3 6 4 2] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 6
Id:1 RootId:1 Layer:0 [3 4 6 2] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 7
Id:1 RootId:1 Layer:0 [3 4 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 8
Id:1 RootId:1 Layer:0 [3 2 4] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 9
Id:1 RootId:1 Layer:0 [3 6 2 4] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 10
Id:1 RootId:1 Layer:0 [3 2 6 4] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 11
Id:1 RootId:1 Layer:0 [3 2 4] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 12
Id:1 RootId:1 Layer:0 [4 3 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 13
Id:1 RootId:1 Layer:0 [4 6 3 2] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 14
Id:1 RootId:1 Layer:0 [4 3 6 2] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 15
Id:1 RootId:1 Layer:0 [4 3 2] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 16
Id:1 RootId:1 Layer:0 [2 4 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 17
Id:1 RootId:1 Layer:0 [2 6 4 3] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 18
Id:1 RootId:1 Layer:0 [2 4 6 3] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:156.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 19
Id:1 RootId:1 Layer:0 [2 4 3] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 20
Id:1 RootId:1 Layer:0 [2 3 4] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 21
Id:1 RootId:1 Layer:0 [2 6 3 4] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:104.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 22
Id:1 RootId:1 Layer:0 [2 3 6 4] X:0.00 Y:0.00 TotalW:188.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 23
Id:1 RootId:1 Layer:0 [2 3 4] X:0.00 Y:0.00 TotalW:136.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:1 [5] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:1 [] X:104.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:2 [] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:6 Layer:1 [] X:156.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestTwoLevelsFromOneParentSameRootMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsTwoLevelsFromOneParentSameRoot,
		"[-4 0 1 2 3 1 3 4]",
		"0 [1], 1 [2 5], 2 [3 10006], 3 [4 6], 4 [7]",
		int64(2),
		`0: 0 [1], 1 [5 2], 2 [10006 3], 3 [6 4], 4 [7]
1: 0 [1], 1 [2 5], 2 [3 10006], 3 [4 6], 4 [7]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [5 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [7] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:4 [] X:0.00 Y:400.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [2 5] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:1 [6] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:3 [7] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:4 [] X:52.00 Y:400.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func TestTwoLevelsFromOneParentSameRootTwoFakesMxPermutator(t *testing.T) {
	helperAll(t,
		testNodeDefsTwoLevelsFromOneParentSameRootTwoFakes,
		"[-4 0 1 2 3 4 1 4 5]",
		"0 [1], 1 [2 6], 2 [3 10007], 3 [4 10007], 4 [5 7], 5 [8]",
		int64(2),
		`0: 0 [1], 1 [6 2], 2 [10007 3], 3 [10007 4], 4 [7 5], 5 [8]
1: 0 [1], 1 [2 6], 2 [3 10007], 3 [4 10007], 4 [5 7], 5 [8]
`,
		`Hierarchy 0
Id:1 RootId:1 Layer:0 [6 2] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:52.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [5] X:52.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:4 [] X:52.00 Y:400.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:1 [7] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:4 [8] X:0.00 Y:400.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:1 Layer:5 [] X:0.00 Y:500.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Hierarchy 1
Id:1 RootId:1 Layer:0 [2 6] X:0.00 Y:0.00 TotalW:84.00 NodeW:32.00 NodeH:40.00
Id:2 RootId:1 Layer:1 [3] X:0.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:3 RootId:1 Layer:2 [4] X:0.00 Y:200.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:4 RootId:1 Layer:3 [5] X:0.00 Y:300.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:5 RootId:1 Layer:4 [] X:0.00 Y:400.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:6 RootId:1 Layer:1 [7] X:52.00 Y:100.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:7 RootId:1 Layer:4 [8] X:52.00 Y:400.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
Id:8 RootId:1 Layer:5 [] X:52.00 Y:500.00 TotalW:32.00 NodeW:32.00 NodeH:40.00
`)
}

func Test40milPermsMxPermutator(t *testing.T) {
	//defer profile.Start(profile.CPUProfile).Stop()
	helperIteratorAndIncrementalCount(t,
		testNodeDefs40milPerms,
		"[-4 0 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 1]",
		"0 [1], 1 [2 3 4 20], 2 [5 6 7 8 9 10 11 12 13 14 15 16 17 18 19]",
		int64(41472000),
		int64(41472000))
}

func Test300bilPermsMxPermutator(t *testing.T) {
	layerMap := buildLayerMap(testNodeDefs300bilPerms, buildPriParentMap(testNodeDefs300bilPerms))
	assert.Equal(t, "[-4 0 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 1 1 1 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefs300bilPerms))
	mx, _ := NewLayerMx(testNodeDefs300bilPerms, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3 4 22 23 24 25], 2 [5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21]", mx.String())

	_, err := NewLayerMxPermIterator(testNodeDefs300bilPerms, mx)
	assert.Contains(t, err.Error(), "313528320000")
}
