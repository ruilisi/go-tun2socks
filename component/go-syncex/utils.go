package syncex

import (
	"bytes"
	"runtime"
	"strconv"
	"sync/atomic"
)

func getGID() uint64 {
	var stack [64]byte
	b := stack[:runtime.Stack(stack[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, e := strconv.ParseUint(string(b), 10, 64)
	if e != nil {
		panic(e)
	}
	return n
}

var ownerID = uint64(0)

// NewOwnerID gives a new ownerID incrementally.
func NewOwnerID() uint64 {
	atomic.CompareAndSwapUint64(&ownerID, ^uint64(0), 0)
	return atomic.AddUint64(&ownerID, 1)
}
