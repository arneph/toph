package main

import (
	"fmt"
	"sync"
)

func main() {
	testA()
	testB()
}

func testA() {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fmt.Println(i)
		}(i)
	}
	wg.Wait()
}

func testB() {
	var wg sync.WaitGroup
	wg.Add(4)
	go wg.Done()
	go wg.Add(-1)
	defer wg.Wait()
	defer wg.Add(-1)
	defer wg.Add(-1)
}
