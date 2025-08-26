module github.com/ruilisi/go-tun2socks

go 1.25.0

require (
	github.com/djherbis/buffer v1.1.0
	github.com/djherbis/nio v2.0.3+incompatible
	github.com/hashicorp/golang-lru v1.0.2
	github.com/miekg/dns v1.1.62
	github.com/ruilisi/stellar-proxy v0.0.0-00010101000000-000000000000
	github.com/songgao/water v0.0.0-20190725173103-fd331bda3f4b
	github.com/v2pro/plz v0.0.0-20221028024117-e5f9aec5b631
	golang.org/x/net v0.41.0
	golang.org/x/sys v0.33.0
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/tools v0.32.0 // indirect
)

replace github.com/ruilisi/stellar-proxy => ../stellar-proxy
