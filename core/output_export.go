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
	// OPTIMIZATION: Always copy data while holding mutex, then release mutex
	// before calling OutputFn. This significantly reduces mutex hold time and
	// allows other goroutines to process packets during I/O operations.
	lwipMutex.Lock()
	totlen := int(p.tot_len)

	// Allocate buffer from pool
	buf := pool.NewBytes(totlen)

	// Copy packet data from pbuf(s) - handles both single and chained pbufs
	C.pbuf_copy_partial(p, unsafe.Pointer(&buf[0]), p.tot_len, 0)

	// Release mutex BEFORE I/O operation - this is the key optimization
	lwipMutex.Unlock()

	// Perform I/O without holding mutex - allows concurrent packet processing
	OutputFn(buf[:totlen])

	// Return buffer to pool
	pool.FreeBytes(buf)

	return C.ERR_OK
}
