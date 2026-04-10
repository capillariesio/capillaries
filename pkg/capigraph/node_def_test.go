package capigraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoop(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{1, "1", EdgeDef{3, "", TextColorDefault}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "2", EdgeDef{}, []EdgeDef{{1, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{3, "3", EdgeDef{2, "", TextColorDefault}, []EdgeDef{{2, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}
	assert.Equal(t, "3<=2<-1<=3", checkNodeDef(3, nodeDefs).Error())
}

func TestCheckBadParents(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{1, "1", EdgeDef{}, []EdgeDef{{2, "", TextColorDefault}}, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "2", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}
	assert.Equal(t, "cannot process node def 1: it has no primary parent, but has secondary parents like 2", checkNodeDef(1, nodeDefs).Error())
}

func TestCheckNodeIds(t *testing.T) {
	nodeDefs := []NodeDef{
		{0, "0", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{2, "2", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
		{3, "3", EdgeDef{}, nil, "", 0, NodeBorderRegular, TextColorDefault, NodeBackgroundSolid, ""},
	}
	assert.Equal(t, "cannot process node at index 1, it has id 2; nodes must be arranged by id from 1", checkNodeIds(nodeDefs).Error())
}
