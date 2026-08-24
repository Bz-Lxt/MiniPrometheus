package ring

import (
	"sync"

	"github.com/alkaid/miniprometheus/internal/model"
)

type Slot struct {
	Samples []model.Sample
}

type Buffer struct {
	mu    sync.RWMutex
	slots []Slot
	cap   int
	head  int
	len   int
}

func New(n int) *Buffer {
	if n < 2 {
		n = 8
	}
	return &Buffer{slots: make([]Slot, n), cap: n}
}

func (b *Buffer) AppendChunk(ss []model.Sample) {
	if len(ss) == 0 {
		return
	}
	cp := append([]model.Sample(nil), ss...)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.slots[b.head] = Slot{Samples: cp}
	b.head = (b.head + 1) % b.cap
	if b.len < b.cap {
		b.len++
	}
}

func (b *Buffer) Latest() []model.Sample {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.len == 0 {
		return nil
	}
	idx := (b.head - 1 + b.cap) % b.cap
	return append([]model.Sample(nil), b.slots[idx].Samples...)
}

func (b *Buffer) Range(mint, maxt int64) []model.Sample {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var out []model.Sample
	if b.len == 0 {
		return out
	}
	start := (b.head - b.len + b.cap) % b.cap
	for i := 0; i < b.len; i++ {
		s := b.slots[(start+i)%b.cap]
		for _, p := range s.Samples {
			if p.T >= mint && p.T <= maxt {
				out = append(out, p)
			}
		}
	}
	return out
}

func (b *Buffer) All() []model.Sample {
	return b.Range(0, 1<<62)
}

func (b *Buffer) ChunkCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.len
}
