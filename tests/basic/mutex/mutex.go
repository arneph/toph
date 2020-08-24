package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func mutex() {
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

func rwMutex() {
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
		mutex()
	} else {
		rwMutex()
	}
	var db quickDB
	quickDBTest(&db)
}

type quickDB struct {
	sync.RWMutex
	m map[string]interface{}
}

func quickDBTest(db *quickDB) {
	db.RLock()
	time.Sleep(1 * time.Second)
	db.RUnlock()
}
