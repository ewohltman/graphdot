package main

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"os/exec"
	"strings"
)

type (
	dependencyMap map[string][]string
	packageHashes map[string]uint32
)

func mapDependencies(dependencies []string) (packageHashes, dependencyMap) {
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
}

func packageHash(dependency string) uint32 {
	hash := fnv.New32a()

	_, err := hash.Write([]byte(dependency))
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error hashing package: %s\n", err)
	}

	return hash.Sum32()
}

func dotFormat(hashes packageHashes, dependsUpon dependencyMap) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	buf.WriteString("digraph {\n")
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
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	_, err := os.Stat("go.mod")
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error finding go.mod file: %s\n", err)
	}

	err = os.Setenv("GO111MODULE", "on")
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error setting environment variable GO111MODULE to on: %s\n", err)
	}

	cmdOutput, err := exec.
		Command("go", "mod", "graph").
		Output()
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error running go mod graph: %s\n", err)
	}

	if len(cmdOutput) == 0 {
		log.Println("No module dependencies to graph")
		os.Exit(0)
	}

	log.Println(
		dotFormat(
			mapDependencies(
				strings.Split(
					string(cmdOutput), "\n",
				),
			),
		).String(),
	)
}
