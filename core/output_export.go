package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/tcp.h"
*/
import "C"
import (
	"github.com/ruilisi/go-tun2socks/component/pool"
	"unsafe"
)

//export output
func output(p *C.struct_pbuf) C.err_t {
	// Always copy pbuf payload into Go-managed buffer to minimize time
	// holding lwipMutex. This removes the lock from around OutputFn call.
	lwipMutex.Lock()
	totlen := int(p.tot_len)
	buf := pool.NewBytes(totlen)
	if p.tot_len == p.len {
		// Single pbuf - direct copy
		copy(buf[:totlen], (*[1 << 30]byte)(unsafe.Pointer(p.payload))[:totlen:totlen])
	} else {
		// Chained pbufs - use pbuf_copy_partial
		C.pbuf_copy_partial(p, unsafe.Pointer(&buf[0]), p.tot_len, 0)
	}
	lwipMutex.Unlock()
	
	// Call OutputFn with Go-owned buffer while unlocked
	OutputFn(buf[:totlen])
	
	// Free the pooled buffer
	pool.FreeBytes(buf)
	return C.ERR_OK
}
