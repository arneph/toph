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
		for {
			select {
			case chA <- 42:
			case <-chB:
				time.Sleep(50 * time.Millisecond)
				fmt.Println("send failed A")
			default:
				time.Sleep(50 * time.Millisecond)
				fmt.Println("send failed B")
			}
		}
	}()
	go func() {
		select {
		case i := <-chA:
			fmt.Println(i)
		}
	}()
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}
