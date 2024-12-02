package capigraph

import "fmt"

const MaxAllowedLayerPermutations int64 = 100000000

type LayerMxPermIterator struct {
	Lps             []*LayerPermutator // We need separate permutator for each layer to preserve state when we do recursion between layers
	IntervalStarts  [][]int
	IntervalLengths [][]int
	SrcMx           LayerMx
	WorkMx          LayerMx
	PriParentMap    []int16
	PriChildrenMap  [][]int16
	NodeLayerMap    []int
	RootNodes       []int16
}

func NewLayerMxPermIterator(nodeDefs []NodeDef, srcMx LayerMx) (*LayerMxPermIterator, error) {
	mxi := LayerMxPermIterator{}
	mxi.Lps = make([]*LayerPermutator, len(srcMx))
	mxi.IntervalStarts = make([][]int, len(srcMx))
	mxi.IntervalLengths = make([][]int, len(srcMx))
	for i := range len(mxi.Lps) {
		mxi.Lps[i] = NewLayerPermutator()
		mxi.IntervalStarts[i] = make([]int, MaxIntervalsInLayer)
		mxi.IntervalLengths[i] = make([]int, MaxIntervalsInLayer)
	}

	mxi.SrcMx = srcMx
	mxi.WorkMx = srcMx.clone()
	mxi.PriParentMap = buildPriParentMap(nodeDefs)
	mxi.PriChildrenMap = buildPriChildrenMap(nodeDefs)
	//mxi.NodeLayerMap = buildLayerMap(nodeDefs, mxi.PriParentMap)
	mxi.NodeLayerMap = buildLayerMap(nodeDefs)
	mxi.RootNodes = buildRootNodeList(mxi.PriParentMap)

	permutations := mxi.MxIteratorCount()
	if permutations > MaxAllowedLayerPermutations {
		return nil, fmt.Errorf("cannot create LayerMxPermIterator, too many permutations, only %d : %d", MaxAllowedLayerPermutations, permutations)
	}
	return &mxi, nil
}

func harvestRowPermutationSettings(mxi *LayerMxPermIterator, layerIdx int) (int, int, int) {
	var lastParent int16
	var lastParentFirstIdx int
	totalIntervals := 0
	var insertStart, insertLen int
	for i, curNodeId := range mxi.WorkMx[layerIdx] {
		var curParent int16
		if curNodeId > FakeNodeBase {
			curParent = mxi.PriParentMap[curNodeId-FakeNodeBase]
		} else {
			curParent = mxi.PriParentMap[curNodeId]
		}
		if curParent == MissingNodeId {
			// This is a root
			// Wrap up an interval for lastParent
			if i-lastParentFirstIdx > 1 {
				// Safeguard. Unlikely, but possible
				if totalIntervals < len(mxi.IntervalStarts[layerIdx]) {
					mxi.IntervalStarts[layerIdx][totalIntervals] = lastParentFirstIdx
					mxi.IntervalLengths[layerIdx][totalIntervals] = i - lastParentFirstIdx
					totalIntervals++
				}
			}
			lastParent = curParent
			lastParentFirstIdx = i

			if insertLen == 0 {
				insertStart = i
				insertLen = 1
			} else {
				insertLen++
			}
		} else {
			// This is not a root
			if curParent != lastParent {
				// Wrap up an interval for lastParent
				if i-lastParentFirstIdx > 2 {
					// Safeguard. Unlikely, but possible
					if totalIntervals < len(mxi.IntervalStarts[layerIdx]) {
						mxi.IntervalStarts[layerIdx][totalIntervals] = lastParentFirstIdx
						mxi.IntervalLengths[layerIdx][totalIntervals] = i - lastParentFirstIdx
						totalIntervals++
					}
				}
				lastParent = curParent
				lastParentFirstIdx = i
			} else {
				// Continuing current interval
			}
		}

		if i == len(mxi.WorkMx[layerIdx])-1 {
			// If the last id is not root and it's the ending of an interval - wrap it up here
			if curParent != MissingNodeId && i > lastParentFirstIdx {
				// Safeguard. Unlikely, but possible
				if totalIntervals < len(mxi.IntervalStarts[layerIdx]) {
					mxi.IntervalStarts[layerIdx][totalIntervals] = lastParentFirstIdx
					mxi.IntervalLengths[layerIdx][totalIntervals] = i - lastParentFirstIdx + 1
					totalIntervals++
				}
			}
		}
	}
	return totalIntervals, insertStart, insertLen
}

func (mxi *LayerMxPermIterator) MxIterator(f func(int, LayerMx)) {
	mxIterRecursive(mxi, 0, 0, f)
}

func mxIterRecursive(mxi *LayerMxPermIterator, layerIdx int, totalCnt int, f func(totalCnt int, perm LayerMx)) {
	cbInner := func(int, LayerMx) {
		f(totalCnt, mxi.WorkMx)
		totalCnt++
	}
	cb := func(int, []int16) {
		mxi.WorkMx[layerIdx] = mxi.Lps[layerIdx].WorkPerm
		mxIterRecursive(mxi, layerIdx+1, totalCnt, cbInner)
	}
	if layerIdx == len(mxi.SrcMx) {
		f(totalCnt, mxi.WorkMx)
		totalCnt++
		return
	}

	newNodeIdx := 0
	if layerIdx > 0 {
		for _, nodeId := range mxi.WorkMx[layerIdx-1] {
			if nodeId > FakeNodeBase {
				// Fake parent node, has only one child. Decide either this (layerIdx) child is fake or not.
				childLayer := mxi.NodeLayerMap[nodeId-FakeNodeBase]
				if childLayer == layerIdx {
					mxi.WorkMx[layerIdx][newNodeIdx] = nodeId - FakeNodeBase // This is the true node
				} else {
					mxi.WorkMx[layerIdx][newNodeIdx] = nodeId // Keep faking, keep in mind that nodeId will have children that reference same Def.NodeId
				}
				newNodeIdx++
			} else {
				// Normal (non-fake) parent
				children := mxi.PriChildrenMap[nodeId]
				for _, childId := range children {
					childLayer := mxi.NodeLayerMap[childId]
					if childLayer > layerIdx {
						// Add another fake node until childLayer == layerIdx
						mxi.WorkMx[layerIdx][newNodeIdx] = childId + FakeNodeBase
					} else {
						// Add the real node, childLayer == layerIdx
						mxi.WorkMx[layerIdx][newNodeIdx] = childId
					}
					newNodeIdx++
				}
			}
		}
	}

	// Add new roots
	for _, rootId := range mxi.RootNodes {
		if mxi.NodeLayerMap[rootId] == layerIdx {
			mxi.WorkMx[layerIdx][newNodeIdx] = rootId
			newNodeIdx++
		}
	}

	totalIntervals, insertStart, insertLen := harvestRowPermutationSettings(mxi, layerIdx)
	if totalIntervals > 0 && insertLen > 0 && insertStart >= 0 {
		mxi.Lps[layerIdx].SwapAndInsertIterator(mxi.IntervalStarts[layerIdx], mxi.IntervalLengths[layerIdx], totalIntervals, insertStart, insertLen, mxi.WorkMx[layerIdx], cb)
	} else if totalIntervals > 0 {
		mxi.Lps[layerIdx].SwapIterator(mxi.IntervalStarts[layerIdx], mxi.IntervalLengths[layerIdx], totalIntervals, mxi.WorkMx[layerIdx], cb)
	} else if insertLen > 0 {
		mxi.Lps[layerIdx].InsertIterator(insertStart, insertLen, mxi.WorkMx[layerIdx], cb)
	} else {
		// No permutations available, just re-use mxi.WorkMx[layerIdx] without modifications
		// Here, we want to call cb, but without mxi.WorkMx[layerIdx] = mxi.Lps[layerIdx].WorkPerm
		mxIterRecursive(mxi, layerIdx+1, totalCnt, cbInner)
	}
}

func (mxi *LayerMxPermIterator) MxIteratorCount() int64 {
	acc := int64(1)
	for layerIdx := range len(mxi.SrcMx) {
		totalIntervals, insertStart, insertLen := harvestRowPermutationSettings(mxi, layerIdx)
		for i := range totalIntervals {
			intervalLen := mxi.IntervalLengths[layerIdx][i]
			acc *= int64(mxi.Lps[layerIdx].Fact[intervalLen])
		}
		for range insertLen {
			acc *= int64(insertStart + 1)
			insertStart++
		}
	}
	return acc
}
