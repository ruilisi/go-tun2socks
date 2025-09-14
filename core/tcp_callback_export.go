package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/tcp.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"unsafe"

	"github.com/ruilisi/go-tun2socks/component/pool"
)

// These exported callback functions must be placed in a seperated file.
//
// See also:
// https://github.com/golang/go/issues/20639
// https://golang.org/cmd/cgo/#hdr-C_references_to_Go

//export tcpAcceptFn
func tcpAcceptFn(arg unsafe.Pointer, newpcb *C.struct_tcp_pcb, err C.err_t) C.err_t {
	if err != C.ERR_OK {
		return err
	}

	if tcpConnHandler == nil {
		log.Printf("must register a TCP connection handler")
		C.tcp_abort(newpcb)
		return C.ERR_ABRT
	}

	if _, nerr := newTCPConn(newpcb, tcpConnHandler); nerr != nil {
		switch nerr.(*lwipError).Code {
		case LWIP_ERR_ABRT:
			return C.ERR_ABRT
		case LWIP_ERR_OK:
			return C.ERR_OK
		default:
			return C.ERR_CONN
		}
	}

	return C.ERR_OK
}

//export tcpRecvFn
func tcpRecvFn(arg unsafe.Pointer, tpcb *C.struct_tcp_pcb, p *C.struct_pbuf, passedInErr C.err_t) C.err_t {
	// Only free the pbuf when returning ERR_OK or ERR_ABRT,
	// otherwise must not free the pbuf.
	lwipMutex.Lock()
	shouldFreePbuf := false
	
	// Handle error cases under lock
	if passedInErr != C.ERR_OK && passedInErr != C.ERR_ABRT {
		// TODO: unknown err passed in, not sure if it's correct
		log.Printf("tcpRecvFn passed in err: %v , begin abort", int(passedInErr))
		C.tcp_abort(tpcb)
		shouldFreePbuf = true
		lwipMutex.Unlock()
		if p != nil && shouldFreePbuf {
			lwipMutex.Lock()
			C.pbuf_free(p)
			lwipMutex.Unlock()
		}
		return C.ERR_ABRT
	}

	var conn = (*tcpConn)(arg)

	if p == nil {
		// Peer closed, EOF.
		lwipMutex.Unlock()
		err := conn.LocalClosed()
		lwipMutex.Lock()
		switch err.(*lwipError).Code {
		case LWIP_ERR_ABRT:
			shouldFreePbuf = true
		case LWIP_ERR_OK:
			shouldFreePbuf = true
		default:
			log.Printf("unexpected error conn.LocalClosed() %v", err.(*lwipError).Error())
			shouldFreePbuf = true
		}
		lwipMutex.Unlock()
		if p != nil && shouldFreePbuf {
			lwipMutex.Lock()
			C.pbuf_free(p)
			lwipMutex.Unlock()
		}
		switch err.(*lwipError).Code {
		case LWIP_ERR_ABRT:
			return C.ERR_ABRT
		default:
			return C.ERR_OK
		}
	}

	// Copy data from pbuf into Go buffer while holding lock
	var buf []byte
	var totlen = int(p.tot_len)
	buf = pool.NewBytes(totlen)
	if p.tot_len == p.len {
		copy(buf[:totlen], (*[1 << 30]byte)(unsafe.Pointer(p.payload))[:totlen:totlen])
	} else {
		C.pbuf_copy_partial(p, unsafe.Pointer(&buf[0]), p.tot_len, 0)
	}
	lwipMutex.Unlock()

	// Call conn.Receive with Go-owned buffer while unlocked
	rerr := conn.Receive(buf[:totlen])
	
	// Free the pooled buffer
	pool.FreeBytes(buf)
	
	// Handle the result and any required lwIP API calls under lock
	lwipMutex.Lock()
	if rerr != nil {
		switch rerr.(*lwipError).Code {
		case LWIP_ERR_ABRT:
			shouldFreePbuf = true
		case LWIP_ERR_OK:
			shouldFreePbuf = true
		case LWIP_ERR_CONN:
			// Tell lwip we can't receive data at the moment,
			// lwip will store it and try again later.
			lwipMutex.Unlock()
			return C.ERR_CONN
		case LWIP_ERR_CLSD:
			// lwip won't handle ERR_CLSD error for us, manually
			// shuts down the rx side.
			shouldFreePbuf = true
			C.tcp_recved(tpcb, p.tot_len)
			C.tcp_shutdown(tpcb, 1, 0)
		default:
			log.Printf("unexpected error conn.Receive() %v", rerr.(*lwipError).Error())
			shouldFreePbuf = true
		}
	} else {
		shouldFreePbuf = true
	}
	lwipMutex.Unlock()
	
	// Free pbuf if needed
	if p != nil && shouldFreePbuf {
		lwipMutex.Lock()
		C.pbuf_free(p)
		lwipMutex.Unlock()
	}
	
	if rerr != nil {
		switch rerr.(*lwipError).Code {
		case LWIP_ERR_ABRT:
			return C.ERR_ABRT
		case LWIP_ERR_CLSD:
			return C.ERR_OK
		default:
			return C.ERR_OK
		}
	}

	return C.ERR_OK
}

//export tcpSentFn
func tcpSentFn(arg unsafe.Pointer, tpcb *C.struct_tcp_pcb, len C.u16_t) C.err_t {
	var conn = (*tcpConn)(arg)

	err := conn.Sent(uint16(len))
	switch err.(*lwipError).Code {
	case LWIP_ERR_ABRT:
		return C.ERR_ABRT
	case LWIP_ERR_OK:
		return C.ERR_OK
	default:
		log.Printf("unexpected error conn.Sent() %v", err.(*lwipError).Error())
		panic("unexpected error conn.Sent()")
	}

}

//export tcpErrFn
func tcpErrFn(arg unsafe.Pointer, err C.err_t) {
	var conn = (*tcpConn)(arg)

	switch err {
	case C.ERR_ABRT:
		// Aborted through tcp_abort or by a TCP timer
		conn.Err(errors.New("connection aborted"))
	case C.ERR_RST:
		// The connection was reset by the remote host
		conn.Err(errors.New("connection reseted"))
	default:
		conn.Err(errors.New(fmt.Sprintf("lwip error code %v", int(err))))
	}
}
