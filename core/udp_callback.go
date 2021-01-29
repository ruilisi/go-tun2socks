package core

/*
#cgo CFLAGS: -I./c/custom -I./c/include
#include "lwip/udp.h"
#include "lwip/ip_addr.h"

extern void udpRecvFn(void *arg, struct udp_pcb *pcb, struct pbuf *p, const ip_addr_t *addr, u16_t port, const ip_addr_t *dest_addr, u16_t dest_port);

void
set_udp_recv_callback(struct udp_pcb *pcb, void *recv_arg) {
	udp_recv(pcb, udpRecvFn, recv_arg);
}

extern void go_tun2socks_ip_addr_copy(ip_addr_t *dest, ip_addr_t *src);

void
go_tun2socks_ip_addr_copy(ip_addr_t *dest, ip_addr_t *src) {
	ip_addr_copy(*dest, *src);
}
*/
import "C"
import (
	"unsafe"
)

func setUDPRecvCallback(pcb *C.struct_udp_pcb, recvArg unsafe.Pointer) {
	C.set_udp_recv_callback(pcb, recvArg)
}

func copyLwipIpAddr(dest, src *C.ip_addr_t) {
	C.go_tun2socks_ip_addr_copy(dest, src)
}
