package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

var db *bbolt.DB

const storageDir = "disk_storage"

type FileVersion struct {
	Timestamp int64    `json:"timestamp"`
	Hashes    []string `json:"hashes"`
}

type FileHistory struct {
	Versions []FileVersion `json:"versions"`
}

func InitVault() error {
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage dir: %w", err)
	}

	var err error
	db, err = bbolt.Open("metadata.db", 0600, nil)
	if err != nil {
		return fmt.Errorf("failed to open boltdb: %w", err)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("FileMeta"))
		return err
	})
}

func CloseVault() {
	if db != nil {
		db.Close()
	}
}

func SaveFile(filePath, fileName string, timestamp int64) error {
	chunks, err := ChunkFile(filePath)
	if err != nil {
		return err
	}

	originalFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer originalFile.Close()

	var hashList []string

	for _, chunk := range chunks {
		hashList = append(hashList, chunk.Hash)
		chunkPath := filepath.Join(storageDir, chunk.Hash)

		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			chunkData := make([]byte, chunk.Length)
			originalFile.ReadAt(chunkData, int64(chunk.Offset))
			os.WriteFile(chunkPath, chunkData, 0644)
		}
	}

	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("FileMeta"))

		var history FileHistory
		existingData := b.Get([]byte(fileName))
		if existingData != nil {
			json.Unmarshal(existingData, &history)
		}

		history.Versions = append(history.Versions, FileVersion{
			Timestamp: timestamp,
			Hashes:    hashList,
		})

		newBytes, _ := json.Marshal(history)
		return b.Put([]byte(fileName), newBytes)
	})
}

func RestoreFile(fileName, outputPath string, targetTimestamp int64) error {
	var targetHashes []string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("FileMeta"))
		recipeBytes := b.Get([]byte(fileName))
		if recipeBytes == nil {
			return fmt.Errorf("file '%s' not found in metadata", fileName)
		}

		var history FileHistory
		json.Unmarshal(recipeBytes, &history)

		for i := len(history.Versions) - 1; i >= 0; i-- {
			if history.Versions[i].Timestamp <= targetTimestamp {
				targetHashes = history.Versions[i].Hashes
				return nil
			}
		}
		return fmt.Errorf("no version of '%s' existed at timestamp %d", fileName, targetTimestamp)
	})

	if err != nil {
		return err
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for _, hash := range targetHashes {
		chunkPath := filepath.Join(storageDir, hash)
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("missing chunk %s: %w", hash, err)
		}

		io.Copy(outFile, chunkFile)
		chunkFile.Close()
	}

	return nil
}
