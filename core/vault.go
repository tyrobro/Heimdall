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

func SaveFile(filePath, fileName string) error {
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

	recipeBytes, _ := json.Marshal(hashList)
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("FileMeta"))
		return b.Put([]byte(fileName), recipeBytes)
	})
}

func RestoreFile(fileName, outputPath string) error {
	var hashList []string

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("FileMeta"))
		recipeBytes := b.Get([]byte(fileName))
		if recipeBytes == nil {
			return fmt.Errorf("file '%s' not found in metadata", fileName)
		}
		return json.Unmarshal(recipeBytes, &hashList)
	})
	if err != nil {
		return err
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for _, hash := range hashList {
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
