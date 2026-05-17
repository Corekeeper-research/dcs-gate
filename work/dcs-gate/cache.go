package main

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
)

// LRU correcta usando container/list. Evicta la más vieja cuando llena.
type LRU struct {
	mu    sync.Mutex
	cap   int
	items map[string]*list.Element
	order *list.List
	hits  uint64
	miss  uint64
}

type lruEntry struct {
	key string
	val []float64
}

func newLRU(cap int) *LRU {
	if cap <= 0 {
		cap = 1000
	}
	return &LRU{
		cap:   cap,
		items: make(map[string]*list.Element, cap),
		order: list.New(),
	}
}

func keyOf(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

func (l *LRU) Get(text string) ([]float64, bool) {
	k := keyOf(text)
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.items[k]; ok {
		l.order.MoveToFront(e)
		atomic.AddUint64(&l.hits, 1)
		return e.Value.(*lruEntry).val, true
	}
	atomic.AddUint64(&l.miss, 1)
	return nil, false
}

func (l *LRU) Put(text string, vec []float64) {
	k := keyOf(text)
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.items[k]; ok {
		e.Value.(*lruEntry).val = vec
		l.order.MoveToFront(e)
		return
	}
	e := l.order.PushFront(&lruEntry{key: k, val: vec})
	l.items[k] = e
	if l.order.Len() > l.cap {
		oldest := l.order.Back()
		if oldest != nil {
			l.order.Remove(oldest)
			delete(l.items, oldest.Value.(*lruEntry).key)
		}
	}
}

func (l *LRU) Stats() (size int, hits, miss uint64) {
	l.mu.Lock()
	size = l.order.Len()
	l.mu.Unlock()
	return size, atomic.LoadUint64(&l.hits), atomic.LoadUint64(&l.miss)
}
