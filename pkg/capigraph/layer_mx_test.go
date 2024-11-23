package capigraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 0:    1
// -     |  \
// 1:    2     4
// -     |     |
// 2:    3     5
func TestTrivial(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{}, ""},
	}
	priParentMap := buildPriParentMap(nodeDefs)

	mx := LayerMx{
		{1},
		{2, 4},
		{3, 5},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1},
		{2, 4},
		{5, 3},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))
}

// 0:    1
// - 	 | \
// 1:    2   4
// -     |   |
// 2:    F3  5
// -     | /
// 3:    3
func TestOneFake(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{{5, ""}}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{}, ""},
	}
	priParentMap := buildPriParentMap(nodeDefs)

	mx := LayerMx{
		{1},
		{2, 4},
		{10003, 5},
		{3},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1},
		{4, 2},
		{10003, 5},
		{3},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))
}

// 0:    1
// - 	 | \
// 1:    2   4
// -     |   |
// 2:    F3  5
// -     |   |
// 3:    F3  6
// -     | /
// 4:    3
func TestTwoFake(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{2, ""}, []EdgeDef{{6, ""}}, ""},
		{4, "4", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{}, ""},
		{6, "6", EdgeDef{5, ""}, []EdgeDef{}, ""},
	}
	priParentMap := buildPriParentMap(nodeDefs)
	mx := LayerMx{
		{1},
		{2, 4},
		{10003, 5},
		{10003, 6},
		{3},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1},
		{4, 2},
		{10003, 5},
		{6, 10003},
		{3},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1},
		{2, 4},
		{5, 10003},
		{10003, 6},
		{3},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))
}

// 0:    1  3
// -     |  |
// 1:    2  4  6
// -        | /
// 2:       5
func TestParentless(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "top node", EdgeDef{}, []EdgeDef{}, ""},
		{1, "1", EdgeDef{}, []EdgeDef{}, ""},
		{2, "2", EdgeDef{1, ""}, []EdgeDef{}, ""},
		{3, "3", EdgeDef{}, []EdgeDef{}, ""},
		{4, "4", EdgeDef{3, ""}, []EdgeDef{}, ""},
		{5, "5", EdgeDef{4, ""}, []EdgeDef{{6, ""}}, ""},
		{6, "6", EdgeDef{}, []EdgeDef{}, ""},
	}
	priParentMap := buildPriParentMap(nodeDefs)
	var mx LayerMx

	mx = LayerMx{
		{3, 1},
		{6, 4, 2},
		{5},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{3, 1},
		{4, 6, 2},
		{5},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{3, 1},
		{4, 2, 6},
		{5},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{3, 1},
		{6, 2, 4},
		{5},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{3, 1},
		{2, 6, 4},
		{5},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{3, 1},
		{2, 4, 6},
		{5},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1, 3},
		{2, 4, 6},
		{5},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1, 3},
		{6, 2, 4},
		{5},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1, 3},
		{2, 6, 4},
		{5},
	}
	assert.True(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1, 3},
		{4, 2, 6},
		{5},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1, 3},
		{4, 6, 2},
		{5},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))

	mx = LayerMx{
		{1, 3},
		{6, 4, 2},
		{5},
	}
	assert.False(t, mx.isMonotonous(priParentMap, len(nodeDefs)))
}
