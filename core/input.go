package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/pbuf.h"
#include "lwip/tcp.h"

err_t
input(struct pbuf *p)
{
	return (*netif_list).input(p, netif_list);
}
*/
import "C"
import (
	"encoding/binary"
	"errors"
	"unsafe"
)

type ipver byte

const (
	ipv4 = 4
	ipv6 = 6
)

type proto byte

const (
	proto_icmp = 1
	proto_tcp  = 6
	proto_udp  = 17
)

func peekIPVer(p []byte) (ipver, error) {
	if len(p) < 1 {
		return 0, errors.New("short IP packet")
	}
	return ipver((p[0] & 0xf0) >> 4), nil
}

func moreFrags(ipv ipver, p []byte) bool {
	switch ipv {
	case ipv4:
		if (p[6] & 0x20) > 0 /* has MF (More Fragments) bit set */ {
			return true
		}
	case ipv6:
		// FIXME Just too lazy to implement this for IPv6, for now
		// returning true simply indicate do the copy anyway.
		return true
	}
	return false
}

func fragOffset(ipv ipver, p []byte) uint16 {
	switch ipv {
	case ipv4:
		return binary.BigEndian.Uint16(p[6:8]) & 0x1fff
	case ipv6:
		// FIXME Just too lazy to implement this for IPv6, for now
		// returning a value greater than 0 simply indicate do the
		// copy anyway.
		return 1
	}
	return 0
}

func peekNextProto(ipv ipver, p []byte) (proto, error) {
	switch ipv {
	case ipv4:
		if len(p) < 9 {
			return 0, errors.New("short IPv4 packet")
		}
		return proto(p[9]), nil
	case ipv6:
		if len(p) < 6 {
			return 0, errors.New("short IPv6 packet")
		}
		return proto(p[6]), nil
	default:
		return 0, errors.New("unknown IP version")
	}
}

func input(pkt []byte) (int, error) {
	lwipMutex.Lock()
	defer lwipMutex.Unlock()
	pktLen := len(pkt)
	if pktLen == 0 {
		return 0, nil
	}
	remaining := pktLen
	startPos := 0
	singleCopyLen := 0

	var buf *C.struct_pbuf
	var newBuf *C.struct_pbuf
	var ierr C.err_t = C.ERR_ARG // initial value to free pbuf if errors occur

	// TODO Copy the data only when lwip need to keep it, e.g. in
	// case we are returning ERR_CONN in tcpRecvFn.
	//
	// XXX: always copy since the address might got moved to other location during GC

	// PBUF_POOL  pbuf payload refers to RAM. This one comes from a pool and should be used for RX.
	// Payload can be chained (scatter-gather RX) but like PBUF_RAM, struct pbuf and its payload are allocated in one piece of contiguous memory
	// (so the first payload byte can be calculated from struct pbuf). Don't use this for TX, if the pool becomes empty e.g. because of TCP queuing,
	// you are unable to receive TCP acks!

	buf = C.pbuf_alloc(C.PBUF_RAW, C.u16_t(pktLen), C.PBUF_POOL)
	newBuf = buf
	defer func(pb *C.struct_pbuf, err *C.err_t) {
		if pb != nil && *err != C.ERR_OK {
			C.pbuf_free(pb)
			pb = nil
		}
	}(newBuf, &ierr)
	if newBuf == nil {
		return 0, errors.New("lwip Input() pbuf_alloc returns NULL")
	}
	for remaining > 0 {
		if remaining > int(newBuf.tot_len) {
			singleCopyLen = int(newBuf.tot_len)
		} else {
			singleCopyLen = remaining
		}
		r := C.pbuf_take_at(newBuf, unsafe.Pointer(&pkt[startPos]), C.u16_t(singleCopyLen), C.u16_t(startPos))
		if r == C.ERR_MEM {
			return 0, errors.New("Input pbuf_take_at this should not happen")
		}
		startPos += singleCopyLen
		remaining -= singleCopyLen
	}
	newBuf = C.pbuf_coalesce(buf, C.PBUF_RAW)
	if newBuf.next != nil {
		return 0, errors.New("lwip Input() pbuf_coalesce failed")
	}

	ierr = C.input(newBuf)
	if ierr != C.ERR_OK {
		return 0, errors.New("packet not handled")
	}
	return pktLen, nil
}
