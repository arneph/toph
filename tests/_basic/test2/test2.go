package main

import (
	"fmt"
	"time"
)

func main() {
	var ch chan int
	ch = make(chan int)
	go func() {
		ch <- 42
		fmt.Println("sent")
	}()
	go func() {
		<-ch
		fmt.Println("received")
	}()
	time.Sleep(1 * time.Second)
}
