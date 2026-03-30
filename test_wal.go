package main

import (
	"fmt"
	"os"
	"time"

	"heimdall/core"
)

func main() {
	core.InitVault()
	defer core.CloseVault()
	core.InitTSO()

	err := core.InitWAL()
	if err != nil {
		panic(err)
	}
	defer core.CloseWAL()

	fileName := "benchmark_data.txt"
	createDummyFile(fileName, "Simulated telemetry data block.\n", 1000)

	numRequests := 100

	fmt.Println("Running Benchmark")

	startSync := time.Now()
	for i := 0; i < numRequests; i++ {
		ts := core.Oracle.GetNextTimestamp()
		storageName := fmt.Sprintf("sync_file_%d", i)
		core.SaveFile(fileName, storageName, ts)
	}
	durationSync := time.Since(startSync)
	fmt.Printf("Synchronous Save Time (100 files): %v\n", durationSync)

	startAsync := time.Now()
	for i := 0; i < numRequests; i++ {
		ts := core.Oracle.GetNextTimestamp()
		storageName := fmt.Sprintf("async_file_%d", i)
		core.AsyncSave(fileName, storageName, ts)
	}
	durationAsync := time.Since(startAsync)
	fmt.Printf("Asynchronous WAL Time (100 files): %v\n", durationAsync)

	speedup := float64(durationSync.Microseconds()) / float64(durationAsync.Microseconds())
	fmt.Printf("\nSpeedup Factor: The WAL is %.2fx faster at accepting traffic.\n", speedup)

	time.Sleep(1 * time.Second)
	os.Remove(fileName)
}

func createDummyFile(filename, content string, repeat int) {
	f, _ := os.Create(filename)
	defer f.Close()
	for i := 0; i < repeat; i++ {
		f.WriteString(content)
	}
}
