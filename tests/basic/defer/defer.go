package main

import "math/rand"

func main() {
	chA := make(chan int, 2)
	chB := make(chan int, 6)
	chC := make(chan int, 2)
	chD := make(chan int, 3)

	f(chA)
	h(chB)
	i(chC, chD)

	chE := make(chan int, 6)
	var x func(chan int)
	if rand.Int() == 42 {
		x = f
	} else {
		x = h
	}
	defer x(chE)
	x = nil
	chE = nil
}

func f(ch chan int) {
	defer close(ch)
	defer g(ch)
	ch <- 42
	ch <- 43
}

func g(ch chan int) {
	<-ch
	<-ch
}

func h(ch chan int) {
	for i := 0; i < 3; i++ {
		defer g(ch)
	}
	ch <- 123
	close(ch)
}

func i(chA, chB chan int) {
	if rand.Int() == 0 {
		defer f(chA)
	}
	defer close(chB)
	// toph: max_iter=20
	for i := 0; i < rand.Int(); i++ {
		defer func() {
			g(chB)
		}()
	}
	chB <- 1
	chB <- 2
	chB <- 3
}
