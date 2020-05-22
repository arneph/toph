package main

func test(a, b chan int) {
	if 24 < 42 {
		for i := 0; i < 100; i++ {
			for j := 0; j < 100; j++ {
				if <-a == 42 {
					b <- 24
				} else {
					select {
					case b <- 24:
					default:
						break
					}
				}
			}
		}
	} else if 42 < 24 {
		select {
		case <-a:
			if 24 < 42 {
				b <- 24
			}
		default:
			for {
				select {}
			}
		}
		select {}
	} else {
		for i := 0; i < 100; i++ {
			for j := 0; j < 100; j++ {
				a <- <-b
			}
		}

	}
}

func main() {
	ch := make(chan int)
	if 24 < 42 {
		ch <- 24
	} else if 36 < 42 {
		ch <- 36
	} else if 40 < 42 {
	} else {
		ch <- 42
	}
}
