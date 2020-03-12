package main

func test(a, b chan int) {
	if 24 < 42 {
		// toph: min_iter=123, max_iter=234
		for i := 0; i < 100; i++ {
			// toph: min_iter=345, max_iter=456
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
			// toph: min_iter=567, max_iter=678
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
