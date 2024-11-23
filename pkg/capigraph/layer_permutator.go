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

// Move item located at itemToInsertSrcIdx to permIdx, shifting [permIdx, itemToInsertSrcIdx-1] to the right
/*
func (lp *LayerPermutator) insertPermutationByIdx(in []int16, out []int16, itemToInsertSrcIdx int, permIdx int) {
	if permIdx == itemToInsertSrcIdx {
		return
	}
	if permIdx > itemToInsertSrcIdx {
		panic(fmt.Sprintf("insertPermutationByIdx(): cannot move item from % d to %d", itemToInsertSrcIdx, permIdx))
	}
	// Shift
	copy(out[permIdx+1:], in[permIdx:itemToInsertSrcIdx])
	// Set the item to be inserted
	out[permIdx] = in[itemToInsertSrcIdx]
}
*/

/*
func (lp *LayerPermutator) insertPermIteratorRecursiveWithAlloc(insertSrcStart int, insertSrcLen int, curInsertSrc int, in []int16, totalCnt int) func(yield func(int, []int16) bool) {
	if insertSrcLen > MaxNodesToInsert {
		panic("too many to insert")
	}
	return func(yield func(int, []int16) bool) {
		o := make([]int16, len(in)) // On stack!

		for permIdx := range curInsertSrc + 1 {
			copy(o, in) // Well, that's a lot of copying...
			lp.insertPermutationByIdx(in, o, curInsertSrc, permIdx)
			if curInsertSrc == insertSrcStart+insertSrcLen-1 {
				if !yield(totalCnt, o) {
					return
				}
				totalCnt++
			} else {
				for _, newo := range lp.insertPermIteratorRecursive(insertSrcStart, insertSrcLen, curInsertSrc+1, o, totalCnt) {
					if !yield(totalCnt, newo) {
						return
					}
					totalCnt++
				}
			}
		}
	}
}
*/

func (lp *LayerPermutator) insertPermutationByIdx(perm []int16, itemToInsertSrcIdx int, permIdx int) {
	if permIdx == itemToInsertSrcIdx {
		// Identity permutation, totally valid
		return
	}
	if permIdx > itemToInsertSrcIdx {
		panic(fmt.Sprintf("insertPermutationByIdx(): cannot move item from % d to %d", itemToInsertSrcIdx, permIdx))
	}
	idToInsert := perm[itemToInsertSrcIdx]
	// Shift
	copy(perm[permIdx+1:], perm[permIdx:itemToInsertSrcIdx])
	// Set the item to be inserted
	perm[permIdx] = idToInsert
}

/*
func (lp *LayerPermutator) insertPermIteratorRecursive(insertSrcStart int, insertSrcLen int, curInsertSrc int, totalCnt int) func(yield func(int, []int16) bool) {
	if insertSrcLen > MaxNodesToInsert {
		panic("too many to insert")
	}
	return func(yield func(int, []int16) bool) {
		backupBetweenInserts := []int16{10: int16(0)}[:len(lp.WorkPerm)] //make([]int16, len(lp.WorkPerm)) // On stack!
		for permIdx := range curInsertSrc + 1 {
			copy(backupBetweenInserts, lp.WorkPerm)
			// Shift-insert in-place
			lp.insertPermutationByIdx(lp.WorkPerm, curInsertSrc, permIdx)
			if curInsertSrc == insertSrcStart+insertSrcLen-1 {
				if !yield(totalCnt, lp.WorkPerm) {
					return
				}
				totalCnt++
			} else {
				for range lp.insertPermIteratorRecursive(insertSrcStart, insertSrcLen, curInsertSrc+1, totalCnt) {
					if !yield(totalCnt, lp.WorkPerm) {
						return
					}
					totalCnt++
				}
			}
			copy(lp.WorkPerm, backupBetweenInserts)
		}
	}
}
*/

/*
func (lp *LayerPermutator) InsertPermIterator(insertSrcStart int, insertSrcLen int) func(yield func(int, []int16) bool) {
	//lp.initSrcAndWorkPerm(in)

	return func(yield func(int, []int16) bool) {
		for i, _ := range lp.insertPermIteratorRecursive(insertSrcStart, insertSrcLen, insertSrcStart, 0) {
			if !yield(i, lp.WorkPerm) {
				return
			}
		}
	}
}
*/

// Execute swap perm strategy for a specific intervalLen and permIdx

func (lp *LayerPermutator) swapPermutationByIdx(perm []int16, firstIdx int, intervalLen int, permIdx int) {
	if intervalLen > MaxSupportedFact {
		panic(fmt.Sprintf("permutationByIdx(): factorial value not supported %d", intervalLen))
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

/*
	func (lp *LayerPermutator) swapPermutationByIdx(in []int16, firstIdx int, intervalLen int, out []int16, permIdx int) {
		if intervalLen > MaxSupportedFact {
			panic(fmt.Sprintf("permutationByIdx(): factorial value not supported %d", intervalLen))
		}
		copy(out[firstIdx:firstIdx+intervalLen], in[firstIdx:firstIdx+intervalLen])
		if intervalLen == 2 {
			if intervalLen-1 > lp.P[2][permIdx][0] {
				swap(out, firstIdx+intervalLen-1, firstIdx+lp.P[2][permIdx][0])
			}
		} else if intervalLen == 3 {
			if intervalLen-2 > lp.P[3][permIdx][0] {
				swap(out, firstIdx+intervalLen-2, firstIdx+lp.P[3][permIdx][0])
			}
			if intervalLen-1 > lp.P[3][permIdx][1] {
				swap(out, firstIdx+intervalLen-1, firstIdx+lp.P[3][permIdx][1])
			}
		} else if intervalLen == 4 {
			if intervalLen-3 > lp.P[4][permIdx][0] {
				swap(out, firstIdx+intervalLen-3, firstIdx+lp.P[4][permIdx][0])
			}
			if intervalLen-2 > lp.P[4][permIdx][1] {
				swap(out, firstIdx+intervalLen-2, firstIdx+lp.P[4][permIdx][1])
			}
			if intervalLen-1 > lp.P[4][permIdx][2] {
				swap(out, firstIdx+intervalLen-1, firstIdx+lp.P[4][permIdx][2])
			}
		} else if intervalLen == 5 {
			if intervalLen-4 > lp.P[5][permIdx][0] {
				swap(out, firstIdx+intervalLen-4, firstIdx+lp.P[5][permIdx][0])
			}
			if intervalLen-3 > lp.P[5][permIdx][1] {
				swap(out, firstIdx+intervalLen-3, firstIdx+lp.P[5][permIdx][1])
			}
			if intervalLen-2 > lp.P[5][permIdx][2] {
				swap(out, firstIdx+intervalLen-2, firstIdx+lp.P[5][permIdx][2])
			}
			if intervalLen-1 > lp.P[5][permIdx][3] {
				swap(out, firstIdx+intervalLen-1, firstIdx+lp.P[5][permIdx][3])
			}
		} else if intervalLen == 6 {
			if intervalLen-5 > lp.P[6][permIdx][0] {
				swap(out, firstIdx+intervalLen-5, firstIdx+lp.P[6][permIdx][0])
			}
			if intervalLen-4 > lp.P[6][permIdx][1] {
				swap(out, firstIdx+intervalLen-4, firstIdx+lp.P[6][permIdx][1])
			}
			if intervalLen-3 > lp.P[6][permIdx][2] {
				swap(out, firstIdx+intervalLen-3, firstIdx+lp.P[6][permIdx][2])
			}
			if intervalLen-2 > lp.P[6][permIdx][3] {
				swap(out, firstIdx+intervalLen-2, firstIdx+lp.P[6][permIdx][3])
			}
			if intervalLen-1 > lp.P[6][permIdx][4] {
				swap(out, firstIdx+intervalLen-1, firstIdx+lp.P[6][permIdx][4])
			}
		}
	}

	func (lp *LayerPermutator) swapPermIteratorRecursiveWithAlloc(permIntervalStarts []int, permIntervalLengths []int, intervalIdx int, in []int16, totalCnt int) func(yield func(int, []int16) bool) {
		return func(yield func(int, []int16) bool) {
			intStart := permIntervalStarts[intervalIdx]
			intLen := permIntervalLengths[intervalIdx]
			if intLen > MaxSupportedFact {
				panic("interval too big, 6 max")
			}
			o := make([]int16, len(in)) // On stack!
			copy(o, in)
			for i := range lp.Fact[intLen] {
				lp.swapPermutationByIdx(in, intStart, intLen, o, i)
				if intervalIdx == len(permIntervalStarts)-1 {
					if !yield(totalCnt, o) {
						return
					}
					totalCnt++
				} else {
					for _, newo := range lp.swapPermIteratorRecursiveWithAlloc(permIntervalStarts, permIntervalLengths, intervalIdx+1, o, totalCnt) {
						if !yield(totalCnt, newo) {
							return
						}
						totalCnt++

					}
				}
			}
		}
	}

func (lp *LayerPermutator) swapPermIteratorRecursive(intervalIdx int, totalCnt int) func(yield func(int, []int16) bool) {
	return func(yield func(int, []int16) bool) {
		intStart := lp.IntervalStarts[intervalIdx]
		intLen := lp.IntervalLengths[intervalIdx]
		if intLen > MaxSupportedFact {
			panic("interval too big, 6 max")
		}
		for i := range lp.Fact[intLen] {
			// Re-initialize slice interval we work with
			copy(lp.WorkPerm[intStart:], lp.SrcPerm[intStart:intStart+intLen])
			// Swap in-place
			lp.swapPermutationByIdx(lp.WorkPerm, intStart, intLen, i)
			if intervalIdx == len(lp.IntervalStarts)-1 {
				if !yield(totalCnt, lp.WorkPerm) {
					return
				}
				totalCnt++
			} else {
				for range lp.swapPermIteratorRecursive(intervalIdx+1, totalCnt) { // No allocations here!
					if !yield(totalCnt, lp.WorkPerm) {
						return
					}
					totalCnt++

				}
			}
		}
	}
}


	func (lp *LayerPermutator) SwapPermIteratorWithAlloc(permIntervalStarts []int, permIntervalLengths []int, in []int16) func(yield func(int, []int16) bool) {
		return func(yield func(int, []int16) bool) {
			for i, perm := range lp.swapPermIteratorRecursiveWithAlloc(permIntervalStarts, permIntervalLengths, 0, in, 0) {
				if !yield(i, perm) {
					return
				}
			}
		}
	}


func (lp *LayerPermutator) SwapPermIterator() func(yield func(int, []int16) bool) {
	// lp.initSrcAndWorkPerm(in)
	// lp.IntervalStarts = lp.IntervalStarts[:totalIntervals]
	// copy(lp.IntervalStarts, permIntervalStarts[:totalIntervals])
	// lp.IntervalLengths = lp.IntervalLengths[:totalIntervals]
	// copy(lp.IntervalLengths, permIntervalLengths[:totalIntervals])

	return func(yield func(int, []int16) bool) {
		for i, _ := range lp.swapPermIteratorRecursive(0, 0) {
			if !yield(i, lp.WorkPerm) {
				return
			}
		}
	}
}


func (lp *LayerPermutator) SwapAndInsertPermIterator(permIntervalStarts []int, permIntervalLengths []int, insertSrcStart int, insertSrcLen int, in []int16) func(yield func(int, []int16) bool) {
	lp.SwapSrcPerm = in
	lp.SwapWorkPerm = make([]int16, len(in))
	copy(lp.SwapWorkPerm, lp.SwapSrcPerm)

	return func(yield func(int, []int16) bool) {
		i := 0
		//for _, swapPerm := range lp.swapPermIteratorRecursive(permIntervalStarts, permIntervalLengths, 0, lp.WorkPerm, 0) {
		//for _, finalPerm := range lp.insertPermIteratorRecursive(insertSrcStart, insertSrcLen, insertSrcStart, swapPerm, 0) {
		//	if !yield(i, finalPerm) {
		for range lp.swapPermIteratorRecursive(permIntervalStarts, permIntervalLengths, 0, lp.SwapWorkPerm, 0) {
			lp.InsertSrcPerm = lp.SwapWorkPerm
			lp.InsertWorkPerm = make([]int16, len(lp.InsertSrcPerm))
			copy(lp.InsertWorkPerm, lp.InsertSrcPerm)
			for range lp.insertPermIteratorRecursive(insertSrcStart, insertSrcLen, insertSrcStart, lp.InsertWorkPerm, 0) {
				if !yield(i, lp.InsertWorkPerm) {
					return
				}
				i++
			}
		}
	}
}



func (lp *LayerPermutator) SwapAndInsertPermIterator(insertSrcStart int, insertSrcLen int) func(yield func(int, []int16) bool) {
	// lp.initSrcAndWorkPerm(in)
	// lp.IntervalStarts = lp.IntervalStarts[:totalIntervals]
	// copy(lp.IntervalStarts, permIntervalStarts[:totalIntervals])
	// lp.IntervalLengths = lp.IntervalLengths[:totalIntervals]
	// copy(lp.IntervalLengths, permIntervalLengths[:totalIntervals])

	return func(yield func(int, []int16) bool) {
		i := 0
		for range lp.swapPermIteratorRecursive(0, 0) { // No alloc here!
			for range lp.insertPermIteratorRecursive(insertSrcStart, insertSrcLen, insertSrcStart, 0) { // No alloc here!
				if !yield(i, lp.WorkPerm) {
					return
				}
				i++
			}
		}
	}
}
*/

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
		panic("interval too big, 6 max")
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
		panic("too many to insert")
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
