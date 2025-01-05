package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include <stdlib.h>
#include "lwip/tcp.h"
#include "lwip/udp.h"
#include "lwip/timeouts.h"
*/
import "C"
import (
	"errors"
	//"fmt"
	//"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
	//"github.com/ruilisi/go-tun2socks/common/log"
	syncex "github.com/ruilisi/go-tun2socks/component/go-syncex"
)

// lwIP runs in a single thread, locking is needed in Go runtime.
var lwipMutex = &syncex.RecursiveMutex{}

type MutexWrapper struct {
	lock  *sync.Mutex
	count int32
}

// pass anything to disable log trace
// pass nothing to enable log trace
func (m *MutexWrapper) Lock( /*params ...bool*/ ) {
	/*if len(params) != 0 {
		m.lock.Lock()
		atomic.AddInt32(&m.count, 1)
		return
	}

	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	s := fmt.Sprintf("%s:%d %s\n", file, line, f.Name())

	log.Infof("MutexWrapper before Lock %v %s", atomic.LoadInt32(&m.count), s) */
	m.lock.Lock()
	atomic.AddInt32(&m.count, 1)
	/*log.Infof("MutexWrapper after Lock %v %s", atomic.LoadInt32(&m.count), s) */
}

// pass anything to disable log trace
// pass nothing to enable log trace
func (m *MutexWrapper) Unlock( /*params ...bool*/ ) {
	/*if len(params) != 0 {
		m.lock.Unlock()
		atomic.AddInt32(&m.count, -1)
		return
	}

	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	file, line := f.FileLine(pc[0])
	s := fmt.Sprintf("%s:%d %s\n", file, line, f.Name())

	log.Infof("MutexWrapper before Unlock %v %s", atomic.LoadInt32(&m.count), s) */
	m.lock.Unlock()
	atomic.AddInt32(&m.count, -1)
	/*log.Infof("MutexWrapper after Unlock %v %s", atomic.LoadInt32(&m.count), s) */
}

// ipaddr_ntoa() is using a global static buffer to return result,
// reentrants are not allowed, caller is required to lock lwipMutex.
//export ipAddrNTOA
func ipAddrNTOA(ipaddr C.struct_ip_addr) string {
	lwipMutex.Lock()
	defer lwipMutex.Unlock()
	return C.GoString(C.ipaddr_ntoa(&ipaddr))
}

//export ipAddrATON
func ipAddrATON(cp string, addr *C.struct_ip_addr) error {
	lwipMutex.Lock()
	defer lwipMutex.Unlock()
	ccp := C.CString(cp)
	defer C.free(unsafe.Pointer(ccp))
	if r := C.ipaddr_aton(ccp, addr); r == 0 {
		return errors.New("failed to convert IP address")
	} else {
		return nil
	}
}
