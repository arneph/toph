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
		// toph: max_iter=100
		for {
			select {
			// toph: check=unreachable
			case chA <- 42:
			// toph: check=reachable
			case <-chB:
				time.Sleep(50 * time.Millisecond)
				fmt.Println("send failed")
			}
		}
	}()
	go func() {
		// toph: max_iter=100
		for {
			select {
			// toph: check=unreachable
			case i := <-chA:
				fmt.Println(i)
			// toph: check=reachable
			case <-chB:
				time.Sleep(50 * time.Millisecond)
				fmt.Println("receive failed A")
			default:
				time.Sleep(50 * time.Millisecond)
				fmt.Println("receive failed B")
			}
		}
	}()
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}
