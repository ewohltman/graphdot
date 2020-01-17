package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/awalterschulze/gographviz/ast"
)

const (
	usageP          = "Short for -graph-props"
	usageGraphProps = `
Select a file to be inserted as graph properties into the dot output file. If
not set some default properties will be inserted. When set to 'none' no
properties will be inserted. If the filename does not exists, the value will be
inserted as a graph property.
`
)

type Node struct {
	Name         string   `json:"name"`
	Hash         [16]byte `json:"hash"`
	GoRoot       bool     `json:"-"`
	Caller       *Node    `json:"caller"`
	Dependencies []*Node  `json:"dependencies"`
}

func (node *Node) findDependencies(ctx *build.Context, pwd string) {
	if node.Name == "C" {
		return
	}

	pkg, err := ctx.Import(node.Name, pwd, build.ImportComment)
	if err != nil {
		err = fmt.Errorf("unable to import dependency package: %s", err)

		log.SetOutput(os.Stderr)
		log.Fatalf("Error: %+v", err)
	}

	if pkg.Goroot {
		node.GoRoot = true
		return
	}

	if pkg.Imports == nil {
		return
	}

	for _, importPath := range pkg.Imports {
		dependency := &Node{
			Name:   importPath,
			Hash:   md5.Sum([]byte(importPath)),
			Caller: node,
		}

		dependency.findDependencies(ctx, pwd)

		if !dependency.GoRoot {
			node.Dependencies = append(node.Dependencies, dependency)
		}
	}
}

func (node *Node) groupPackages() {
	for _, dependency := range node.Dependencies {
		dependency.groupPackages()
	}

	if node.Caller == nil {
		return
	}

	nodeTokens := strings.Split(node.Name, "/")
	callerTokens := strings.Split(node.Caller.Name, "/")

	// Handle special case for non-standard imports, e.g. k8s.io/api
	if len(nodeTokens) < 3 || len(callerTokens) < 3 {
		nodeTokens = nodeTokens[:2]
		callerTokens = callerTokens[:2]
	} else {
		nodeTokens = nodeTokens[:3]
		callerTokens = callerTokens[:3]
	}

	nodeProject := strings.Join(nodeTokens, "/")
	callerProject := strings.Join(callerTokens, "/")

	if nodeProject != callerProject {
		return
	}

	toKeep := make([]*Node, 0)

	for _, callerDependency := range node.Caller.Dependencies {
		if callerDependency.Name != node.Name {
			toKeep = append(toKeep, callerDependency)
		}
	}

	for _, nodeDependency := range node.Dependencies {
		toKeep = append(toKeep, nodeDependency)
	}

	node.Caller.Dependencies = toKeep
}

func (node *Node) buildGraph(graph *gographviz.Graph) error {
	nodeHash := fmt.Sprintf(`"%x"`, node.Hash)

	if !graph.IsNode(nodeHash) {
		nodeProperties := map[string]string{
			"label":    fmt.Sprintf(`"%s"`, node.Name),
			"fontname": "helvetica",
			"shape":    "box",
		}

		err := graph.AddNode("dependencies", nodeHash, nodeProperties)
		if err != nil {
			return err
		}
	}

	for _, dependency := range node.Dependencies {
		dependencyHash := fmt.Sprintf(`"%x"`, dependency.Hash)

		if !graph.IsNode(dependencyHash) {
			dependencyProperties := map[string]string{
				"label":    fmt.Sprintf(`"%s"`, dependency.Name),
				"fontname": "helvetica",
				"shape":    "box",
			}

			err := graph.AddNode("dependencies", dependencyHash, dependencyProperties)
			if err != nil {
				return err
			}
		}

		err := graph.AddEdge(nodeHash, dependencyHash, true, nil)
		if err != nil {
			return err
		}

		err = dependency.buildGraph(graph)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildGraphAST(graphPropsFilePath string) (*ast.Graph, error) {
	buf := bytes.NewBuffer([]byte{})

	buf.WriteString("strict digraph dependencies {\n")

	switch {
	case len(graphPropsFilePath) == 0:
		buf.WriteString("    pad=.25;\n")
		buf.WriteString("    ratio=fill;\n")
		buf.WriteString("    dpi=360;\n")
		buf.WriteString("    nodesep=.25;\n")
		buf.WriteString("    node [shape=box fontname=helvetica];\n")
	case graphPropsFilePath != "none":
		err := insertGraphProps(buf, graphPropsFilePath)
		if err != nil {
			return nil, err
		}
	}

	buf.WriteString("}")

	graphAst, err := gographviz.ParseString(buf.String())
	if err != nil {
		return nil, err
	}

	return graphAst, nil
}

func insertGraphProps(writer io.Writer, graphPropsFilePath string) (err error) {
	_, err = os.Stat(graphPropsFilePath)
	if os.IsNotExist(err) {
		err = fmt.Errorf("%s file does not exist", graphPropsFilePath)
		return
	}

	var graphPropsFile *os.File

	graphPropsFile, err = os.Open(graphPropsFilePath)
	if err != nil {
		return
	}

	defer func() {
		err = graphPropsFile.Close()
		if err != nil {
			return
		}
	}()

	_, err = io.Copy(writer, graphPropsFile)

	return
}

func main() {
	var graphPropsFilePath string

	flag.StringVar(&graphPropsFilePath, "p", "", usageP)
	flag.StringVar(&graphPropsFilePath, "graph-props", "", strings.TrimSpace(usageGraphProps))
	flag.Parse()

	log.SetFlags(0)

	graphAst, err := buildGraphAST(graphPropsFilePath)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	graph := gographviz.NewGraph()

	err = gographviz.Analyse(graphAst, graph)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("unable to determine working directory: %w", err)

		log.Fatalf("Error: %s", err)
	}

	ctx := &build.Default

	project, err := ctx.ImportDir(pwd, build.ImportComment)
	if err != nil {
		err = fmt.Errorf("unable to import source project: %w", err)

		log.Fatalf("Error: %s", err)
	}

	root := &Node{
		Name: project.ImportPath,
		Hash: md5.Sum([]byte(project.Name)),
	}

	root.findDependencies(ctx, pwd)

	root.groupPackages()

	err = root.buildGraph(graph)
	if err != nil {
		err = fmt.Errorf("unable to build dependency graph: %w", err)

		log.Fatalf("Error: %s", err)
	}

	fmt.Println(graph)
}
