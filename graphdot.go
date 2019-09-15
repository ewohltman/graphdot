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
)

type Node struct {
	Name   string   `json:"name"`
	Hash   [16]byte `json:"hash"`
	GoRoot bool     `json:"-"`
}

var (
	pkgHashes          = make(map[string][16]byte)
	uniqueDependencies = make(map[string]bool)
)

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
		dependency := Node{
			Name: importPath,
			Hash: md5.Sum([]byte(importPath)),
		}

		dependency.findDependencies(ctx, pwd)

		if !dependency.GoRoot {
			pkgHashes[dependency.Name] = dependency.Hash

			mapping := fmt.Sprintf(
				"    \"%x\" -> \"%x\";\n",
				node.Hash,
				dependency.Hash,
			)

			uniqueDependencies[mapping] = true
		}
	}
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

func dotFormat(graphProps string) *bytes.Buffer {
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

	for name, hashed := range pkgHashes {
		buf.WriteString(
			fmt.Sprintf(
				"    \"%x\" [label=\"%s\"];\n",
				hashed,
				name,
			),
		)
	}

	for depMapping := range uniqueDependencies {
		buf.WriteString(depMapping)
	}

	buf.WriteString("}\n")

	return buf
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

	root := Node{
		Name: project.ImportPath,
		Hash: md5.Sum([]byte(project.Name)),
	}

	pkgHashes[root.Name] = root.Hash

	root.findDependencies(ctx, pwd)

	log.Print(dotFormat(*flagGraphProps).String())
}
