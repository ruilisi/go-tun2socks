package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/tcp.h"

extern err_t tcpAcceptFn(void *arg, struct tcp_pcb *newpcb, err_t err);

void
set_tcp_accept_callback(struct tcp_pcb *pcb) {
	tcp_accept(pcb, tcpAcceptFn);
}

extern err_t tcpRecvFn(void *arg, struct tcp_pcb *tpcb, struct pbuf *p, err_t err);

void
set_tcp_recv_callback(struct tcp_pcb *pcb) {
	tcp_recv(pcb, tcpRecvFn);
}

extern err_t tcpSentFn(void *arg, struct tcp_pcb *tpcb, u16_t len);

void
set_tcp_sent_callback(struct tcp_pcb *pcb) {
    tcp_sent(pcb, tcpSentFn);
}

extern void tcpErrFn(void *arg, err_t err);

void
set_tcp_err_callback(struct tcp_pcb *pcb) {
	tcp_err(pcb, tcpErrFn);
}
*/
import "C"

func setTCPAcceptCallback(pcb *C.struct_tcp_pcb) {
	C.set_tcp_accept_callback(pcb)
}

func setTCPRecvCallback(pcb *C.struct_tcp_pcb) {
	C.set_tcp_recv_callback(pcb)
}

func setTCPSentCallback(pcb *C.struct_tcp_pcb) {
	C.set_tcp_sent_callback(pcb)
}

func setTCPErrCallback(pcb *C.struct_tcp_pcb) {
	C.set_tcp_err_callback(pcb)
}
