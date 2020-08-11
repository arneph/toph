package main

import (
	"fmt"
	"sync"
	"time"
)

func produce(ch chan<- int) {
	for i := 0; i < 10; i++ {
		ch <- i
	}
	close(ch)
}

var once sync.Once

func process(chIn <-chan int, chOut chan int) {
	for x := range chIn {
		chOut <- x * x
	}
	once.Do(func() {
		close(chOut)
	})
	<-chOut
}

func consume(ch <-chan int) {
	for x := range ch {
		fmt.Println(x)
	}
}

func main() {
	chA := make(chan int)
	chB := make(chan int)
	go produce(chA)
	go process(chA, chB)
	go process(chA, chB)
	go process(chA, chB)
	go consume(chB)
	go consume(chB)
	time.Sleep(1 * time.Second)
}
