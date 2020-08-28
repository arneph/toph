package main

// Modifications to try:
// - Add default to select
// - Add timeout to select

import (
	"fmt"
	"time"
)

func producerA(ch chan int) {
	for i := 10; i < 20; i++ {
		ch <- i
	}
}

func producerB(ch chan int) {
	for i := 20; i < 30; i++ {
		ch <- i
	}
}

func consumer(chA, chB chan int) {
	for {
		select {
		case x := <-chA:
			fmt.Println(x)
		case y := <-chB:
			fmt.Println(y)
		case <-time.After(1 * time.Second):
			fmt.Println("exit")
			return
		}
	}
}

func main() {
	chA := make(chan int)
	chB := make(chan int)
	go producerA(chA)
	go producerB(chB)
	go consumer(chA, chB)
	time.Sleep(3 * time.Second)
}
