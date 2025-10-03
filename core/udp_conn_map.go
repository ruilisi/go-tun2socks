package core

import (
	"sync"
)

// udpConnId identifies a UDP "connection". Currently only the source string
// is used, matching the original key logic. Extend if a fuller 5‑tuple
// becomes necessary.
type udpConnId struct {
	src string
}

// udpConnRegistry provides a typed, RWMutex-protected map with a
// double-checked GetOrCreate, plus Range and Delete helpers.
type udpConnRegistry struct {
	mu sync.RWMutex
	m  map[udpConnId]UDPConn
}

func newUDPConnRegistry() *udpConnRegistry {
	return &udpConnRegistry{
		m: make(map[udpConnId]UDPConn, 64),
	}
}

// Get returns an existing connection if present.
func (r *udpConnRegistry) Get(id udpConnId) (UDPConn, bool) {
	r.mu.RLock()
	c, ok := r.m[id]
	r.mu.RUnlock()
	return c, ok
}

// GetOrCreate returns (conn, created, err). It avoids duplicate creation
// with a double-checked pattern.
func (r *udpConnRegistry) GetOrCreate(id udpConnId, factory func() (UDPConn, error)) (UDPConn, bool, error) {
	// Fast path read.
	r.mu.RLock()
	c, ok := r.m[id]
	r.mu.RUnlock()
	if ok {
		return c, false, nil
	}

	newConn, err := factory()
	if err != nil {
		return nil, false, err
	}

	r.mu.Lock()
	if existing, ok := r.m[id]; ok {
		r.mu.Unlock()
		return existing, false, nil
	}
	r.m[id] = newConn
	r.mu.Unlock()
	return newConn, true, nil
}

// Delete removes the connection if present.
func (r *udpConnRegistry) Delete(id udpConnId) {
	r.mu.Lock()
	delete(r.m, id)
	r.mu.Unlock()
}

// Range iterates over a snapshot to minimize time under lock.
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

// Global registry replacing: var udpConns sync.Map
var udpConns = newUDPConnRegistry()