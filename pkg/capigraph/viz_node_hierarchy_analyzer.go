package capigraph

import (
	"fmt"
	"math"
	"time"
)

type VizNodeHierarchyAnalyzer struct {
}

func getBestHierarchy(nodeDefs []NodeDef, nodeFo FontOptions, edgeFo FontOptions, optimize bool) ([]VizNode, int64, float64, float64, error) {
	priParentMap := buildPriParentMap(nodeDefs)
	//layerMap := buildLayerMap(nodeDefs, priParentMap)
	layerMap := buildLayerMap(nodeDefs)
	rootNodes := buildRootNodeList(priParentMap)
	mx, err := NewLayerMx(nodeDefs, layerMap, rootNodes)
	if err != nil {
		return nil, int64(0), 0.0, 0.0, err
	}

	vnh := NewVizNodeHierarchy(nodeDefs, nodeFo, edgeFo)

	vnh.buildNewRootSubtreeHierarchy(mx)

	var bestMx LayerMx
	mxPermCnt := 0
	bestDistSec := math.MaxFloat64
	var tElapsed float64
	if optimize {
		bestSignature := "z"
		tStart := time.Now()
		mxi, err := NewLayerMxPermIterator(nodeDefs, mx)
		if err != nil {
			return nil, int64(0), 0.0, 0.0, err
		}
		mxi.MxIterator(func(i int, mxPerm LayerMx) {

			// Hierarchy
			vnh.reuseRootSubtreeHierarchy(mxPerm)

			// X coord
			vnh.PopulateNodeTotalWidth()
			vnh.PopulateNodesXCoords()

			distSec := vnh.CalculateTotalHorizontalShift()
			if distSec < bestDistSec {
				// This: 1. Adds determinism 2. helps user choose ids that go first (to some extent)
				signature := mxPerm.signature()
				if distSec < bestDistSec-0.1 || signature < bestSignature {
					bestDistSec = distSec
					bestMx = mxPerm.clone()
					bestSignature = signature
				}
			}
			mxPermCnt++
		})
		tElapsed = time.Since(tStart).Seconds()

		if bestMx == nil {
			return nil, int64(mxPermCnt), tElapsed, 0.0, fmt.Errorf("no best")
		}
	} else {
		bestMx = mx
		bestDistSec = 0.0
	}

	// Hierarchy
	vnh.reuseRootSubtreeHierarchy(bestMx)

	// X coord
	vnh.PopulateNodeTotalWidth()
	vnh.PopulateNodesXCoords()

	// Y coord
	vnh.PopulateEdgeLabelDimensions()
	vnh.PopulateUpperLayerGapMap(edgeFo.SizeInPixels)
	vnh.PopulateNodesYCoords()

	// Edge label X and Y
	vnh.PopulateEdgeLabelCoords()
	vnh.RemoveDuplicateSecEdgeLabels()

	return vnh.VizNodeMap, int64(mxPermCnt), tElapsed, bestDistSec, nil
}
