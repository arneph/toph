package main

func producerA(a, b int, ch chan int, done chan struct{}) {
	ch <- a
	ch <- b
	for i := 0; i < 10; i++ {
		c := a + b
		ch <- c
		a, b = b, c
	}
	close(ch)
}

func producerB(ch chan string, done chan struct{}) {
	s := ""
	for i := 0; i < 3; i++ {
		s += "."
		ch <- s
	}
	close(ch)
}

func consumer(chA chan int, chB chan string, done chan struct{}) {

}
