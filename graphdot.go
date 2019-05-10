package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"go/build"
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
		err = fmt.Errorf("error importing dependency package: %s", err)

		log.SetOutput(os.Stderr)
		log.Fatalf("%+v", err)
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

func dotFormat() *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	buf.WriteString("digraph {\n")
	buf.WriteString("    pad=.25;\n")
	buf.WriteString("    ratio=\"fill\";\n")
	buf.WriteString("    dpi=360;\n")
	buf.WriteString("    nodesep=.25;\n")
	buf.WriteString("    node [shape=box];\n")

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

	pwd, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("error determining working director: %s", err)

		log.SetOutput(os.Stderr)
		log.Fatalf("%+v", err)
	}

	ctx := &build.Default

	project, err := ctx.ImportDir(pwd, build.ImportComment)
	if err != nil {
		err = fmt.Errorf("error importing source project: %s", err)

		log.SetOutput(os.Stderr)
		log.Fatalf("%+v", err)
	}

	root := Node{
		Name: project.ImportPath,
		Hash: md5.Sum([]byte(project.Name)),
	}

	pkgHashes[root.Name] = root.Hash

	root.findDependencies(ctx, pwd)

	log.Print(dotFormat().String())
}
