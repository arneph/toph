package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

var errChan chan error = make(chan error)
var abortChan chan struct{} = make(chan struct{})

func findFilesInFolder(root string, filesChan chan string) {
	defer close(filesChan)
	files, err := ioutil.ReadDir(root)
	if err != nil {
		errChan <- err
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		select {
		case filesChan <- root + "/" + file.Name():
		case <-abortChan:
			return
		}
	}
}

func countWords(filesChan chan string, wordCountsChan chan int) {
	for file := range filesChan {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			select {
			case errChan <- err:
				continue
			case <-abortChan:
				return
			}
		}
		text := string(content)
		count := strings.Count(text, " ")
		select {
		case wordCountsChan <- count: // possible send on closed channel
		case <-abortChan:
			return
		}
	}
	close(wordCountsChan) // double close
}

func main() {
	filesChan := make(chan string)
	wordCountsChan := make(chan int)
	doneChan := make(chan struct{})

	root := os.Args[1]
	go findFilesInFolder(root, filesChan)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			countWords(filesChan, wordCountsChan)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	sum := 0
loop:
	for {
		select {
		case count := <-wordCountsChan:
			sum += count
		case err := <-errChan:
			fmt.Println(err)
			close(abortChan)
			return
		case <-doneChan:
			break loop
		}
	}
	fmt.Printf("counted %d words\n", sum)
}
