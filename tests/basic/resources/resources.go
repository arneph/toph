package main

func main() { // 67
	a()
	c()
	e()
}

func a() { // 25
	for i := 0; i < 10; i++ {
		b()
	}
	c()
	c()
	ch := make(chan int)
	close(ch)
}

func b() { // 2
	ch := make(chan int)
	close(ch)
	d()
}

func c() { // 2
	b()
}

func d() { // 1
	if false {
		ch := make(chan int)
		close(ch)
	} else {
		ch := make(chan int)
		close(ch)
		//b()
	}
}

func e() { // 40
	for i := 0; i < 10; i++ {
		ch := make(chan int)
		close(ch)
		f()
	}
}

func f() { // 3
	ch := make(chan int)
	ch = make(chan int)
	ch = make(chan int)
	close(ch)
}
