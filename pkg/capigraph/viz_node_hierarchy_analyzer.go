package capigraph

import (
	"fmt"
	"math"
	"time"
)

type VizNodeHierarchyAnalyzer struct {
}

func GetBestHierarchy(nodeDefs []NodeDef, nodeFo FontOptions, edgeFo FontOptions) ([]VizNode, int64, float64, float64, error) {
	if err := checkNodeIds(nodeDefs); err != nil {
		return nil, int64(0), 0.0, 0.0, err
	}

	for i := range len(nodeDefs) - 1 {
		if err := checkNodeDef(int16(i+1), nodeDefs); err != nil {
			return nil, int64(0), 0.0, 0.0, err
		}
	}

	priParentMap := buildPriParentMap(nodeDefs)
	layerMap := buildLayerMap(nodeDefs, priParentMap)
	rootNodes := buildRootNodeList(priParentMap)
	mx, err := NewLayerMx(nodeDefs, layerMap, rootNodes)
	if err != nil {
		return nil, int64(0), 0.0, 0.0, err
	}

	mxi, err := NewLayerMxPermIterator(nodeDefs, mx)
	if err != nil {
		return nil, int64(0), 0.0, 0.0, err
	}

	vnh := NewVizNodeHierarchy(nodeDefs, nodeFo, edgeFo)

	vnh.buildNewRootSubtreeHierarchy(mx)

	bestDistSec := math.MaxFloat64
	bestSignature := "z"
	var bestMx LayerMx
	mxPermCnt := 0
	tStart := time.Now()
	mxi.MxIterator(func(i int, mxPerm LayerMx) {

		// Hierarchy
		vnh.reuseRootSubtreeHierarchy(mxPerm)
		// X coord
		vnh.PopulateNodeTotalWidth()
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
	tElapsed := time.Since(tStart).Seconds()

	if bestMx == nil {
		return nil, int64(mxPermCnt), tElapsed, 0.0, fmt.Errorf("no best")
	}

	// Hierarchy
	vnh.reuseRootSubtreeHierarchy(bestMx)

	// X coord
	vnh.PopulateNodeTotalWidth()
	vnh.PopulateNodesXCoords()

	// Y coord
	vnh.PopulateEdgeLabelDimensions()
	vnh.PopulateUpperLayerGapMap(edgeFo.SizeInPixels, math.Max(vnh.VizNodeMap[0].TotalW/20.0, nodeFo.SizeInPixels*3)) // Purely empiric
	vnh.PopulateNodesYCoords()

	// Edge label X and Y
	vnh.PopulateEdgeLabelCoords()

	return vnh.VizNodeMap, int64(mxPermCnt), tElapsed, bestDistSec, nil
}
