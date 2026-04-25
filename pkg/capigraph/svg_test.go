package capigraph

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getPermsFromSvg(svg string) int64 {
	re := regexp.MustCompile(`Perms (\d+), elapsed [0-9\.s]+, dist ([\d\.]+)`)
	match := re.FindStringSubmatch(svg)
	permutations, _ := strconv.Atoi(match[1])
	return int64(permutations)
}

func getDistFromSvg(svg string) float64 {
	re := regexp.MustCompile(`Perms (\d+), elapsed [0-9\.s]+, dist ([\d\.]+)`)
	match := re.FindStringSubmatch(svg)
	distance, _ := strconv.ParseFloat(match[2], 64)
	return distance
}

// Common MxPermutator and SVG tests

func TestBasicSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsBasic, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 72.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TrivialParallelSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsTrivialParallel, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 47.2, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestOneEnclosingOneLevelSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsOneEnclosedOneLevel, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(6), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestOneEnclosedTwoLevelsSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsOneEnclosedTwoLevels, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(6), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestNoIntervalsSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsNoIntervals, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 0.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestFlat10Svg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsFlat10, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(3628800), getPermsFromSvg(svg))
	assert.Equal(t, 0.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestTwoEnclosingTwoLevelsNodeSizeMattersSvg(t *testing.T) {
	// Only one of 8, 9 is enclosed
	svg, _ := Draw(context.TODO(), testNodeDefsTwoEnclosedNodeSizeMatters, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(24), getPermsFromSvg(svg))
	assert.Equal(t, 432.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)

	// Now make nodes 4 and 5 considerably wider - it will change the best hierarchy, 8 and 9 enclosed
	testNodeDefsTwoEnclosedNodeSizeMatters[4].Text += " wider"
	testNodeDefsTwoEnclosedNodeSizeMatters[5].Text += " wider"

	svg, _ = Draw(context.TODO(), testNodeDefsTwoEnclosedNodeSizeMatters, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, 576.0, math.Round(getDistFromSvg(svg)*100.0)/100.0)
	fmt.Printf("%s\n", svg)
}

func TestOneSecondarySvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsOneSecondary, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(6), getPermsFromSvg(svg))
	assert.Equal(t, 72.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestDiamonSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsDiamond, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(24), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestSubtreeBelowLongSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsSubtreeBelowLong, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 72.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestOneNotTwoLevelsDownSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsOneNotTwoLevelsDown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestMultiSecParentPullDownSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsMultiSecParentPullDown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestMultiSecParentNoPullDownSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsMultiSecParentNoPullDown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestTwoLevelsFromOneParentSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsTwoLevelsFromOneParent, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestTwoLevelsFromOneParentSameRootSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsTwoLevelsFromOneParentSameRoot, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 72.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestTwoLevelsFromOneParentSameRootTwoFakesSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsTwoLevelsFromOneParentSameRootTwoFakes, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 72.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestDuplicateSecLabelsSvg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, err := getBestHierarchy(context.TODO(), testNodeDefsDuplicateSecLabels, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), Optimize)
	assert.Nil(t, err)
	// Zero-width sec labels to avoid overlapping
	assert.Equal(t, 0.0, vizNodeMap[8].IncomingVizEdges[1].W)
	// Displayed sec labels
	assert.Equal(t, 37.08, vizNodeMap[7].IncomingVizEdges[1].W)
	assert.Equal(t, 37.08, vizNodeMap[9].IncomingVizEdges[1].W)
	assert.Equal(t, 37.08, vizNodeMap[10].IncomingVizEdges[1].W)
	assert.Equal(t, 37.08, vizNodeMap[11].IncomingVizEdges[1].W)

	svg := drawVizNodes(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), totalPermutations, elapsed, bestDist)
	assert.Equal(t, int64(720), getPermsFromSvg(svg))
	assert.Equal(t, 660.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestLayerLongRootsSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsLayerLongRoots, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(6), getPermsFromSvg(svg))
	assert.Equal(t, 216.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestPriAndSecInfinitePulldownSvg(t *testing.T) {
	svg, _ := Draw(context.TODO(), testNodeDefsPriAndSecInfinitePulldown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(4), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func Test40milPermsSvg(t *testing.T) {
	// defer profile.Start(profile.CPUProfile).Stop()

	drawCtx, drawCancel := context.WithTimeout(context.Background(), 1*time.Second)
	svg, err := Draw(drawCtx, testNodeDefs40milPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	drawCancel()
	assert.Equal(t, "timeout exceeded", err.Error())

	// If we allowed longer timeouts, this would take 15 sec, too long for a unit test
	// assert.Equal(t, int64(41472000), getPermsFromSvg(svg))
	// assert.Equal(t, 84.0, math.Round(bestDist*100.0)/100.0)
	fmt.Printf("%s\n", svg)
}

// Not a big diagram, but too many permutations
func Test300bilPermsSvg(t *testing.T) {
	var err error
	_, err = Draw(context.TODO(), testNodeDefs300bilPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Contains(t, err.Error(), "313528320000")

	drawCtx, drawCancel := context.WithTimeout(context.Background(), 15*time.Second)
	_, err = Draw(drawCtx, testNodeDefs300bilPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	drawCancel()
	assert.Contains(t, err.Error(), "313528320000")

	svg, err := Draw(context.TODO(), testNodeDefs300bilPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), DoNotOptimize)
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}

// SVG-specific tests

// Unoptimized only, too many permutations
func TestUnoptimizedSvg(t *testing.T) {
	nodeDefs := make([]NodeDef, 0, 10000)
	var populateChildren func(parentIdx int, firstChildIdx int, layer int) int
	populateChildren = func(parentIdx int, firstChildIdx int, layer int) int {
		if layer == 4 {
			return firstChildIdx
		}
		nextChildIdx := firstChildIdx
		for range 5 - layer {
			parentOverrideIdx := parentIdx
			if firstChildIdx%7 == 0 {
				parentOverrideIdx = 0
			}
			newNode := NodeDef{int16(nextChildIdx), fmt.Sprintf("%d", nextChildIdx), EdgeDef{int16(parentOverrideIdx), "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""}
			if parentOverrideIdx != 0 && firstChildIdx%9 == 0 {
				newNode.SecIn = append(newNode.SecIn, EdgeDef{int16(firstChildIdx / 2), "", TextColorDefault})
			}
			nodeDefs = append(nodeDefs, newNode)
			nextChildIdx = populateChildren(nextChildIdx, nextChildIdx+1, layer+1)
		}
		return nextChildIdx
	}
	populateChildren(0, 1, 0)

	svg, err := Draw(context.TODO(), nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), DoNotOptimize)
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}

// Takes 160s to complete (it's optimized!), but working.
// Fake nodes, enclosed subtrees, 20 levels, 484 nodes.
// func TestInsanelyBigBinaryTreeSvg(t *testing.T) {
// 	nodeDefs := make([]NodeDef, 0, 10000)
// 	var populateChildren func(parentIdx int, firstChildIdx int, layer int) int
// 	populateChildren = func(parentIdx int, firstChildIdx int, layer int) int {
// 		if layer == 7 {
// 			return firstChildIdx
// 		}
// 		nextChildIdx := firstChildIdx
// 		for range 2 {
// 			parentOverrideIdx := parentIdx
// 			if firstChildIdx%7 == 0 && firstChildIdx < 20 {
// 				parentOverrideIdx = 0
// 				layer = 0
// 			}
// 			newNode := NodeDef{int16(nextChildIdx), fmt.Sprintf("%d", nextChildIdx), EdgeDef{int16(parentOverrideIdx), ""}, nil, "", 0, NodeBorderRegular, NodeTextColorDefault, NodeBackgroundSolid, ""}
// 			if parentOverrideIdx != 0 && firstChildIdx%9 == 0 {
// 				newNode.SecIn = append(newNode.SecIn, EdgeDef{int16(firstChildIdx / 2), ""})
// 			}
// 			nodeDefs = append(nodeDefs, newNode)
// 			nextChildIdx = populateChildren(nextChildIdx, nextChildIdx+1, layer+1)
// 		}
// 		return nextChildIdx
// 	}
// 	populateChildren(0, 1, 0)

// 	svg, err := Draw(context.TODO(), nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
// 	assert.Equal(t, int64(3819584), getPermsFromSvg(svg))
// 	assert.Equal(t, nil, err)
// 	fmt.Printf("%s\n", svg)
// }

func TestEnclosingOneLevelWideNodes(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{1, "A1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "A21\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A21", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{3, "A22\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A22", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{4, "A31\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A21 to A31", TextColorDefault}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A31", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{5, "A32\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{3, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A22 to A32", TextColorDefault}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A32", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{6, "B1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}
	svg, _ := Draw(context.TODO(), nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(6), getPermsFromSvg(svg))
	assert.Equal(t, 768.0, getDistFromSvg(svg))
	fmt.Printf("%s\n", svg)
}

func TestHalfComplexWithEnclosed(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{1, "A1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "A2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A2", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{3, "A31\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A2 to A31", TextColorDefault}, []EdgeDef{{10, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B2 to A31", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{4, "A32\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A2 to A32", TextColorDefault}, []EdgeDef{{14, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C3 to A32", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{5, "A41\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{4, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A32 to A41", TextColorDefault}, []EdgeDef{{11, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B3 to A41", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{6, "A42\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{4, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A32 to A42", TextColorDefault}, []EdgeDef{{14, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C3 to A42", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{7, "A51\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{5, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A41 to A51", TextColorDefault}, []EdgeDef{{15, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom D1 to A51", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{8, "A52\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A42 to A52", TextColorDefault}, []EdgeDef{{15, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom D1 to A52", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{9, "B1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{10, "B2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{9, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to B2", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{11, "B3\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{10, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B2 to B3", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{12, "C1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{13, "C2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{12, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C1 to C2", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{14, "C3\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{13, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C2 to C3", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{15, "D1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}
	svg, _ := Draw(context.TODO(), nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(48), getPermsFromSvg(svg))
	assert.Equal(t, 3072.0, math.Round(getDistFromSvg(svg)))
	fmt.Printf("%s\n", svg)
}

func TestConflictingSecAndTotalViewboxWidthAdjustedToLabel(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{1, "A", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "B", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{3, "C", EdgeDef{1, "A to C", TextColorDefault}, []EdgeDef{{2, "B to ? duplicate going really wide", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{4, "D", EdgeDef{3, "C to D", TextColorDefault}, []EdgeDef{{2, "B to ? duplicate going really wide", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}
	svg, _ := Draw(context.TODO(), nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), Optimize)
	assert.Equal(t, int64(2), getPermsFromSvg(svg))
	assert.Equal(t, 144.0, math.Round(getDistFromSvg(svg)*100.0)/100.0)
	fmt.Printf("%s\n", svg)
}

func TestCapillariesIcons(t *testing.T) {
	nodeDefs := []NodeDef{

		{
			1,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			nil,
			"icon-database-table-read",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			2,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{1, "payments\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			3,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{2, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			4,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{2, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			5,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{3, "deal_names\n(10 batches)", TextColorDefault},
			[]EdgeDef{{2, "idx_loan_ids_deal_name\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			6,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{2, "loan_ids\n(10 batches)", TextColorDefault},
			[]EdgeDef{{1, "idx_payments_by_loan_id\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			7,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{6, "loan_payment_summaries\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-py",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			8,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{5, "deal_total_upbs\n(10 batches)", TextColorDefault},
			[]EdgeDef{{7, "idx_loan_summaries_calculated_deal_name\n(lookup)\ndeal_name", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			9,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{4, "deal_sellers\n(10 batches)", TextColorDefault},
			[]EdgeDef{{7, "idx_loan_summaries_calculated_deal_name\n(lookup)\ndeal_name\nseller_name", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			10,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{7, "loan_summaries_calculated\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			11,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{8, "deal_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			12,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{9, "deal_seller_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},

		{
			13,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			nil,
			"icon-database-table-read",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			14,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{13, "payments\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			15,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{14, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			16,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{14, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			17,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{15, "deal_names\n(10 batches)", TextColorDefault},
			[]EdgeDef{{14, "idx_loan_ids_deal_name\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			18,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{14, "loan_ids\n(10 batches)", TextColorDefault},
			[]EdgeDef{{13, "idx_payments_by_loan_id\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			19,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{18, "loan_payment_summaries\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-py",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			20,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{17, "deal_total_upbs\n(10 batches)", TextColorDefault},
			[]EdgeDef{{19, "idx_loan_summaries_calculated_deal_name\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			21,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{16, "deal_sellers\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			22,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{19, "loan_summaries_calculated\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			23,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{20, "deal_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			24,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{21, "deal_seller_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},

		{
			25,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			nil,
			"icon-database-table-read",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			26,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{25, "payments\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			27,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{26, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			28,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{26, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			29,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{27, "deal_names\n(10 batches)", TextColorDefault},
			[]EdgeDef{{26, "idx_loan_ids_deal_name\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			30,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{26, "loan_ids\n(10 batches)", TextColorDefault},
			[]EdgeDef{{25, "idx_payments_by_loan_id\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			31,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{30, "loan_payment_summaries\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-py",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			32,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{29, "deal_total_upbs\n(10 batches)", TextColorDefault},
			[]EdgeDef{{31, "idx_loan_summaries_calculated_deal_name\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			33,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{28, "deal_sellers\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			34,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{31, "loan_summaries_calculated\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			35,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{32, "deal_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			36,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{33, "deal_seller_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},

		{
			37,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			nil,
			"icon-database-table-read",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			38,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{37, "payments\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			39,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{38, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			40,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{38, "loan_ids\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-distinct",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			41,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{39, "deal_names\n(10 batches)", TextColorDefault},
			[]EdgeDef{{38, "idx_loan_ids_deal_name\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			42,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{38, "loan_ids\n(10 batches)", TextColorDefault},
			[]EdgeDef{{37, "idx_payments_by_loan_id\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			43,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{42, "loan_payment_summaries\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-py",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			44,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{41, "deal_total_upbs\n(10 batches)", TextColorDefault},
			[]EdgeDef{{43, "idx_loan_summaries_calculated_deal_name\n(lookup)", TextColorDefault}},
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			45,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{40, "deal_sellers\n(10 batches)", TextColorDefault},
			nil,
			"icon-database-table-join",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			46,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{43, "loan_summaries_calculated\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			47,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{44, "deal_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
		{
			48,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{45, "deal_seller_summaries\n(no parallelism)", TextColorDefault},
			nil,
			"icon-parquet",
			0,
			NodeBorderThick, TextColorDefault, NodeBackgroundSolid, "",
		},
	}
	// overrideCss := ".rect-node-background {rx:20; ry:20;} .rect-node {rx:20; ry:20;} .capigraph-rendering-stats {fill:black;}"
	// nodeColorMap := []int32{0x010101, 0x0000FF, 0x008000, 0xFF0000, 0xFF8C00, 0x2F4F4F} //none, blue, darkgreen, red, darkorange, darkslategray (none, start, success, fail, stopreceived, unknown)
	// for nodeIdx := range nodeDefs {
	// 	nodeDefs[nodeIdx].Color = nodeColorMap[nodeIdx%len(nodeColorMap)]
	// }
	svg, _ := Draw(context.TODO(), nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), CapillariesIcons100x100, "" /* overrideCss*/, DefaultPalette(), Optimize)
	assert.Equal(t, int64(31104), getPermsFromSvg(svg))
	assert.Equal(t, 6618.0, math.Round(getDistFromSvg(svg)*100.0)/100.0)
	fmt.Printf("%s\n", svg)
}

func TestPrefixTree(t *testing.T) {
	rootWord := "trip"
	allWords := []string{"tri", "trip", "trim", "trio", "trib", "trig", "trix", "trin", "trid", "trit", "trial", "tried", "trips", "tries", "trick", "tribe", "trina", "triad", "trims", "trier", "trike", "trios", "trice", "trine", "trite", "tripe", "trill", "triac", "trias", "trigo", "triga", "trist", "trica", "trixy", "trior", "tripl", "triol", "tript", "tripy", "trink", "trigs", "trifa", "trials", "triple", "tricks", "tribal", "trivia", "tribes", "tripod", "tricky", "triton", "tricia", "trisha", "trixie", "triage", "trifle", "trichy", "tricot", "triads", "trivet", "triste", "triumf", "triune", "triode", "trigon", "tringa", "tripos", "trilby", "trimer", "triply", "trills", "tripel", "trillo", "triers", "tricon", "triops", "trines", "trifid", "trices", "triose", "triols", "triole", "triwet", "trivat", "tripot", "trityl", "tripes", "tritor", "tripla", "tricae", "triter", "trisul", "tripal", "trinol", "trifly", "triens", "triene", "triduo", "tridra", "tridii", "triace", "trichi", "triced", "tricar", "triazo", "triary", "trigae", "trigly", "trinil", "trigla", "trined", "trinal", "trimly", "trilli", "trilit", "trikir", "triker", "trijet", "trigyn", "trigos", "triact", "tribune", "trigger", "tribute", "trinity", "triumph", "trilogy", "trivial", "trivium", "trimmed", "tristan", "trimmer", "tripods", "trident", "triples", "trickle", "tricked", "triplet", "tritium", "tripoli", "tripled", "trinket", "tripped", "triplex", "tripple", "triceps", "trisomy", "tripper", "tribble", "trifles", "trifold", "trianon", "trivets", "trigram", "tritone", "tripler", "triadic", "tritons", "tricker", "triable", "tripack", "trimers", "trifled", "trionfo", "tricorn", "trinkle", "triller", "tribuna", "trireme", "tristam", "trilled", "triodes", "triduum", "triolet", "trionfi", "triplum", "trionyx", "tripart", "trionym", "triurid", "triones", "trioxid", "trioses", "trional", "tripara", "tripery", "tripody", "tritely", "tritest", "tritish", "tritolo", "tritoma", "triture", "triunal", "triunes", "triuris", "trivant", "trivvet", "trizoic", "tritaph", "trisulc", "trisula", "trippet", "tripsis", "triquet", "trisalt", "trisazo", "trisect", "triseme", "trishaw", "trishna", "trismic", "trismus", "trisome", "trizone", "trioecs", "triacid", "tribrac", "tribual", "trichia", "tricing", "trickie", "trickly", "tricksy", "triclad", "tricots", "triduam", "triduan", "triedly", "trienes", "trifler", "triblet", "tribase", "triaene", "triages", "triakid", "triamid", "triamin", "triarch", "triarii", "triaryl", "triatic", "triaxal", "triaxon", "triazin", "tribade", "tribady", "triflet", "trifoil", "trifoly", "trillet", "trillil", "trilobe", "trimera", "trinary", "trindle", "trinely", "tringle", "trining", "trinkum", "trinode", "trintle", "triobol", "triodia", "trilium", "trilith", "triform", "trigamy", "trigged", "triglid", "triglot", "trigona", "trigone", "trigons", "trijets", "trikaya", "triketo", "trilabe", "trilisa", "trilite", "triodon", "trinidad", "triangle", "tribunal", "triggers", "trillion", "trimming", "tripping", "tributes", "trimmers", "triumphs", "tristate", "triplets", "tricycle", "trillium", "tripwire", "tristram", "trinkets", "trickery", "tribulus", "triassic", "trifecta", "tricking", "triptych", "trifling", "tricolor", "tributed", "triticum", "trickier", "tripling", "trickled", "trioxide", "triaxial", "triazine", "trilling", "trickles", "trippers", "triskele", "trigonal", "triploid", "tristeza", "trimaran", "tribally", "tribunes", "trifocal", "triazole", "trigrams", "triology", "trimeric", "triumvir", "triplane", "trialist", "triatoma", "triturus", "tripolis", "trichome", "trivette", "trimurti", "tringoid", "trinerve", "triplite", "triplice", "trindled", "tripodal", "triplopy", "trindles", "tringine", "triphora", "triphony", "trinodal", "trinkums", "trinklet", "trioecia", "trioleic", "triolein", "triolets", "tripenny", "trioxids", "tripacks", "trinkety", "tripedal", "tripeman", "triodion", "triphane", "trinitro", "triphase", "trinomen", "trizonia", "tritomas", "tritiums", "triticin", "tritical", "tritiate", "trithing", "tristyly", "tristive", "tristich", "tristful", "tritonal", "tritones", "trizonal", "trizomal", "trivirga", "trivalve", "triunity", "triunion", "tritural", "tritoral", "tritonic", "tritonia", "triposes", "trispast", "triptote", "triptane", "tripsome", "tripsill", "trippler", "trippist", "trippets", "trippant", "tripoter", "tripolar", "triptyca", "tripudia", "trisomic", "trisomes", "trisetum", "trisemic", "trisemes", "trisects", "triscele", "triremes", "triratna", "triradii", "tripodic", "tridacna", "trickful", "trickers", "trichord", "trichoma", "trichoid", "trichode", "trichite", "trichion", "trichina", "triceria", "trickily", "trickish", "tricklet", "trictrac", "tricouni", "tricotee", "tricosyl", "tricorns", "tricorne", "triconch", "tricolon", "tricolic", "triclads", "tributer", "tribular", "triander", "triamino", "triamine", "triamide", "triality", "trialism", "trialate", "triadist", "triadism", "triadics", "triapsal", "triarchy", "triareal", "tribrach", "tribelet", "tribasic", "tribadic", "tribades", "triazoic", "triazins", "triazane", "triaster", "triarian", "triacids", "trimtram", "trilloes", "trilliin", "trilleto", "trillers", "trillado", "trilemma", "trilbies", "trikeria", "trihoral", "trihedra", "trilobal", "trilobed", "trilogic", "trimotor", "trimorph", "trimoric", "trimodal", "trimness", "trimmest", "trimeter", "trimesyl", "trimesic", "trimacer", "trihalid", "trigynia", "trifilar", "triethyl", "triequal", "trientes", "triental", "triennia", "triduums", "tridents", "tridecyl", "triddler", "triflers", "triforia", "trifuran", "trigraph", "trigonum", "trigonon", "trigonid", "trigonic", "trigonia", "trigness", "triglyph", "trigging", "triggest", "tridaily", "triggered", "triangles", "trimester", "tribunals", "tributary", "trimmings", "trigraphs", "trickster", "triennial", "trivially", "tricyclic", "trillions", "triumphed", "trickling", "tricycles", "triumphal", "tribesmen", "trifolium", "triticale", "tricuspid", "tribology", "trivalent", "trimethyl", "tristania", "tribalism", "trilobite", "tricolour", "triplexes", "trilinear", "tribesman", "trickiest", "triennium", "trilogies", "trichloro", "tritiated", "tristesse", "trisodium", "triazines", "triquetra", "trinomial", "tributing", "trichomes", "triazoles", "tripitaka", "trimarans", "tribolium", "trichuris", "triphasic", "triclinic", "tricyrtis", "tripeshop", "triparted", "trioxides", "tripelike", "triperies", "triozonid", "tripewife", "triploidy", "triploids", "triplites", "trisilane", "triorchis", "triplegia", "triplasic", "triplaris", "triplanes", "triphenyl", "triphasia", "triphaser", "triplopia", "triosteum", "trinketer", "trinketed", "trinitrin", "trimotors", "trindling", "trinitrid", "trinities", "trinidado", "trineural", "trinerved", "trinchera", "trination", "trinalize", "trinality", "trinketry", "trinodine", "trinovant", "triopidae", "trimetric", "trionymal", "triolefin", "trioleate", "triolcous", "trioicous", "trioecism", "trimorphs", "triocular", "trioctile", "triobolon", "trinunity", "trimstone", "triweekly", "triticums", "triticoid", "triticism", "triticeum", "trithings", "tritheite", "tritheist", "tritheism", "triteness", "triteleia", "tritanope", "tritactic", "trisulfid", "tristichs", "tristezas", "tritocone", "tritomite", "triverbal", "trivantly", "trivalves", "triumviry", "triumvirs", "triumviri", "triumpher", "triturium", "triturate", "tritoxide", "tritorium", "tritopine", "tritonous", "tritonoid", "tritoness", "trisquare", "trisporic", "tripudium", "tripudist", "tripudial", "triptyque", "triptychs", "triptycas", "triptanes", "tripsacum", "trippings", "tripotage", "tripolite", "tripoline", "tripodies", "tripodian", "tripodial", "tripylaea", "tripylean", "trisonant", "trisomies", "trisomics", "trismuses", "triskelia", "triskeles", "trisetose", "triserial", "trisector", "trisected", "trisceles", "trisagion", "triregnum", "triradius", "triradial", "tripmadam", "trimeters", "tricrural", "trichroic", "trichosis", "trichomic", "trichogen", "trichitis", "trichitic", "trichites", "trichions", "trichinas", "trichinal", "trichinae", "trichilia", "tricerium", "tricerion", "tricepses", "trichrome", "tricinium", "tricrotic", "tricresol", "tricotine", "tricosane", "tricornes", "tricolors", "triclinia", "triactine", "tricksome", "tricksily", "tricksier", "trickment", "tricklike", "tricklier", "trickless", "tricephal", "tricenary", "triazolic", "triatomic", "triarctic", "triannual", "triangula", "triangler", "triangled", "triandria", "triamorph", "trialogue", "triagonal", "triaenose", "triadisms", "triadical", "tribalist", "tribarred", "tricaudal", "tricarbon", "tricalcic", "tributist", "tribunate", "tribunary", "tribuloid", "tribulate", "tribually", "tribromid", "tribrachs", "tribonema", "tribeship", "tribelike", "tribeless", "triadenum", "trimerous", "trihydrol", "trihydric", "trihybrid", "trihourly", "trihedron", "trihedral", "trihalide", "trigynous", "trigynian", "trigonous", "trigonoid", "trigonite", "triglyphs", "triglidae", "trigintal", "trijugate", "trijugous", "trimerite", "trimeride", "trimellic", "trilogist", "trilobita", "trilobate", "trilliums", "trillibub", "trilletto", "trillando", "trilithon", "trilithic", "trilaurin", "triketone", "trikerion", "trigemini", "trigatron", "triennias", "trieennia", "triedness", "triecious", "tridymite", "tridrachm", "tridermic", "tridental", "tridecoic", "tridecene", "tridecane", "tridactyl", "tricycler", "tricycled", "tricuspal", "trierarch", "trierucin", "trigamous", "trigamist", "trifurcal", "trifornia", "triformin", "triformed", "triforium", "triforial", "trifocals", "trifloral", "triflings", "trifledom", "triferous", "trifacial", "trieteric", "trictracs", "triangular", "triggering", "triumphant", "tripartite", "trigeminal", "trilateral", "triplicate", "trilingual", "trimesters", "tricksters", "triviality", "trihydrate", "trivialize", "triumphing", "trinocular", "tricolored", "tridentine", "trinitrate", "triangulum", "tripeptide", "tricalcium", "trigonella", "triggerman", "triskelion", "tricholoma", "trihydroxy", "triborough", "trichechus", "trillionth", "trifoliate", "tripsomely", "triplefold", "tripterous", "trippingly", "triphysite", "tripinnate", "triplasian", "tripleback", "tripolitan", "triplicist", "tripleness", "tripliform", "triplexity", "triploidic", "triplewise", "triplumbic", "tripletree", "tripletail", "tripodical", "tripointed", "triplicity", "triphylite", "triodontes", "trinucleus", "trinacrian", "trinoctile", "trinoctial", "trinketing", "trinkermen", "trinkerman", "trinitride", "trinervate", "trimscript", "trimotored", "trimorphic", "trioecious", "triolefine", "triphyline", "triphthong", "triphibian", "triphammer", "tripewoman", "tripestone", "tripennate", "tripaschal", "triozonide", "trioxazine", "triovulate", "triorchism", "trionychid", "trimonthly", "trivirgate", "triturated", "triturable", "trittichan", "tritozooid", "tritonymph", "tritonidae", "tritoconid", "triticeous", "tritically", "trithrinax", "trithionic", "triterpene", "triternate", "tritanopic", "tritanopia", "triturates", "triturator", "trivialist", "trivialism", "trivialise", "trivetwise", "triverbial", "trivariant", "trivalerin", "trivalency", "trivalence", "triunities", "triungulin", "triumviral", "trivoltine", "triumfetta", "tritylodon", "tritangent", "trisylabic", "triseptate", "trisensory", "trisectrix", "trisection", "trisecting", "triradiate", "triquinoyl", "triquinate", "triquetrum", "triquetric", "triquetral", "tripylaean", "tripunctal", "tripudiate", "tripudiary", "triseriate", "trisilicic", "trisulphid", "trisulfone", "trisulfide", "trisulfate", "trisulcate", "tristylous", "tristichic", "tristfully", "tristeness", "tristearin", "trisporous", "trispinose", "trispaston", "trismegist", "trisinuate", "tripudiant", "triconodon", "tricipital", "trichromic", "trichromat", "trichroism", "trichotomy", "trichopter", "trichopore", "trichoplax", "trichology", "trichogyne", "trichocyst", "trichlorid", "trichiurus", "trichiurid", "trickeries", "trickiness", "tricolette", "tricoccous", "tricoccose", "triclinium", "triclinial", "triclinate", "tricladida", "tricktrack", "tricksiest", "tricksical", "trickproof", "trickliest", "trickishly", "trickingly", "trichinous", "trichinoid", "tribasilar", "triaxonian", "triarthrus", "triarchies", "triarchate", "triapsidal", "trianthous", "triangulid", "triandrous", "triandrian", "triamylose", "triactinal", "triaconter", "triacontad", "tribesfolk", "triblastic", "trichinize", "trichinise", "trichiasis", "trichevron", "trichauxis", "tricentral", "tricennial", "tricaudate", "tricarpous", "tributyrin", "tributable", "tribromide", "tribrachic", "tribometer", "triacetate", "tricornute", "trimnesses", "trilineate", "trilaminar", "trilabiate", "trihydride", "trihemimer", "trihemeral", "trihedrons", "trigraphic", "trigrammic", "trigonitis", "trigonally", "trignesses", "triglyphic", "triglyphed", "trilinguar", "triliteral", "trimmingly", "trimethoxy", "trimestral", "trimesitic", "trimesinic", "trimensual", "trimembral", "trimacular", "triluminar", "trilogical", "trilocular", "trilobitic", "trilobated", "trillachan", "triglyphal", "triglochin", "trielaidin", "tridiurnal", "tridepside", "tridentate", "tridecylic", "tricyclist", "tricycling", "tricyclene", "tricyanide", "tricussate", "tricurvate", "tricrotous", "tricrotism", "tricosylic", "trienniums", "trientalis", "triglochid", "trigesimal", "trigeneric", "trigeminus", "trifurcate", "triformous", "triformity", "triflorous", "triflorate", "triflingly", "trifarious", "trifanious", "trieterics", "trierarchy", "tricostate", "tributaries", "tribulation", "triumvirate", "trinitarian", "triceratops", "tridiagonal", "trichomonas", "trichoderma", "trichoptera", "trinidadian", "triggerfish", "trichloride", "triangulate", "trichinella", "trifluralin", "trichinosis", "trifluoride", "tribunitian", "tristimulus", "trinovantes", "triphyletic", "triphyllous", "tripinnated", "trinorantum", "trinopticon", "trinucleate", "tripemonger", "tripersonal", "tripartient", "tripartible", "tripartedly", "tripalmitin", "tripetaloid", "trionychoid", "tripetalous", "triphibious", "triodontoid", "triplicated", "trinomially", "trinobantes", "trimetallic", "trimetalism", "trimestrial", "trimercuric", "trimellitic", "trimargarin", "trimaculate", "triluminous", "trilophodon", "triloculate", "trimetrical", "trimetrogon", "trimodality", "trinklement", "trinketries", "trinitytide", "trinityhood", "trinational", "trimyristin", "trimuscular", "trimscripts", "trimorphous", "trimorphism", "trilobation", "triweeklies", "tritogeneia", "triticality", "trithionate", "tritheistic", "tritemorion", "tritanoptic", "tritanopsia", "tritagonist", "trisyllable", "trisyllabic", "tritonality", "triturating", "trituration", "trivialness", "trivialised", "trivalvular", "triuridales", "triumphwise", "triumphator", "triumphancy", "triumphance", "triturature", "triturators", "trisulphone", "trisulphide", "trisulphate", "trisections", "trisceptral", "triradiuses", "triradiated", "triradially", "triquetrous", "tripyrenous", "tripylarian", "tripunctate", "triploidite", "trisepalous", "triserially", "triseriatim", "trisulfoxid", "trisulcated", "tristiloquy", "tristichous", "tristearate", "trispermous", "trisotropis", "trisinuated", "trisilicate", "trisilicane", "triplicates", "trillionths", "trichophyte", "trichinotic", "trichinoses", "trichinosed", "trichinized", "trichinised", "trichechine", "tricephalus", "tricephalic", "tricenarium", "tricenaries", "trichiuroid", "trichlorfon", "trichoblast", "trichophore", "trichopathy", "trichonotid", "trichonosus", "trichonosis", "trichomonal", "trichomonad", "trichomanes", "tricholaena", "trichogynic", "tricellular", "tricarinate", "tribadistic", "triaxiality", "triarcuated", "triantelope", "triannulate", "trianguloid", "triammonium", "triadically", "triacontane", "triachenium", "tribasicity", "tribeswoman", "tribeswomen", "tricapsular", "tributorian", "tributarily", "tribunitive", "tribunitial", "tribunician", "tribunicial", "tribuneship", "tribrachial", "tribologist", "triableness", "trillionize", "triglyceryl", "triggerless", "trigeminous", "trifurcated", "trifoliosis", "trifoliated", "trierarchic", "trierarchal", "triennially", "trieciously", "trigonellin", "trigoneutic", "trigoniidae", "trilliaceae", "trilinolate", "trilineated", "trilaminate", "trilamellar", "trijunction", "trihydrated", "trihemiobol", "trigonotype", "trigonodont", "tridynamous", "tridominium", "tricliniary", "triclclinia", "trickstress", "tricksiness", "tricklingly", "tricircular", "trichronous", "trichromate", "trichotomic", "trichostema", "tricolumnar", "tricompound", "triconodont", "tridigitate", "tridiapason", "tridentlike", "tridentated", "tridecylene", "tridacnidae", "tricuspidal", "tricosanone", "tricorporal", "tricornered", "trichorrhea", "trigonometry", "triphosphate", "tribulations", "triglyceride", "triumphantly", "triangulated", "trivialities", "tribological", "trichophyton", "trivializing", "trichogramma", "triangularis", "trinitarians", "tribespeople", "trimolecular", "trimyristate", "trinitration", "trigrammatic", "trinomialism", "trinomialist", "trinomiality", "triodontidae", "trioeciously", "trimaculated", "tripaleolate", "tripalmitate", "tripartitely", "trigonometer", "tripartition", "trimethylene", "trimetallism", "trihemimeris", "trilarcenous", "trilaterally", "trihemimeral", "trilingually", "trilinoleate", "trilinolenin", "triliterally", "trilliaceous", "trillionaire", "trilophodont", "triguttulate", "trimargarate", "trimastigate", "trimeresurus", "trimesitinic", "trionychidae", "triphthongal", "trisulphonic", "trisulphoxid", "trisyllabism", "trisyllabity", "triternately", "triterpenoid", "tritheocracy", "trithionates", "triticalness", "tritonymphal", "tritopatores", "trituberculy", "triweekliess", "triumvirates", "triumvirship", "triunitarian", "triuridaceae", "trisulfoxide", "tristisonous", "tripinnately", "triplicately", "triplicating", "triplication", "triplicative", "triplicature", "triplicities", "triplinerved", "tripotassium", "trippingness", "tripudiation", "triradiately", "triradiation", "trismegistic", "tristachyous", "tristfulness", "tristigmatic", "trivialising", "trigoniacean", "triacetamide", "trichinizing", "trichinopoli", "trichinopoly", "trichiuridae", "trichobezoar", "trichoclasia", "trichoclasis", "trichocystic", "trichogenous", "trichogynial", "trichologist", "trichomatose", "trichomatous", "trichopathic", "trichophobia", "trichophoric", "trichophytia", "trichinising", "trichiniasis", "trichechidae", "triadelphous", "triamorphous", "triangleways", "trianglewise", "trianglework", "triangularly", "triangulates", "triangulator", "triatomicity", "tribophysics", "tribracteate", "tribunitiary", "tricarbimide", "tricarinated", "tricenarious", "tricentenary", "tricephalous", "trichophytic", "trichopteran", "tridactylous", "tridentinian", "tridiametral", "trienniality", "trierarchies", "trifasciated", "trifistulary", "triflingness", "trifluouride", "trifoliolate", "trifoveolate", "trifurcating", "trifurcation", "triglandular", "triglyphical", "trigonelline", "trigoneutism", "tricuspidate", "tricoryphean", "tricorporate", "trichopteron", "trichosporum", "trichostasis", "trichotomies", "trichotomism", "trichotomist", "trichotomize", "trichotomous", "trichromatic", "trichuriases", "trichuriasis", "trickishness", "trickstering", "tricliniarch", "triconodonta", "triconodonty", "tricophorous", "trigoniaceae", "triglycerides", "triangulation", "trigonometric", "triamcinolone", "trinucleotide", "triethylamine", "triangulating", "tricarboxylic", "trichodesmium", "tricentennial", "trilateration", "trinitroxylol", "trimethadione", "triodontoidea", "triodontoidei", "trioperculate", "triorthogonal", "trimucronatus", "trihypostatic", "trimerization", "triliterality", "trilamellated", "triliteralism", "trilaterality", "trilinolenate", "trilingualism", "tripersonally", "triphenylated", "tripinnatifid", "tristichaceae", "tristigmatose", "trisulphoxide", "trisyllabical", "tritangential", "tritheistical", "tritocerebral", "tritocerebrum", "trisplanchnic", "trisaccharose", "trisaccharide", "triplications", "triplicostate", "triploblastic", "triplocaulous", "triquadrantal", "triquetrously", "trirhomboidal", "triricinolein", "tritubercular", "trigrammatism", "trigonometria", "tricarpellate", "triceratopses", "trichatrophia", "trichechodont", "trichinoscope", "trichinoscopy", "trichocarpous", "trichoglossia", "tricarpellary", "tributariness", "triangulately", "triarticulate", "triatomically", "tribesmanship", "triboelectric", "tribonemaceae", "tribromacetic", "tribromphenol", "trichological", "trichomaphyte", "trichromatist", "triconodontid", "tricuspidated", "triflagellate", "triggerfishes", "trigintennial", "trigoniaceous", "trigonocerous", "trichromatism", "trichothallic", "trichomatosis", "trichomonadal", "trichomycosis", "trichopterous", "trichorrhexic", "trichorrhexis", "trichosanthes", "trichoschisis", "triangularity", "triangulations", "trichomoniasis", "trivialization", "trimethylamine", "tripelennamine", "trionychoidean", "trinitroxylene", "trinitrotoluol", "trinitrophenol", "trinitrocresol", "trinitarianism", "triliteralness", "trilateralness", "triiodomethane", "trihemiobolion", "trigonometries", "tripersonalism", "tripersonalist", "trivialisation", "triunsaturated", "triunification", "trituberculism", "trituberculata", "tritriacontane", "tritencephalon", "trisubstituted", "trisoctahedron", "trisoctahedral", "trirectangular", "tripinnatisect", "triphenylamine", "tripersonality", "trigonocephaly", "tridimensioned", "trichobranchia", "trichobacteria", "trichinophobia", "trichinization", "trichinisation", "trichiniferous", "tricentennials", "tricentenarian", "tricarballylic", "tribromphenate", "tribromophenol", "tribracteolate", "triantaphyllos", "trichocephalus", "trichodontidae", "trichoglossine", "tridimensional", "tridentiferous", "tridecilateral", "triconsonantal", "triconodontoid", "triclinohedric", "trichotomously", "trichosporange", "trichoschistic", "trichopterygid", "trichophytosis", "trichophyllous", "trichomonacide", "triacetonamine", "trichloroethane", "triethanolamine", "trifluoperazine", "trichloroacetic", "trinitrotoluene", "trinitromethane", "trinitrobenzene", "trinitroaniline", "trimethylacetic", "triacontaeterid", "triodontophorus", "trioxymethylene", "triphenylmethyl", "trirhombohedral", "tristetrahedron", "trisubstitution", "trisyllabically", "trithioaldehyde", "trigonometrical", "trigonocephalus", "tribromoethanol", "trichlormethane", "trichloromethyl", "trichoglossidae", "trichoglossinae", "trichomonacidal", "trichomonadidae", "trichoschistism", "trichostrongyle", "trichromatopsia", "tricotyledonous", "triethylstibine", "trigonocephalic", "trithiocarbonic", "triiodothyronine", "trichotillomania", "trimethylbenzene", "trimethylglycine", "trimethylmethane", "trimethylstibine", "trinitrocarbolic", "trinitroglycerin", "trinitroresorcin", "triphenylmethane", "triplocaulescent", "triskaidekaphobe", "tritetartemorion", "triboelectricity", "trigonometrician", "trigonocephalous", "tribofluorescent", "triboluminescent", "trichlorethylene", "trichloromethane", "trichobranchiate", "trichopterygidae", "trichosporangial", "trichosporangium", "trichostrongylid", "trichostrongylus", "tridimensionally", "trithiocarbonate", "trichloroethylene", "triakisoctahedral", "tridimensionality", "trigonometrically", "trinitrocellulose", "trionychoideachid", "triphenylcarbinol", "triplochitonaceae", "trisacramentarian", "triskaidekaphobes", "triconsonantalism", "trichopathophobia", "trichogrammatidae", "triakisoctahedrid", "triakisoctahedron", "tribofluorescence", "triboluminescence", "trichlorethylenes", "trichloromethanes", "trichocephaliasis", "trichoepithelioma", "triskaidekaphobia", "triphenylphosphine", "triakisicosahedral", "triakisicosahedron", "triakistetrahedral", "triakistetrahedron", "triangulopyramidal"}

	// The whole list would produce an insanely big SVG that chromium browsers can't handle (500% zoom is not enough), so focus on words with prefix "trip"
	words := make([]string, 0, 1000)
	words = append(words, "")
	for _, w := range allWords {
		if strings.HasPrefix(w, "trip") {
			words = append(words, w)
		}
	}

	prefixMap := map[string]int{}
	parentMap := make([]int, len(words))
	for idx, w := range words {
		prefixMap[w] = idx
	}
	for idx, w := range words {
		if len(w) <= len(rootWord) {
			continue
		}
		prefixLen := len(w) - 1
		for {
			prefixIdx, ok := prefixMap[w[:prefixLen]]
			if ok {
				parentMap[idx] = prefixIdx
				break
			}
			prefixLen--
		}
	}
	nodeDefs := make([]NodeDef, len(words))
	for idx, w := range words {
		prefix := words[parentMap[idx]]
		nodeDefs[idx] = NodeDef{int16(idx), w[len(prefix):], EdgeDef{int16(parentMap[idx]), "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""}
	}
	// Don't even try optimized, it will ask for fact(51)
	svg, err := Draw(context.TODO(), nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette(), DoNotOptimize)
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}

func TestReadmeMonochromeDiamond(t *testing.T) {
	var testNodeDefsDiamond = []NodeDef{
		{1, "1", EdgeDef{}, nil, "", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{3, "3", EdgeDef{1, "", TextColorDefault}, nil, "", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{4, "4", EdgeDef{1, "", TextColorDefault}, nil, "", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{5, "5", EdgeDef{3, "", TextColorDefault}, []EdgeDef{
			{4, "", TextColorDefault},
			{6, "", TextColorDefault}},
			"", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{6, "6", EdgeDef{}, nil, "", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}
	svg, err := Draw(context.TODO(),
		testNodeDefsDiamond,
		DefaultNodeFontOptions(),
		DefaultEdgeLabelFontOptions(),
		DefaultEdgeOptions(),
		"", "", nil, Optimize)
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}

func TestReadmeRootColors(t *testing.T) {
	const defsToAdd = `
<g id="icon-database-table-read">
	<g transform="scale(0.56) translate(2,61)">
		<path fill-rule="evenodd"
		d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
	</g>
	<g transform="scale(0.1) translate(540,20)">
		<path fill-rule="nonzero"
		d="M117.91 0h201.68c3.93 0 7.44 1.83 9.72 4.67l114.28 123.67c2.21 2.37 3.27 5.4 3.27 8.41l.06 310c0 35.43-29.4 64.81-64.8 64.81H117.91c-35.57 0-64.81-29.24-64.81-64.81V64.8C53.1 29.13 82.23 0 117.91 0zM325.5 37.15v52.94c2.4 31.34 23.57 42.99 52.93 43.5l36.16-.04-89.09-96.4zm96.5 121.3l-43.77-.04c-42.59-.68-74.12-21.97-77.54-66.54l-.09-66.95H117.91c-21.93 0-39.89 17.96-39.89 39.88v381.95c0 21.82 18.07 39.89 39.89 39.89h264.21c21.71 0 39.88-18.15 39.88-39.89v-288.3z" />
	</g>
</g>
<g id="icon-database-table-py">
	<g transform="scale(0.56) translate(2,61)">
		<path fill-rule="evenodd"
		d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
	</g>
	<g transform="scale(2.1) translate(24,0)">
		<path d="m9.8594 2.0009c-1.58 0-2.8594 1.2794-2.8594 2.8594v1.6797h4.2891c.39 0 .71094.57094.71094.96094h-7.1406c-1.58 0-2.8594 1.2794-2.8594 2.8594v3.7812c0 1.58 1.2794 2.8594 2.8594 2.8594h1.1797v-2.6797c0-1.58 1.2716-2.8594 2.8516-2.8594h5.25c1.58 0 2.8594-1.2716 2.8594-2.8516v-3.75c0-1.58-1.2794-2.8594-2.8594-2.8594zm-.71875 1.6094c.4 0 .71875.12094.71875.71094s-.31875.89062-.71875.89062c-.39 0-.71094-.30062-.71094-.89062s.32094-.71094.71094-.71094z"/>
		<path d="m17.959 7v2.6797c0 1.58-1.2696 2.8594-2.8496 2.8594h-5.25c-1.58 0-2.8594 1.2696-2.8594 2.8496v3.75a2.86 2.86 0 0 0 2.8594 2.8613h4.2812a2.86 2.86 0 0 0 2.8594 -2.8613v-1.6797h-4.291c-.39 0-.70898-.56898-.70898-.95898h7.1406a2.86 2.86 0 0 0 2.8594 -2.8613v-3.7793a2.86 2.86 0 0 0 -2.8594 -2.8594zm-9.6387 4.5137-.0039.0039c.01198-.0024.02507-.0016.03711-.0039zm6.5391 7.2754c.39 0 .71094.30062.71094.89062a.71 .71 0 0 1 -.71094 .70898c-.4 0-.71875-.11898-.71875-.70898s.31875-.89062.71875-.89062z"/>
	</g>
</g>
<g id="icon-database-table-join">
	<g transform="scale(0.56) translate(2,61)">
		<path fill-rule="evenodd"
		d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
	</g>
	<g transform="scale(0.1) translate(500,50)">
		<path fill-rule="nonzero"
		d="M303.633 363.721c10.832-11.26 28.745-11.608 40.007-.776 11.26 10.832 11.608 28.745.776 40.006l-91.965 95.954c-5.07 7.72-13.8 12.824-23.727 12.824-1.387 0-2.75-.1-4.079-.296l-.239-.033a28.149 28.149 0 01-15.859-7.649l-96.64-100.8c-10.832-11.261-10.484-29.174.777-40.006s29.174-10.484 40.006.776l47.665 49.733V258.99c0-50.724-20.558-101.577-53.822-139.863-31.152-35.856-73.279-60.35-119.571-62.576C11.355 55.817-.702 42.569.032 26.962.766 11.355 14.014-.702 29.621.032c62.738 3.021 118.895 35.136 159.687 82.086 15.498 17.837 28.798 37.876 39.416 59.302 10.579-21.35 23.828-41.327 39.253-59.121C308.656 35.368 364.703 3.22 427.379.051c15.607-.734 28.855 11.323 29.589 26.93.734 15.607-11.323 28.855-26.93 29.589-46.168 2.335-88.19 26.868-119.285 62.738-33.168 38.253-53.66 89.029-53.66 139.682v153.292l46.54-48.561z" />
	</g>
</g>
`

	nodeFontOptions := FontOptions{
		Typeface:     FontTypefaceCourier,
		Weight:       FontWeightNormal,
		SizeInPixels: 20,
		Interval:     0.3}
	edgeLabelFontOptions := FontOptions{
		Typeface:     FontTypefaceVerdana,
		Weight:       FontWeightNormal,
		SizeInPixels: 10,
		Interval:     0.3}
	edgeOptions := EdgeOptions{StrokeWidth: 2.0}
	cssOverrides := `
.text-node {font-family:Verdana; font-size:16px; fill:gray;}
`
	rootNodePalette := []int32{
		0x023EFF, 0xFF7C00, 0x1AC938, 0xE8000B, 0x8B2BE2, 0x9F4800, 0xF14CC1, 0xA3A3A3, 0xFFC400, 0x00D7FF} // blue, orange, etc.

	var testDiagramWithOneEnclosedLevel = []NodeDef{
		{1, "1 - Read\nprimary data\nfrom file",
			EdgeDef{},
			nil, "icon-database-table-read", 0,
			NodeBorderRegular, TextColorAsContainer, NodeBackgroundSolid, ""},
		{2, "2 - Apply\n one set of Python formulas\nto primary data",
			EdgeDef{1, "Data from file", TextColorAsContainer},
			nil, "icon-database-table-py", 0x001080,
			NodeBorderThick, TextColorAsContainer, NodeBackgroundSolid, ""},
		{3, "3 - Apply\n another set of Python\nformulas to primary data",
			EdgeDef{1, "Other data from file", TextColorDefault},
			nil, "icon-database-table-py", 0,
			NodeBorderThick, TextColorAsContainer, NodeBackgroundSolid, ""},
		{4, "4 - Join\n primary and\nsecondary data",
			EdgeDef{2, "", TextColorDefault},
			[]EdgeDef{{6, "Data to join", TextColorAsContainer}}, "icon-database-table-join", 0x001080,
			NodeBorderRegular, TextColorAsContainer, NodeBackgroundSolid, ""},
		{5, "5 - Join\n primary and\nsecondary data",
			EdgeDef{3, "", TextColorDefault},
			[]EdgeDef{{6, "Data to join", TextColorDefault}}, "icon-database-table-join", 0,
			NodeBorderRegular, TextColorAsContainer, NodeBackgroundSolid, ""},
		{6, "6 - Read\n secondary data\nfrom file",
			EdgeDef{},
			nil, "icon-database-table-read", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}

	svg, err := Draw(context.TODO(),
		testDiagramWithOneEnclosedLevel,
		nodeFontOptions,
		edgeLabelFontOptions,
		edgeOptions,
		defsToAdd,
		cssOverrides,
		rootNodePalette, Optimize)
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}

func TestReadmeCustomBackground(t *testing.T) {
	const defsToAdd = `
<pattern id="diagonalBlueLines" patternUnits="userSpaceOnUse" width="10" height="10">
	<line x1="10" y1="0" x2="20" y2="10" stroke="blue" opacity="0.3" stroke-width="2" stroke-linecap="square">
	<animateTransform attributeType="xml" attributeName="transform" type="translate" from="0 0" to="10 0" begin="0" dur="1" repeatCount="indefinite"/>
	</line>
	<line x1="0" y1="0" x2="10" y2="10" stroke="blue" opacity="0.3" stroke-width="2" stroke-linecap="square">
	<animateTransform attributeType="xml" attributeName="transform" type="translate" from="0 0" to="10 0" begin="0" dur="1" repeatCount="indefinite"/>
	</line>
	<line x1="-10" y1="0" x2="0" y2="10" stroke="blue" opacity="0.3" stroke-width="2" stroke-linecap="square">
	<animateTransform attributeType="xml" attributeName="transform" type="translate" from="0 0" to="10 0" begin="0" dur="1" repeatCount="indefinite"/>
	</line>
	<line x1="-20" y1="0" x2="-10" y2="10" stroke="blue" opacity="0.3" stroke-width="2" stroke-linecap="square">
	<animateTransform attributeType="xml" attributeName="transform" type="translate" from="0 0" to="10 0" begin="0" dur="1" repeatCount="indefinite"/>
	</line>
</pattern>
<pattern id="topProgressBar" width="1" height="1" patternUnits="objectBoundingBox" patternContentUnits="objectBoundingBox">
	<rect x="0" y="0" rx="0" ry="0" width="1" height="1" fill="blue" opacity="0.3"/>
	<rect x="0.1" y="0.1" rx="0" ry="0" width=".8" height=".1" fill="#f2f2f2" opacity="1"/>
	<rect x="0.1" y="0.1" rx="0" ry="0" width=".8" height=".1" fill="#2589d0" opacity="1">
		<animate attributeName="x"
			values="0.1;0.1;0.3;.9"
			keyTimes="0;.4;.8;1"
			keySplines="0 0 1 1;.3 .1 .8 1;.1 .1 .6 1"
			dur="2s" repeatCount="indefinite" calcMode="spline"/>
		<animate attributeName="width"
			values="0;.6;.6;0"
			keyTimes="0;.4;.8;1"
			keySplines="0 0 1 1;.3 .1 .8 1;.1 .1 .6 1"
			dur="2s" repeatCount="indefinite" calcMode="spline"/>
	</rect>
</pattern>
<radialGradient id="redGradient" cx="50%" cy="50%" r="70%">
	<stop offset="0%" stop-color="red">
	<animate attributeName="stop-color" values="#ec0000;#ecca00;#ec0000" dur="1s" repeatCount="indefinite" />
	<animate attributeName="offset" values="0%;50%;0%" dur="1s" repeatCount="indefinite" />
	</stop>
	<stop offset="100%" stop-color="#ecca00"></stop>
</radialGradient>
`
	cssOverrides := `
.diagonal-progress-background {fill:url(#diagonalBlueLines)}
.top-progress-background {fill:url(#topProgressBar)}
.failed-background {fill:url(#redGradient)}
`

	var testNodeDefsOneSecondary = []NodeDef{
		{1, "1 - Complete",
			EdgeDef{},
			nil, "", 0x386641,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "2 - Running",
			EdgeDef{1, "Some\ndata", TextColorDefault},
			nil, "", 0x0000FF,
			NodeBorderRegular, TextColorDefault, NodeBackgroundPattern, "diagonal-progress-background"},
		{3, "3 - Complete",
			EdgeDef{},
			nil, "", 0x386641,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{4, "4 - Running",
			EdgeDef{3, "Some\nother\ndata", TextColorDefault},
			nil, "", 0x0000FF,
			NodeBorderRegular, TextColorDefault, NodeBackgroundPattern, "top-progress-background"},
		{5, "5 - Not started",
			EdgeDef{4, "Some primary data", TextColorDefault},
			[]EdgeDef{{6, "Some data to join", TextColorDefault}}, "", 0,
			NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{6, "6 - Failed",
			EdgeDef{},
			nil, "", 0xEC0000,
			NodeBorderRegular, TextColorDefault, NodeBackgroundPattern, "failed-background"},
	}
	svg, err := Draw(context.TODO(),
		testNodeDefsOneSecondary,
		DefaultNodeFontOptions(),
		DefaultEdgeLabelFontOptions(),
		DefaultEdgeOptions(),
		defsToAdd,
		cssOverrides,
		nil,
		Optimize)
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}
