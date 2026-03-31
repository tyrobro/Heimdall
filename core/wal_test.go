package core

import (
	"os"
	"testing"
)

func TestWALRecovery(t *testing.T) {
	os.Remove("heimdall.wal")
	os.Remove("metadata.db")
	InitTSO()

	testFile := "crash_data.txt"
	content := []byte("this data survived a crash")
	os.WriteFile(testFile, content, 0644)
	defer os.Remove(testFile)

	ts := int64(99)
	err := AsyncSave(testFile, "persistent_file.txt", ts)
	if err != nil {
		t.Fatalf("AsyncSave failed: %v", err)
	}

	err = ReplayWAL()
	if err != nil {
		t.Fatalf("ReplayWAL failed: %v", err)
	}

	_, err = GetFileRecipe("persistent_file.txt", ts)
	if err != nil {
		t.Errorf("Data loss detected! WAL failed to recover transaction.")
	}
}
