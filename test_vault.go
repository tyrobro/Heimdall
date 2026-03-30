package main

import (
	"fmt"
	"log"
	"os"

	"heimdall/core"
)

func main() {
	err := core.InitVault()
	if err != nil {
		log.Fatalf("Failed to init vault: %v", err)
	}
	defer core.CloseVault()

	originalFile := "test_document.txt"
	createDummyFile(originalFile, 1000000)

	fmt.Println("Saving the file to Heimdall's Vault")
	err = core.SaveFile(originalFile, "document_v1")
	if err != nil {
		log.Fatalf("Save failed: %v", err)
	}

	fmt.Println("Deleting original file from system")
	os.Remove(originalFile)

	restoredFile := "restored_document.txt"
	fmt.Println("Restoring file from Heimdall's Vault")
	err = core.RestoreFile("document_v1", restoredFile)
	if err != nil {
		log.Fatalf("Restore failed: %v", err)
	}

	fmt.Println("Success")
	fmt.Println("You should be able to see a 'restored_document.txt', a 'disk_storage' folder with raw chunks, and 'metadata.db'.")
}

func createDummyFile(filename string, repeat int) {
	f, _ := os.Create(filename)
	defer f.Close()
	for i := 0; i < repeat; i++ {
		f.WriteString(fmt.Sprintf("This is unique data block number %d. \n", i))
	}
}
