package main

import (
	"fmt"
	"heimdall/core"
	"sync"
)

func main() {
	core.InitTSO()
	var wg sync.WaitGroup
	numRequests := 10000

	results := make(chan int64, numRequests)
	fmt.Printf("Launching %d concurrent requests to the Oracle...\n", numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			ts := core.Oracle.GetNextTimestamp()
			results <- ts
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[int64]bool)
	duplicates := 0

	for ts := range results {
		if seen[ts] {
			duplicates++
		}
		seen[ts] = true
	}

	fmt.Printf("Total Unique Timestamps Generated: %d\n", len(seen))
	fmt.Printf("Duplicates Found: %d\n", duplicates)

	if len(seen) == numRequests && duplicates == 0 {
		fmt.Println("SUCCESS: The Oracle is thread-safe!")
	} else {
		fmt.Println("FAILURE: Race condition detected! The Mutex failed.")
	}
}
