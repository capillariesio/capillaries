package capigraph

// 0:    1  3
// -     |/
// 1:    2
var testNodeDefsBasic = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, []EdgeDef{{3, "from 3", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1  3
// -     |  |
// 1:    2  4
var testNodeDefsTrivialParallel = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1
// -     |    \
// 1:    2  6  3
// -     | /  \|
// 2:    4     5
var testNodeDefsOneEnclosedOneLevel = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{2, "", TextColorDefault}, []EdgeDef{{6, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{6, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1
// -     |  \
// 1:    2     5
// -     |     |
// 2:    3  8  6
// -     | / \ |
// 3:    4     7
var testNodeDefsOneEnclosedTwoLevels = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{8, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{5, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{6, "", TextColorDefault}, []EdgeDef{{8, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1
// -     |  \
// 1:    2     4
// -     |     |
// 2:    3     5
var testNodeDefsNoIntervals = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1 2 3 4 5 6 7 8 9 10
var testNodeDefsFlat10 = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{9, "9", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{10, "10", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
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
	{0, "", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{8, "", TextColorDefault}, {9, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{5, "", TextColorDefault}, []EdgeDef{{8, "", TextColorDefault}, {9, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{9, "9", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1  3
// -     |  |
// 1:    2  4  6
// -        | /
// 2:       5
var testNodeDefsOneSecondary = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{6, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:       1
// -      / | \
// 1:    2  3  4 6
// -        | //
// 2:       5
var testNodeDefsDiamond = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{4, "", TextColorDefault}, {6, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

var testNodeDefs40milPerms = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{5, "5", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{9, "9", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{10, "10", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{11, "11", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{12, "12", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{13, "13", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{14, "14", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{15, "15", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{16, "16", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{17, "17", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{18, "18", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{20, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	// {19, "19", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{21, "", TextColorDefault}}, "", 0, ThickBorder:false, UseRootColor:false},
	{19, "19", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{20, "20", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	// {21, "21", EdgeDef{}, nil, "", 0, ThickBorder:false, UseRootColor:false},
}

var testNodeDefs300bilPerms = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{5, "5", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{9, "9", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{10, "10", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{11, "11", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{12, "12", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{13, "13", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{14, "14", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{15, "15", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{16, "16", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{17, "17", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{18, "18", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{22, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{19, "19", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{23, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{20, "20", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{24, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{21, "21", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{25, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{22, "22", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{23, "23", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{24, "24", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{25, "25", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1  2
// -     | /|
// 1:    3  |
// -     | /
// 2:    4
var testNodeDefsTwoLevelsFromOneParent = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{1, "", TextColorDefault}, []EdgeDef{{2, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{2, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
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
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{5, "", TextColorDefault}, []EdgeDef{{3, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{6, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
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
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{6, "", TextColorDefault}, []EdgeDef{{4, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{7, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
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
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{5, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{5, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{6, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
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
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{6, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{8, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{6, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{7, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1
// - 	 |
// 1:    2    5
// -     |   /|
// 2:    3  / 6
// -     | //
// 3:    4
var testNodeDefsMultiSecParentPullDown = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{5, "", TextColorDefault}, {6, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{5, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1   4
// - 	 |  /|
// 1:    2 / 5
// -     |//
// 2:    3
var testNodeDefsMultiSecParentNoPullDown = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, []EdgeDef{{4, "", TextColorDefault}, {5, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{4, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1
// - 	  \\\\\\
// 1:      2 3 4 5 6
var testNodeDefsDuplicateSecLabels = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{2, "2", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},

	{7, "7", EdgeDef{2, "", TextColorDefault}, []EdgeDef{{1, "txt", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{1, "txt", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{9, "9", EdgeDef{4, "", TextColorDefault}, []EdgeDef{{1, "txt", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{10, "10", EdgeDef{5, "", TextColorDefault}, []EdgeDef{{1, "txt", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{11, "11", EdgeDef{6, "", TextColorDefault}, []EdgeDef{{1, "txt", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    5
// - 	 |
// 1:    6 - -
// -     |     \
// 2:    7   1  |
// -     |   |  |
// 3:    8   2  | 9
// -       \ |  | |
// 4:        3  | 10
// -         | /  |
// 5:        4    11
// -           \  |
// 6:             12
var testNodeDefsLayerLongRoots = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, []EdgeDef{{8, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, []EdgeDef{{6, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{5, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{6, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{7, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{9, "9", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{10, "10", EdgeDef{9, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{11, "11", EdgeDef{10, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{12, "12", EdgeDef{11, "", TextColorDefault}, []EdgeDef{{4, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}

// 0:    1
// - 	 |
// 1:    2 ------\
// -     |  \     |
// 2:    3    6   |
// -     |\   |\//
// 3:    4 5  7 8
var testNodeDefsPriAndSecInfinitePulldown = []NodeDef{
	{0, "top node", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{1, "1", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{2, "2", EdgeDef{1, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{3, "3", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{4, "4", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{5, "5", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{6, "6", EdgeDef{2, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{7, "7", EdgeDef{6, "", TextColorDefault}, []EdgeDef{{2, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	{8, "8", EdgeDef{6, "", TextColorDefault}, []EdgeDef{{2, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
}
