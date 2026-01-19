package core

import (
	spcore "github.com/ruilisi/stellar-proxy/core"
)

// TCPConnHandler is re-exported from stellar-proxy/core
type TCPConnHandler = spcore.TCPConnHandler

// UDPConnHandler is re-exported from stellar-proxy/core
type UDPConnHandler = spcore.UDPConnHandler

var tcpConnHandler TCPConnHandler
var udpConnHandler UDPConnHandler

func RegisterTCPConnHandler(h TCPConnHandler) {
	tcpConnHandler = h
}

func RegisterUDPConnHandler(h UDPConnHandler) {
	udpConnHandler = h
}
