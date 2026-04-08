This package draws Capillaries diagrams. May be externalized eventually.

1. DAG only
2. Each node can have zero or one incoming primary edge (solid arrows), and any number of incoming secondary edges (dashed arrows).
3. Nodes are arranged as if they are executed in stages, top to bottom.
4. Secondary edges can overlap with nodes and other edges.
5. Optional coloring by root node.
6. Optional thick border for selected nodes
7. Optional text for nodes and edges
8. Optional node icons

const customIcons100x100 = `
<g id="icon-database-table-read">
  <g transform="scale(0.56) translate(2,61)">
    <path fill-rule="evenodd"
      d="M16.49,24.88C24.05,27.41,34.57,29,46.26,29S68.48,27.41,76,24.88c6.63-2.22,10.73-4.9,10.73-7.52S82.67,12.06,76,9.84C68.48,7.33,58,5.75,46.27,5.75S24.06,7.33,16.49,9.84c-14.06,4.7-14.46,10.21,0,15ZM64.91,55.34h48.73a9.27,9.27,0,0,1,9.24,9.24v42.58a9.27,9.27,0,0,1-9.24,9.25H64.91a9.27,9.27,0,0,1-9.24-9.25V64.58a9.27,9.27,0,0,1,9.24-9.24ZM91.09,99.18H118v12H91.09v-12Zm-30.89,0H87.13v12H60.2v-12Zm0-31.89H87.13v12H60.2v-12Zm0,15.94H87.13v12H60.2v-12ZM91.09,67.29H118v12H91.09v-12Zm0,15.94H118v12H91.09v-12ZM5.82,45.77c.52,2.45,4.5,4.91,10.68,7,7.22,2.42,17.16,3.95,28.24,4.08v5.77c-11.67-.13-22.25-1.78-30.05-4.39A35.86,35.86,0,0,1,5.84,54V71.27c.52,2.45,4.5,4.91,10.68,7,7.22,2.4,17.15,3.94,28.22,4.07v5.75c-11.67-.14-22.25-1.78-30.05-4.4A36.08,36.08,0,0,1,5.83,79.5V96.75c.52,2.45,4.51,4.91,10.68,7,7.22,2.41,17.16,4,28.23,4.08v5.75c-11.67-.13-22.24-1.78-30-4.4C10.4,107.72,0,103,0,97.38V95.55C0,69.86,0,43.06,0,17.41c0-5.43,5.61-10,14.66-13C22.82,1.68,34,0,46.27,0S69.7,1.68,77.87,4.41s13.64,6.78,14.55,11.53a3,3,0,0,1,.16,1v28.6H86.8V26.09a36.69,36.69,0,0,1-8.93,4.22c-8.15,2.75-19.31,4.41-31.58,4.41S22.83,33,14.66,30.31A36.26,36.26,0,0,1,5.8,26.14V45.77Z" />
  </g>
  <g transform="scale(0.1) translate(540,20)">
    <path fill-rule="nonzero"
      d="M117.91 0h201.68c3.93 0 7.44 1.83 9.72 4.67l114.28 123.67c2.21 2.37 3.27 5.4 3.27 8.41l.06 310c0 35.43-29.4 64.81-64.8 64.81H117.91c-35.57 0-64.81-29.24-64.81-64.81V64.8C53.1 29.13 82.23 0 117.91 0zM325.5 37.15v52.94c2.4 31.34 23.57 42.99 52.93 43.5l36.16-.04-89.09-96.4zm96.5 121.3l-43.77-.04c-42.59-.68-74.12-21.97-77.54-66.54l-.09-66.95H117.91c-21.93 0-39.89 17.96-39.89 39.88v381.95c0 21.82 18.07 39.89 39.89 39.89h264.21c21.71 0 39.88-18.15 39.88-39.89v-288.3z" />
  </g>
</g>
`;

var testDiagramWithOneEnclosedLevel = []NodeDef{
	{0, "top node", EdgeDef{}, []EdgeDef{}, "icon-database-table-read", 0, NodeOptions{false}},
	{1, "1", EdgeDef{}, []EdgeDef{}, "", 0, NodeOptions{false}},
	{2, "2", EdgeDef{1, ""}, []EdgeDef{}, "", 0, NodeOptions{true}},
	{3, "3", EdgeDef{1, ""}, []EdgeDef{}, "", 0, NodeOptions{true}},
	{4, "4", EdgeDef{2, ""}, []EdgeDef{{6, ""}}, "", 0, NodeOptions{false}},
	{5, "5", EdgeDef{3, ""}, []EdgeDef{{6, ""}}, "", 0, NodeOptions{false}},
	{6, "6", EdgeDef{}, []EdgeDef{}, "", 0, NodeOptions{false}},
}

	svg, _, totalPermutations, _, bestDist, _ := DrawOptimized(testDiagramWithOneEnclosedLevel, DefaultNodeFontOptions(), DefaultEdgeLabelFontOptions(), DefaultEdgeOptions(), customIcons100x100, "", DefaultPalette())

