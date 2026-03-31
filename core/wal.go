package core

import (
	"encoding/binary"
	"io"
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
	if err := ReplayWAL(); err != nil {
		return err
	}
	log.Println("WAL System Initialized.")
	return nil
}

func AsyncSave(tempPath string, fileName string, ts int64) error {
	data, err := os.ReadFile(tempPath)
	if err != nil {
		return err
	}

	f, err := os.OpenFile("heimdall.wal", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	binary.Write(f, binary.LittleEndian, ts)
	binary.Write(f, binary.LittleEndian, int32(len(fileName)))
	f.WriteString(fileName)
	binary.Write(f, binary.LittleEndian, int32(len(data)))
	f.Write(data)

	go func() {
		err := SaveFile(tempPath, fileName, ts)
		if err != nil {
			log.Printf("Background Vault save failed for %s: %v", fileName, err)
		}
	}()

	return nil
}

func ReplayWAL() error {
	file, err := os.Open("heimdall.wal")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	log.Println("WAL detected. Checking for uncommitted transactions...")

	for {
		var ts int64
		var nameLen, dataLen int32

		err := binary.Read(file, binary.LittleEndian, &ts)
		if err == io.EOF {
			break
		}

		binary.Read(file, binary.LittleEndian, &nameLen)
		nameBuf := make([]byte, nameLen)
		file.Read(nameBuf)
		fileName := string(nameBuf)

		binary.Read(file, binary.LittleEndian, &dataLen)
		dataBuf := make([]byte, dataLen)
		file.Read(dataBuf)

		_, err = GetFileRecipe(fileName, ts)
		if err != nil {
			log.Printf("Recovering transaction: %s (TS: %d)", fileName, ts)

			tmpName := "recovery_" + fileName
			os.WriteFile(tmpName, dataBuf, 0644)

			SaveFile(tmpName, fileName, ts)
			os.Remove(tmpName)
		}
	}

	log.Println("WAL Replay sequence complete.")
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
