package core

import (
	"sync"
)

type TSO struct {
	mu        sync.Mutex
	timestamp int64
}

var Oracle *TSO

func InitTSO() {
	Oracle = &TSO{
		timestamp: 0,
	}
}

func (t *TSO) GetNextTimestamp() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.timestamp++
	return t.timestamp
}
