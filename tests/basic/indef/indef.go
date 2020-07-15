package main

func main() {
	ch := make(chan int)
	go func() {
		for {
			ch <- 0
		}
	}()
	go func() {
		for {
			<-ch
		}
	}()
}
