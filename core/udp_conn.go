package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/udp.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"net"
	"sync"
	"unsafe"
)

type udpConnState uint

const (
	udpConnecting udpConnState = iota
	udpConnected
	udpClosed
)

type udpPacket struct {
	data []byte
	addr *net.UDPAddr
}

type udpConn struct {
	sync.Mutex

	pcb       *C.struct_udp_pcb
	handler   UDPConnHandler
	localAddr *net.UDPAddr
	localIP   C.ip_addr_t
	localPort C.u16_t
	state     udpConnState
	pending   chan *udpPacket
}

func newUDPConn(pcb *C.struct_udp_pcb, handler UDPConnHandler, localIP C.ip_addr_t, localPort C.u16_t, localAddr, remoteAddr *net.UDPAddr) (UDPConn, error) {
	conn := &udpConn{
		handler:   handler,
		pcb:       pcb,
		localAddr: localAddr,
		localIP:   localIP,
		localPort: localPort,
		state:     udpConnecting,
		pending:   make(chan *udpPacket, 64), // To hold the early packets on the connection
	}

	go func() {
		err := handler.Connect(conn, remoteAddr)
		if err != nil {
			conn.Close()
		} else {
			conn.Lock()
			conn.state = udpConnected
			conn.Unlock()
			// Once connected, send all pending data.
		DrainPending:
			for {
				select {
				case pkt := <-conn.pending:
					err := conn.handler.ReceiveTo(conn, pkt.data, pkt.addr)
					if err != nil {
						break DrainPending
					}
					continue DrainPending
				default:
					conn.pending = nil
					break DrainPending
				}
			}
		}
	}()

	return conn, nil
}

func (conn *udpConn) LocalAddr() *net.UDPAddr {
	return conn.localAddr
}

func (conn *udpConn) checkState() error {
	conn.Lock()
	defer conn.Unlock()

	switch conn.state {
	case udpClosed:
		return errors.New("connection closed")
	case udpConnected:
		return nil
	case udpConnecting:
		return errors.New("not connected")
	}
	return nil
}

// If the connection isn't ready yet, and there is room in the queue, make a copy
// and hold onto it until the connection is ready.
func (conn *udpConn) enqueueEarlyPacket(data []byte, addr *net.UDPAddr) bool {
	conn.Lock()
	defer conn.Unlock()
	if conn.state == udpConnecting {
		pkt := &udpPacket{data: append([]byte(nil), data...), addr: addr}
		select {
		// Data will be dropped if pending is full.
		case conn.pending <- pkt:
			return true
		default:
		}
	}
	return false
}

func (conn *udpConn) ReceiveTo(data []byte, addr *net.UDPAddr) error {
	if conn.enqueueEarlyPacket(data, addr) {
		return nil
	}
	if err := conn.checkState(); err != nil {
		return err
	}
	err := conn.handler.ReceiveTo(conn, data, addr)
	if err != nil {
		return errors.New(fmt.Sprintf("write proxy failed: %v", err))
	}
	return nil
}

func (conn *udpConn) WriteFrom(data []byte, addr *net.UDPAddr) (int, error) {
	if err := conn.checkState(); err != nil {
		return 0, err
	}
	// FIXME any memory leaks?
	cremoteIP := C.struct_ip_addr{}
	if err := ipAddrATON(addr.IP.String(), &cremoteIP); err != nil {
		return 0, err
	}
	dataLen := len(data)
	if dataLen <= 0 {
		return dataLen, nil
	}
	remaining := dataLen
	startPos := 0
	singleCopyLen := 0
	// PBUF_TRANSPORT  Includes spare room for transport layer header, e.g. UDP header. Use this if you intend to pass the pbuf to functions like udp_send().

	// PBUF_RAM  pbuf data is stored in RAM, used for TX mostly, struct pbuf and its payload are allocated in one piece of contiguous memory
	// (so the first payload byte can be calculated from struct pbuf). pbuf_alloc() allocates PBUF_RAM pbufs as unchained pbufs
	// (although that might change in future versions). This should be used for all OUTGOING packets (TX).

	// Why this way? Please check lwip/core/dns.c
	buf := C.pbuf_alloc(C.PBUF_TRANSPORT, C.u16_t(dataLen), C.PBUF_RAM)
	defer func(pb *C.struct_pbuf) {
		if pb != nil {
			C.pbuf_free(pb)
			pb = nil
		}
	}(buf)
	if buf == nil {
		panic("udpConn WriteFrom pbuf_alloc returns NULL")
	}
	for remaining > 0 {
		if remaining > int(buf.tot_len) {
			singleCopyLen = int(buf.tot_len)
		} else {
			singleCopyLen = remaining
		}
		//log.Printf("udpConn WriteFrom remaining: %v, tot_len %v singleCopyLen : %v", remaining, int(buf.tot_len), singleCopyLen)
		r := C.pbuf_take_at(buf, unsafe.Pointer(&data[startPos]), C.u16_t(singleCopyLen), C.u16_t(startPos))
		if r == C.ERR_MEM {
			panic("udpConn WriteFrom pbuf_take_at this should not happen")
		}
		startPos += singleCopyLen
		remaining -= singleCopyLen
	}
	C.udp_sendto(conn.pcb, buf, &conn.localIP, conn.localPort, &cremoteIP, C.u16_t(addr.Port))

	return dataLen, nil
}

func (conn *udpConn) Close() error {
	connId := udpConnId{
		src: conn.LocalAddr().String(),
	}
	conn.Lock()
	conn.state = udpClosed
	conn.Unlock()
	udpConns.Delete(connId)
	return nil
}
