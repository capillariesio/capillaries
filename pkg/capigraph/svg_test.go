package capigraph

import (
	"fmt"
	"testing"

	"github.com/pkg/profile"
)

func TestEnclosingOneLevel(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, ""},
		{1, "A1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
		{2, "A21\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A21"}, []EdgeDef{}, ""},
		{3, "A22\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{1, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A1 to A22"}, []EdgeDef{}, ""},
		{4, "A31\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{2, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A21 to A31"}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A31"}}, ""},
		{5, "A32\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{3, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom A22 to A32"}, []EdgeDef{{6, "Lorem\n\n\nipsum\ndolor\nsit\namet\nfrom B1 to A32"}}, ""},
		{6, "B1\nlorem ipsum dolor sit amet,\nconsectetur adipisci elit,\nsed eiusmod tempor incidunt\nut labore\net dolore magna aliqua", EdgeDef{}, []EdgeDef{}, ""},
	}

	nodeFo := FontOptions{FontTypefaceCourier, FontWeightNormal, 20}
	edgeFo := FontOptions{FontTypefaceArial, FontWeightNormal, 18}
	eo := EdgeOptions{2.0}
	vizNodeMap, _ := GetBestHierarchy(nodeDefs, &nodeFo, &edgeFo)
	svg := draw(vizNodeMap, nodeFo, edgeFo, eo, CapillariesIcons100x100)
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
	nodeFo := FontOptions{FontTypefaceCourier, FontWeightNormal, 20}
	edgeFo := FontOptions{FontTypefaceArial, FontWeightNormal, 18}
	eo := EdgeOptions{2.0}
	vizNodeMap, _ := GetBestHierarchy(nodeDefs, &nodeFo, &edgeFo)
	svg := draw(vizNodeMap, nodeFo, edgeFo, eo, CapillariesIcons100x100)
	fmt.Printf("%s\n", svg)
}

func TestLarge1(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, []EdgeDef{}, ""},
		{1, "0-5", EdgeDef{}, []EdgeDef{}, ""},
		//{"7-2", "7-2", EdgeDef{}, []EdgeDef{}, ""},
		{2, "0-0", EdgeDef{}, []EdgeDef{}, ""},
		{3, "0-1", EdgeDef{}, []EdgeDef{}, ""},
		{4, "0-2", EdgeDef{}, []EdgeDef{}, ""},
		//{"4-0", "4-0", EdgeDef{}, []EdgeDef{}, ""},

		{5, "5-0", EdgeDef{}, []EdgeDef{}, ""},
		{6, "6-0", EdgeDef{3, ""}, []EdgeDef{}, ""},
		//{"0-3", "0-3", EdgeDef{}, []EdgeDef{}, ""},
		{7, "0-4", EdgeDef{}, []EdgeDef{}, ""},
		{8, "1-0", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{9, "1-1", EdgeDef{4, ""}, []EdgeDef{}, ""},
		//{"2-3", "2-3", EdgeDef{"0-0", ""}, []EdgeDef{}, ""},

		{10, "4-1", EdgeDef{}, []EdgeDef{}, ""},
		{11, "4-2", EdgeDef{8, ""}, []EdgeDef{}, ""},
		{12, "5-1", EdgeDef{7, ""}, []EdgeDef{}, ""},
		{13, "7-0", EdgeDef{5, ""}, []EdgeDef{}, ""},
		{14, "2-0", EdgeDef{8, ""}, []EdgeDef{}, ""},
		{15, "2-1", EdgeDef{8, ""}, []EdgeDef{}, ""},
		{16, "2-2", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{17, "5-2", EdgeDef{10, ""}, []EdgeDef{}, ""},

		{18, "5-3", EdgeDef{10, ""}, []EdgeDef{}, ""},

		{19, "6-1", EdgeDef{17, ""}, []EdgeDef{}, ""},
		{20, "7-1", EdgeDef{17, ""}, []EdgeDef{}, ""},
	}
	nodeFo := FontOptions{FontTypefaceCourier, FontWeightNormal, 20}
	edgeFo := FontOptions{FontTypefaceArial, FontWeightNormal, 18}
	eo := EdgeOptions{2.0}
	vizNodeMap, _ := GetBestHierarchy(nodeDefs, &nodeFo, &edgeFo)
	svg := draw(vizNodeMap, nodeFo, edgeFo, eo, CapillariesIcons100x100)
	fmt.Printf("%s\n", svg)
}

func TestLarge2(t *testing.T) {
	defer profile.Start(profile.CPUProfile).Stop()
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
	nodeFo := FontOptions{FontTypefaceCourier, FontWeightNormal, 20}
	edgeFo := FontOptions{FontTypefaceArial, FontWeightNormal, 18}
	eo := EdgeOptions{2.0}
	vizNodeMap, _ := GetBestHierarchy(nodeDefs, &nodeFo, &edgeFo)
	svg := draw(vizNodeMap, nodeFo, edgeFo, eo, CapillariesIcons100x100)
	fmt.Printf("%s\n", svg)
}
