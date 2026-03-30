package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

type LogEntry struct {
	Timestamp int64  `json:"timestamp"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
}

var walFile *os.File
var walMutex sync.Mutex
var processQueue chan LogEntry

func InitWAL() error {
	var err error
	walFile, err = os.OpenFile("heimdall.wal", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	processQueue = make(chan LogEntry, 10000)
	go backgroundProcessor()
	return nil
}

func AsyncSave(filePath, fileName string, timestamp int64) error {
	entry := LogEntry{
		Timestamp: timestamp,
		FileName:  fileName,
		FilePath:  filePath,
	}

	data, _ := json.Marshal(entry)
	data = append(data, '\n')

	walMutex.Lock()
	_, err := walFile.Write(data)
	walMutex.Unlock()

	if err != nil {
		return fmt.Errorf("WAL write failed: %w", err)
	}

	processQueue <- entry
	return nil
}

func backgroundProcessor() {
	for entry := range processQueue {
		err := SaveFile(entry.FilePath, entry.FileName, entry.Timestamp)
		if err != nil {
			log.Printf("Background processing failed for %s: %v", entry.FileName, err)
		}
	}
}

func CloseWAL() {
	if walFile != nil {
		walFile.Close()
	}
	close(processQueue)
}
