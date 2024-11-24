package capigraph

import (
	"fmt"
	"math"
)

type VizNodeHierarchyAnalyzer struct {
}

func GetBestHierarchy(nodeDefs []NodeDef, nodeFo *FontOptions, edgeFo *FontOptions) ([]VizNode, error) {
	priParentMap := buildPriParentMap(nodeDefs)
	layerMap := buildLayerMap(nodeDefs, priParentMap)
	rootNodes := buildRootNodeList(priParentMap)
	mx, err := NewLayerMx(nodeDefs, layerMap, rootNodes)
	if err != nil {
		return nil, err
	}

	mxi, err := NewLayerMxPermIterator(nodeDefs, mx)
	if err != nil {
		return nil, err
	}

	vnh := NewVizNodeHierarchy(nodeDefs, nodeFo, edgeFo)

	vnh.buildNewRootSubtreeHierarchy(mx)
	vnh.PopulateEdgeLabelDimensions()

	bestDistSec := math.MaxFloat64
	bestSignature := "z"
	var bestMx LayerMx
	mxPermCnt := 0
	mxi.MxIterator(func(i int, mxPerm LayerMx) {
		vnh.reuseRootSubtreeHierarchy(mxPerm)
		vnh.PopulateNodeDimensions()
		vnh.PopulateNodesXCoords()
		distSec := vnh.CalculateTotalHorizontalShift()
		//fmt.Printf("%d %.2f %s\n", mxPermCnt, distSec, mxPerm.String())
		if distSec <= bestDistSec {
			// This: 1. Adds determinism 2. helps user choose ids that go first (to some extent)
			signature := mxPerm.signature()
			if math.Abs(bestDistSec-distSec) > 0.01 || signature < bestSignature {
				bestDistSec = distSec
				bestMx = mxPerm.clone()
				bestSignature = signature
			}
		}
		mxPermCnt++
	})

	if bestMx == nil {
		return nil, fmt.Errorf("no best")
	}

	vnh.reuseRootSubtreeHierarchy(bestMx)
	vnh.PopulateNodeDimensions()
	vnh.PopulateNodesXCoords()
	vnh.PopulateUpperLayerGapMap(edgeFo.SizeInPixels, vnh.VizNodeMap[0].TotalW/12.0)
	vnh.PopulateNodesYCoords()
	vnh.PopulateEdgeLabelCoords()

	return vnh.VizNodeMap, nil
}
