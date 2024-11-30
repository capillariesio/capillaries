package capigraph

// 0:    1  3
// -     |/
// 1:    2
var testNodeDefsBasic = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{{3, "from 3"}}, "", 0},
	{3, "3", EdgeDef{}, []EdgeDef{}, "", 0},
}

// 0:    1  3
// -     |  |
// 1:    2  4
var testNodeDefsTrivialParallel = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
}

// 0:    1
// -     |    \
// 1:    2  6  3
// -     | /  \|
// 2:    4     5
var testNodeDefsOneEnclosedOneLevel = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{2, ""}, []EdgeDef{{6, ""}}, "", 0},
	{5, "5", EdgeDef{3, ""}, []EdgeDef{{6, ""}}, "", 0},
	{6, "6", EdgeDef{}, []EdgeDef{}, "", 0},
}

// 0:    1
// -     |  \
// 1:    2     5
// -     |     |
// 2:    3  8  6
// -     | / \ |
// 3:    4     7
var testNodeDefsOneEnclosedTwoLevels = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{{8, ""}}, "", 0},
	{5, "5", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{5, ""}, []EdgeDef{}, "", 0},
	{7, "7", EdgeDef{6, ""}, []EdgeDef{{8, ""}}, "", 0},
	{8, "8", EdgeDef{}, []EdgeDef{}, "", 0},
}

// 0:    1
// -     |  \
// 1:    2     4
// -     |     |
// 2:    3     5
var testNodeDefsNoIntervals = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
}

// 0:    1 2 3 4 5 6 7 8 9 10
var testNodeDefsFlat10 = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{}, []EdgeDef{}, "", 0},
	{7, "7", EdgeDef{}, []EdgeDef{}, "", 0},
	{8, "8", EdgeDef{}, []EdgeDef{}, "", 0},
	{9, "9", EdgeDef{}, []EdgeDef{}, "", 0},
	{10, "10", EdgeDef{}, []EdgeDef{}, "", 0},
}

// 0:    1
// -     |  \
// 1:    2     3
// -     |     |
// 2:    4  8  5  9
// -     |    \| /
// -     | / / | /
// -     | //  |
// 2:    6     7
var testNodeDefsTwoEnclosedNodeSizeMatters = []NodeDef{
	{0, "", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{4, ""}, []EdgeDef{{8, ""}, {9, ""}}, "", 0},
	{7, "7", EdgeDef{5, ""}, []EdgeDef{{8, ""}, {9, ""}}, "", 0},
	{8, "8", EdgeDef{}, []EdgeDef{}, "", 0},
	{9, "9", EdgeDef{}, []EdgeDef{}, "", 0},
}

// 0:    1  3
// -     |  |
// 1:    2  4  6
// -        | /
// 2:       5
var testNodeDefsOneSecondary = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{4, ""}, []EdgeDef{{6, ""}}, "", 0},
	{6, "6", EdgeDef{}, []EdgeDef{}, "", 0},
}

// 0:       1
// -      / | \
// 1:    2  3  4 6
// -        | //
// 2:       5
var testNodeDefsDiamond = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{3, ""}, []EdgeDef{{4, ""}, {6, ""}}, "", 0},
	{6, "6", EdgeDef{}, []EdgeDef{}, "", 0},
}

var testNodeDefs40milPerms = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},

	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{1, ""}, []EdgeDef{}, "", 0},

	{5, "5", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{7, "7", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{8, "8", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{9, "9", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{10, "10", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{11, "11", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{12, "12", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{13, "13", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{14, "14", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{15, "15", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
	{16, "16", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
	{17, "17", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
	{18, "18", EdgeDef{4, ""}, []EdgeDef{{20, ""}}, "", 0},
	//{19, "19", EdgeDef{4, ""}, []EdgeDef{{21, ""}}, "", 0},
	{19, "19", EdgeDef{4, ""}, []EdgeDef{}, "", 0},

	{20, "20", EdgeDef{}, []EdgeDef{}, "", 0},
	//{21, "21", EdgeDef{}, []EdgeDef{}, "", 0},
}

var testNodeDefs300bilPerms = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},

	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{1, ""}, []EdgeDef{}, "", 0},

	{5, "5", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{7, "7", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{8, "8", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{9, "9", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{10, "10", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{11, "11", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{12, "12", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{13, "13", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{14, "14", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{15, "15", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{16, "16", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
	{17, "17", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
	{18, "18", EdgeDef{4, ""}, []EdgeDef{{22, ""}}, "", 0},
	{19, "19", EdgeDef{4, ""}, []EdgeDef{{23, ""}}, "", 0},
	{20, "20", EdgeDef{4, ""}, []EdgeDef{{24, ""}}, "", 0},
	{21, "21", EdgeDef{4, ""}, []EdgeDef{{25, ""}}, "", 0},

	{22, "22", EdgeDef{}, []EdgeDef{}, "", 0},
	{23, "23", EdgeDef{}, []EdgeDef{}, "", 0},
	{24, "24", EdgeDef{}, []EdgeDef{}, "", 0},
	{25, "25", EdgeDef{}, []EdgeDef{}, "", 0},
}

// 0:    1  2
// -     | /|
// 1:    3  |
// -     | /
// 2:    4
var testNodeDefsTwoLevelsFromOneParent = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{1, ""}, []EdgeDef{{2, ""}}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{{2, ""}}, "", 0},
}

// 0:    1
// -     | \
// 1:    2   5
// -     |   |
// 2:    3   F
// -     | \ |
// 3:    4   6
// -         |
// 4:        7
var testNodeDefsTwoLevelsFromOneParentSameRoot = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{5, ""}, []EdgeDef{{3, ""}}, "", 0},
	{7, "7", EdgeDef{6, ""}, []EdgeDef{}, "", 0},
}

// 0:    1
// -     | \
// 1:    2   6
// -     |   |
// 2:    3   F
// -     |   |
// 3:    4   F
// -     | \ |
// 4:    5   7
// -         |
// 5:        8
var testNodeDefsTwoLevelsFromOneParentSameRootTwoFakes = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{7, "7", EdgeDef{6, ""}, []EdgeDef{{4, ""}}, "", 0},
	{8, "8", EdgeDef{7, ""}, []EdgeDef{}, "", 0},
}

// 0:    1
// - 	 |
// 1:    2
// -     |
// 2:    3     5
// -     |  /  |
// 3:    4     6
// -           |
// 4:          7
var testNodeDefsSubtreeBelowLong = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{{5, ""}}, "", 0},
	{5, "5", EdgeDef{}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{5, ""}, []EdgeDef{}, "", 0},
	{7, "7", EdgeDef{6, ""}, []EdgeDef{}, "", 0},
}

// 0:    1
// - 	 |
// 1:    2      6
// -     |    / |
// 2:    3   /  7
// -     |  /   |
// 3:    4      8
// -     |   /
// 4:    5
var testNodeDefsOneNotTwoLevelsDown = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{{6, ""}}, "", 0},
	{5, "5", EdgeDef{4, ""}, []EdgeDef{{8, ""}}, "", 0},
	{6, "6", EdgeDef{}, []EdgeDef{}, "", 0},
	{7, "7", EdgeDef{6, ""}, []EdgeDef{}, "", 0},
	{8, "8", EdgeDef{7, ""}, []EdgeDef{}, "", 0},
}

// 0:    1
// - 	 |
// 1:    2    5
// -     |   /|
// 2:    3  / 6
// -     | //
// 3:    4
var testNodeDefsMultiSecParentPullDown = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{3, ""}, []EdgeDef{{5, ""}, {6, ""}}, "", 0},
	{5, "5", EdgeDef{}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{5, ""}, []EdgeDef{}, "", 0},
}

// 0:    1   4
// - 	 |  /|
// 1:    2 / 5
// -     |//
// 2:    3
var testNodeDefsMultiSecParentNoPullDown = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{2, ""}, []EdgeDef{{4, ""}, {5, ""}}, "", 0},
	{4, "4", EdgeDef{}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{4, ""}, []EdgeDef{}, "", 0},
}

// 0:    1
// - 	  \\\\\\
// 1:      2 3 4 5 6
var testNodeDefsDuplicateSecLabels = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "", 0},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0},

	{2, "2", EdgeDef{}, []EdgeDef{}, "", 0},
	{3, "3", EdgeDef{}, []EdgeDef{}, "", 0},
	{4, "4", EdgeDef{}, []EdgeDef{}, "", 0},
	{5, "5", EdgeDef{}, []EdgeDef{}, "", 0},
	{6, "6", EdgeDef{}, []EdgeDef{}, "", 0},

	{7, "7", EdgeDef{2, ""}, []EdgeDef{{1, "txt"}}, "", 0},
	{8, "8", EdgeDef{3, ""}, []EdgeDef{{1, "txt"}}, "", 0},
	{9, "9", EdgeDef{4, ""}, []EdgeDef{{1, "txt"}}, "", 0},
	{10, "10", EdgeDef{5, ""}, []EdgeDef{{1, "txt"}}, "", 0},
	{11, "11", EdgeDef{6, ""}, []EdgeDef{{1, "txt"}}, "", 0},
}
