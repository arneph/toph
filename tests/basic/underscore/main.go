package main

import "fmt"

func t() chan int {
	return make(chan int)
}

func main() {
	_ = t()
	fmt.Println("hello")
}
