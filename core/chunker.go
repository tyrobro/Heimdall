package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/restic/chunker"
)

type Chunk struct {
	Hash   string
	Length uint
	Offset uint
}

func ChunkFile(filePath string) ([]Chunk, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	myPol := chunker.Pol(0x3DA3358B4CC173)
	chk := chunker.New(file, myPol)

	var chunks []Chunk
	buf := make([]byte, 8*1024*1024)
	var currentOffset uint = 0

	for {
		chunk, err := chk.Next(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("chunking error: %w", err)
		}

		hasher := sha256.New()
		hasher.Write(chunk.Data)
		hashString := hex.EncodeToString(hasher.Sum(nil))

		chunks = append(chunks, Chunk{
			Hash:   hashString,
			Length: chunk.Length,
			Offset: currentOffset,
		})

		currentOffset += chunk.Length
	}

	return chunks, nil
}
