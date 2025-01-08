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
	defer lwipMutex.Unlock()
	defer func(pb *C.struct_pbuf) {
		lwipMutex.Lock()
		defer lwipMutex.Unlock()
		if pb != nil {
			C.pbuf_free(pb)
			pb = nil
		}
	}(p)

	if pcb == nil {
		return
	}
	addrCopy := C.ip_addr_t{}
	destAddrCopy := C.ip_addr_t{}
	copyLwipIpAddr(&addrCopy, addr)
	copyLwipIpAddr(&destAddrCopy, destAddr)
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
			return
		}
		udpConns.Store(connId, conn)
	}

	var buf []byte
	var totlen = int(p.tot_len)
	if p.tot_len == p.len {
		buf = (*[1 << 30]byte)(unsafe.Pointer(p.payload))[:totlen:totlen]
	} else {
		buf = pool.NewBytes(totlen)
		defer pool.FreeBytes(buf)
		C.pbuf_copy_partial(p, unsafe.Pointer(&buf[0]), p.tot_len, 0)
	}

	conn.(UDPConn).ReceiveTo(buf[:totlen], dstAddr)
}
