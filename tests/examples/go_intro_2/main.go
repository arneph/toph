package main

import (
	"fmt"
	"time"
)

func f(ch chan int) {
	for i := 1; ; i++ {
		ch <- i
		fmt.Println("Ping:", i)
		x := <-ch
		y := <-ch
		fmt.Println("Pong:", x, y)

		time.Sleep(1 * time.Second)
	}
}

func g(ch chan int) {
	for {
		x := <-ch
		time.Sleep(1 * time.Second)
		ch <- x
	}
}

func main() {
	ch := make(chan int)
	go f(ch)
	go g(ch)
	time.Sleep(6 * time.Second)
}
