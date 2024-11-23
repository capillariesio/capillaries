package capigraph

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 0:    1  3
// -     |  |
// 1:    2  4
func Test0(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{}, ""},
	}
	priParentMap := buildPriParentMap(nodeDefs)
	layerMap := buildLayerMap(nodeDefs, priParentMap)
	assert.Equal(t, "[-4 0 1 0 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(priParentMap)
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1 3], 1 [2 4]", mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
	sb := strings.Builder{}

	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i, mxPerm.String()))
	})

	// WRONG

	mxPerms := `0: 0 [3 1], 1 [4 2]
1: 0 [1 3], 1 [2 4]
`
	assert.Equal(t, mxPerms, sb.String())
	assert.Equal(t, int64(2), mxi.MxIteratorCount())
}

// 0:    1  3
// -     |  |
// 1:    2  4  6
// -        | /
// 2:       5
func Test1(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{{6, ""}}, ""},
		{6, "6", EdgeDef{}, []EdgeDef{}, ""},
	}
	priParentMap := buildPriParentMap(nodeDefs)
	layerMap := buildLayerMap(nodeDefs, priParentMap)
	assert.Equal(t, "[-4 0 1 0 1 2 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(priParentMap)
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1 3], 1 [2 4 6], 2 [5]", mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
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

// 0:       1
// -      / | \
// 1:    2  3  4 6
// -        | //
// 2:       5
func Test2(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{3, ""}, []EdgeDef{{4, ""}, {6, ""}}, ""},
		{6, "6", EdgeDef{}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 1 1 2 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3 4 6], 2 [5]", mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
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

// 0:    1
// -     |  \
// 1:    2     5
// -     |     |
// 2:    3  8  6
// -     | / \ |
// 3:    4     7
func TestPotentialEnclosure(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{{8, ""}}, ""},
		{5, "5", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{5, ""}, []EdgeDef{}, ""},
		{7, "7", EdgeDef{6, ""}, []EdgeDef{{8, ""}}, ""},
		{8, "8", EdgeDef{}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 2 3 1 2 3 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 5], 2 [3 6 8], 3 [4 7]", mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
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

// 0:    1
// -     |  \
// 1:    2     4
// -     |     |
// 2:    3     5
func TestNoIntervals(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 2 1 2]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 4], 2 [3 5]", mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
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

func Test10Inserts(t *testing.T) {
	//defer profile.Start(profile.MemProfileHeap).Stop()
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{}, []EdgeDef{}, ""},
		{7, "7", EdgeDef{}, []EdgeDef{}, ""},
		{8, "8", EdgeDef{}, []EdgeDef{}, ""},
		{9, "9", EdgeDef{}, []EdgeDef{}, ""},
		{10, "10", EdgeDef{}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 0 0 0 0 0 0 0 0 0]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1 2 3 4 5 6 7 8 9 10]", mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
	cnt := 0
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		cnt++
	})

	assert.Equal(t, 3628800, cnt)
	assert.Equal(t, int64(3628800), mxi.MxIteratorCount())

}

func TestBigIntervalsTreeAndTwoInserts(t *testing.T) {
	//defer profile.Start(profile.CPUProfile).Stop()
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},

		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},

		{5, "5", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{7, "7", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{8, "8", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{9, "9", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{10, "10", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{11, "11", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{12, "12", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{13, "13", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{14, "14", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{15, "15", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{16, "16", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{17, "17", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{18, "18", EdgeDef{4, ""}, []EdgeDef{{20, ""}}, ""},
		//{19, "19", EdgeDef{4, ""}, []EdgeDef{{21, ""}}, ""},
		{19, "19", EdgeDef{4, ""}, []EdgeDef{}, ""},

		{20, "20", EdgeDef{}, []EdgeDef{}, ""},
		//{21, "21", EdgeDef{}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3 4 20], 2 [5 6 7 8 9 10 11 12 13 14 15 16 17 18 19]", mx.String())

	mxi, _ := NewLayerMxPermIterator(nodeDefs, mx)
	cnt := 0

	mxi.MxIterator(func(nestedTotalCnt int, perm LayerMx) {
		cnt++
	})

	// With 20: 41472000 1.5 s
	// With 21: 207360000 and 7s
	assert.Equal(t, 41472000, cnt)
	assert.Equal(t, int64(41472000), mxi.MxIteratorCount())
}

func TestExtraLargeCount(t *testing.T) {
	//defer profile.Start(profile.CPUProfile).Stop()
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},

		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},

		{5, "5", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{7, "7", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{8, "8", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{9, "9", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{10, "10", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{11, "11", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{12, "12", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{13, "13", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{14, "14", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{15, "15", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{16, "16", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{17, "17", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{18, "18", EdgeDef{4, ""}, []EdgeDef{{22, ""}}, ""},
		{19, "19", EdgeDef{4, ""}, []EdgeDef{{23, ""}}, ""},
		{20, "20", EdgeDef{4, ""}, []EdgeDef{{24, ""}}, ""},
		{21, "21", EdgeDef{4, ""}, []EdgeDef{{25, ""}}, ""},

		{22, "22", EdgeDef{}, []EdgeDef{}, ""},
		{23, "23", EdgeDef{}, []EdgeDef{}, ""},
		{24, "24", EdgeDef{}, []EdgeDef{}, ""},
		{25, "25", EdgeDef{}, []EdgeDef{}, ""},
	}
	layerMap := buildLayerMap(nodeDefs, buildPriParentMap(nodeDefs))
	assert.Equal(t, "[-4 0 1 1 1 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 2 1 1 1 1]", fmt.Sprintf("%v", layerMap))

	rootNodes := buildRootNodeList(buildPriParentMap(nodeDefs))
	mx, _ := NewLayerMx(nodeDefs, layerMap, rootNodes)
	assert.Equal(t, "0 [1], 1 [2 3 4 22 23 24 25], 2 [5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21]", mx.String())

	_, err := NewLayerMxPermIterator(nodeDefs, mx)
	assert.Contains(t, err.Error(), "313528320000")
}
