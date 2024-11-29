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
	sb.WriteString(fmt.Sprintf("{Id:%d RootId:%d Layer:%d ", vn.Def.Id, vn.RootId, vn.Layer))
	sb.WriteString(fmt.Sprintf("X:%.2f Y:%.2f TotalW:%.2f W:%.2f H:%.2f ", vn.X, vn.Y, vn.TotalW, vn.NodeW, vn.NodeH))
	sb.WriteString("In:[")
	for i, e := range vn.IncomingVizEdges {
		if i > 0 {
			sb.WriteString(" ")
		}
		ht := "pri"
		if e.HierarchyType == HierarchySec {
			ht = "sec"
		}
		sb.WriteString(fmt.Sprintf("{HT:%s SrcId:%d X:%.2f Y:%.2f W:%.2f H:%.2f}", ht, e.Edge.SrcId, e.X, e.Y, e.W, e.H))
	}
	sb.WriteString("] ")
	sb.WriteString("PriRootOut:[")
	for i, c := range vn.PriChildrenAndEnclosedRoots {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%d", c.Def.Id))
	}
	sb.WriteString("]")
	sb.WriteString("}")
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
