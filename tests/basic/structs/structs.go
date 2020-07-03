package main

import (
	"fmt"
	"sync"
)

type A struct {
	abc string
	ch  chan int
}

func (a *A) processAll() {
	for i := range a.ch {
		fmt.Println(i)
	}
}

type B struct {
	A
	sync.Mutex
	x bool
	c *C
}

type C struct {
	i    int
	b    B
	wg   sync.WaitGroup
	test *C
}

func main() {
	a := &A{
		ch:  make(chan int),
		abc: "hello",
	}

	a.ch = make(chan int, 1)
	a.ch <- 42

	y := *a
	close(y.ch)

	var t C

	t.b.Lock()
	defer t.b.Unlock()

	t.wg.Add(1)

	t.b.c = &t

	t.b.c.wg.Done()
	t.b.c.wg.Wait()

	t.b.c = new(C)
	t.b.c.i++

	t.b.ch = make(chan int)
	close(t.b.ch)
	t.b.processAll()

	var x C = C{
		i: 42,
		b: B{
			x: true,
			A: A{"", make(chan int, 5)},
		},
	}
	x.wg.Wait()
}
