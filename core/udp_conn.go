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
	"sync/atomic"
	"unsafe"

	"github.com/ruilisi/stellar-proxy/log"
)

type udpConnState uint32

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
	pcb       *C.struct_udp_pcb
	handler   UDPConnHandler
	localAddr *net.UDPAddr
	localIP   C.ip_addr_t
	localPort C.u16_t

	// state is stored atomically:
	// udpConnecting -> udpConnected (CAS)
	// any -> udpClosed (Store)
	state atomic.Uint32

	pending chan *udpPacket
}

func newUDPConn(pcb *C.struct_udp_pcb, handler UDPConnHandler, localIP C.ip_addr_t, localPort C.u16_t, localAddr, remoteAddr *net.UDPAddr) (UDPConn, error) {
	conn := &udpConn{
		handler:   handler,
		pcb:       pcb,
		localAddr: localAddr,
		localIP:   localIP,
		localPort: localPort,
		pending:   make(chan *udpPacket, 128),
	}
	conn.state.Store(uint32(udpConnecting))

	go func() {
		err := handler.Connect(conn, remoteAddr)
		if err != nil {
			log.E("[tun2socks/Connect] %s err: %v ", remoteAddr, err)
			conn.Close()
			return
		}

		// Only transition to connected if we are still in connecting state.
		if !conn.state.CompareAndSwap(uint32(udpConnecting), uint32(udpConnected)) {
			// Connection was closed while dialing; do not proceed.
			return
		}

		// Drain any pending early packets now that we are connected.
	DrainPending:
		for {
			select {
			case pkt := <-conn.pending:
				err := conn.handler.ReceiveTo(conn, pkt.data, pkt.addr)
				if err != nil {
					log.E("[tun2socks/ReceiveTo] %s err: %v ", remoteAddr, err)
					break DrainPending
				}
			default:
				// No more pending packets
				break DrainPending
			}
		}
	}()

	return conn, nil
}

func (conn *udpConn) LocalAddr() *net.UDPAddr {
	return conn.localAddr
}

func (conn *udpConn) ensureStateConnected() error {
	switch udpConnState(conn.state.Load()) {
	case udpClosed:
		return errors.New("connection closed")
	case udpConnected:
		return nil
	case udpConnecting:
		return errors.New("not connected")
	default:
		panic(fmt.Sprintf("unknown udp connection state: %d", conn.state.Load()))
	}
}

func (conn *udpConn) ReceiveTo(data []byte, addr *net.UDPAddr) error {
	if udpConnState(conn.state.Load()) == udpConnecting {
		pkt := &udpPacket{data: append([]byte(nil), data...), addr: addr}
		select {
		case conn.pending <- pkt:
			return nil
		default:
			return errors.New("failed to pend packet when udp is connecting")
		}
	}
	if err := conn.ensureStateConnected(); err != nil {
		return err
	}
	if err := conn.handler.ReceiveTo(conn, data, addr); err != nil {
		return fmt.Errorf("write proxy failed: %v", err)
	}
	return nil
}

func (conn *udpConn) WriteFrom(data []byte, addr *net.UDPAddr) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if err := conn.ensureStateConnected(); err != nil {
		return 0, err
	}

	lwipMutex.Lock()
	defer lwipMutex.Unlock()

	cremoteIP := C.struct_ip_addr{}
	if err := ipAddrATON(addr.IP.String(), &cremoteIP); err != nil {
		return 0, err
	}
	dataLen := len(data)
	remaining := dataLen
	startPos := 0

	buf := C.pbuf_alloc(C.PBUF_TRANSPORT, C.u16_t(dataLen), C.PBUF_RAM)
	defer func(pb *C.struct_pbuf) {
		lwipMutex.Lock()
		defer lwipMutex.Unlock()
		if pb != nil {
			C.pbuf_free(pb)
			pb = nil
		}
	}(buf)
	if buf == nil {
		panic("udpConn WriteFrom pbuf_alloc returns NULL")
	}

	for remaining > 0 {
		singleCopyLen := min(remaining, int(buf.tot_len))
		r := C.pbuf_take_at(buf, unsafe.Pointer(&data[startPos]), C.u16_t(singleCopyLen), C.u16_t(startPos))
		if r == C.ERR_MEM {
			panic("udpConn WriteFrom pbuf_take_at this should not happen")
		}
		startPos += singleCopyLen
		remaining -= singleCopyLen
	}

	ret := C.udp_sendto(conn.pcb, buf, &conn.localIP, conn.localPort, &cremoteIP, C.u16_t(addr.Port))
	if ret != 0 {
		return 0, fmt.Errorf("[tun2socks] udp_sendto error %d", ret)
	}
	return dataLen, nil
}

func (conn *udpConn) Close() error {
	// Set closed regardless of prior state.
	conn.state.Store(uint32(udpClosed))
	connId := udpConnId{
		src: conn.LocalAddr().String(),
	}
	udpConns.Delete(connId)
	return nil
}
