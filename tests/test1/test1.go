package main

import "fmt"

var ch chan int

func main() {
	var n, sum int
	fmt.Scan(&n)
	for i := 0; i < n; i++ {
		sum += i
	}
	fmt.Println(sum)
}
