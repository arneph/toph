package main

import (
	"fmt"
)

func f(id int, chA, chB chan int) {
	for i := range chA {
		fmt.Println(id, i)
		//time.Sleep(20 * time.Millisecond)
	}
	fmt.Println("loop done")
	chB <- 123
}

func main() {
	chA := make(chan int, 5)
	chB := make(chan int)

	go func() {
		for i := 0; i < 5; i++ {
			chA <- 42 + i
		}
		close(chA)
	}()
	go f(0, chA, chB)
	go f(1, chA, chB)
	go f(2, chA, chB)
	go f(3, chA, chB)
	<-chB
	<-chB
	<-chB
	<-chB
	fmt.Println("done")
}
