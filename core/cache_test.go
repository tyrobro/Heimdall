package core

import (
	"testing"
)

func TestLRUEviction(t *testing.T) {
	err := InitCache(2)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	hash1 := "chunk_A"
	hash2 := "chunk_B"
	hash3 := "chunk_C"
	data := []byte("dummy_payload")

	chunkCache.Add(hash1, data)
	chunkCache.Add(hash2, data)

	_, ok := chunkCache.Get(hash1)
	if !ok {
		t.Fatalf("Expected chunk_A to be in cache")
	}

	chunkCache.Add(hash3, data)

	_, ok = chunkCache.Get(hash2)
	if ok {
		t.Errorf("Cache failed to evict least recently used item (chunk_B)")
	}

	if _, ok := chunkCache.Get(hash1); !ok {
		t.Errorf("Cache incorrectly evicted a recently used item (chunk_A)")
	}
	if _, ok := chunkCache.Get(hash3); !ok {
		t.Errorf("Cache failed to store new item (chunk_C)")
	}
}
