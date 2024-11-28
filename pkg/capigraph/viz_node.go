package capigraph

import (
	"fmt"
	"strings"
)

type HierarchyType int

const (
	HierarchyPri HierarchyType = iota
	HierarchySec
)

type VizNode struct {
	Def                         *NodeDef
	IncomingVizEdges            []VizEdge
	RootId                      int16
	Layer                       int
	TotalW                      float64
	X                           float64 // Total X
	Y                           float64
	NodeW                       float64
	NodeH                       float64
	PriChildrenAndEnclosedRoots []*VizNode
}

func (vn *VizNode) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Id:%d RootId:%d Layer:%d [", vn.Def.Id, vn.RootId, vn.Layer))
	for i, c := range vn.PriChildrenAndEnclosedRoots {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%d", c.Def.Id))
	}
	sb.WriteString(fmt.Sprintf("] X:%.2f Y:%.2f TotalW:%.2f NodeW:%.2f NodeH:%.2f", vn.X, vn.Y, vn.TotalW, vn.NodeW, vn.NodeH))
	return sb.String()
}

func (vn *VizNode) cleanPropertiesSubjectToPermutation() {
	// Do not clean up static properties like Def, NodeW, NodeH etc
	vn.PriChildrenAndEnclosedRoots = vn.PriChildrenAndEnclosedRoots[:0] // Reset size, not capacity
	vn.TotalW = 0.0
	vn.X = 0.0
	vn.Y = 0.0
}

type VizEdge struct {
	Edge          EdgeDef
	HierarchyType HierarchyType
	X             float64
	Y             float64
	W             float64
	H             float64
}
