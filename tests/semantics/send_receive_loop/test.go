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
		for i := range chA {
			fmt.Println(i)
		}
		fmt.Println("loop done")
	}()
	time.Sleep(1 * time.Second)
	fmt.Println("done")
}
