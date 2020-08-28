package main

// Modifications to try:
// - Close channel after three reads
// - Close channel before first read
// - Make channel buffered
// - Add sleep before closed
// - Add sleep after closed

import (
	"fmt"
)

func f(ch chan int) {
	ch <- 1
	ch <- 2
	ch <- 3
}

func main() {
	ch := make(chan int)
	go f(ch)
	fmt.Println(<-ch)
	fmt.Println(<-ch)
	fmt.Println(<-ch)
	fmt.Println(<-ch)
	fmt.Println(<-ch)
	fmt.Println("Done")
}
