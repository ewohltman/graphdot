# graphdot

**Visualize Go module dependencies in Graphviz DOT format**

----

## Installation
Install `graphdot` using go get:

`go get -u github.com/ewohltman/graphdot`

## Usage
Run `graphdot` in the directory of any project using Go modules with a `go.mod`
file to print out a dependency graph in [Graphviz](https://www.graphviz.org/)
DOT format.

The output can be piped directly into `dot` to generate a
[PNG](https://en.wikipedia.org/wiki/Portable_Network_Graphics) image file:

`graphdot | dot -T png -o dependency_graph.png`

For large graphs with many nodes of dependencies, you may want to generate an
[SVG](https://en.wikipedia.org/wiki/Scalable_Vector_Graphics) file to allow you
to zoom in with high-fidelity and save disk space instead:

`graphdot | dot -Gdpi=0 -T svg -o dependency_graph.svg`

If you like a more UML'ish style, you can use the provided graph properties
from `uml.gprops`:

`graphdot -graph-props uml.gprops | dot -T svg -o dependency_graph.svg`

## Example

![graphdot](https://raw.githubusercontent.com/ewohltman/graphdot/master/dependency_graph.png)

## Contributing to the project

Contributions are very welcome, however please follow the guidelines below:

* Open an issue describing the bug or enhancement
* Fork the `develop` branch and make your changes
  * Try to match current naming conventions as closely as possible
* Create a Pull Request with your changes against the `develop` branch
