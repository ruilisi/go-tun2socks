package main

/*
#cgo CFLAGS: -I../core/c/custom -I../core/c/include
#cgo LDFLAGS: -pthread
#include "lwip_init.h"
*/
import "C"

func main() {
	C.lwip_run()
}
