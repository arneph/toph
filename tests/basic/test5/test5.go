package main

func main() {
	a()
	c()
	e()
}

func a() {
	for i := 0; i < 10; i++ {
		b()
	}
	c()
	c()
	ch := make(chan int)
	close(ch)
}

func b() {
	ch := make(chan int)
	close(ch)
	d()
}

func c() {
	b()
}

func d() {
	if false {
		ch := make(chan int)
		close(ch)
	} else {
		ch := make(chan int)
		close(ch)
		//b()
	}
}

func e() {
	for i := 0; i < 10; i++ {
		ch := make(chan int)
		close(ch)
		f()
	}
}

func f() {
	ch := make(chan int)
	ch = make(chan int)
	ch = make(chan int)
	close(ch)
}