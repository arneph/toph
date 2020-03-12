package main

import (
	"fmt"
)

var ch chan int

func testA() {
	a, b := 42, make(chan int)
	c, b := 65, make(chan int)

	b <- a
	b <- c
}

func testB() {
	var ch chan int
	var f func() chan int

	f = func() chan int {
		return make(chan int)
	}

	ch = f()
	ch <- 42
}

func testC() {
	ch := make(chan int)
	if 42 == 24 {
		ch := make(chan int)
		close(ch)
	}
	close(ch)
}

func main() {
	var n, sum int
	fmt.Scan(&n)
	for i := 0; i < n; i++ {
		sum += i
	}
	fmt.Println(sum)
}
