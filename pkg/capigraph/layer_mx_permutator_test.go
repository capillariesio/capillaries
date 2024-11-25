package capigraph

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Common MxPermutator and SVG tests

func TrivialParallelMxPermutator(t *testing.T) {
	priParentMap := buildPriParentMap(testNodeDefsTrivialParallel)
	layerMap := buildLayerMap(testNodeDefsTrivialParallel, priParentMap)
	assert.Equal(t, "[-4 0 1 0 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(priParentMap)
	mx, _ := NewLayerMx(testNodeDefsTrivialParallel, layerMap, rootNodes)
	assert.Equal(t, "0 [1 3], 1 [2 4]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsTrivialParallel, mx)
	sb := strings.Builder{}

	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})

	mxPerms := `0: 0 [3 1], 1 [4 2]
1: 0 [1 3], 1 [2 4]
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(2), mxi.MxIteratorCount())
}

func TestOneEnclosingOneLevelMxPermutator(t *testing.T) {
	layerMap := buildLayerMap(testNodeDefsOneEnclosedOneLevel, buildPriParentMap(testNodeDefsOneEnclosedOneLevel))
	assert.Equal(t, "[-4 0 1 1 2 2 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefsOneEnclosedOneLevel))
	mx, _ := NewLayerMx(testNodeDefsOneEnclosedOneLevel, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3 6], 2 [4 5]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsOneEnclosedOneLevel, mx)
	sb := strings.Builder{}
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})
	mxPerms := `0: 0 [1], 1 [6 3 2], 2 [5 4]
1: 0 [1], 1 [3 6 2], 2 [5 4]
2: 0 [1], 1 [3 2 6], 2 [5 4]
3: 0 [1], 1 [6 2 3], 2 [4 5]
4: 0 [1], 1 [2 6 3], 2 [4 5]
5: 0 [1], 1 [2 3 6], 2 [4 5]
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(6), mxi.MxIteratorCount())
}

func TestOneEnclosedTwoLevelsMxPermutator(t *testing.T) {
	layerMap := buildLayerMap(testNodeDefsOneEnclosedTwoLevels, buildPriParentMap(testNodeDefsOneEnclosedTwoLevels))
	assert.Equal(t, "[-4 0 1 2 3 1 2 3 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefsOneEnclosedTwoLevels))
	mx, _ := NewLayerMx(testNodeDefsOneEnclosedTwoLevels, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 5], 2 [3 6 8], 3 [4 7]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsOneEnclosedTwoLevels, mx)
	sb := strings.Builder{}
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})
	mxPerms := `0: 0 [1], 1 [5 2], 2 [8 6 3], 3 [7 4]
1: 0 [1], 1 [5 2], 2 [6 8 3], 3 [7 4]
2: 0 [1], 1 [5 2], 2 [6 3 8], 3 [7 4]
3: 0 [1], 1 [2 5], 2 [8 3 6], 3 [4 7]
4: 0 [1], 1 [2 5], 2 [3 8 6], 3 [4 7]
5: 0 [1], 1 [2 5], 2 [3 6 8], 3 [4 7]
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(6), mxi.MxIteratorCount())
}

func TestNoIntervalsMxPermutator(t *testing.T) {
	layerMap := buildLayerMap(testNodeDefsNoIntervals, buildPriParentMap(testNodeDefsNoIntervals))
	assert.Equal(t, "[-4 0 1 2 1 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefsNoIntervals))
	mx, _ := NewLayerMx(testNodeDefsNoIntervals, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 4], 2 [3 5]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsNoIntervals, mx)
	sb := strings.Builder{}
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})
	mxPerms := `0: 0 [1], 1 [4 2], 2 [5 3]
1: 0 [1], 1 [2 4], 2 [3 5]
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(2), mxi.MxIteratorCount())
}

func TestFlat10MxPermutator(t *testing.T) {
	layerMap := buildLayerMap(testNodeDefsFlat10, buildPriParentMap(testNodeDefsFlat10))
	assert.Equal(t, "[-4 0 0 0 0 0 0 0 0 0 0]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefsFlat10))
	mx, _ := NewLayerMx(testNodeDefsFlat10, layerMap, rootNodes)
	assert.Equal(t, "0 [1 2 3 4 5 6 7 8 9 10]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsFlat10, mx)
	cnt := 0
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		cnt++
	})

	assert.Equal(t, 3628800, cnt)
	assert.Equal(t, int64(3628800), mxi.MxIteratorCount())
}

func TestTwoEnclosingTwoLevelsNodeSizeMattersMxPermutator(t *testing.T) {
	layerMap := buildLayerMap(testNodeDefsTwoEnclosedNodeSizeMatters, buildPriParentMap(testNodeDefsTwoEnclosedNodeSizeMatters))
	assert.Equal(t, "[-4 0 1 1 2 2 3 3 2 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefsTwoEnclosedNodeSizeMatters))
	mx, _ := NewLayerMx(testNodeDefsTwoEnclosedNodeSizeMatters, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3], 2 [4 5 8 9], 3 [6 7]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsTwoEnclosedNodeSizeMatters, mx)
	sb := strings.Builder{}
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})
	mxPerms := `0: 0 [1], 1 [3 2], 2 [9 8 5 4], 3 [7 6]
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
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(24), mxi.MxIteratorCount())
}

func TestOneSecondaryMxPermutator(t *testing.T) {
	priParentMap := buildPriParentMap(testNodeDefsOneSecondary)
	layerMap := buildLayerMap(testNodeDefsOneSecondary, priParentMap)
	assert.Equal(t, "[-4 0 1 0 1 2 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(priParentMap)
	mx, _ := NewLayerMx(testNodeDefsOneSecondary, layerMap, rootNodes)
	assert.Equal(t, "0 [1 3], 1 [2 4 6], 2 [5]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsOneSecondary, mx)
	sb := strings.Builder{}

	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})

	mxPerms := `0: 0 [3 1], 1 [6 4 2], 2 [5]
1: 0 [3 1], 1 [4 6 2], 2 [5]
2: 0 [3 1], 1 [4 2 6], 2 [5]
3: 0 [1 3], 1 [6 2 4], 2 [5]
4: 0 [1 3], 1 [2 6 4], 2 [5]
5: 0 [1 3], 1 [2 4 6], 2 [5]
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(6), mxi.MxIteratorCount())
}

func TestDiamonMxPermutator(t *testing.T) {
	layerMap := buildLayerMap(testNodeDefsDiamond, buildPriParentMap(testNodeDefsDiamond))
	assert.Equal(t, "[-4 0 1 1 1 2 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefsDiamond))
	mx, _ := NewLayerMx(testNodeDefsDiamond, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3 4 6], 2 [5]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefsDiamond, mx)
	sb := strings.Builder{}
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})
	mxPerms := `0: 0 [1], 1 [6 4 2 3], 2 [5]
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
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(24), mxi.MxIteratorCount())
}

func Test40milPermsMxPermutator(t *testing.T) {
	//defer profile.Start(profile.CPUProfile).Stop()
	layerMap := buildLayerMap(testNodeDefs40milPerms, buildPriParentMap(testNodeDefs40milPerms))
	assert.Equal(t, "[-4 0 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(testNodeDefs40milPerms))
	mx, _ := NewLayerMx(testNodeDefs40milPerms, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3 4 20], 2 [5 6 7 8 9 10 11 12 13 14 15 16 17 18 19]", mx.String())

	mxi, _ := NewLayerMxPermIterator(testNodeDefs40milPerms, mx)
	cnt := 0

	mxi.MxIterator(func(nestedTotalCnt int, perm LayerMx) {
		cnt++
	})

	// With 20: 41472000 1.5 s
	// With 21: 207360000 and 7s
	assert.Equal(t, 41472000, cnt)
	assert.Equal(t, int64(41472000), mxi.MxIteratorCount())
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
