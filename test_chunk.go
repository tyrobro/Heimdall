package main

import (
	"fmt"
	"log"
	"os"

	"heimdall/core"
)

func main() {
	file1 := "test1.txt"
	createDummyFile(file1, 500000)
	file2 := "test2.txt"
	createDummyFile(file2, 500000)

	originalContent, _ := os.ReadFile(file2)
	newContent := append([]byte("This line has been inserted purely for testing purposes \n"), originalContent...)
	os.WriteFile(file2, newContent, 0644)

	chunks1, err := core.ChunkFile(file1)
	if err != nil {
		log.Fatal(err)
	}
	chunks2, err := core.ChunkFile(file2)
	if err != nil {
		log.Fatal(err)
	}

	commonChunks := 0
	for _, c2 := range chunks2 {
		for _, c1 := range chunks1 {
			if c1.Hash == c2.Hash {
				commonChunks++
				break
			}
		}
	}

	fmt.Printf("File 1 total chunks: %d\n", len(chunks1))
	fmt.Printf("File 2 total chunks: %d\n", len(chunks2))
	fmt.Printf("Identical chunks (Deduplicated): %d\n", commonChunks)

	os.Remove(file1)
	os.Remove(file2)
}

func createDummyFile(filename string, repeat int) {
	f, _ := os.Create(filename)
	defer f.Close()
	for i := 0; i < repeat; i++ {
		f.WriteString(fmt.Sprintf("This is unique data block number %d. \n", i))
	}
}
