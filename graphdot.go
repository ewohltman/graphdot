package main

import (
	"go/build"
	"log"
	"os"
)

/*type (
	dependencyMap map[string][]string
	packageHashes map[string]uint32
)*/

/*func mapDependencies(dependencies []string) (packageHashes, dependencyMap) {
	hashes := make(packageHashes)
	dependsUpon := make(dependencyMap)

	for _, dependency := range dependencies {
		if dependency == "" {
			continue
		}

		relationship := strings.Split(dependency, " ")

		if _, found := hashes[relationship[0]]; !found {
			hashes[relationship[0]] = packageHash(relationship[0])
		}

		if _, found := hashes[relationship[1]]; !found {
			hashes[relationship[1]] = packageHash(relationship[1])
		}

		dependsUpon[relationship[0]] = append(
			dependsUpon[relationship[0]],
			relationship[1],
		)
	}

	return hashes, dependsUpon
}*/

/*func packageHash(dependency string) uint32 {
	hash := fnv.New32a()

	_, err := hash.Write([]byte(dependency))
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error hashing package: %s\n", err)
	}

	return hash.Sum32()
}*/

/*func dotFormat(hashes packageHashes, dependsUpon dependencyMap) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	buf.WriteString("digraph {\n")
	buf.WriteString("    size=\"11,6!\";\n")
	buf.WriteString("    pad=.25;\n")
	buf.WriteString("    ratio=\"fill\";\n")
	buf.WriteString("    dpi=360;\n")
	buf.WriteString("    nodesep=.25;\n")
	buf.WriteString("    node [shape=box];\n")

	for libraryDetails, hash := range hashes {
		library := strings.Split(libraryDetails, "@")
		libraryName := ""
		libraryVersion := ""

		if len(library) < 2 {
			libraryName = library[0]
		} else {
			libraryName = library[0]
			libraryVersion = library[1]
		}

		nodeLabel := ""

		if libraryVersion != "" {
			nodeLabel = fmt.Sprintf(
				"    %v [label=\"%s\\n%s\"];\n",
				hash,
				libraryName,
				libraryVersion,
			)
		} else {
			nodeLabel = fmt.Sprintf(
				"    %v [label=\"%s\"];\n",
				hash,
				libraryName,
			)
		}

		buf.WriteString(nodeLabel)
	}

	for child, parents := range dependsUpon {
		for _, parent := range parents {
			buf.WriteString(
				fmt.Sprintf(
					"    %v -> %v;\n",
					hashes[child],
					hashes[parent],
				),
			)
		}
	}

	buf.WriteString("}")

	return buf
}*/

type Node struct {
	Name         string
	GoRoot       bool
	Dependencies []Node
}

func (node *Node) findImports(ctx *build.Context) error {
	pkg, err := ctx.Import(node.Name, ".", build.ImportComment)
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
		}

		err = dependencyNode.findImports(ctx)
		if err != nil {
			return err
		}

		if !dependencyNode.GoRoot {
			node.Dependencies = append(node.Dependencies, dependencyNode)
		}
	}

	return nil
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	/*err := os.Setenv("GO111MODULE", "on")
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error setting environment variable GO111MODULE to on: %s\n", err)
	}*/

	ctx := &build.Default

	root, err := ctx.ImportDir(".", build.ImportComment)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error determining root package: %s\n", err)
	}

	rootNode := Node{
		Name: root.Name,
	}

	for _, imprt := range root.Imports {
		dependencyNode := Node{
			Name: imprt,
		}

		err = dependencyNode.findImports(ctx)
		if err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("Error determining dependency package: %s\n", err)
		}

		if !dependencyNode.GoRoot {
			rootNode.Dependencies = append(rootNode.Dependencies, dependencyNode)
		}
	}

	for _, depen := range rootNode.Dependencies {
		log.Printf("%+v\n", depen)
	}
}
