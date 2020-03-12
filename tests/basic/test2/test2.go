package main

import (
	"fmt"
	"time"
)

func test() {
	var ch chan int
	ch = make(chan int)
	go func() {
		time.Sleep(1 * time.Second)
		ch <- 42
		fmt.Printf("sent %p\n", ch)
	}()
	go func() {
		time.Sleep(1 * time.Second)
		<-ch
		fmt.Printf("received %p\n", ch)
	}()
}

func main() {
	test()
	test()
	test()
	test()
	time.Sleep(5 * time.Second)
	fmt.Println("done")
}
