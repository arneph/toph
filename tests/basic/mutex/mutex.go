package main

import (
	"fmt"
	"math/rand"
	"sync"
)

func mutexTest() {
	var mu sync.Mutex
	var x int
	for i := 0; i < 3; i++ {
		go func() {
			mu.Lock()
			defer mu.Unlock()
			x++
			fmt.Println(x)
		}()
	}
}

func rwMutexTest() {
	var mu sync.RWMutex
	var x int
	for i := 0; i < 3; i++ {
		go func() {
			mu.Lock()
			x++
			mu.Unlock()
		}()
	}
	for i := 0; i < 5; i++ {
		go func() {
			mu.RLock()
			defer mu.RUnlock()
			fmt.Println(x)
		}()
	}
}

func main() {
	if rand.Int()%2 == 0 {
		mutexTest()
	} else {
		rwMutexTest()
	}
}
