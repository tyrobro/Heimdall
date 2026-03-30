package core

import (
	"fmt"
	"os"
	"path/filepath"

	lru "github.com/hashicorp/golang-lru/v2"
)

var chunkCache *lru.Cache[string, []byte]

func InitCache(maxChunks int) error {
	var err error
	chunkCache, err = lru.New[string, []byte](maxChunks)
	if err != nil {
		return fmt.Errorf("Failed to inititalise LRU cache: %w", err)
	}
	return nil
}

func GetChunk(hash string) ([]byte, bool, error) {
	if data, ok := chunkCache.Get(hash); ok {
		return data, true, nil
	}

	chunkPath := filepath.Join(storageDir, hash)
	data, err := os.ReadFile(chunkPath)
	if err != nil {
		return nil, false, fmt.Errorf("chunk not found on disk: %w", err)
	}
	chunkCache.Add(hash, data)
	return data, false, nil

}
