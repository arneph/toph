package main

// Modifications to try:
// - Multiple consumers
// - Multiple producers
// - Add sleep before close

import (
	"fmt"
	"time"
)

func producer(ch chan int) {
	for i := 1; i < 10; i++ {
		ch <- i
	}
	time.Sleep(1 * time.Second)
	close(ch)
}

func consumer(ch chan int) {
	for x := range ch {
		fmt.Println(x)
	}
}

func main() {
	ch := make(chan int)
	go producer(ch)
	for i := 0; i < 3; i++ {
		go consumer(ch)
	}
	time.Sleep(3 * time.Second)
}
