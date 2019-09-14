# graphdot

**Visualize Go module dependencies in Graphviz DOT format**

----
## Why Fork

This is a fork to implement a feature to configure the graph properties. Tha original
version has hard coded properties that make generation of e.g. SVG graphics fail,
i.e. dot will render unusable SVGs.

The idea is to have an option that either supresses the graph proerties completely
or let's the user specify a file that has custom graph proerties.

The goal is to merge back into the [original project](https://github.com/ewohltman/graphdot).

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

## Contributing

Contributions are very welcome, however please follow the below guidelines.

* Open an issue describing the bug or enhancement
* Fork the `develop` branch and make your changes
* Create a Pull Request with your changes against the `develop` branch
