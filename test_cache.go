package main

import (
	"fmt"
	"os"
	"path/filepath"

	"heimdall/core"
)

func main() {
	os.MkdirAll("disk_storage", 0755)

	core.InitCache(2)

	hashes := []string{"hashA", "hashB", "hashC"}
	for _, h := range hashes {
		os.WriteFile(filepath.Join("disk_storage", h), []byte("fake data for "+h), 0644)
	}

	fmt.Println("Phase 1: Initial Loading")
	for _, h := range hashes {
		_, hit, _ := core.GetChunk(h)
		fmt.Printf("Requested %s -> Cache Hit: %t (Loaded from Disk)\n", h, hit)
	}

	fmt.Println("\nPhase 2: The Eviction Check")

	_, hitB, _ := core.GetChunk("hashB")
	fmt.Printf("Requested hashB -> Cache Hit: %t (Expected: true, still in RAM)\n", hitB)

	_, hitC, _ := core.GetChunk("hashC")
	fmt.Printf("Requested hashC -> Cache Hit: %t (Expected: true, still in RAM)\n", hitC)

	_, hitA, _ := core.GetChunk("hashA")
	fmt.Printf("Requested hashA -> Cache Hit: %t (Expected: false, it was evicted!)\n", hitA)

	for _, h := range hashes {
		os.Remove(filepath.Join("disk_storage", h))
	}
}
