package main

func ifExample(chA, chB chan int) {
	if true {
		chA <- 42
	} else {
		close(chB)
	}
}

func forExample(chA, chB chan int) {
	for i := 0; i < 10; i++ {
		if true {
			<-chA
		} else {
			close(chA)
			continue
		}

		if false {
			chA <- 123
			break
		}
		chB <- 42
	}
}

func main() {
	chA := make(chan int)
	chB := make(chan int)

	ifExample(chA, chB)
	go forExample(chA, chB)
}
