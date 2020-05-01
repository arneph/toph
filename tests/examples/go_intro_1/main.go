package main

import (
	"fmt"
	"time"
)

func f() {
	time.Sleep(1 * time.Second)
	fmt.Println("Hello")
}

func g() {
	time.Sleep(2 * time.Second)
	fmt.Println("World")
}

func main() {
	go f()
	go g()
	time.Sleep(3 * time.Second)
	fmt.Println("!!!")
}
