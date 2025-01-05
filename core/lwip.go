package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/tcp.h"
#include "lwip/udp.h"
#include "lwip/timeouts.h"

err_t
tcp_bind_cgo(struct tcp_pcb *pcb, _Bool enableIPv6, _Bool allowLan)
{
	if (allowLan) {
		return tcp_bind(pcb, IP_ADDR_ANY, 0);
	}
	ip_addr_t bindAddr = IPADDR_ANY_TYPE_INIT;
	if (enableIPv6) {
		IP_ADDR6(&bindAddr, 0, 0, 0, 0x00000001UL);
	} else {
		IP_ADDR4(&bindAddr, 127, 0, 0, 1);
	}

	return tcp_bind(pcb, &bindAddr, 0);
}

err_t
udp_bind_cgo(struct udp_pcb *pcb, _Bool enableIPv6, _Bool allowLan)
{
	if (allowLan) {
		return udp_bind(pcb, IP_ADDR_ANY, 0);
	}
	ip_addr_t bindAddr = IPADDR_ANY_TYPE_INIT;
	if (enableIPv6) {
		IP_ADDR6(&bindAddr, 0, 0, 0, 0x00000001UL);
	} else {
		IP_ADDR4(&bindAddr, 127, 0, 0, 1);
	}

	return udp_bind(pcb, &bindAddr, 0);
}
*/
import "C"
import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ruilisi/go-tun2socks/common/log"
	syncex "github.com/ruilisi/go-tun2socks/component/go-syncex"
	"github.com/ruilisi/go-tun2socks/component/runner"
)

const CHECK_TIMEOUTS_INTERVAL = 250 // in millisecond
const TCP_POLL_INTERVAL = 8         // poll every 4 seconds

type LWIPSysCheckTimeoutsClosingType uint

const (
	INSTANT LWIPSysCheckTimeoutsClosingType = iota
	DELAY
)

type LWIPStack interface {
	Write([]byte) (int, error)
	Close(LWIPSysCheckTimeoutsClosingType) error
	RestartTimeouts()
	GetRunningStatus() bool
	StartTimeouts()
	StopTimeouts(LWIPSysCheckTimeoutsClosingType)
}

var lwipSysCheckTimeoutsLock = &syncex.RecursiveMutex{}

type lwipStack struct {
	IsRunning                     *int32
	tpcb                          *C.struct_tcp_pcb
	upcb                          *C.struct_udp_pcb
	LWIPSysCheckTimeoutsTask      *runner.Task
	LWIPSysStopCheckTimeoutsTimer *time.Timer
	enableIPv6                    bool
}

const (
	STOP    int32 = 0
	RUNNING int32 = 1
)

func lwipStackSetupInternal(enableIPv6 bool, allowLan bool) *lwipStack {
	lwipMutex.Lock()
	defer lwipMutex.Unlock()
	var tcpPCB *C.struct_tcp_pcb
	var udpPCB *C.struct_udp_pcb
	var err C.err_t

	if enableIPv6 {
		tcpPCB = C.tcp_new_ip_type(C.IPADDR_TYPE_ANY)
	} else {
		tcpPCB = C.tcp_new_ip_type(C.IPADDR_TYPE_V4)
	}

	if tcpPCB == nil {
		log.Fatalf("tcp_new return nil")
	}

	err = C.tcp_bind_cgo(tcpPCB, C._Bool(enableIPv6), C._Bool(allowLan))

	switch err {
	case C.ERR_OK:
		break
	case C.ERR_VAL:
		log.Fatalf("invalid PCB state")
	case C.ERR_USE:
		log.Fatalf("port in use")
	default:
		C.memp_free(C.MEMP_TCP_PCB, unsafe.Pointer(tcpPCB))
		log.Fatalf("unknown tcp_bind return value")
	}

	tcpPCB = C.tcp_listen_with_backlog(tcpPCB, C.TCP_DEFAULT_LISTEN_BACKLOG)
	if tcpPCB == nil {
		log.Fatalf("can not allocate tcp pcb")
	}

	setTCPAcceptCallback(tcpPCB)

	if enableIPv6 {
		udpPCB = C.udp_new_ip_type(C.IPADDR_TYPE_ANY)
	} else {
		udpPCB = C.udp_new_ip_type(C.IPADDR_TYPE_V4)
	}
	if udpPCB == nil {
		log.Fatalf("could not allocate udp pcb")
	}

	err = C.udp_bind_cgo(udpPCB, C._Bool(enableIPv6), C._Bool(allowLan))

	if err != C.ERR_OK {
		log.Fatalf("address already in use")
	}

	setUDPRecvCallback(udpPCB, nil)
	var run int32
	stack := &lwipStack{
		tpcb:       tcpPCB,
		upcb:       udpPCB,
		enableIPv6: enableIPv6,
		IsRunning:  &run,
	}
	return stack
}

// NewLWIPStack listens for any incoming connections/packets and registers
// corresponding accept/recv callback functions.
// The timeout check goroutine is NOT started
func NewLWIPStack(enableIPv6 bool, allowLan bool) LWIPStack {
	stack := lwipStackSetupInternal(enableIPv6, allowLan)
	atomic.StoreInt32(stack.IsRunning, RUNNING)
	stack.StartTimeouts()
	return stack
}

func (s *lwipStack) doStartTimeouts() {
	task := runner.Go(func(shouldStop runner.S) error {
		// do setup
		// defer func(){
		//   // do teardown
		// }
		zeroErr := errors.New("no error")
		for {
			// do some work
			lwipMutex.Lock( /*true*/ ) // pass anything to disable log trace
			C.sys_check_timeouts()
			lwipMutex.Unlock( /*true*/ )

			time.Sleep(CHECK_TIMEOUTS_INTERVAL * time.Millisecond)
			if shouldStop() {
				break
			}
		}
		log.Infof("got sys_check_timeouts stop signal")
		return zeroErr // any errors?
	})
	lwipSysCheckTimeoutsLock.Lock()
	defer lwipSysCheckTimeoutsLock.Unlock()
	s.LWIPSysCheckTimeoutsTask = task
	log.Infof("sys_check_timeouts started")
}

// StartTimeouts start the lwip stack timeout checking goroutine
func (s *lwipStack) StartTimeouts() {
	// only start the task when we are really in running state
	if s.GetRunningStatus() {
		// cancel scheduled stop timer
		lwipSysCheckTimeoutsLock.Lock()
		defer lwipSysCheckTimeoutsLock.Unlock()
		if s.LWIPSysStopCheckTimeoutsTimer != nil {
			log.Infof("StartTimeouts: cancel scheduled stop timer Stop() before call")
			if !s.LWIPSysStopCheckTimeoutsTimer.Stop() {
				<-s.LWIPSysStopCheckTimeoutsTimer.C
			}
			s.LWIPSysStopCheckTimeoutsTimer.Reset(1)
			log.Infof("StartTimeouts: cancel scheduled stop timer Stop() called")
		}

		// never started or not running at the moment
		if s.LWIPSysCheckTimeoutsTask == nil || !s.LWIPSysCheckTimeoutsTask.Running() {
			s.doStartTimeouts()
		}
	}
}

// StopTimeouts stops the lwip stack when timeout
// We should NOT stop it instantly, TCP stack relys on the timeout to destory
// connections, we stop it only when we are in long-idle
func (s *lwipStack) StopTimeouts(t LWIPSysCheckTimeoutsClosingType) {
	if t == DELAY {
		if s.LWIPSysCheckTimeoutsTask != nil && s.LWIPSysCheckTimeoutsTask.Running() {
			log.Infof("StopTimeouts: schedule stop timer at %v", time.Now())
			lwipSysCheckTimeoutsLock.Lock()
			defer lwipSysCheckTimeoutsLock.Unlock()
			s.LWIPSysStopCheckTimeoutsTimer = time.NewTimer(30 * time.Minute)

			// stop when timeout expires
			go func(s *lwipStack) {

				t := <-s.LWIPSysStopCheckTimeoutsTimer.C
				// if we are really in stop state then stop it
				if !s.GetRunningStatus() {
					log.Infof("StopTimeouts: scheduled stop timer expires at %v with stopped lwipStack", t)
					s.LWIPSysCheckTimeoutsTask.Stop()
				} else {
					log.Infof("StopTimeouts: scheduled stop timer expires at %v with running lwipStack", t)
				}
				// destory timer when expires
				lwipSysCheckTimeoutsLock.Lock()
				defer lwipSysCheckTimeoutsLock.Unlock()
				s.LWIPSysStopCheckTimeoutsTimer = nil
			}(s)
		}
	} else if t == INSTANT {
		// cancel scheduled stop timer
		lwipSysCheckTimeoutsLock.Lock()
		defer lwipSysCheckTimeoutsLock.Unlock()
		if s.LWIPSysStopCheckTimeoutsTimer != nil {
			log.Infof("StopTimeouts: cancel scheduled stop timer Stop() before call")
			if !s.LWIPSysStopCheckTimeoutsTimer.Stop() {
				<-s.LWIPSysStopCheckTimeoutsTimer.C
			}
			s.LWIPSysStopCheckTimeoutsTimer.Reset(1)
			log.Infof("StopTimeouts: cancel scheduled stop timer Stop() called")
		}

		// and finally one shot kill
		log.Infof("StopTimeouts: stop LWIPSysCheckTimeoutsTask instantly")
		s.LWIPSysCheckTimeoutsTask.Stop()
	}

}

// GetRunningStatus returns true if the lwip stack is running
func (s *lwipStack) GetRunningStatus() bool {
	r := atomic.LoadInt32(s.IsRunning)
	return r == RUNNING
}

// Write writes IP packets to the stack.
func (s *lwipStack) Write(data []byte) (int, error) {
	if s.GetRunningStatus() {
		n, err := input(data)
		if err != nil {
			log.Errorf("lwip input err: %v", err)
		}
		return n, err
	} else {
		return 0, errors.New("stack closed")
	}
}

func (s *lwipStack) RestartTimeouts() {
	C.sys_restart_timeouts()
}

func (s *lwipStack) closeInternal() {
	lwipMutex.Lock()
	defer lwipMutex.Unlock()

	// free TCP listener pcb
	err := C.tcp_close(s.tpcb)
	switch err {
	case C.ERR_OK:
		// ERR_OK if connection has been closed
		break
	default:
		// another err_t if closing failed and pcb is not freed
		// make sure free is invoked
		C.tcp_abort(s.tpcb)
	}

	// free UDP pcb
	C.udp_remove(s.upcb)
}

func (s *lwipStack) Close(t LWIPSysCheckTimeoutsClosingType) error {
	if s.GetRunningStatus() {
		tcpConns.Range(func(c, _ interface{}) bool {
			c.(*tcpConn).Abort()
			return true
		})
		udpConns.Range(func(_, c interface{}) bool {
			c.(*udpConn).Close()
			return true
		})

		s.closeInternal()

		atomic.StoreInt32(s.IsRunning, STOP)
	}

	s.StopTimeouts(t)

	return nil
}

func init() {
	// Initialize lwIP.
	//
	// There is a little trick here, a loop interface (127.0.0.1)
	// is created in the initialization stage due to the option
	// `#define LWIP_HAVE_LOOPIF 1` in `lwipopts.h`, so we need
	// not create our own interface.
	//
	// Now the loop interface is just the first element in
	// `C.netif_list`, i.e. `*C.netif_list`.
	lwipInit()

	// Set MTU.
	C.netif_list.mtu = 1500
}
