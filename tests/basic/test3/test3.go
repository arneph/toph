package main

import "fmt"

func producer(out chan<- int) {
	for i := 0; i < 10; i++ {
		out <- i
	}
	close(out)
}

func processor(in <-chan int, out chan<- int) {
	for i := range in {
		out <- i * i
	}
	close(out)
}

func consumer(in <-chan int) {
	for i := range in {
		fmt.Println(i)
	}
}

func main() {
	var ch1, ch2 chan int
	ch1 = make(chan int)
	ch2 = make(chan int)

	go producer(ch1)
	go processor(ch1, ch2)
	consumer(ch2)
}
