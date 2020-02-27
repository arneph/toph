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

func main() {
	var n, sum int
	fmt.Scan(&n)
	for i := 0; i < n; i++ {
		sum += i
	}
	fmt.Println(sum)
}
