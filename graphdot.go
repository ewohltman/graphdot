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
	Name         string   `json:"name"`
	Hash         [16]byte `json:"hash"`
	GoRoot       bool     `json:"-"`
	Dependencies []Node   `json:"dependencies"`
}

var (
	pkgHashes          = make(map[string][16]byte)
	uniqueDependencies = make(map[string]bool)
)

func (node *Node) findDeps(ctx *build.Context, pwd string) {
	pkg, err := ctx.Import(node.Name, pwd, build.ImportComment)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("Error determining dependency package: %s", err)
	}

	if pkg.Goroot {
		node.GoRoot = true
		return
	}

	if pkg.Imports == nil {
		return
	}

	for _, name := range pkg.Imports {
		dependencyNode := Node{
			Name: name,
			Hash: md5.Sum([]byte(name)),
		}

		dependencyNode.findDeps(ctx, pwd)

		if !dependencyNode.GoRoot {
			pkgHashes[dependencyNode.Name] = dependencyNode.Hash
			node.Dependencies = append(node.Dependencies, dependencyNode)
		}
	}
}

func (node *Node) walkDeps() {
	if node.Dependencies == nil {
		return
	}

	if node.GoRoot {
		return
	}

	for _, dep := range node.Dependencies {
		if dep.GoRoot {
			continue
		}

		mapping := fmt.Sprintf(
			"    \"%x\" -> \"%x\";\n",
			node.Hash,
			dep.Hash,
		)

		uniqueDependencies[mapping] = true

		dep.walkDeps()
	}
}

func dotFormat(root Node) *bytes.Buffer {
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

	root.walkDeps()

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
		log.SetOutput(os.Stderr)
		log.Fatalf("Error determining working directory: %s\n", err)
	}

	ctx := &build.Default

	pkgMain, err := ctx.ImportDir(pwd, build.ImportComment)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error determining main package: %s\n", err)
	}

	root := Node{
		Name: pkgMain.Name,
		Hash: md5.Sum([]byte(pkgMain.Name)),
	}

	pkgHashes[root.Name] = root.Hash

	for _, imprt := range pkgMain.Imports {
		dependencyNode := Node{
			Name: imprt,
			Hash: md5.Sum([]byte(imprt)),
		}

		dependencyNode.findDeps(ctx, pwd)

		if !dependencyNode.GoRoot {
			pkgHashes[dependencyNode.Name] = dependencyNode.Hash
			root.Dependencies = append(root.Dependencies, dependencyNode)
		}
	}

	buf := dotFormat(root)

	log.Printf("%s", buf.String())
}
