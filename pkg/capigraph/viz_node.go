package capigraph

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

func (vn *VizNode) clean() {
	vn.Layer = 0
	vn.PriChildrenAndEnclosedRoots = vn.PriChildrenAndEnclosedRoots[:0] // Reset size, not capacity
	vn.TotalW = 0.0
	vn.X = 0.0
	vn.Y = 0.0
	vn.NodeW = 0.0
	vn.NodeH = 0.0
}

type VizEdge struct {
	Edge          EdgeDef
	HierarchyType HierarchyType
	X             float64
	Y             float64
	W             float64
	H             float64
}
