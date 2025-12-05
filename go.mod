module github.com/ruilisi/go-tun2socks

go 1.25.4

replace github.com/ruilisi/stellar-proxy => ../stellar-proxy

require (
	github.com/djherbis/buffer v1.2.0
	github.com/djherbis/nio v2.0.3+incompatible
	github.com/hashicorp/golang-lru v1.0.2
	github.com/miekg/dns v1.1.68
	github.com/ruilisi/stellar-proxy v0.0.0-00010101000000-000000000000
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8
	github.com/v2pro/plz v0.0.0-20221028024117-e5f9aec5b631
	golang.org/x/net v0.43.0
	golang.org/x/sys v0.35.0
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/tools v0.35.0 // indirect
)
