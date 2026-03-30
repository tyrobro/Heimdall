package main

import (
	"fmt"
	"log"
	"os"

	"heimdall/core"
)

func createDummyFile(filename, content string, repeat int) {
	f, _ := os.Create(filename)
	defer f.Close()
	for i := 0; i < repeat; i++ {
		f.WriteString(content)
	}
}

func main() {
	core.InitVault()
	defer core.CloseVault()
	core.InitTSO()

	fileName := "company_budget.txt"
	storageName := "budget_doc"

	ts1 := core.Oracle.GetNextTimestamp()
	createDummyFile(fileName, "Q1 Budget: $10,000\n", 5000)
	core.SaveFile(fileName, storageName, ts1)
	fmt.Printf("Saved Version 1 at Timestamp %d\n", ts1)

	ts2 := core.Oracle.GetNextTimestamp()
	createDummyFile(fileName, "Q2 Budget: $50,000 (INCREASED!)\n", 5000)
	core.SaveFile(fileName, storageName, ts2)
	fmt.Printf("Saved Version 2 at Timestamp %d\n", ts2)

	ts3 := core.Oracle.GetNextTimestamp()
	createDummyFile(fileName, "Q3 Budget: $0 (BANKRUPT)\n", 5000)
	core.SaveFile(fileName, storageName, ts3)
	fmt.Printf("Saved Version 3 at Timestamp %d\n", ts3)

	os.Remove(fileName)

	fmt.Println("\nInitiating Time Travel")

	fmt.Printf("Requesting file state at Timestamp %d...\n", ts2)

	restoredName := "restored_budget.txt"
	err := core.RestoreFile(storageName, restoredName, ts2)
	if err != nil {
		log.Fatal(err)
	}

	content, _ := os.ReadFile(restoredName)
	fmt.Printf("File Contents from the Past: %s\n", string(content[:45]))

	os.Remove(restoredName)
}
