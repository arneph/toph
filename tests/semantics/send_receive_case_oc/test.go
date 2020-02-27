package main

import (
	"fmt"
	"time"
)

func main() {
	chA := make(chan int)
	chB := make(chan int)
	close(chB)

	go func() {
		chA <- 42
	}()
	go func() {
		for {
			select {
			case i := <-chA:
				fmt.Println(i)
			case <-chB:
				time.Sleep(50 * time.Millisecond)
				fmt.Println("receive failed")
			}
		}
	}()
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}
