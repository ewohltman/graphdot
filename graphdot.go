package main

import (
	"bytes"
	"crypto/sha256"
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
	Hash         [32]byte `json:"hash"`
	GoRoot       bool     `json:"-"`
	Caller       *Node    `json:"caller"`
	Dependencies []*Node  `json:"dependencies"`
}

func (node *Node) findDependencies(ctx *build.Context, pwd string) error {
	if node.Name == "C" {
		return nil
	}

	pkg, err := ctx.Import(node.Name, pwd, build.ImportComment)
	if err != nil {
		return fmt.Errorf("unable to import dependency package: %w", err)
	}

	if pkg.Goroot {
		node.GoRoot = true
		return nil
	}

	if pkg.Imports == nil {
		return nil
	}

	for _, importPath := range pkg.Imports {
		dependency := &Node{
			Name:   importPath,
			Hash:   sha256.Sum256([]byte(importPath)),
			Caller: node,
		}

		err = dependency.findDependencies(ctx, pwd)
		if err != nil {
			return err
		}

		if !dependency.GoRoot {
			node.Dependencies = append(node.Dependencies, dependency)
		}
	}

	if pkg.TestImports == nil {
		return nil
	}

	for _, testImportPath := range pkg.TestImports {
		dependency := &Node{
			Name:   testImportPath,
			Hash:   sha256.Sum256([]byte(testImportPath)),
			Caller: node,
		}

		err = dependency.findDependencies(ctx, pwd)
		if err != nil {
			return err
		}

		if !dependency.GoRoot {
			node.Dependencies = append(node.Dependencies, dependency)
		}
	}

	return nil
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

	toKeep = append(toKeep, node.Dependencies...)

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
			return fmt.Errorf("unable to add graph node: %w", err)
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
				return fmt.Errorf("unable to add graph node: %w", err)
			}
		}

		err := graph.AddEdge(nodeHash, dependencyHash, true, nil)
		if err != nil {
			return fmt.Errorf("unable to add graph edge: %w", err)
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
	case graphPropsFilePath == "":
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
	log.SetFlags(0)

	var graphPropsFilePath string

	flag.StringVar(&graphPropsFilePath, "p", "", usageP)
	flag.StringVar(&graphPropsFilePath, "graph-props", "", strings.TrimSpace(usageGraphProps))
	flag.Parse()

	targetDirectories := flag.Args()

	if len(targetDirectories) > 1 {
		log.Fatalf("Error: more than one directory provided to be recursively evaluated")
	}

	graphAst, err := buildGraphAST(graphPropsFilePath)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	graph := gographviz.NewGraph()

	err = gographviz.Analyse(graphAst, graph)
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	if len(targetDirectories) == 0 || targetDirectories[0] == "." {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Error: unable to determine working directory: %s", err)
		}

		targetDirectories = append(targetDirectories, pwd)
	}

	for _, targetDirectory := range targetDirectories {
		ctx := &build.Default

		project, err := ctx.ImportDir(targetDirectory, build.ImportComment)
		if err != nil {
			log.Fatalf("Error: unable to import source project: %s", err)
		}

		root := &Node{
			Name: project.ImportPath,
			Hash: sha256.Sum256([]byte(project.Name)),
		}

		err = root.findDependencies(ctx, targetDirectory)
		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		root.groupPackages()

		err = root.buildGraph(graph)
		if err != nil {
			log.Fatalf("Error: unable to build dependency graph: %s", err)
		}
	}

	fmt.Println(graph)
}
