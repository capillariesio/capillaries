package capigraph

import (
	"fmt"
)

func swap(a []int16, i1 int, i2 int) {
	t := a[i2]
	a[i2] = a[i1]
	a[i1] = t
}

const MaxSupportedFact int = 6     // Max amount of pri children originating from a node
const MaxNodesToInsert int = 10    // Cannot insert more than this amt of new roots at a layer
const MaxIntervalsInLayer int = 20 // Max parent nodes containing more than one child on this layer
const MaxLayerLen int = 200

type LayerPermutator struct {
	P               [][][]int // Swap permutation strategy by permIdx: [interval size 2-6][permIdx 2!-6!][positions to swap with]
	Fact            []int
	SrcPerm         []int16
	WorkPerm        []int16
	IntervalStarts  []int
	IntervalLengths []int
}

func NewLayerPermutator() *LayerPermutator {
	lp := LayerPermutator{}
	lp.init()
	return &lp
}

func (lp *LayerPermutator) insertPermutationByIdx(perm []int16, itemToInsertSrcIdx int, permIdx int) {
	if permIdx == itemToInsertSrcIdx {
		// Identity permutation, totally valid
		return
	}
	if permIdx > itemToInsertSrcIdx {
		panic(fmt.Sprintf("insertPermutationByIdx(): cannot move item %d from % d to %d", perm[itemToInsertSrcIdx], itemToInsertSrcIdx, permIdx))
	}
	idToInsert := perm[itemToInsertSrcIdx]
	// Shift
	copy(perm[permIdx+1:], perm[permIdx:itemToInsertSrcIdx])
	// Set the item to be inserted
	perm[permIdx] = idToInsert
}

// Execute swap perm strategy for a specific intervalLen and permIdx

func (lp *LayerPermutator) swapPermutationByIdx(perm []int16, firstIdx int, intervalLen int, permIdx int) {
	if intervalLen > MaxSupportedFact {
		panic(fmt.Sprintf("permutationByIdx(): factorial value not supported %d, max supported %d", intervalLen, MaxSupportedFact))
	}
	if intervalLen == 2 {
		if intervalLen-1 > lp.P[2][permIdx][0] {
			swap(perm, firstIdx+intervalLen-1, firstIdx+lp.P[2][permIdx][0])
		}
	} else if intervalLen == 3 {
		if intervalLen-2 > lp.P[3][permIdx][0] {
			swap(perm, firstIdx+intervalLen-2, firstIdx+lp.P[3][permIdx][0])
		}
		if intervalLen-1 > lp.P[3][permIdx][1] {
			swap(perm, firstIdx+intervalLen-1, firstIdx+lp.P[3][permIdx][1])
		}
	} else if intervalLen == 4 {
		if intervalLen-3 > lp.P[4][permIdx][0] {
			swap(perm, firstIdx+intervalLen-3, firstIdx+lp.P[4][permIdx][0])
		}
		if intervalLen-2 > lp.P[4][permIdx][1] {
			swap(perm, firstIdx+intervalLen-2, firstIdx+lp.P[4][permIdx][1])
		}
		if intervalLen-1 > lp.P[4][permIdx][2] {
			swap(perm, firstIdx+intervalLen-1, firstIdx+lp.P[4][permIdx][2])
		}
	} else if intervalLen == 5 {
		if intervalLen-4 > lp.P[5][permIdx][0] {
			swap(perm, firstIdx+intervalLen-4, firstIdx+lp.P[5][permIdx][0])
		}
		if intervalLen-3 > lp.P[5][permIdx][1] {
			swap(perm, firstIdx+intervalLen-3, firstIdx+lp.P[5][permIdx][1])
		}
		if intervalLen-2 > lp.P[5][permIdx][2] {
			swap(perm, firstIdx+intervalLen-2, firstIdx+lp.P[5][permIdx][2])
		}
		if intervalLen-1 > lp.P[5][permIdx][3] {
			swap(perm, firstIdx+intervalLen-1, firstIdx+lp.P[5][permIdx][3])
		}
	} else if intervalLen == 6 {
		if intervalLen-5 > lp.P[6][permIdx][0] {
			swap(perm, firstIdx+intervalLen-5, firstIdx+lp.P[6][permIdx][0])
		}
		if intervalLen-4 > lp.P[6][permIdx][1] {
			swap(perm, firstIdx+intervalLen-4, firstIdx+lp.P[6][permIdx][1])
		}
		if intervalLen-3 > lp.P[6][permIdx][2] {
			swap(perm, firstIdx+intervalLen-3, firstIdx+lp.P[6][permIdx][2])
		}
		if intervalLen-2 > lp.P[6][permIdx][3] {
			swap(perm, firstIdx+intervalLen-2, firstIdx+lp.P[6][permIdx][3])
		}
		if intervalLen-1 > lp.P[6][permIdx][4] {
			swap(perm, firstIdx+intervalLen-1, firstIdx+lp.P[6][permIdx][4])
		}
	}
}

func (lp *LayerPermutator) init() {
	lp.Fact = make([]int, MaxSupportedFact+1)
	acc := 1
	for i := range MaxSupportedFact + 1 {
		lp.Fact[i] = acc
		acc *= (i + 1)
	}

	lp.P = make([][][]int, MaxSupportedFact+1)
	i := 2
	for i <= 6 {
		lp.P[i] = make([][]int, lp.Fact[i])
		p := lp.P[i]
		for j := range len(p) {
			p[j] = make([]int, i-1)
		}

		k := 0
		for k < i-1 {
			streakSize := len(p) / lp.Fact[k+2]
			maxStreakVal := k + 1
			curVal := 0
			curPosWithinStreak := 0

			for j := range len(p) {
				p[j][k] = curVal
				curPosWithinStreak++
				if curPosWithinStreak == streakSize {
					curPosWithinStreak = 0
					curVal++
				}
				if curVal == maxStreakVal+1 {
					curVal = 0
				}
			}
			k++
		}
		i++
	}
}

func (lp *LayerPermutator) initSource(in []int16) {
	lp.SrcPerm = in
	if lp.WorkPerm == nil {
		lp.WorkPerm = make([]int16, len(lp.SrcPerm), MaxLayerLen) // Assume this capacity is enough
	} else if len(lp.WorkPerm) != len(lp.SrcPerm) {
		lp.WorkPerm = lp.WorkPerm[:len(lp.SrcPerm)] // Shrink, or grow, hopefully no reallocation
	}
	copy(lp.WorkPerm, lp.SrcPerm)
}

func (lp *LayerPermutator) initIntervals(permIntervalStarts []int, permIntervalLengths []int, totalIntervals int) {
	lp.IntervalStarts = make([]int, MaxIntervalsInLayer)
	lp.IntervalLengths = make([]int, MaxIntervalsInLayer)
	lp.IntervalStarts = lp.IntervalStarts[:totalIntervals]
	copy(lp.IntervalStarts, permIntervalStarts[:totalIntervals])
	lp.IntervalLengths = lp.IntervalLengths[:totalIntervals]
	copy(lp.IntervalLengths, permIntervalLengths[:totalIntervals])
}

func (lp *LayerPermutator) SwapIterator(intervalStarts []int, intervalLengths []int, totalIntervals int, in []int16, f func(int, []int16)) {
	lp.initSource(in)
	lp.initIntervals(intervalStarts, intervalLengths, totalIntervals)
	swapFuncIteratorRecursive(lp, 0, 0, f)
}

func swapFuncIteratorRecursive(lp *LayerPermutator, intervalIdx int, totalCnt int, f func(int, []int16)) {
	cb := func(int, []int16) {
		f(totalCnt, lp.WorkPerm)
		totalCnt++
	}
	intStart := lp.IntervalStarts[intervalIdx]
	intLen := lp.IntervalLengths[intervalIdx]
	if intLen > MaxSupportedFact {
		panic(fmt.Sprintf("swapFuncIteratorRecursive: swap interval too big: %d, max supported %d", intLen, MaxSupportedFact))
	}
	for i := range lp.Fact[intLen] {
		// Re-initialize slice interval we work with
		copy(lp.WorkPerm[intStart:], lp.SrcPerm[intStart:intStart+intLen])
		// Swap in-place
		lp.swapPermutationByIdx(lp.WorkPerm, intStart, intLen, i)
		if intervalIdx == len(lp.IntervalStarts)-1 {
			f(totalCnt, lp.WorkPerm)
			totalCnt++
		} else {
			swapFuncIteratorRecursive(lp, intervalIdx+1, totalCnt, cb)
		}
	}
}

func (lp *LayerPermutator) InsertIterator(insertSrcStart int, insertSrcLen int, in []int16, f func(int, []int16)) {
	lp.initSource(in)
	insertFuncIteratorRecursive(lp, insertSrcStart, insertSrcLen, insertSrcStart, 0, f)
}

func insertFuncIteratorRecursive(lp *LayerPermutator, insertSrcStart int, insertSrcLen int, curInsertSrc int, totalCnt int, f func(int, []int16)) {
	if insertSrcLen > MaxNodesToInsert {
		panic(fmt.Sprintf("insertFuncIteratorRecursive: too many ids to insert: %d, max supported %d", insertSrcLen, MaxNodesToInsert))
	}
	cb := func(int, []int16) {
		f(totalCnt, lp.WorkPerm)
		totalCnt++
	}
	backupBetweenInserts := []int16{10: int16(0)}[:len(lp.WorkPerm)] //make([]int16, len(lp.WorkPerm)) // On stack!
	for permIdx := range curInsertSrc + 1 {
		copy(backupBetweenInserts, lp.WorkPerm)
		// Shift-insert in-place
		lp.insertPermutationByIdx(lp.WorkPerm, curInsertSrc, permIdx)
		if curInsertSrc == insertSrcStart+insertSrcLen-1 {
			f(totalCnt, lp.WorkPerm)
			totalCnt++
		} else {
			insertFuncIteratorRecursive(lp, insertSrcStart, insertSrcLen, curInsertSrc+1, totalCnt, cb)
		}
		copy(lp.WorkPerm, backupBetweenInserts)
	}
}

func (lp *LayerPermutator) SwapAndInsertIterator(intervalStarts []int, intervalLengths []int, totalIntervals int, insertSrcStart int, insertSrcLen int, in []int16, f func(int, []int16)) {
	lp.initSource(in)
	lp.initIntervals(intervalStarts, intervalLengths, totalIntervals)
	swapAndInsertFuncIterator(lp, 0, insertSrcStart, insertSrcLen, insertSrcStart, 0, f)
}

func swapAndInsertFuncIterator(lp *LayerPermutator, intervalIdx int, insertSrcStart int, insertSrcLen int, curInsertSrc int, totalCnt int, f func(int, []int16)) {
	realTotalCnt := 0
	cb := func(int, []int16) {
		f(realTotalCnt, lp.WorkPerm)
		realTotalCnt++
	}
	swapFuncIteratorRecursive(lp, intervalIdx, totalCnt, func(int, []int16) {
		insertFuncIteratorRecursive(lp, insertSrcStart, insertSrcLen, curInsertSrc, 0, cb)
	})
}
