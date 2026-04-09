Capigraph is a Go library, it started as part of the [Capillaries](https://github.com/capillariesio/capillaries) project, now it's a spin-off.
It is pretty opinionated and it is used by Capillaries to draw process diagrams in Capillaries WebUI. It generates SVG diagrams from definitions in Go programs.

## Highlights.

1. DAG only.
2. Each node can have zero or one incoming primary edge (solid arrows), and zero or more incoming secondary edges (dashed arrows).
3. Nodes are arranged as if they are executed in stages, top to bottom.
4. Secondary edges can overlap with nodes and other edges (this is why nodes and edge labels are semi-transparent).
5. Optional coloring by root node.
6. Optional thick border for selected nodes
7. Optional multi-line text for nodes and edges
8. Optional node icons (SVG)
9. Support for user-defined node backgrounds (SVG).

## Examples


## Q&A

Q. Why do we even need unoptimized mode?
A. This is for the cases when the number of possible permutations of node positions on each level is too large. For example, check out the prefix tree unit test, it builds a ludicrously big (and unusable) diagram.

Q. Any plans to make this library more generic and support other types of diagrams?
A. No.

Q. Any plans to come up with a diagram definition language like DOT language for Graphviz?
A. No. But it should not be hard to create a Go tool that reads diagram definitions from files and generates diagrams.

[MIT License](LICENSE)

(C) 2022-2026 KH (kleines.hertz[at]protonmail.com)