package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	telemetryFile  *os.File
	telemetryMutex sync.Mutex
	csvWriter      *csv.Writer
)

func InitTelemetry() error {
	var err error
	fileExists := false

	if _, err := os.Stat("telemetry.csv"); err == nil {
		fileExists = true
	}

	telemetryFile, err = os.OpenFile("telemetry.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	csvWriter = csv.NewWriter(telemetryFile)

	if !fileExists {
		err = csvWriter.Write([]string{"timestamp", "file_name", "action"})
		if err != nil {
			return err
		}
		csvWriter.Flush()
	}

	return nil
}

func LogEvent(fileName string, action string) {
	telemetryMutex.Lock()
	defer telemetryMutex.Unlock()

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	record := []string{timestamp, fileName, action}

	csvWriter.Write(record)
	csvWriter.Flush()
}

func CloseTelemetry() {
	if csvWriter != nil {
		csvWriter.Flush()
	}
	if telemetryFile != nil {
		telemetryFile.Close()
	}
}
