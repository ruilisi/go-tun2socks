// +build ignore

package main

import (
	"fmt"
	"time"

	syncex "github.com/orkunkaraduman/go-syncex"
)

func main() {
	var mu syncex.RecursiveMutex
	var f int
	mu.Lock()
	for i := 0; i < 5; i++ {
		mu.Lock()
		go func() {
			mu.Lock()
			f++
			fmt.Println("goroutine: ", f)
			mu.Unlock()
		}()
		f++
		fmt.Println("forloop: ", f)
		mu.Unlock()
	}
	fmt.Println("mainfunc: ", f)
	mu.Unlock()
	time.Sleep(1 * time.Second)
	fmt.Println("mainfunc: ", f)
}
