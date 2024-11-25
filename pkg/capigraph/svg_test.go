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
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsBasic, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 47.2, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TrivialParallelSvg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsTrivialParallel, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 47.2, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestOneEnclosingOneLevelSvg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsOneEnclosedOneLevel, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestOneEnclosedTwoLevelsSvg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsOneEnclosedTwoLevels, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestNoIntervalsSvg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsNoIntervals, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(2), totalPermutations)
	assert.Equal(t, 0.0, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestFlat10Svg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsFlat10, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(3628800), totalPermutations)
	assert.Equal(t, 0.0, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestTwoEnclosingTwoLevelsNodeSizeMattersSvg(t *testing.T) {
	// Only one of 8, 9 is enclosed
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsTwoEnclosedNodeSizeMatters, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(24), totalPermutations)
	assert.Equal(t, 302.4, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)

	// Now make nodes 4 and 5 considerably wider - it will change the best hierarchy, 8 and 9 enclosed
	testNodeDefsTwoEnclosedNodeSizeMatters[4].Text += " wider"
	testNodeDefsTwoEnclosedNodeSizeMatters[5].Text += " wider"

	vizNodeMap, totalPermutations, elapsed, bestDist, _ = GetBestHierarchy(testNodeDefsTwoEnclosedNodeSizeMatters, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, 456.0, math.Round(bestDist*100.0)/100.0)
	svg = strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestOneSecondarySvg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsOneSecondary, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 47.2, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestDiamonSvg(t *testing.T) {
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefsDiamond, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(24), totalPermutations)
	assert.Equal(t, 104.0, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

// Takes 15 seconds, disable for quick testing
/*
func Test40milPermsSvg(t *testing.T) {
	defer profile.Start(profile.CPUProfile).Stop()
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(testNodeDefs40milPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(41472000), totalPermutations)
	assert.Equal(t, 57.40, math.Round(bestDist*100.0)/100.0)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}
*/

func Test300bilPermsSvg(t *testing.T) {
	_, _, _, _, err := GetBestHierarchy(testNodeDefs300bilPerms, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Contains(t, err.Error(), "313528320000")
}

// SVG-specific tests

func TestEnclosingOneLevelWideNodes(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, ""},
		{1, "A1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
		{2, "A21\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A21"}, []EdgeDef{}, ""},
		{3, "A22\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A22"}, []EdgeDef{}, ""},
		{4, "A31\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A21 to A31"}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A31"}}, ""},
		{5, "A32\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{3, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A22 to A32"}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A32"}}, ""},
		{6, "B1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
	}

	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(6), totalPermutations)
	assert.Equal(t, 728.0, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestHalfComplexWithEnclosed(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, ""},
		{1, "A1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
		{2, "A2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A2"}, []EdgeDef{}, ""},
		{3, "A31\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A2 to A31"}, []EdgeDef{{10, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B2 to A31"}}, ""},
		{4, "A32\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A2 to A32"}, []EdgeDef{{14, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C3 to A32"}}, ""},
		{5, "A41\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{4, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A32 to A41"}, []EdgeDef{{11, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B3 to A41"}}, ""},
		{6, "A42\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{4, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A32 to A42"}, []EdgeDef{{14, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C3 to A42"}}, ""},
		{7, "A51\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{5, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A41 to A51"}, []EdgeDef{{15, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom D1 to A51"}}, ""},
		{8, "A52\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A42 to A52"}, []EdgeDef{{15, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom D1 to A52"}}, ""},
		{9, "B1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
		{10, "B2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{9, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to B2"}, []EdgeDef{}, ""},
		{11, "B3\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{10, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B2 to B3"}, []EdgeDef{}, ""},
		{12, "C1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
		{13, "C2\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{12, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C1 to C2"}, []EdgeDef{}, ""},
		{14, "C3\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{13, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom C2 to C3"}, []EdgeDef{}, ""},
		{15, "D1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
	}
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(54), totalPermutations)
	assert.Equal(t, 4368.0, math.Round(bestDist))
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestConflictingSecAndTotalViewboxWidthAdjustedToLabel(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, ""},
		{1, "A", EdgeDef{}, []EdgeDef{}, ""},
		{2, "B", EdgeDef{}, []EdgeDef{}, ""},
		{3, "C", EdgeDef{1, "A to C"}, []EdgeDef{{2, "B to ? duplicate going really wide"}}, ""},
		{4, "D", EdgeDef{3, "C to D"}, []EdgeDef{{2, "B to ? duplicate going really wide"}}, ""},
	}
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), "", ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}

func TestCapillariesIcons(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{
			1,
			"01_read_payments\n" +
				"Read from files into a table\n" +
				"Files: /tmp/capi_in/.../CAS_2023_R08_G1_20231020_000.parquet\n" +
				"Table created: payments",
			EdgeDef{},
			[]EdgeDef{},
			"icon-database-table-read",
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
		},
		{
			8,
			"05_deal_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_summaries",
			EdgeDef{5, "deal_total_upbs\n(10 batches)"},
			[]EdgeDef{{7, "idx_loan_summaries_calculated_deal_name\n(lookup)"}},
			"icon-database-table-join",
		},
		{
			9,
			"05_deal_seller_summaries\n" +
				"Join with lookup table\n" +
				"Group: true, join: left\n" +
				"Table created: deal_seller_summaries",
			EdgeDef{4, "deal_sellers\n(10 batches)"},
			[]EdgeDef{},
			"icon-database-table-join",
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
		},
	}
	vizNodeMap, totalPermutations, elapsed, bestDist, _ := GetBestHierarchy(nodeDefs, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions())
	assert.Equal(t, int64(1296), totalPermutations)
	assert.Equal(t, 4114.2, bestDist)
	svg := strings.ReplaceAll(draw(vizNodeMap, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), CapillariesIcons100x100, ""), "</svg>", drawStatistics(totalPermutations, elapsed, bestDist)+"\n</svg>")
	fmt.Printf("%s\n", svg)
}
