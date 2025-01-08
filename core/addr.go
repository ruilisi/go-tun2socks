package core

import (
	"net"
	"strconv"
)

func ParseTCPAddr(addr string, port uint16) *net.TCPAddr {
	netAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(addr, strconv.Itoa(int(port))))
	if err != nil {
		return nil
	}
	return netAddr
}

func ParseUDPAddr(addr string, port uint16) *net.UDPAddr {
	netAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(addr, strconv.Itoa(int(port))))
	if err != nil {
		return nil
	}
	return netAddr
}
