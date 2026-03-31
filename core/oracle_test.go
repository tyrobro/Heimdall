package core

import (
	"sync"
	"testing"
)

func TestOracleMonotonicity(t *testing.T) {
	InitTSO()

	var wg sync.WaitGroup
	numRequests := 1000
	results := make(chan int64, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- Oracle.GetNextTimestamp()
		}()
	}

	wg.Wait()
	close(results)

	seen := make(map[int64]bool)
	maxVal := int64(0)

	for ts := range results {
		if seen[ts] {
			t.Fatalf("CRITICAL FAULT: Oracle generated duplicate timestamp: %d", ts)
		}
		seen[ts] = true
		if ts > maxVal {
			maxVal = ts
		}
	}

	if maxVal != int64(numRequests) {
		t.Errorf("Expected max timestamp to be %d, got %d", numRequests, maxVal)
	}
}
