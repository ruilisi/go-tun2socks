package syncex

import (
	hackgid "github.com/ruilisi/go-tun2socks/component/gls"
	"sync"
)

// A RecursiveMutex is particular type of mutual exclusion (mutex) that
// may be locked multiple times by the same goroutine.
//
// A RecursiveMutex must not be copied after first use.
type RecursiveMutex struct {
	mu sync.Mutex
	c  chan struct{}
	v  int32
	id uint64
}

// Lock locks rm.
// If rm is already owned by different goroutine, it waits
// until the RecursiveMutex is available.
func (rm *RecursiveMutex) Lock() {
	//id := getGID()
	id := uint64(hackgid.GoID())
	for {
		rm.mu.Lock()
		if rm.c == nil {
			rm.c = make(chan struct{}, 1)
		}
		if rm.v == 0 || rm.id == id {
			rm.v++
			rm.id = id
			rm.mu.Unlock()
			break
		}
		rm.mu.Unlock()
		<-rm.c
	}
}

// Unlock unlocks rm.
// It panics if rm is not locked on entry to Unlock.
func (rm *RecursiveMutex) Unlock() {
	rm.mu.Lock()
	if rm.c == nil {
		rm.c = make(chan struct{}, 1)
	}
	if rm.v <= 0 {
		rm.mu.Unlock()
		panic(ErrNotLocked)
	}
	rm.v--
	if rm.v == 0 {
		rm.id = 0
	}
	rm.mu.Unlock()
	select {
	case rm.c <- struct{}{}:
	default:
	}
}

func init() {
	id1 := getGID()
	id2 := uint64(hackgid.GoID())
	if id1 != id2 {
		panic("hackgid and ordinary GID different")
	}
}
