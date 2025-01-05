package syncex

import (
	"sync"
)

// A CriticalSection is particular type of mutual exclusion (mutex) that
// may be locked multiple times by the same owner.
//
// The CriticalSection is faster than RecursiveMutex.
//
// A CriticalSection must not be copied after first use.
type CriticalSection struct {
	mu sync.Mutex
	c  chan struct{}
	v  int32
	id uint64
}

// Lock locks cs with given ownerID.
// If cs is already owned by different owner, it waits
// until the CriticalSection is available.
func (cs *CriticalSection) Lock(ownerID uint64) {
	for {
		cs.mu.Lock()
		if cs.c == nil {
			cs.c = make(chan struct{}, 1)
		}
		if cs.v == 0 || cs.id == ownerID {
			cs.v++
			cs.id = ownerID
			cs.mu.Unlock()
			break
		}
		cs.mu.Unlock()
		<-cs.c
	}
}

// Unlock unlocks cs.
// It panics if cs is not locked on entry to Unlock.
func (cs *CriticalSection) Unlock() {
	cs.mu.Lock()
	if cs.c == nil {
		cs.c = make(chan struct{}, 1)
	}
	if cs.v <= 0 {
		cs.mu.Unlock()
		panic(ErrNotLocked)
	}
	cs.v--
	if cs.v == 0 {
		cs.id = 0
	}
	cs.mu.Unlock()
	select {
	case cs.c <- struct{}{}:
	default:
	}
}
