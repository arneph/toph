package main

var x func(chan int, func()) chan bool = f
var testCh chan bool

func main() {
	var y func() = r
	if 42 == 24 {
		x = g
	} else if 35 == 53 {
		x = func(a chan int, b func()) chan bool {
			y = b
			testCh <- false
			return nil
		}
	}
	for i := 0; i < 2; i++ {
		if i%2 == 0 {
			y = s
		} else {
			y = nil
		}
	}

	ch := make(chan int, 3)
	testCh = x(ch, y)
}

func f(ch chan int, l func()) chan bool {
	ch <- 1
	return make(chan bool)
}

func g(ch chan int, l func()) chan bool {
	<-ch
	l()
	return nil
}

func r() {
	close(testCh)
}

func s() {
	testCh <- true
}
