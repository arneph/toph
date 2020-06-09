package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func a() {
	randomChan := make(chan int)
	errChan := make(chan error)
	select {
	case <-randomChan:
		close(errChan)
	case <-time.After(time.Second):
		errChan <- fmt.Errorf("oh no")
	}
}

func b() {
	pathChan := make(chan string)
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		pathChan <- path
		return nil
	})
}

func main() {
	a()
	b()
}
