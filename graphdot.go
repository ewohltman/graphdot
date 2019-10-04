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
	"sort"
	"strings"
)

type Node struct {
	Name         string   `json:"name"`
	Hash         [16]byte `json:"hash"`
	GoRoot       bool     `json:"-"`
	Caller       *Node    `json:"caller"`
	Dependencies []*Node  `json:"dependencies"`
}

func (node *Node) String() string {
	if node.Dependencies == nil {
		return ""
	}

	buf := bytes.NewBuffer([]byte{})

	for _, dependency := range node.Dependencies {
		buf.WriteString(node.Name + " -> " + dependency.Name + "\n")
		buf.WriteString(dependency.String())
	}

	return buf.String()
}

func (node *Node) dotFormat() string {
	buf := bytes.NewBuffer([]byte{})

	buf.WriteString(
		fmt.Sprintf(
			"    \"%x\" [label=\"%s\"];\n",
			node.Hash,
			node.Name,
		),
	)

	if node.Dependencies == nil {
		return buf.String()
	}

	for _, dependency := range node.Dependencies {
		buf.WriteString(
			fmt.Sprintf(
				"    \"%x\" -> \"%x\";\n",
				node.Hash,
				dependency.Hash,
			),
		)

		buf.WriteString(dependency.dotFormat())
	}

	return buf.String()
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

func insertGraphProps(wr io.Writer, graphProps string) {
	if _, err := os.Stat(graphProps); os.IsNotExist(err) {
		fmt.Fprintf(wr, "    %s;\n", graphProps)
	} else {
		rd, err := os.Open(graphProps)
		if err != nil {
			log.Fatal("graph props file:", err)
		}
		defer rd.Close()
		_, err = io.Copy(wr, rd)
		if err != nil {
			log.Fatal("graph props file:", err)
		}
	}
}

func dotFormat(root *Node, graphProps string) string {
	buf := bytes.NewBuffer([]byte{})

	buf.WriteString("digraph {\n")
	switch {
	case len(graphProps) == 0:
		buf.WriteString("    pad=.25;\n")
		buf.WriteString("    ratio=\"fill\";\n")
		buf.WriteString("    dpi=360;\n")
		buf.WriteString("    nodesep=.25;\n")
		buf.WriteString("    node [shape=box];\n")
	case graphProps != "none":
		insertGraphProps(buf, graphProps)
	}

	tokens := dotSort(strings.Split(root.dotFormat(), "\n"))

	buf.WriteString(strings.Join(tokens, "\n") + "\n")

	buf.WriteString("}\n")

	return buf.String()
}

func dotSort(tokens []string) []string {
	uniqueTokens := make(map[string]bool)

	for _, token := range tokens {
		if token == "" {
			continue
		}

		uniqueTokens[token] = true
	}

	labels := make([]string, 0)
	mappings := make([]string, 0)

	for uniqueToken := range uniqueTokens {
		if strings.Contains(uniqueToken, "label=") {
			labels = append(labels, uniqueToken)
		} else {
			mappings = append(mappings, uniqueToken)
		}
	}

	sort.Strings(labels)
	sort.Strings(mappings)

	sortedTokens := make([]string, 0)

	for _, label := range labels {
		sortedTokens = append(sortedTokens, label)
	}

	for _, mapping := range mappings {
		sortedTokens = append(sortedTokens, mapping)
	}

	return sortedTokens
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	flagGraphProps := flag.String("graph-props", "",
		`Select a file to be inserted as graph properties into the dot output
file. If not set some default properties will be inserted. When set to
'none' no properties will be inserted. If the filename does not exists,
the value will be inserted as a graph property.`)
	flag.StringVar(flagGraphProps, "p", "", "Short for -graph-props")
	flag.Parse()

	pwd, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("unable to determine working directory: %s", err)

		log.SetOutput(os.Stderr)
		log.Fatalf("Error: %+v", err)
	}

	ctx := &build.Default

	project, err := ctx.ImportDir(pwd, build.ImportComment)
	if err != nil {
		err = fmt.Errorf("unable to import source project: %s", err)

		log.SetOutput(os.Stderr)
		log.Fatalf("Error: %+v", err)
	}

	root := &Node{
		Name: project.ImportPath,
		Hash: md5.Sum([]byte(project.Name)),
	}

	root.findDependencies(ctx, pwd)
	root.groupPackages()

	log.Print(dotFormat(*flagGraphProps).String())
}
