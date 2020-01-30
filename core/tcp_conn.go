package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/tcp.h"
u32_t
tcp_sndbuf_cgo(struct tcp_pcb *pcb)
{
	return tcp_sndbuf(pcb);
}

void
tcp_nagle_disable_cgo(struct tcp_pcb *pcb)
{
	tcp_nagle_disable(pcb);
}
void
tcp_keepalive_settings_cgo(struct tcp_pcb *pcb)
{
#if defined(LWIP_TCP_KEEPALIVE) && LWIP_TCP_KEEPALIVE == 1
	pcb->so_options |= SOF_KEEPALIVE;
#endif
}
void tcp_arg_cgo(struct tcp_pcb *pcb, uintptr_t ptr) {
	tcp_arg(pcb, (void*)ptr);
}
*/
import "C"
import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
	"unsafe"

	"github.com/djherbis/nio"
	"github.com/ruilisi/go-tun2socks/buffer"
)

type tcpConnState uint

var tcpConns sync.Map

const (
	// tcpNewConn is the initial state.
	tcpNewConn tcpConnState = iota

	// tcpConnecting indicates the handler is still connecting remote host.
	tcpConnecting

	// tcpConnected indicates the connection has been established, handler
	// may write data to TUN, and read data from TUN.
	tcpConnected

	// tcpWriteClosed indicates the handler has closed the writing side
	// of the connection, no more data will send to TUN, but handler can still
	// read data from TUN.
	tcpWriteClosed

	// tcpReceiveClosed indicates lwIP has received a FIN segment from
	// local peer, the reading side is closed, no more data can be read
	// from TUN, but handler can still write data to TUN.
	tcpReceiveClosed

	// tcpClosing indicates both reading side and writing side are closed,
	// resources deallocation will be triggered at any time in lwIP callbacks.
	tcpClosing

	// tcpAborting indicates the connection is aborting, resources deallocation
	// will be triggered at any time in lwIP callbacks.
	tcpAborting

	// tcpClosed indicates the connection has been closed, resources were freed.
	tcpClosed

	// tcpErrord indicates an fatal error occured on the connection, resources
	// were freed.
	tcpErrored
)

type tcpConn struct {
	sync.Mutex

	pcb           *C.struct_tcp_pcb
	handler       TCPConnHandler
	remoteAddr    *net.TCPAddr
	localAddr     *net.TCPAddr
	state         tcpConnState
	sndPipeReader *nio.PipeReader
	sndPipeWriter *nio.PipeWriter
	closeOnce     sync.Once
	closeErr      error
}

func newTCPConn(pcb *C.struct_tcp_pcb, handler TCPConnHandler) (TCPConn, error) {
	// lwipMutex.Lock()
	// defer lwipMutex.Unlock()
	// From badvpn-tun2socks
	C.tcp_nagle_disable_cgo(pcb)
	C.tcp_keepalive_settings_cgo(pcb)

	// From badvpn-tun2socks
	C.tcp_nagle_disable_cgo(pcb)

	// Register callbacks.
	setTCPRecvCallback(pcb)
	setTCPSentCallback(pcb)
	setTCPErrCallback(pcb)

	buf := buffer.New(0xffff)
	pipeReader, pipeWriter := nio.Pipe(buf)
	conn := &tcpConn{
		pcb:           pcb,
		handler:       handler,
		localAddr:     ParseTCPAddr(ipAddrNTOA(pcb.remote_ip), uint16(pcb.remote_port)),
		remoteAddr:    ParseTCPAddr(ipAddrNTOA(pcb.local_ip), uint16(pcb.local_port)),
		state:         tcpNewConn,
		sndPipeReader: pipeReader,
		sndPipeWriter: pipeWriter,
	}

	C.tcp_arg_cgo(pcb, C.uintptr_t(uintptr(unsafe.Pointer(conn))))
	tcpConns.Store(conn, true)

	// Connecting remote host could take some time, do it in another goroutine
	// to prevent blocking the lwip thread.
	conn.Lock()
	conn.state = tcpConnecting
	conn.Unlock()
	go func() {
		err := handler.Handle(TCPConn(conn), conn.remoteAddr)
		if err != nil {
			conn.Abort()
		} else {
			conn.Lock()
			if conn.state != tcpConnecting {
				conn.Unlock()
				return
			}
			conn.state = tcpConnected
			conn.Unlock()
		}
	}()

	return conn, NewLWIPError(LWIP_ERR_OK)
}

func (conn *tcpConn) RemoteAddr() net.Addr {
	return conn.remoteAddr
}

func (conn *tcpConn) LocalAddr() net.Addr {
	return conn.localAddr
}

func (conn *tcpConn) SetDeadline(t time.Time) error {
	return nil
}
func (conn *tcpConn) SetReadDeadline(t time.Time) error {
	return nil
}
func (conn *tcpConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (conn *tcpConn) receiveCheck() error {
	conn.Lock()
	defer conn.Unlock()

	switch conn.state {
	case tcpConnected:
		fallthrough
	case tcpWriteClosed:
		return nil
	case tcpNewConn:
		fallthrough
	case tcpConnecting:
		return NewLWIPError(LWIP_ERR_CONN)
	case tcpAborting:
		fallthrough
	case tcpClosed:
		fallthrough
	case tcpReceiveClosed:
		fallthrough
	case tcpClosing:
		return NewLWIPError(LWIP_ERR_CLSD)
	case tcpErrored:
		conn.abortInternal()
		return NewLWIPError(LWIP_ERR_ABRT)
	default:
		panic("unexpected error")
	}
	return nil
}

func (conn *tcpConn) Receive(data []byte) error {
	if err := conn.receiveCheck(); err != nil {
		return err
	}
	_, err := conn.sndPipeWriter.Write(data)
	if err != nil {
		return NewLWIPError(LWIP_ERR_CLSD)
	}
	return NewLWIPError(LWIP_ERR_OK)
}

func (conn *tcpConn) Read(data []byte) (int, error) {
	conn.Lock()
	if conn.state == tcpReceiveClosed {
		conn.Unlock()
		return 0, io.EOF
	}
	if conn.state >= tcpClosing {
		conn.Unlock()
		return 0, io.ErrClosedPipe
	}
	conn.Unlock()

	// Handler should get EOF.
	n, err := conn.sndPipeReader.Read(data)
	if err == io.ErrClosedPipe {
		err = io.EOF
	}

	lwipMutex.Lock()
	C.tcp_recved(conn.pcb, C.u16_t(n))
	lwipMutex.Unlock()

	return n, err
}

// writeInternal enqueues data to snd_buf, and treats ERR_MEM returned by tcp_write not an error,
// but instead tells the caller that data is not successfully enqueued, and should try
// again another time. By calling this function, the lwIP thread is assumed to be already
// locked by the caller.
func (conn *tcpConn) writeInternal(data []byte) (int, error) {
	lwipMutex.Lock()
	err := C.tcp_write(conn.pcb, unsafe.Pointer(&data[0]), C.u16_t(len(data)), C.TCP_WRITE_FLAG_COPY)
	if err == C.ERR_OK {
		lwipMutex.Unlock()
		return len(data), nil
	} else if err == C.ERR_MEM {
		lwipMutex.Unlock()
		return 0, nil
	}
	lwipMutex.Unlock()
	return 0, fmt.Errorf("tcp_write failed (%v)", int(err))
}

func (conn *tcpConn) tcpOutputInternal() error {
	lwipMutex.Lock()
	err := C.tcp_output(conn.pcb)
	if err != C.ERR_OK {
		lwipMutex.Unlock()
		return fmt.Errorf("tcp_output failed (%v)", int(err))
	}
	lwipMutex.Unlock()
	return nil

}

func (conn *tcpConn) writeCheck() error {
	conn.Lock()
	defer conn.Unlock()

	switch conn.state {
	case tcpConnecting:
		fallthrough
	case tcpConnected:
		fallthrough
	case tcpReceiveClosed:
		return nil
	case tcpWriteClosed:
		fallthrough
	case tcpClosing:
		fallthrough
	case tcpClosed:
		fallthrough
	case tcpErrored:
		fallthrough
	case tcpAborting:
		return io.ErrClosedPipe
	default:
		panic("unexpected error")
	}
	return nil
}

func (conn *tcpConn) Write(data []byte) (int, error) {
	totalWritten := 0

	for len(data) > 0 {
		if err := conn.writeCheck(); err != nil {
			return totalWritten, err
		}

		toWrite := len(data)

		lwipMutex.Lock()
		sendBufLen := C.tcp_sndbuf_cgo(conn.pcb)
		lwipMutex.Unlock()

		if toWrite > int(sendBufLen) {
			// Write at most the size of the LWIP buffer.
			toWrite = int(sendBufLen)
		}
		if toWrite > 0 {
			written, err := conn.writeInternal(data[0:toWrite])
			if err != nil {
				return totalWritten, err
			}
			totalWritten += written
			data = data[written:len(data)]
		}
	}

	err := conn.tcpOutputInternal()
	if err != nil {
		return totalWritten, err
	}
	return totalWritten, nil
}

func (conn *tcpConn) CloseWrite() error {
	conn.Lock()
	if conn.state >= tcpClosing || conn.state == tcpWriteClosed {
		conn.Unlock()
		return nil
	}
	if conn.state == tcpReceiveClosed {
		conn.state = tcpClosing
	} else {
		conn.state = tcpWriteClosed
	}
	conn.Unlock()

	lwipMutex.Lock()
	// FIXME Handle tcp_shutdown error.
	C.tcp_shutdown(conn.pcb, 0, 1)
	lwipMutex.Unlock()

	return nil
}

func (conn *tcpConn) CloseRead() error {
	return conn.sndPipeReader.Close()
}

func (conn *tcpConn) Sent(len uint16) error {
	// Some packets are acknowledged by local client, check if any pending data to send.
	return conn.checkState()
}

func (conn *tcpConn) checkClosing() error {
	conn.Lock()

	if conn.state == tcpClosing {
		conn.Unlock()

		conn.release()
		conn.closeInternal()
		return NewLWIPError(LWIP_ERR_OK)
	}
	conn.Unlock()
	return nil
}

func (conn *tcpConn) checkAborting() error {
	conn.Lock()

	if conn.state == tcpAborting {
		conn.Unlock()

		conn.release()
		conn.abortInternal()
		return NewLWIPError(LWIP_ERR_ABRT)
	}
	conn.Unlock()
	return nil
}

func (conn *tcpConn) isClosed() bool {
	conn.Lock()
	ret := conn.state == tcpClosed
	conn.Unlock()
	return ret
}

func (conn *tcpConn) checkState() error {
	if conn.isClosed() {
		return nil
	}

	err := conn.checkClosing()
	if err != nil {
		return err
	}

	err = conn.checkAborting()
	if err != nil {
		return err
	}

	return NewLWIPError(LWIP_ERR_OK)
}

func (conn *tcpConn) Close() error {
	conn.closeOnce.Do(conn.close)
	return conn.closeErr
}

func (conn *tcpConn) close() {
	err := conn.CloseRead()
	if err != nil {
		conn.closeErr = err
	}
	err = conn.CloseWrite()
	if err != nil {
		conn.closeErr = err
	}
}

func (conn *tcpConn) setLocalClosed() error {
	conn.Lock()
	defer conn.Unlock()

	if conn.state >= tcpClosing || conn.state == tcpReceiveClosed {
		return nil
	}

	// Causes the read half of the pipe returns.
	conn.sndPipeWriter.Close()

	if conn.state == tcpWriteClosed {
		conn.state = tcpClosing
	} else {
		conn.state = tcpReceiveClosed
	}
	return nil
}

// Never call this function outside of the lwIP thread.
func (conn *tcpConn) closeInternal() error {
	// lwipMutex.Lock()
	// defer lwipMutex.Unlock()
	C.tcp_arg(conn.pcb, nil)
	C.tcp_recv(conn.pcb, nil)
	C.tcp_sent(conn.pcb, nil)
	C.tcp_err(conn.pcb, nil)

	// FIXME Handle error.
	err := C.tcp_close(conn.pcb)
	switch err {
	case C.ERR_OK:
		// ERR_OK if connection has been closed
		break
	case C.ERR_ARG:
		// invalid pointer or state
		panic("closeInternal: tcp pcb is invalid")
	default:
		// another err_t if closing failed and pcb is not freed
		// make sure tcp_free is invoked
		C.tcp_abort(conn.pcb)
	}
	if err == C.ERR_OK {
		return nil
	} else {
		return errors.New(fmt.Sprintf("close TCP connection failed, lwip error code %d", int(err)))
	}
}

// Never call this function outside of the lwIP thread since it calls
// tcp_abort() and in that case we must return ERR_ABRT to lwIP.
func (conn *tcpConn) abortInternal() {
	// lwipMutex.Lock()
	// defer lwipMutex.Unlock()
	C.tcp_abort(conn.pcb)
}

func (conn *tcpConn) Abort() {
	conn.Lock()
	// If it's in tcpErrored state, the pcb was already freed.
	if conn.state < tcpAborting {
		conn.state = tcpAborting
	}
	conn.Unlock()

	lwipMutex.Lock()
	conn.checkState()
	lwipMutex.Unlock()
}

func (conn *tcpConn) Err(err error) {
	conn.Lock()
	conn.state = tcpErrored
	conn.Unlock()

	conn.release()

}

func (conn *tcpConn) LocalClosed() error {
	conn.setLocalClosed()
	return conn.checkState()
}

func (conn *tcpConn) release() {
	// lwipMutex.Lock()
	// defer lwipMutex.Unlock()

	tcpConns.Delete(conn)

	conn.sndPipeWriter.Close()
	conn.sndPipeReader.Close()
	conn.state = tcpClosed

}

func (conn *tcpConn) Poll() error {
	return conn.checkState()
}
