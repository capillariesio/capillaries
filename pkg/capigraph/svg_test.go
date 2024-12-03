package capigraph

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Common MxPermutator and SVG tests

func TestBasicSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsBasic, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 52.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TrivialParallelSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsTrivialParallel, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 47.2, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestOneEnclosingOneLevelSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsOneEnclosedOneLevel, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestOneEnclosedTwoLevelsSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsOneEnclosedTwoLevels, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestNoIntervalsSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsNoIntervals, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 0.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestFlat10Svg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsFlat10, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(3628800), totalPermutations)
	assert.Equal(t, 0.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestTwoEnclosingTwoLevelsNodeSizeMattersSvg(t *testing.T) {
	// Only one of 8, 9 is enclosed
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsTwoEnclosedNodeSizeMatters, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(24), totalPermutations)
	assert.Equal(t, 312.0, bestDist)
	fmt.Printf("%s\n", svg)

	// Now make nodes 4 and 5 considerably wider - it will change the best hierarchy, 8 and 9 enclosed
	testNodeDefsTwoEnclosedNodeSizeMatters[4].Text += " wider"
	testNodeDefsTwoEnclosedNodeSizeMatters[5].Text += " wider"

	svg, _, totalPermutations, _, bestDist, _ = DrawOptimized(testNodeDefsTwoEnclosedNodeSizeMatters, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, 456.0, math.Round(bestDist*100.0)/100.0)
	fmt.Printf("%s\n", svg)
}

func TestOneSecondarySvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsOneSecondary, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 52.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestDiamonSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsDiamond, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(24), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestSubtreeBelowLongSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsSubtreeBelowLong, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 52.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestOneNotTwoLevelsDownSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsOneNotTwoLevelsDown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestMultiSecParentPullDownSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsMultiSecParentPullDown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestMultiSecParentNoPullDownSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsMultiSecParentNoPullDown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestTwoLevelsFromOneParentSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsTwoLevelsFromOneParent, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestTwoLevelsFromOneParentSameRootSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsTwoLevelsFromOneParentSameRoot, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 52.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestTwoLevelsFromOneParentSameRootTwoFakesSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsTwoLevelsFromOneParentSameRootTwoFakes, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 52.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestDuplicateSecLabelsSvg(t *testing.T) {
	svg, vizNodeMap, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsDuplicateSecLabels, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(720), totalPermutations)
	assert.Equal(t, 480.0, bestDist)
	// Zero-width sec labels to avoid interlapped
	assert.Equal(t, 0.0, vizNodeMap[8].IncomingVizEdges[1].W)
	assert.Equal(t, 0.0, vizNodeMap[9].IncomingVizEdges[1].W)
	// Displayed sec labels
	assert.Equal(t, 37.08, vizNodeMap[7].IncomingVizEdges[1].W)
	assert.Equal(t, 37.08, vizNodeMap[10].IncomingVizEdges[1].W)
	assert.Equal(t, 37.08, vizNodeMap[11].IncomingVizEdges[1].W)
	fmt.Printf("%s\n", svg)
}

func TestLayerLongRootsSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsLayerLongRoots, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 156.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestPriAndSecInfinitePulldownSvg(t *testing.T) {
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testNodeDefsPriAndSecInfinitePulldown, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(4), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	fmt.Printf("%s\n", svg)
}

// Takes 15 seconds, disable for quick testing
// func Test40milPermsSvg(t *testing.T) {
// 	defer profile.Start(profile.CPUProfile).Stop()
// 	svg, _, totalPermutations, _, bestDist, _ := Draw(testNodeDefs40milPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
// 	assert.Equal(t, int64(41472000), totalPermutations)
// 	assert.Equal(t, 64.0, math.Round(bestDist*100.0)/100.0)
// 	fmt.Printf("%s\n", svg)
// }

func Test300bilPermsSvg(t *testing.T) {
	_, _, _, _, _, err := DrawOptimized(testNodeDefs300bilPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Contains(t, err.Error(), "313528320000")
	svg, _, err := DrawUnoptimized(testNodeDefs300bilPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}

// SVG-specific tests

// Unoptimized only, too many permutations
func TestUnoptimizedSvg(t *testing.T) {
	nodeDefs := make([]NodeDef, 0, 10000)
	nodeDefs = append(nodeDefs, NodeDef{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0, false})
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
			newNode := NodeDef{int16(nextChildIdx), fmt.Sprintf("%d", nextChildIdx), EdgeDef{int16(parentOverrideIdx), ""}, []EdgeDef{}, "", 0, false}
			if parentOverrideIdx != 0 && firstChildIdx%9 == 0 {
				newNode.SecIn = append(newNode.SecIn, EdgeDef{int16(firstChildIdx / 2), ""})
			}
			nodeDefs = append(nodeDefs, newNode)
			nextChildIdx = populateChildren(nextChildIdx, nextChildIdx+1, layer+1)
		}
		return nextChildIdx
	}
	populateChildren(0, 1, 0)

	svg, _, err := DrawUnoptimized(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}

// Takes 160s to complete, but working. Fake nodes, enclosed subtrees, 20 levels, 484 nodes.
// func TestInsanelyBigBinaryTreeSvg(t *testing.T) {
// 	nodeDefs := make([]NodeDef, 0, 10000)
// 	nodeDefs = append(nodeDefs, NodeDef{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0, false})
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
// 			newNode := NodeDef{int16(nextChildIdx), fmt.Sprintf("%d", nextChildIdx), EdgeDef{int16(parentOverrideIdx), ""}, []EdgeDef{}, "", 0, false}
// 			if parentOverrideIdx != 0 && firstChildIdx%9 == 0 {
// 				newNode.SecIn = append(newNode.SecIn, EdgeDef{int16(firstChildIdx / 2), ""})
// 			}
// 			nodeDefs = append(nodeDefs, newNode)
// 			nextChildIdx = populateChildren(nextChildIdx, nextChildIdx+1, layer+1)
// 		}
// 		return nextChildIdx
// 	}
// 	populateChildren(0, 1, 0)

// 	svg, _, totalPermutations, _, _, err := DrawOptimized(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
// 	assert.Equal(t, int64(3819584), totalPermutations)
// 	assert.Equal(t, nil, err)
// 	fmt.Printf("%s\n", svg)
// }

func TestEnclosingOneLevelWideNodes(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{1, "A1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{2, "A21\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A21"}, []EdgeDef{}, "", 0, false},
		{3, "A22\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A22"}, []EdgeDef{}, "", 0, false},
		{4, "A31\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A21 to A31"}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A31"}}, "", 0, false},
		{5, "A32\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{3, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A22 to A32"}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A32"}}, "", 0, false},
		{6, "B1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, "", 0, false},
	}
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 728.0, bestDist)
	fmt.Printf("%s\n", svg)
}

func TestHalfComplexWithEnclosed(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{1, "A1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{2, "A2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A2"}, []EdgeDef{}, "", 0, false},
		{3, "A31\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A2 to A31"}, []EdgeDef{{10, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B2 to A31"}}, "", 0, false},
		{4, "A32\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A2 to A32"}, []EdgeDef{{14, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C3 to A32"}}, "", 0, false},
		{5, "A41\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{4, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A32 to A41"}, []EdgeDef{{11, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B3 to A41"}}, "", 0, false},
		{6, "A42\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{4, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A32 to A42"}, []EdgeDef{{14, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C3 to A42"}}, "", 0, false},
		{7, "A51\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{5, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A41 to A51"}, []EdgeDef{{15, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom D1 to A51"}}, "", 0, false},
		{8, "A52\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A42 to A52"}, []EdgeDef{{15, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom D1 to A52"}}, "", 0, false},
		{9, "B1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{10, "B2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{9, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to B2"}, []EdgeDef{}, "", 0, false},
		{11, "B3\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{10, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B2 to B3"}, []EdgeDef{}, "", 0, false},
		{12, "C1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{13, "C2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{12, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C1 to C2"}, []EdgeDef{}, "", 0, false},
		{14, "C3\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{13, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C2 to C3"}, []EdgeDef{}, "", 0, false},
		{15, "D1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, "", 0, false},
	}
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(48), totalPermutations)
	assert.Equal(t, 2912.0, math.Round(bestDist))
	fmt.Printf("%s\n", svg)
}

func TestConflictingSecAndTotalViewboxWidthAdjustedToLabel(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{1, "A", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{2, "B", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{3, "C", EdgeDef{1, "A to C"}, []EdgeDef{{2, "B to ? duplicate going really wide"}}, "", 0, false},
		{4, "D", EdgeDef{3, "C to D"}, []EdgeDef{{2, "B to ? duplicate going really wide"}}, "", 0, false},
	}
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 104.0, math.Round(bestDist*100.0)/100.0)
	fmt.Printf("%s\n", svg)
}

func TestCapillariesIcons(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0, false},
		{
			1,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			[]EdgeDef{},
			"icon-database-table-read",
			0,
			true,
		},
		{
			2,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{1, "payments\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			true,
		},
		{
			3,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{2, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			true,
		},
		{
			4,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{2, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			true,
		},
		{
			5,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{3, "deal_names\n(10 batches)"},
			[]EdgeDef{{2, "idx_loan_ids_deal_name\n(lookup)"}},
			"icon-database-table-join",
			0,
			true,
		},
		{
			6,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{2, "loan_ids\n(10 batches)"},
			[]EdgeDef{{1, "idx_payments_by_loan_id\n(lookup)"}},
			"icon-database-table-join",
			0,
			true,
		},
		{
			7,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{6, "loan_payment_summaries\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-py",
			0,
			true,
		},
		{
			8,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{5, "deal_total_upbs\n(10 batches)"},
			[]EdgeDef{{7, "idx_loan_summaries_calculated_deal_name\n(lookup)\ndeal_name"}},
			"icon-database-table-join",
			0,
			true,
		},
		{
			9,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{4, "deal_sellers\n(10 batches)"},
			[]EdgeDef{{7, "idx_loan_summaries_calculated_deal_name\n(lookup)\ndeal_name\nseller_name"}},
			"icon-database-table-join",
			0,
			true,
		},
		{
			10,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{7, "loan_summaries_calculated\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			true,
		},
		{
			11,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{8, "deal_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			true,
		},
		{
			12,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{9, "deal_seller_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			true,
		},

		{
			13,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			[]EdgeDef{},
			"icon-database-table-read",
			0,
			false,
		},
		{
			14,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{13, "payments\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			15,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{14, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			16,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{14, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			17,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{15, "deal_names\n(10 batches)"},
			[]EdgeDef{{14, "idx_loan_ids_deal_name\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			18,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{14, "loan_ids\n(10 batches)"},
			[]EdgeDef{{13, "idx_payments_by_loan_id\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			19,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{18, "loan_payment_summaries\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-py",
			0,
			false,
		},
		{
			20,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{17, "deal_total_upbs\n(10 batches)"},
			[]EdgeDef{{19, "idx_loan_summaries_calculated_deal_name\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			21,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{16, "deal_sellers\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-join",
			0,
			false,
		},
		{
			22,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{19, "loan_summaries_calculated\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},
		{
			23,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{20, "deal_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},
		{
			24,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{21, "deal_seller_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},

		{
			25,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			[]EdgeDef{},
			"icon-database-table-read",
			0,
			false,
		},
		{
			26,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{25, "payments\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			27,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{26, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			28,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{26, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			29,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{27, "deal_names\n(10 batches)"},
			[]EdgeDef{{26, "idx_loan_ids_deal_name\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			30,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{26, "loan_ids\n(10 batches)"},
			[]EdgeDef{{25, "idx_payments_by_loan_id\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			31,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{30, "loan_payment_summaries\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-py",
			0,
			false,
		},
		{
			32,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{29, "deal_total_upbs\n(10 batches)"},
			[]EdgeDef{{31, "idx_loan_summaries_calculated_deal_name\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			33,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{28, "deal_sellers\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-join",
			0,
			false,
		},
		{
			34,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{31, "loan_summaries_calculated\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},
		{
			35,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{32, "deal_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},
		{
			36,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{33, "deal_seller_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},

		{
			37,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			[]EdgeDef{},
			"icon-database-table-read",
			0,
			false,
		},
		{
			38,
			"02_loan_ids\n" +
				"Select distinct rows\n" +
				"Index used: unique(loan_id)\n" +
				"Table created: loan_ids",
			EdgeDef{37, "payments\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			39,
			"02_deal_names\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name)\n" +
				"Table created: deal_names",
			EdgeDef{38, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			40,
			"02_deal_sellers\n" +
				"Select distinct rows\n" +
				"Index used: unique(deal_name, seller_name)\n" +
				"Table created: deal_sellers",
			EdgeDef{38, "loan_ids\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-distinct",
			0,
			false,
		},
		{
			41,
			"03_deal_total_upbs\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_total_upbs",
			EdgeDef{39, "deal_names\n(10 batches)"},
			[]EdgeDef{{38, "idx_loan_ids_deal_name\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			42,
			"04_loan_payment_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: loan_payment_summaries",
			EdgeDef{38, "loan_ids\n(10 batches)"},
			[]EdgeDef{{37, "idx_payments_by_loan_id\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			43,
			"04_loan_summaries_calculated\n" +
				"Apply Python calculations\n" +
				"Group: true, join: left\n" +
				"Table created: loan_summaries_calculated",
			EdgeDef{42, "loan_payment_summaries\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-py",
			0,
			false,
		},
		{
			44,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{41, "deal_total_upbs\n(10 batches)"},
			[]EdgeDef{{43, "idx_loan_summaries_calculated_deal_name\n(lookup)"}},
			"icon-database-table-join",
			0,
			false,
		},
		{
			45,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{40, "deal_sellers\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-join",
			0,
			false,
		},
		{
			46,
			"04_write_file_loan_summaries_calculated\n" +
				"Write from table to files\n" +
				"Table: loan_summaries_calculated\n" +
				"Files: /tmp/.../.../loan_summaries_calculated.parquet",
			EdgeDef{43, "loan_summaries_calculated\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},
		{
			47,
			"05_write_file_deal_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_summaries\n" +
				"Files: /tmp/.../.../deal_summaries.parquet",
			EdgeDef{44, "deal_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},
		{
			48,
			"05_write_file_deal_seller_summaries\n" +
				"Write from table to files\n" +
				"Table: deal_seller_summaries\n" +
				"Files: /tmp/.../.../deal_seller_summaries.parquet",
			EdgeDef{45, "deal_seller_summaries\n(no parallelism)"},
			[]EdgeDef{},
			"icon-parquet",
			0,
			false,
		},
	}
	// overrideCss := ".rect-node-background {rx:20; ry:20;} .rect-node {rx:20; ry:20;} .capigraph-rendering-stats {fill:black;}"
	// nodeColorMap := []int32{0x010101, 0x0000FF, 0x008000, 0xFF0000, 0xFF8C00, 0x2F4F4F} //none, blue, darkgreen, red, darkorange, darkslategray (none, start, success, fail, stopreceived, unknown)
	// for nodeIdx := range nodeDefs {
	// 	nodeDefs[nodeIdx].Color = nodeColorMap[nodeIdx%len(nodeColorMap)]
	// }
	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), CapillariesIcons100x100, "" /* overrideCss*/, DefaultPalette())
	assert.Equal(t, int64(31104), totalPermutations)
	assert.Equal(t, 6438.0, math.Round(bestDist*100.0)/100.0)
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
		nodeDefs[idx] = NodeDef{int16(idx), w[len(prefix):], EdgeDef{int16(parentMap[idx]), ""}, []EdgeDef{}, "", 0, false}
	}
	// Don't even try optimized, it will ask for fact(51)
	svg, _, err := DrawUnoptimized(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", "", DefaultPalette())
	assert.Equal(t, nil, err)
	fmt.Printf("%s\n", svg)
}
