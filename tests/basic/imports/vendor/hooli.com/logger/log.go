package logger

import "fmt"

var ch chan struct{}

func init() {
	ch = make(chan struct{}, 1)
}

func init() {
	Log("starting log")
}

// Log logs logs.
func Log(s string) {
	ch <- struct{}{}
	fmt.Println(s)
	<-ch
}
