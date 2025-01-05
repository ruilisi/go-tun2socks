// +build ignore

package main

import (
	"fmt"
	"time"

	syncex "github.com/orkunkaraduman/go-syncex"
)

func main() {
	var cs syncex.CriticalSection
	var f int
	oid := syncex.NewOwnerID()
	cs.Lock(oid)
	for i := 0; i < 5; i++ {
		cs.Lock(oid)
		go func() {
			oid1 := syncex.NewOwnerID()
			cs.Lock(oid1)
			f++
			fmt.Println("goroutine: ", f)
			cs.Unlock()
		}()
		f++
		fmt.Println("forloop: ", f)
		cs.Unlock()
	}
	fmt.Println("mainfunc: ", f)
	cs.Unlock()
	time.Sleep(1 * time.Second)
	fmt.Println("mainfunc: ", f)
}
