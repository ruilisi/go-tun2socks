package core

import (
	"sync"
)

// udpConnId identifies a UDP "connection".
type udpConnId struct {
	src string
}

// udpConnRegistry is a typed, lock-protected map.
type udpConnRegistry struct {
	mu sync.RWMutex
	m  map[udpConnId]UDPConn
}

func newUDPConnRegistry() *udpConnRegistry {
	return &udpConnRegistry{
		m: make(map[udpConnId]UDPConn, 64),
	}
}

func (r *udpConnRegistry) Get(id udpConnId) (UDPConn, bool) {
	r.mu.RLock()
	c, ok := r.m[id]
	r.mu.RUnlock()
	return c, ok
}

// GetOrCreate returns (conn, created, err).
// This version calls the factory ONLY after acquiring the write lock
// and confirming the entry still does not exist, preventing duplicate
// expensive factory calls under concurrent races.
func (r *udpConnRegistry) GetOrCreate(id udpConnId, factory func() (UDPConn, error)) (UDPConn, bool, error) {
	// Fast read check.
	r.mu.RLock()
	c, ok := r.m[id]
	r.mu.RUnlock()
	if ok {
		return c, false, nil
	}

	// Acquire write lock to serialize creation.
	r.mu.Lock()
	// Re-check: another goroutine may have inserted meanwhile.
	if c, ok = r.m[id]; ok {
		r.mu.Unlock()
		return c, false, nil
	}

	// Create while holding the write lock to ensure single factory invocation.
	newConn, err := factory()
	if err != nil {
		r.mu.Unlock()
		return nil, false, err
	}
	r.m[id] = newConn
	r.mu.Unlock()
	return newConn, true, nil
}

func (r *udpConnRegistry) Delete(id udpConnId) {
	r.mu.Lock()
	delete(r.m, id)
	r.mu.Unlock()
}

func (r *udpConnRegistry) Range(fn func(id udpConnId, c UDPConn) bool) {
	r.mu.RLock()
	if len(r.m) == 0 {
		r.mu.RUnlock()
		return
	}
	snap := make([]struct {
		id udpConnId
		c  UDPConn
	}, 0, len(r.m))
	for id, c := range r.m {
		snap = append(snap, struct {
			id udpConnId
			c  UDPConn
		}{id: id, c: c})
	}
	r.mu.RUnlock()

	for _, e := range snap {
		if !fn(e.id, e.c) {
			return
		}
	}
}

var udpConns = newUDPConnRegistry()
