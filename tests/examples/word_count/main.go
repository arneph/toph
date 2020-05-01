package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var errChan chan error = make(chan error, 3)
var abortChan chan struct{} = make(chan struct{})

func findFilesInFolder(root string,
	filesChan chan string) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		errChan <- err
		close(filesChan)
		return
	}
	// toph: max_iter=3
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		select {
		case filesChan <- file.Name():
		case <-abortChan:
			break
		}
	}
	close(filesChan)
}

func countWords(filesChan chan string,
	wordCountsChan chan int) {
	for file := range filesChan {
		select {
		case <-abortChan:
			return
		default:
		}
		content, err := ioutil.ReadFile(file)
		if err != nil {
			errChan <- err
			break
		}
		text := string(content)
		count := strings.Count(text, " ")
		wordCountsChan <- count
	}
}

func main() {
	filesChan := make(chan string, 2)
	wordCountsChan := make(chan int, 2)

	root := os.Args[1]
	go findFilesInFolder(root, filesChan)

	waitChan := make(chan struct{})
	doneChan := make(chan struct{})
	// toph: min_iter=2, max_iter=2
	for i := 0; i < 2; i++ {
		go func() {
			countWords(filesChan, wordCountsChan)
			waitChan <- struct{}{}
		}()
	}
	go func() {
		// toph: min_iter=2, max_iter=2
		for i := 0; i < 2; i++ {
			<-waitChan
		}
		close(doneChan)
	}()

	totalCount := 0
	for {
		select {
		case err := <-errChan:
			close(abortChan)
			fmt.Println(err)
			return
		case c := <-wordCountsChan:
			totalCount += c
		case <-doneChan:
			fmt.Printf("counted %d words\n",
				totalCount)
			return
		}
	}
}
