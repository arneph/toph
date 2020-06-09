package main

func a(chA, chB, chC chan int) bool {
mySwitch:
	switch {
	case "abc" == "cba", false:
		<-chC
		fallthrough
	case 1 < 2:
	default:
		chA <- 42
		if 17 < 23 {
			break
		} else if 17 < 23 {
			return false
		}
		fallthrough
	case 24 == 42, 2 < <-chA:
		<-chB
	case 24 != 42, a(chA, chB, chC):
		for x := range chC {
			if x < 42 {
				break mySwitch
			}
		}
	}
	return true
}

func main() {
	chA := make(chan int)
	chB := make(chan int)
	chC := make(chan int)
	a(chA, chB, chC)
}
