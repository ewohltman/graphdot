package main

import (
	"bytes"
	"fmt"
	"go/build"
	"hash"
	"hash/fnv"
	"log"
	"os"
)

type Node struct {
	Name         string `json:"name"`
	Hash         uint32 `json:"hash"`
	GoRoot       bool   `json:"-"`
	Dependencies []Node `json:"dependencies"`
}

func (node *Node) findDeps(ctx *build.Context, pwd string, hashAlgo hash.Hash32) error {
	pkg, err := ctx.Import(node.Name, pwd, build.ImportComment)
	if err != nil {
		return err
	}

	if pkg.Goroot {
		node.GoRoot = true
		return nil
	}

	if pkg.Imports == nil {
		return nil
	}

	for _, name := range pkg.Imports {
		dependencyNode := Node{
			Name: name,
			Hash: hashName(hashAlgo, name),
		}

		err = dependencyNode.findDeps(ctx, pwd, hashAlgo)
		if err != nil {
			return err
		}

		if !dependencyNode.GoRoot {
			node.Dependencies = append(node.Dependencies, dependencyNode)
		}
	}

	return nil
}

func (node *Node) mapHashes(nameHashes map[string]uint32) {
	nameHashes[node.Name] = node.Hash

	if node.Dependencies == nil {
		return
	}

	for _, dep := range node.Dependencies {
		dep.mapHashes(nameHashes)
	}
}

func (node *Node) walkDeps(buf *bytes.Buffer) {
	if node.Dependencies == nil {
		return
	}

	for _, dep := range node.Dependencies {
		buf.WriteString(
			fmt.Sprintf(
				"    %d -> %d;\n",
				node.Hash,
				dep.Hash,
			),
		)

		dep.walkDeps(buf)
	}

}

func hashName(hashAlgo hash.Hash32, name string) uint32 {
	_, err := hashAlgo.Write([]byte(name))
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error hashing package: %s\n", err)
	}

	return hashAlgo.Sum32()
}

func dotFormat(root Node) *bytes.Buffer {
	nameHashes := make(map[string]uint32)
	root.mapHashes(nameHashes)

	buf := bytes.NewBuffer([]byte{})

	buf.WriteString("digraph {\n")
	// buf.WriteString("    size=\"11,6!\";\n")
	buf.WriteString("    pad=.25;\n")
	buf.WriteString("    ratio=\"fill\";\n")
	buf.WriteString("    dpi=360;\n")
	buf.WriteString("    nodesep=.25;\n")
	buf.WriteString("    node [shape=box];\n")

	for name, hashed := range nameHashes {
		buf.WriteString(
			fmt.Sprintf(
				"    %d [label=\"%s\"];\n",
				hashed,
				name,
			),
		)
	}

	root.walkDeps(buf)

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

	hashAlgo := fnv.New32a()

	root := Node{
		Name: pkgMain.Name,
		Hash: hashName(hashAlgo, pkgMain.Name),
	}

	for _, imprt := range pkgMain.Imports {
		dependencyNode := Node{
			Name: imprt,
			Hash: hashName(hashAlgo, imprt),
		}

		err = dependencyNode.findDeps(ctx, pwd, hashAlgo)
		if err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("Error determining dependency package: %s\n", err)
		}

		if !dependencyNode.GoRoot {
			root.Dependencies = append(root.Dependencies, dependencyNode)
		}
	}

	buf := dotFormat(root)

	log.Printf("%s", buf.String())
}
