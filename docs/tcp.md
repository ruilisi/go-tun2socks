# TCP in go-tun2socks

## Table of Contents
1. [TCP Protocol Fundamentals](#tcp-protocol-fundamentals)
2. [Standard Go TCP Implementation](#standard-go-tcp-implementation)
3. [go-tun2socks TCP Architecture](#go-tun2socks-tcp-architecture)
4. [TCP Connection Lifecycle](#tcp-connection-lifecycle)
5. [TCP State Machine](#tcp-state-machine)
6. [Flow Control and Congestion Control](#flow-control-and-congestion-control)
7. [Data Flow: Read and Write Paths](#data-flow-read-and-write-paths)
8. [TCP Connection Handling](#tcp-connection-handling)
9. [Error Handling and Connection Termination](#error-handling-and-connection-termination)
10. [Performance Considerations](#performance-considerations)
11. [Debugging TCP Issues](#debugging-tcp-issues)

---

## TCP Protocol Fundamentals

### What is TCP?

**TCP (Transmission Control Protocol)** is a connection-oriented, reliable transport layer protocol that provides:

1. **Reliable delivery**: Guarantees data arrives without corruption or loss
2. **Ordered delivery**: Data arrives in the same order it was sent
3. **Flow control**: Prevents sender from overwhelming receiver
4. **Congestion control**: Prevents network overload
5. **Error detection**: Detects corrupted packets via checksums
6. **Full-duplex communication**: Simultaneous bidirectional data transfer

### TCP Segment Structure

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Source Port          |       Destination Port        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Sequence Number                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Acknowledgment Number                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Data |       |C|E|U|A|P|R|S|F|                               |
| Offset| Rsrvd |W|C|R|C|S|S|Y|I|            Window             |
|       |       |R|E|G|K|H|T|N|N|                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Checksum            |         Urgent Pointer        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Options                    |    Padding    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                             data                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Key Fields**:
- **Sequence Number**: Position of first byte in this segment
- **Acknowledgment Number**: Next expected byte from the other side
- **Window**: Available receive buffer space (flow control)
- **Flags**: SYN, ACK, FIN, RST, PSH, URG

### Connection Establishment (Three-Way Handshake)

```
Client                                Server
  |                                     |
  |----------- SYN (seq=x) ----------->|
  |                                     |
  |<----- SYN-ACK (seq=y, ack=x+1) ----|
  |                                     |
  |----------- ACK (ack=y+1) --------->|
  |                                     |
  |    Connection ESTABLISHED          |
```

**Steps**:
1. Client sends SYN with initial sequence number (ISN)
2. Server responds with SYN-ACK, acknowledging client's SYN and sending its own ISN
3. Client acknowledges server's SYN
4. Connection is established, data transfer can begin

### Data Transfer

```
Client                                Server
  |                                     |
  |---- DATA (seq=x, len=100) -------->|
  |                                     |
  |<------- ACK (ack=x+100) -----------|
  |                                     |
  |---- DATA (seq=x+100, len=200) ---->|
  |                                     |
  |<------- ACK (ack=x+300) -----------|
```

**Mechanisms**:
- **Cumulative ACK**: ACK number indicates next expected byte
- **Retransmission**: If ACK not received within timeout, segment is retransmitted
- **Selective ACK (SACK)**: Allows acknowledging non-contiguous blocks (enabled in go-tun2socks)
- **Window Scaling**: Allows window sizes > 64KB
- **Timestamps**: For RTT measurement and PAWS (enabled in go-tun2socks)

### Connection Termination (Four-Way Handshake)

```
Client                                Server
  |                                     |
  |----------- FIN (seq=x) ----------->|
  |                                     |
  |<---------- ACK (ack=x+1) ----------|
  |                                     |
  |<---------- FIN (seq=y) ------------|
  |                                     |
  |----------- ACK (ack=y+1) --------->|
  |                                     |
  |    Connection CLOSED               |
```

**Half-Close**: TCP supports closing one direction while keeping the other open:
- Client sends FIN → Can't send more data, but can still receive
- Server ACKs FIN → Can still send data to client
- Server sends FIN when done → Both directions closed

---

## Standard Go TCP Implementation

### Basic TCP Server

```go
package main

import (
    "bufio"
    "net"
)

func main() {
    // Listen on TCP port 8080
    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        panic(err)
    }
    defer listener.Close()

    for {
        // Accept blocks until new connection arrives
        conn, err := listener.Accept()
        if err != nil {
            continue
        }

        // Handle connection in goroutine
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    reader := bufio.NewReader(conn)
    for {
        // Read data
        data, err := reader.ReadBytes('\n')
        if err != nil {
            return
        }

        // Write response
        conn.Write([]byte("Echo: "))
        conn.Write(data)
    }
}
```

### Basic TCP Client

```go
package main

import (
    "net"
)

func main() {
    // Connect to server
    conn, err := net.Dial("tcp", "example.com:8080")
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // Write data
    _, err = conn.Write([]byte("Hello, server!\n"))
    if err != nil {
        panic(err)
    }

    // Read response
    buffer := make([]byte, 1024)
    n, err := conn.Read(buffer)
    if err != nil {
        panic(err)
    }

    println(string(buffer[:n]))
}
```

### How Go Handles TCP Under the Hood

1. **Goroutine-based concurrency**: Each connection typically gets its own goroutine
2. **Network poller**: Uses epoll (Linux) / kqueue (BSD/macOS) / IOCP (Windows) for efficient I/O multiplexing
3. **Blocking I/O with goroutine scheduling**: `Read()`/`Write()` block the goroutine but not the OS thread
4. **Kernel TCP stack**: All TCP protocol logic (segments, ACKs, retransmits) handled by OS kernel
5. **Socket API**: Go's `net` package wraps syscalls like `socket()`, `bind()`, `listen()`, `accept()`, `connect()`, `read()`, `write()`

**Advantages**:
- Simple, high-level API
- Excellent performance (kernel optimizations)
- Automatic memory management
- Built-in timeouts and deadlines

**Limitations for TUN2SOCKS**:
- Can't intercept raw IP packets
- Can't redirect connections without kernel routing changes
- Limited control over TCP behavior

---

## go-tun2socks TCP Architecture

### Why lwIP Instead of Kernel TCP?

go-tun2socks needs to:
1. **Receive raw IP packets** from TUN device
2. **Process TCP in userspace** to intercept connections
3. **Redirect connections** to SOCKS/HTTP proxies transparently
4. **Send response packets** back to TUN device

The kernel TCP stack can't do this because:
- It would route packets normally based on routing table
- No way to transparently intercept and redirect connections
- TUN device provides raw packets, not TCP streams

**Solution**: Use **lwIP (lightweight IP)**, a complete TCP/IP stack designed to run in userspace.

### lwIP Overview

**lwIP** is a small, fast TCP/IP implementation originally designed for embedded systems:

- **Single-threaded**: Not thread-safe, requires external locking
- **Event-driven**: Uses callbacks for I/O events
- **Memory pools**: Pre-allocated buffers (pbufs) for efficient memory usage
- **NO_SYS mode**: Used in go-tun2socks without OS threading primitives
- **Complete stack**: IP, ICMP, UDP, TCP, ARP, DHCP, DNS

**Configuration** (`core/c/custom/lwipopts.h`):
```c
// Memory
#define MEM_SIZE                 (32 * MB)    // 32MB heap
#define PBUF_POOL_SIZE           5120         // 5120 packet buffers
#define PBUF_POOL_BUFSIZE        1600         // 1600 bytes each

// Connections
#define MEMP_NUM_TCP_PCB         5120         // Max 5120 TCP connections
#define MEMP_NUM_UDP_PCB         2560         // Max 2560 UDP connections

// TCP Parameters
#define TCP_MSS                  1460         // Maximum Segment Size
#define TCP_SND_BUF              (64 * KB)    // Send buffer: 64KB
#define TCP_WND                  (64 * KB - 1)// Receive window: 64KB

// Performance Features
#define LWIP_TCP_SACK_OUT        1            // Selective ACK
#define LWIP_TCP_TIMESTAMPS      1            // TCP timestamps
#define LWIP_TCP_KEEPALIVE       1            // TCP keepalive
```

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Application (Browser, etc.)              │
└────────────────────────┬────────────────────────────────────┘
                         │ Writes to TUN device
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    TUN Device (tun1)                        │
│              Raw IP packets (userspace)                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              go-tun2socks (main.go)                         │
│          io.CopyBuffer(lwipWriter, tunDev)                  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  lwipStack.Write()                          │
│              ┌──────────────────┐                           │
│              │   lwipMutex      │  ← GLOBAL LOCK            │
│              └──────────────────┘                           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  lwIP C Stack                               │
│   IP layer → TCP layer → Callbacks                         │
│                                                              │
│   tcpAcceptFn()  - New connection                          │
│   tcpRecvFn()    - Data received                           │
│   tcpSentFn()    - Data acknowledged                       │
│   tcpErrFn()     - Error occurred                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              tcpConn (core/tcp_conn.go)                     │
│      Implements net.Conn interface                          │
│      Bridges lwIP ↔ Go                                     │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│          Proxy Handler (proxy/socks/tcp.go)                 │
│      Connects to SOCKS proxy and relays data               │
│      TUN ↔ Proxy ↔ Internet                               │
└─────────────────────────────────────────────────────────────┘
```

### Thread Safety with lwipMutex

lwIP is **NOT thread-safe**. go-tun2socks uses a **global recursive mutex** (`core/lwip_access_wrapper.go:23`):

```go
var lwipMutex = &syncex.RecursiveMutex{}
```

**Every interaction with lwIP requires this lock**:
- Packet input: `input()`
- Packet output: `output()`
- TCP operations: `tcp_write()`, `tcp_read()`, `tcp_close()`
- All callbacks: `tcpRecvFn()`, `tcpSentFn()`, etc.

**Why recursive?**
- lwIP callbacks may call back into lwIP (e.g., `tcp_close()` in error path)
- Prevents deadlock when same goroutine needs to re-acquire lock
- Adds overhead but necessary for correctness

---

## TCP Connection Lifecycle

### 1. Connection Establishment

#### Step 1: TUN Device Receives SYN Packet

```
Application → OS → TUN device → go-tun2socks
```

The main loop reads packets (`cmd/tun2socks/main_*.go`):
```go
io.CopyBuffer(lwipWriter, tunDev, buf)
```

#### Step 2: Packet Forwarded to lwIP

`core/input.go:89`:
```go
func input(pkt []byte) (int, error) {
    lwipMutex.Lock()
    defer lwipMutex.Unlock()

    // Allocate pbuf from pool
    buf := C.pbuf_alloc(C.PBUF_RAW, C.u16_t(len(pkt)), C.PBUF_POOL)

    // Copy packet data
    C.pbuf_take_at(buf, unsafe.Pointer(&pkt[0]), ...)

    // Process in lwIP stack
    err := C.input(buf)

    return len(pkt), nil
}
```

#### Step 3: lwIP Processes SYN

lwIP's TCP layer (`C.tcp_input()`) detects new connection and calls:

`core/tcp_callback_export.go:23`:
```go
//export tcpAcceptFn
func tcpAcceptFn(arg unsafe.Pointer, newpcb *C.struct_tcp_pcb, err C.err_t) C.err_t {
    // Still holding lwipMutex!

    if handler == nil {
        return C.ERR_ABRT
    }

    // Create new connection object
    conn, err := newTCPConn(newpcb, handler)
    if err != nil {
        return err
    }

    return C.ERR_OK
}
```

#### Step 4: Create tcpConn Object

`core/tcp_conn.go:98`:
```go
func newTCPConn(pcb *C.struct_tcp_pcb, handler TCPConnHandler) (TCPConn, error) {
    lwipMutex.Lock()
    defer lwipMutex.Unlock()

    // Disable Nagle's algorithm (reduce latency)
    C.tcp_nagle_disable_cgo(pcb)

    // Enable TCP keepalive
    C.tcp_keepalive_settings_cgo(pcb)

    // Register callbacks
    setTCPRecvCallback(pcb)
    setTCPSentCallback(pcb)
    setTCPErrCallback(pcb)

    // Create pipe for data flow: lwIP → handler
    buf := buffer.New(0xffff)  // 64KB buffer
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

    // Store connection pointer in PCB
    C.tcp_arg_cgo(pcb, C.uintptr_t(uintptr(unsafe.Pointer(conn))))

    // Track connection globally
    tcpConns.Store(conn, true)

    // Launch handler in goroutine (non-blocking)
    conn.state = tcpConnecting
    go func() {
        err := handler.Handle(TCPConn(conn), conn.remoteAddr)
        if err != nil {
            conn.Abort()
        } else {
            conn.Lock()
            if conn.state == tcpConnecting {
                conn.state = tcpConnected
            }
            conn.Unlock()
        }
    }()

    return conn, NewLWIPError(LWIP_ERR_OK)
}
```

#### Step 5: Handler Connects to Proxy

`proxy/socks/tcp.go:88`:
```go
func (h *tcpHandler) Handle(conn net.Conn, target *net.TCPAddr) error {
    // Create SOCKS5 dialer
    dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, nil)
    if err != nil {
        return err
    }

    // Connect to target through SOCKS proxy
    c, err := dialer.Dial(target.Network(), target.String())
    if err != nil {
        return err
    }

    // Start bidirectional relay
    go h.relay(conn, c)

    return nil
}
```

### 2. Data Transfer

#### Client → Server (Uplink)

```
TUN → lwIP → tcpRecvFn → conn.Receive() → pipe → handler.Read() → proxy.Write() → Internet
```

**Step 1**: lwIP receives TCP segment with data

**Step 2**: `tcpRecvFn()` callback (`core/tcp_callback_export.go:49`):
```go
//export tcpRecvFn
func tcpRecvFn(arg unsafe.Pointer, pcb *C.struct_tcp_pcb, p *C.struct_pbuf, err C.err_t) C.err_t {
    lwipMutex.Lock()
    defer lwipMutex.Unlock()

    // Get connection from PCB
    conn := (*tcpConn)(arg)

    // Handle FIN (connection close)
    if p == nil {
        conn.setLocalClosed()
        return C.ERR_OK
    }

    // Copy data from pbuf
    totlen := int(p.tot_len)
    var buf []byte

    if p.tot_len == p.len {
        // Single buffer, direct access
        buf = (*[1 << 30]byte)(unsafe.Pointer(p.payload))[:totlen:totlen]
    } else {
        // Fragmented, need to copy
        buf = pool.NewBytes(totlen)
        defer pool.FreeBytes(buf)
        C.pbuf_copy_partial(p, unsafe.Pointer(&buf[0]), p.tot_len, 0)
    }

    // Free pbuf
    C.pbuf_free(p)

    // Pass to connection
    rerr := conn.Receive(buf[:totlen])

    return C.err_t(rerr)
}
```

**Step 3**: `conn.Receive()` writes to pipe (`core/tcp_conn.go:196`):
```go
func (conn *tcpConn) Receive(data []byte) error {
    if err := conn.receiveCheck(); err != nil {
        return err
    }

    // Write to pipe (may block if buffer full)
    _, err := conn.sndPipeWriter.Write(data)
    if err != nil {
        return NewLWIPError(LWIP_ERR_CLSD)
    }

    return NewLWIPError(LWIP_ERR_OK)
}
```

**Step 4**: Handler reads from connection (`core/tcp_conn.go:207`):
```go
func (conn *tcpConn) Read(data []byte) (int, error) {
    // Check connection state
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

    // Read from pipe (blocks until data available)
    n, err := conn.sndPipeReader.Read(data)
    if err == io.ErrClosedPipe {
        err = io.EOF
    }

    // Acknowledge to lwIP (flow control!)
    lwipMutex.Lock()
    C.tcp_recved(conn.pcb, C.u16_t(n))
    lwipMutex.Unlock()

    return n, err
}
```

The `C.tcp_recved()` call is **critical** - it tells lwIP the application has consumed `n` bytes, allowing lwIP to:
- Update the receive window
- Send ACK with updated window to peer
- Accept more data from peer

**Step 5**: Handler writes to proxy:
```go
// In relay goroutine
io.Copy(proxyConn, tunConn)  // tunConn.Read() → proxyConn.Write()
```

#### Server → Client (Downlink)

```
Internet → proxy.Read() → handler → conn.Write() → lwIP → output() → TUN
```

**Step 1**: Handler reads from proxy

**Step 2**: Handler writes to connection (`core/tcp_conn.go:289`):
```go
func (conn *tcpConn) Write(data []byte) (int, error) {
    totalWritten := 0

    for len(data) > 0 {
        // Check if connection is writable
        if err := conn.writeCheck(); err != nil {
            return totalWritten, err
        }

        toWrite := len(data)

        // Check available send buffer space
        lwipMutex.Lock()
        sendBufLen := C.tcp_sndbuf_cgo(conn.pcb)
        lwipMutex.Unlock()

        if toWrite > int(sendBufLen) {
            // Don't overflow lwIP's send buffer
            toWrite = int(sendBufLen)
        }

        if toWrite > 0 {
            // Write to lwIP
            written, err := conn.writeInternal(data[0:toWrite])
            if err != nil {
                return totalWritten, err
            }
            totalWritten += written
            data = data[written:]
        }
    }

    // Trigger transmission
    err := conn.tcpOutputInternal()
    if err != nil {
        return totalWritten, err
    }

    return totalWritten, nil
}
```

**Step 3**: `writeInternal()` enqueues data (`core/tcp_conn.go:236`):
```go
func (conn *tcpConn) writeInternal(data []byte) (int, error) {
    lwipMutex.Lock()

    // Enqueue data in lwIP's send buffer
    err := C.tcp_write(conn.pcb, unsafe.Pointer(&data[0]), C.u16_t(len(data)), C.TCP_WRITE_FLAG_COPY)

    if err == C.ERR_OK {
        lwipMutex.Unlock()
        return len(data), nil
    } else if err == C.ERR_MEM {
        // Buffer full, try again later
        lwipMutex.Unlock()
        return 0, nil
    }

    lwipMutex.Unlock()
    return 0, fmt.Errorf("tcp_write failed (%v)", int(err))
}
```

The `C.TCP_WRITE_FLAG_COPY` flag tells lwIP to **copy** the data (Go's GC can move the original).

**Step 4**: `tcpOutputInternal()` triggers transmission (`core/tcp_conn.go:250`):
```go
func (conn *tcpConn) tcpOutputInternal() error {
    lwipMutex.Lock()

    // Ask lwIP to send buffered data
    err := C.tcp_output(conn.pcb)

    if err != C.ERR_OK {
        lwipMutex.Unlock()
        return fmt.Errorf("tcp_output failed (%v)", int(err))
    }

    lwipMutex.Unlock()
    return nil
}
```

**Step 5**: lwIP builds TCP segments and calls output callback

**Step 6**: `output()` writes to TUN (`core/output_export.go:14`):
```go
//export output
func output(p *C.struct_pbuf) C.err_t {
    lwipMutex.Lock()

    totlen := int(p.tot_len)
    var buf []byte

    if p.tot_len == p.len {
        // Direct access
        buf = (*[1 << 30]byte)(unsafe.Pointer(p.payload))[:totlen:totlen]
    } else {
        // Copy fragmented pbuf
        buf = pool.NewBytes(totlen)
        defer pool.FreeBytes(buf)
        C.pbuf_copy_partial(p, unsafe.Pointer(&buf[0]), p.tot_len, 0)
    }

    lwipMutex.Unlock()

    // Write to TUN device
    OutputFn(buf[:totlen])

    return C.ERR_OK
}
```

### 3. Connection Termination

#### Normal Close (FIN)

**Scenario 1: Client closes connection**

```
Client sends FIN → TUN → lwIP → tcpRecvFn(p=nil) → setLocalClosed() → state=tcpReceiveClosed
```

`core/tcp_conn.go:502`:
```go
func (conn *tcpConn) setLocalClosed() error {
    conn.Lock()
    defer conn.Unlock()

    if conn.state >= tcpClosing || conn.state == tcpReceiveClosed {
        return nil
    }

    // Close pipe writer to signal EOF to handler
    conn.sndPipeWriter.Close()

    if conn.state == tcpWriteClosed {
        conn.state = tcpClosing
    } else {
        conn.state = tcpReceiveClosed
    }

    return nil
}
```

Handler reads will now return `io.EOF`.

**Scenario 2: Handler closes connection**

```go
// In relay goroutine
io.Copy(proxyConn, tunConn)  // Returns when tunConn.Read() returns EOF
tunConn.CloseWrite()         // Send FIN to client
```

`core/tcp_conn.go:324`:
```go
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

    // Send FIN to peer
    lwipMutex.Lock()
    C.tcp_shutdown(conn.pcb, 0, 1)  // Shutdown write side
    lwipMutex.Unlock()

    return nil
}
```

**Scenario 3: Both sides closed**

When `state == tcpClosing`, the connection is torn down:

`core/tcp_conn.go:443`:
```go
func (conn *tcpConn) closeInternal() error {
    lwipMutex.Lock()
    defer lwipMutex.Unlock()

    // Unregister callbacks
    C.tcp_arg(conn.pcb, nil)
    C.tcp_recv(conn.pcb, nil)
    C.tcp_sent(conn.pcb, nil)
    C.tcp_err(conn.pcb, nil)

    // Close TCP connection
    err := C.tcp_close(conn.pcb)

    switch err {
    case C.ERR_OK:
        // Success
        break
    case C.ERR_ARG:
        panic("closeInternal: tcp pcb is invalid")
    default:
        // Failed, force abort
        C.tcp_abort(conn.pcb)
    }

    return nil
}
```

#### Abnormal Close (RST)

**Abort connection** (immediate termination):

`core/tcp_conn.go:480`:
```go
func (conn *tcpConn) Abort() {
    conn.Lock()
    if conn.state < tcpAborting {
        conn.state = tcpAborting
    }
    conn.Unlock()

    lwipMutex.Lock()
    conn.checkState()
    lwipMutex.Unlock()
}

func (conn *tcpConn) abortInternal() {
    lwipMutex.Lock()
    defer lwipMutex.Unlock()

    // Send RST to peer, free PCB immediately
    C.tcp_abort(conn.pcb)
}
```

`tcp_abort()` sends a RST segment and immediately frees the PCB (no TIME_WAIT).

---

## TCP State Machine

### Connection States

Defined in `core/tcp_conn.go:47-82`:

```go
type tcpConnState uint

const (
    tcpNewConn       tcpConnState = iota  // Initial state
    tcpConnecting                         // Handler connecting to proxy
    tcpConnected                          // Fully connected
    tcpWriteClosed                        // Write side closed (FIN sent)
    tcpReceiveClosed                      // Read side closed (FIN received)
    tcpClosing                            // Both sides closed
    tcpAborting                           // Abort in progress
    tcpClosed                             // Connection closed cleanly
    tcpErrored                            // Fatal error occurred
)
```

### State Transitions

```
                  ┌──────────────┐
                  │  tcpNewConn  │
                  └──────┬───────┘
                         │
                         │ Handler launched
                         ▼
                  ┌──────────────┐
                  │tcpConnecting │
                  └──────┬───────┘
                         │
                         │ Proxy connection established
                         ▼
                  ┌──────────────┐
           ┌──────┤ tcpConnected ├──────┐
           │      └──────────────┘      │
           │                            │
           │ CloseWrite()               │ setLocalClosed()
           │ (send FIN)                 │ (receive FIN)
           │                            │
           ▼                            ▼
    ┌──────────────┐            ┌──────────────┐
    │tcpWriteClosed│            │tcpReceiveClosed
    └──────┬───────┘            └──────┬───────┘
           │                            │
           │ setLocalClosed()           │ CloseWrite()
           │                            │
           └────────────┬───────────────┘
                        │
                        ▼
                  ┌──────────────┐
                  │  tcpClosing  │
                  └──────┬───────┘
                         │
                         │ closeInternal()
                         ▼
                  ┌──────────────┐
                  │  tcpClosed   │
                  └──────────────┘

                  ┌──────────────┐
      Error──────►│ tcpErrored   │
                  └──────┬───────┘
                         │
                         │ abortInternal()
                         ▼
                  ┌──────────────┐
      Abort──────►│ tcpAborting  │
                  └──────┬───────┘
                         │
                         ▼
                  ┌──────────────┐
                  │  tcpClosed   │
                  └──────────────┘
```

### State Validation

Before operations, state is checked:

**Read validation** (`core/tcp_conn.go:166`):
```go
func (conn *tcpConn) receiveCheck() error {
    conn.Lock()
    defer conn.Unlock()

    switch conn.state {
    case tcpConnected:
        fallthrough
    case tcpWriteClosed:
        return nil  // Can receive
    case tcpNewConn:
        fallthrough
    case tcpConnecting:
        return NewLWIPError(LWIP_ERR_CONN)  // Not ready
    case tcpAborting:
        fallthrough
    case tcpClosed:
        fallthrough
    case tcpReceiveClosed:
        fallthrough
    case tcpClosing:
        return NewLWIPError(LWIP_ERR_CLSD)  // Closed
    case tcpErrored:
        conn.abortInternal()
        return NewLWIPError(LWIP_ERR_ABRT)  // Error
    }
}
```

**Write validation** (`core/tcp_conn.go:262`):
```go
func (conn *tcpConn) writeCheck() error {
    conn.Lock()
    defer conn.Unlock()

    switch conn.state {
    case tcpConnecting:
        fallthrough
    case tcpConnected:
        fallthrough
    case tcpReceiveClosed:
        return nil  // Can write
    case tcpWriteClosed:
        fallthrough
    case tcpClosing:
        fallthrough
    case tcpClosed:
        fallthrough
    case tcpErrored:
        fallthrough
    case tcpAborting:
        return io.ErrClosedPipe  // Can't write
    }
}
```

---

## Flow Control and Congestion Control

### Receive Window (Flow Control)

**Purpose**: Prevent sender from overwhelming receiver's buffer.

**Mechanism**:
1. Receiver advertises available buffer space in TCP window field
2. Sender limits outstanding (unacknowledged) data to window size
3. Window size updated in every ACK

**In go-tun2socks**:

Configuration (`core/c/custom/lwipopts.h`):
```c
#define TCP_WND  (64 * 1024 - 1)  // 64KB receive window
```

When handler reads data, it **must** notify lwIP:
```go
n, _ := conn.Read(buffer)
C.tcp_recved(conn.pcb, C.u16_t(n))  // Update window
```

Without `tcp_recved()`:
- lwIP's window stays at 0
- Peer stops sending data
- Connection stalls!

### Send Buffer (Flow Control)

**Purpose**: Limit buffered outgoing data.

Configuration:
```c
#define TCP_SND_BUF  (64 * 1024)  // 64KB send buffer
```

Before writing, check available space:
```go
sendBufLen := C.tcp_sndbuf_cgo(conn.pcb)
if toWrite > int(sendBufLen) {
    toWrite = int(sendBufLen)
}
```

If buffer full (`ERR_MEM`), wait and retry.

### Congestion Control

**Purpose**: Prevent network overload.

lwIP implements:
- **Slow Start**: Exponentially increase cwnd until ssthresh
- **Congestion Avoidance**: Linear increase after ssthresh
- **Fast Retransmit**: Retransmit on 3 duplicate ACKs
- **Fast Recovery**: Reduce cwnd on loss, avoid slow start

Configuration:
```c
#define LWIP_TCP_SACK_OUT 1  // Selective ACK for better loss recovery
```

These are handled entirely by lwIP, transparent to Go code.

### Nagle's Algorithm

**Purpose**: Reduce small packet overhead by buffering.

**Disabled in go-tun2socks** (`core/tcp_conn.go:102`):
```c
C.tcp_nagle_disable_cgo(pcb)
```

**Why disable?**
- Reduces latency for interactive applications
- Proxy traffic often small messages (HTTP requests, SSH, etc.)
- Modern networks have higher bandwidth, overhead less critical

### TCP Keepalive

**Purpose**: Detect dead connections.

**Enabled in go-tun2socks** (`core/tcp_conn.go:103`):
```c
C.tcp_keepalive_settings_cgo(pcb)
```

Configuration:
```c
#define LWIP_TCP_KEEPALIVE 1
```

Default keepalive interval: 75 seconds (lwIP default).

---

## Data Flow: Read and Write Paths

### Read Path (Detailed)

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. TUN Device                                                    │
│    Kernel delivers IP packet to TUN file descriptor             │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. main.go                                                       │
│    io.CopyBuffer(lwipWriter, tunDev, buf)                       │
│    Read from TUN, write to lwipWriter                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. lwipStack.Write() (lwip.go:242)                              │
│    func (s *lwipStack) Write(data []byte) (int, error)         │
│    Dispatches to input()                                        │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. input() (input.go:89)                                        │
│    lwipMutex.Lock()                                             │
│    buf := C.pbuf_alloc(PBUF_RAW, pktLen, PBUF_POOL)            │
│    C.pbuf_take_at(buf, &pkt[0], ...)                           │
│    C.input(buf)  ← Call into lwIP C stack                      │
│    lwipMutex.Unlock()                                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. lwIP C Stack Processing                                      │
│    ip_input() → validates IP header, routing                   │
│    tcp_input() → TCP state machine, reassembly                 │
│    Calls tcpRecvFn() callback                                  │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. tcpRecvFn() (tcp_callback_export.go:49)                     │
│    lwipMutex already locked!                                    │
│    Get tcpConn from PCB                                         │
│    Copy data from pbuf to Go slice                             │
│    C.pbuf_free(p)                                               │
│    conn.Receive(buf)                                            │
│    lwipMutex.Unlock()                                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 7. conn.Receive() (tcp_conn.go:196)                            │
│    sndPipeWriter.Write(data)                                    │
│    Writes to buffered pipe (64KB buffer)                       │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 8. Handler goroutine                                            │
│    n, err := conn.Read(buffer)                                  │
│    Reads from sndPipeReader                                     │
│    C.tcp_recved(pcb, n) ← Ack consumed data                    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 9. Proxy relay                                                  │
│    proxyConn.Write(buffer[:n])                                  │
│    Forward to SOCKS proxy → Internet                           │
└─────────────────────────────────────────────────────────────────┘
```

### Write Path (Detailed)

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Proxy relay                                                  │
│    n, err := proxyConn.Read(buffer)                            │
│    Read response from SOCKS proxy                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Handler goroutine                                            │
│    conn.Write(buffer[:n])                                       │
│    Write to TUN connection                                      │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. conn.Write() (tcp_conn.go:289)                              │
│    Loop over data:                                              │
│      sendBufLen := C.tcp_sndbuf_cgo(pcb)  ← Check space        │
│      toWrite := min(len(data), sendBufLen)                     │
│      conn.writeInternal(data[:toWrite])                        │
│    conn.tcpOutputInternal()  ← Trigger send                    │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. writeInternal() (tcp_conn.go:236)                           │
│    lwipMutex.Lock()                                             │
│    err := C.tcp_write(pcb, &data[0], len, TCP_WRITE_FLAG_COPY)│
│    lwipMutex.Unlock()                                           │
│    Returns ERR_MEM if buffer full (caller retries)             │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. tcpOutputInternal() (tcp_conn.go:250)                       │
│    lwipMutex.Lock()                                             │
│    C.tcp_output(pcb)  ← Ask lwIP to transmit                   │
│    lwipMutex.Unlock()                                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 6. lwIP C Stack Processing                                      │
│    tcp_output() → Build TCP segments                           │
│    Add TCP header (seq, ack, window, checksum)                │
│    Add IP header                                               │
│    Calls output() callback                                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 7. output() (output_export.go:14)                              │
│    lwipMutex.Lock()                                             │
│    Copy pbuf to Go slice                                       │
│    OutputFn(buf)  ← Write to TUN                               │
│    lwipMutex.Unlock()                                           │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│ 8. TUN Device Write                                             │
│    tunDev.Write(packet)                                         │
│    Kernel delivers to application                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## TCP Connection Handling

### Proxy Handler Interface

Defined in `core/handler.go`:

```go
type TCPConnHandler interface {
    Handle(TCPConn, *net.TCPAddr) error
}
```

Implementations:
- **SOCKS**: `proxy/socks/tcp.go`
- **Redirect**: `proxy/redirect/tcp.go`

### SOCKS Handler

`proxy/socks/tcp.go:21`:

```go
type tcpHandler struct {
    sync.Mutex
    proxyHost string
    proxyPort uint16
}

func NewTCPHandler(proxyHost string, proxyPort uint16) core.TCPConnHandler {
    return &tcpHandler{
        proxyHost: proxyHost,
        proxyPort: proxyPort,
    }
}
```

**Handle function** (`proxy/socks/tcp.go:88`):

```go
func (h *tcpHandler) Handle(conn net.Conn, target *net.TCPAddr) error {
    // Create SOCKS5 dialer
    dialer, err := proxy.SOCKS5("tcp",
        core.ParseTCPAddr(h.proxyHost, h.proxyPort).String(),
        nil,  // No auth
        nil)  // Direct dialer
    if err != nil {
        return err
    }

    // Connect to target through proxy
    c, err := dialer.Dial(target.Network(), target.String())
    if err != nil {
        return err
    }

    // Start relay in background
    go h.relay(conn, c)

    log.Infof("new proxy connection to %v", target)

    return nil
}
```

**Relay function** (`proxy/socks/tcp.go:41`):

```go
func (h *tcpHandler) relay(lhs, rhs net.Conn) {
    upCh := make(chan struct{})

    cls := func(dir direction, interrupt bool) {
        lhsDConn, lhsOk := lhs.(duplexConn)
        rhsDConn, rhsOk := rhs.(duplexConn)

        if !interrupt && lhsOk && rhsOk {
            // Half-close if both support it
            switch dir {
            case dirUplink:
                lhsDConn.CloseRead()
                rhsDConn.CloseWrite()
            case dirDownlink:
                lhsDConn.CloseWrite()
                rhsDConn.CloseRead()
            }
        } else {
            // Full close
            lhs.Close()
            rhs.Close()
        }
    }

    // Uplink: TUN → Proxy
    go func() {
        _, err := io.Copy(rhs, lhs)
        if err != nil {
            cls(dirUplink, true)   // Error, interrupt
        } else {
            cls(dirUplink, false)  // EOF, half-close
        }
        upCh <- struct{}{}
    }()

    // Downlink: Proxy → TUN (main goroutine)
    _, err := io.Copy(lhs, rhs)
    if err != nil {
        cls(dirDownlink, true)
    } else {
        cls(dirDownlink, false)
    }

    <-upCh  // Wait for uplink to finish
}
```

**Half-close support**:

The `duplexConn` interface allows closing one direction:

```go
type duplexConn interface {
    net.Conn
    CloseRead() error
    CloseWrite() error
}
```

`tcpConn` implements this:
- `CloseRead()`: Close pipe reader (handler won't read more)
- `CloseWrite()`: Send FIN to peer (no more data to send)

This properly handles TCP half-close scenarios.

### Connection Tracking

All active connections tracked globally (`core/tcp.go`):

```go
var tcpConns sync.Map  // map[*tcpConn]bool
```

**Add connection**:
```go
tcpConns.Store(conn, true)
```

**Remove connection**:
```go
tcpConns.Delete(conn)
```

**Iterate connections**:
```go
tcpConns.Range(func(key, value interface{}) bool {
    conn := key.(*tcpConn)
    // ...
    return true
})
```

---

## Error Handling and Connection Termination

### Error Callback

When TCP error occurs (e.g., connection reset), lwIP calls:

`core/tcp_callback_export.go:134`:

```go
//export tcpErrFn
func tcpErrFn(arg unsafe.Pointer, err C.err_t) {
    if arg == nil {
        return
    }

    conn := (*tcpConn)(arg)

    switch err {
    case C.ERR_ABRT:
        // Connection aborted
        log.Warnf("TCP connection aborted")
    case C.ERR_RST:
        // Connection reset
        log.Warnf("TCP connection reset")
    case C.ERR_CLSD:
        // Connection closed
        log.Infof("TCP connection closed")
    default:
        log.Errorf("TCP error: %d", int(err))
    }

    conn.Err(errors.New(fmt.Sprintf("TCP error: %d", int(err))))
}
```

`core/tcp_conn.go:493`:

```go
func (conn *tcpConn) Err(err error) {
    conn.Lock()
    conn.state = tcpErrored
    conn.Unlock()

    conn.release()
}
```

### Resource Cleanup

`core/tcp_conn.go:507`:

```go
func (conn *tcpConn) release() {
    lwipMutex.Lock()
    defer lwipMutex.Unlock()

    // Remove from global tracking
    tcpConns.Delete(conn)

    // Close pipes
    conn.sndPipeWriter.Close()
    conn.sndPipeReader.Close()

    // Mark closed
    conn.state = tcpClosed
}
```

### Graceful Shutdown

**Step 1**: Stop accepting new connections

```go
// Stop handler
SetTCPHandler(nil)
```

**Step 2**: Close all active connections

```go
tcpConns.Range(func(key, value interface{}) bool {
    conn := key.(*tcpConn)
    conn.Close()
    return true
})
```

**Step 3**: Wait for connections to close

```go
for {
    count := 0
    tcpConns.Range(func(key, value interface{}) bool {
        count++
        return true
    })
    if count == 0 {
        break
    }
    time.Sleep(100 * time.Millisecond)
}
```

---

## Performance Considerations

### Bottlenecks

See `ARCHITECTURE.md` for detailed analysis. Key issues:

1. **Global lwipMutex**: Serializes all TCP processing
2. **Head-of-line blocking**: Slow connections block fast ones
3. **Memory copying**: Multiple copies at each layer
4. **Single lwIP instance**: Can't utilize multiple CPU cores

### Optimizations Applied

1. **Nagle disabled**: Reduces latency for small packets
2. **SACK enabled**: Better loss recovery
3. **TCP timestamps**: Accurate RTT measurement
4. **Large buffers**: 64KB send/receive windows
5. **Connection pooling**: Reuse pbuf allocations

### Recommended Improvements

From `ARCHITECTURE.md`:

1. **Reduce mutex hold time**: Release lock before slow I/O
2. **Multi-stack architecture**: Separate stacks for TCP/UDP
3. **Per-connection sharding**: Hash connections to different stacks
4. **Batch processing**: Process multiple packets per lock acquisition
5. **Zero-copy paths**: Avoid unnecessary memory copies

### Monitoring

Add instrumentation to measure:

- Lock contention time
- Packet processing latency
- Connection setup time
- Throughput per connection

Example:

```go
func (conn *tcpConn) Write(data []byte) (int, error) {
    start := time.Now()
    defer func() {
        elapsed := time.Since(start)
        if elapsed > 10*time.Millisecond {
            log.Warnf("Slow write: %v for %d bytes", elapsed, len(data))
        }
    }()

    // ... existing code ...
}
```

---

## Debugging TCP Issues

### Enable lwIP Debug Logging

Edit `core/c/custom/lwipopts.h`:

```c
#define LWIP_DEBUG 1
#define TCP_DEBUG LWIP_DBG_ON
#define TCP_INPUT_DEBUG LWIP_DBG_ON
#define TCP_OUTPUT_DEBUG LWIP_DBG_ON
```

Rebuild:
```bash
go build -v ./cmd/tun2socks
```

### Capture Packets

**On TUN interface**:
```bash
sudo tcpdump -i tun1 -w tun1.pcap
```

**On physical interface**:
```bash
sudo tcpdump -i en0 host <proxy-server> -w proxy.pcap
```

**Analyze with Wireshark**:
```bash
wireshark tun1.pcap
```

Look for:
- Retransmissions (indicates packet loss)
- Zero window (flow control issue)
- Duplicate ACKs (congestion or loss)
- RST packets (connection reset)
- Out-of-order segments

### Common Issues

#### 1. Connection Stalls

**Symptom**: Data stops flowing, connection hangs

**Possible causes**:
- Handler forgot to call `tcp_recved()` → window stays at 0
- Send buffer full, handler not reading → deadlock
- Mutex deadlock

**Debug**:
```go
log.Debugf("State: %v, Window: %d, Send buffer: %d",
    conn.state,
    conn.pcb.rcv_wnd,
    C.tcp_sndbuf_cgo(conn.pcb))
```

#### 2. High Latency

**Symptom**: Slow connection despite low packet loss

**Possible causes**:
- Nagle's algorithm enabled (should be disabled)
- Delayed ACK interacting badly
- lwipMutex contention

**Debug**:
```bash
# Check mutex contention
go tool pprof http://localhost:6060/debug/pprof/mutex
```

#### 3. Connection Reset

**Symptom**: Connection closes with RST

**Possible causes**:
- Proxy server rejected connection
- Handler returned error
- Timeout (keepalive or retransmit)

**Debug**:
Look at `tcpErrFn()` logs:
```
TCP connection reset
TCP error: -14
```

#### 4. Memory Leak

**Symptom**: Memory usage grows over time

**Possible causes**:
- Connections not properly closed
- Pbufs not freed
- Goroutines leaked

**Debug**:
```go
// Count active connections
count := 0
tcpConns.Range(func(key, value interface{}) bool {
    count++
    return true
})
log.Infof("Active connections: %d", count)

// Check goroutine count
log.Infof("Goroutines: %d", runtime.NumGoroutine())
```

### Test Tools

**iperf3** (throughput test):
```bash
# Server
iperf3 -s

# Client (through TUN)
iperf3 -c <server-ip>
```

**netcat** (simple TCP test):
```bash
# Server
nc -l 8080

# Client (through TUN)
nc <server-ip> 8080
```

**curl** (HTTP test):
```bash
# Through TUN
curl -v http://example.com

# Check timing
curl -w "@curl-format.txt" -o /dev/null -s http://example.com
```

---

## Summary

### TCP Protocol
- **Reliable, ordered** byte stream protocol
- **Three-way handshake** for connection setup
- **Sequence numbers and ACKs** for reliability
- **Flow control** via receive window
- **Congestion control** via cwnd/ssthresh
- **Four-way handshake** for graceful termination

### Standard Go TCP
- High-level `net.Conn` interface
- Kernel TCP stack (fast, optimized)
- Goroutine-per-connection model
- epoll/kqueue for efficient I/O

### go-tun2socks TCP
- **Userspace TCP stack** using lwIP
- Intercepts raw IP packets from TUN device
- Processes TCP in userspace
- Redirects connections to SOCKS/HTTP proxy
- Bridges lwIP (C) ↔ Go via callbacks and pipes
- Global mutex for thread safety
- State machine for connection lifecycle
- Implements `net.Conn` for handler compatibility

### Key Components
- **lwIP**: C TCP/IP stack
- **tcpConn**: Go wrapper around lwIP PCB
- **Callbacks**: tcpAcceptFn, tcpRecvFn, tcpSentFn, tcpErrFn
- **Pipes**: nio.Pipe for lwIP → handler data flow
- **Handler**: SOCKS/redirect proxy relay
- **lwipMutex**: Global lock for thread safety

### Critical Operations
- **tcp_recved()**: Must be called after reading (flow control)
- **tcp_write()**: Enqueue data in send buffer
- **tcp_output()**: Trigger transmission
- **tcp_close()**: Graceful shutdown
- **tcp_abort()**: Immediate termination (RST)

### Performance
- Bottlenecked by global mutex
- Single-threaded lwIP processing
- Recommended: multi-stack architecture for scaling

For detailed performance analysis and optimization recommendations, see `ARCHITECTURE.md`.
