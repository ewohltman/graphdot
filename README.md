# graphdot

### Visualize Go module dependencies in Graphviz DOT format

----

## Installation
Install `graphdot` using go get:

`go get github.com/ewohltman/graphdot`

## Usage
Run `graphdot` in the directory of any project using Go modules with a `go.mod`
file to print out a dependency graph in [Graphviz](https://www.graphviz.org/)
DOT format.

The output can be piped directly into `dot` to generate an image file:

`graphdot | dot -T png -o dependency_graph.png`

## Contributing

Contributions are very welcome, however please follow the below guidelines.

* Open an issue describing the bug or enhancement
* Fork the `develop` branch and make your changes
* Create a Pull Request with your changes against the `develop` branch
