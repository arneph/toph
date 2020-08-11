package main

import (
	"fmt"
	"sync"
)

type IO [2]chan int

type WorkerId int
type Worker struct {
	id                 WorkerId
	io                 IO
	completionHandlers []func(w *Worker)
}

var workers map[WorkerId]*Worker = make(map[WorkerId]*Worker)

func (w *Worker) Work() {
	for i := range w.io[0] {
		w.io[1] <- i
	}
	close(w.io[1])

	for _, h := range w.completionHandlers {
		h(w)
	}
}

func completionLog(w *Worker) {
	fmt.Printf("worker %d finished\n", w.id)
}

func completionDone(w *Worker) {
	wg.Done()
}

var wg sync.WaitGroup

func main() {
	chA := make(chan int, 0)
	chB := make(chan int, 0)
	chStart := chA
	chEnd := chB
	for i := 0; i < 3; i++ {
		handlers := []func(w *Worker){completionLog}
		handlers = append(handlers, completionDone)
		workers[WorkerId(i)] = &Worker{
			id:                 WorkerId(i),
			io:                 IO{chA, chB},
			completionHandlers: handlers,
		}
		chEnd = chB
		chA, chB = chB, make(chan int, 0)
	}
	wg.Add(3)
	for _, w := range workers {
		go w.Work()
	}
	delete(workers, WorkerId(42))
	for i := 100; i <= 10000; i *= 10 {
		chStart <- i
		result := <-chEnd
		fmt.Println("input:", i, "resullt:", result)
	}
	close(chStart)
	wg.Wait()
}
