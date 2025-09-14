package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/udp.h"
*/
import "C"
import (
	"unsafe"

	"github.com/ruilisi/go-tun2socks/component/pool"
)

//export udpRecvFn
func udpRecvFn(arg unsafe.Pointer, pcb *C.struct_udp_pcb, p *C.struct_pbuf, addr *C.ip_addr_t, port C.u16_t, destAddr *C.ip_addr_t, destPort C.u16_t) {
	// XXX:  * ATTENTION: Be aware that 'addr' might point into the pbuf 'p' so freeing this pbuf
	//       *            can make 'addr' invalid, too.
	// Let's copy addr in case accessing invalid pointer
	lwipMutex.Lock()
	
	if pcb == nil {
		lwipMutex.Unlock()
		if p != nil {
			lwipMutex.Lock()
			C.pbuf_free(p)
			lwipMutex.Unlock()
		}
		return
	}
	
	// Copy addresses to avoid accessing invalid pointers after unlock
	addrCopy := C.ip_addr_t{}
	destAddrCopy := C.ip_addr_t{}
	copyLwipIpAddr(&addrCopy, addr)
	copyLwipIpAddr(&destAddrCopy, destAddr)
	
	// Copy UDP payload into Go buffer while holding lock
	var buf []byte
	var totlen = int(p.tot_len)
	buf = pool.NewBytes(totlen)
	if p.tot_len == p.len {
		copy(buf[:totlen], (*[1 << 30]byte)(unsafe.Pointer(p.payload))[:totlen:totlen])
	} else {
		C.pbuf_copy_partial(p, unsafe.Pointer(&buf[0]), p.tot_len, 0)
	}
	lwipMutex.Unlock()
	
	// Parse addresses and handle connection logic without holding lock
	srcAddr := ParseUDPAddr(ipAddrNTOA(addrCopy), uint16(port))
	dstAddr := ParseUDPAddr(ipAddrNTOA(destAddrCopy), uint16(destPort))
	if srcAddr == nil || dstAddr == nil {
		panic("invalid UDP address")
	}

	connId := udpConnId{
		src: srcAddr.String(),
	}
	conn, found := udpConns.Load(connId)
	if !found {
		if udpConnHandler == nil {
			panic("must register a UDP connection handler")
		}
		var err error
		conn, err = newUDPConn(pcb,
			udpConnHandler,
			addrCopy,
			port,
			srcAddr,
			dstAddr)
		if err != nil {
			// Free resources before returning
			pool.FreeBytes(buf)
			lwipMutex.Lock()
			if p != nil {
				C.pbuf_free(p)
			}
			lwipMutex.Unlock()
			return
		}
		udpConns.Store(connId, conn)
	}

	// Call ReceiveTo with Go-owned buffer while unlocked
	conn.(UDPConn).ReceiveTo(buf[:totlen], dstAddr)
	
	// Free the pooled buffer
	pool.FreeBytes(buf)
	
	// Free pbuf under lock
	lwipMutex.Lock()
	if p != nil {
		C.pbuf_free(p)
	}
	lwipMutex.Unlock()
}
