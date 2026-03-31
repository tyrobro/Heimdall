package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	pb "heimdall/proto"

	"go.etcd.io/bbolt"
)

const storageDir = "disk_storage"

type FileVersion struct {
	Timestamp int64    `json:"timestamp"`
	Hashes    []string `json:"hashes"`
}

type FileHistory struct {
	Versions []FileVersion `json:"versions"`
}

func InitVault() error {
	return os.MkdirAll(storageDir, 0755)
}

func CloseVault() {}

func SaveFile(filePath string, fileName string, timestamp int64) error {
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

	db, err := bbolt.Open("metadata.db", 0600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return fmt.Errorf("failed to open boltdb for writing: %w", err)
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("FileMeta"))
		if err != nil {
			return err
		}

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

func GetFileRecipe(fileName string, targetTimestamp int64) ([]string, error) {
	var targetHashes []string

	db, err := bbolt.Open("metadata.db", 0600, &bbolt.Options{Timeout: 2 * time.Second, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open boltdb for reading: %w", err)
	}
	defer db.Close()

	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("FileMeta"))
		if b == nil {
			return fmt.Errorf("database bucket not initialized")
		}

		recipeBytes := b.Get([]byte(fileName))
		if recipeBytes == nil {
			return fmt.Errorf("file '%s' not found in metadata", fileName)
		}

		var history FileHistory
		if err := json.Unmarshal(recipeBytes, &history); err != nil {
			return fmt.Errorf("metadata for file '%s' is corrupted: %w", fileName, err)
		}

		for i := len(history.Versions) - 1; i >= 0; i-- {
			if history.Versions[i].Timestamp <= targetTimestamp {
				targetHashes = history.Versions[i].Hashes
				return nil
			}
		}
		return fmt.Errorf("no version of '%s' existed at timestamp %d", fileName, targetTimestamp)
	})

	return targetHashes, err
}

func RestoreFile(fileName string, outputPath string, targetTimestamp int64) error {
	targetHashes, err := GetFileRecipe(fileName, targetTimestamp)
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

func GetAllFiles() (map[string]*pb.VersionList, error) {
	result := make(map[string]*pb.VersionList)

	db, err := bbolt.Open("metadata.db", 0600, &bbolt.Options{Timeout: 2 * time.Second, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open boltdb for reading: %w", err)
	}
	defer db.Close()

	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("FileMeta"))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			fileName := string(k)
			var history FileHistory

			if err := json.Unmarshal(v, &history); err != nil {
				log.Printf("Skipping legacy/corrupted data for file [%s]: %v", fileName, err)
				return nil
			}

			var timestamps []int64
			for _, version := range history.Versions {
				timestamps = append(timestamps, version.Timestamp)
			}

			result[fileName] = &pb.VersionList{Timestamps: timestamps}
			return nil
		})
	})

	return result, err
}
