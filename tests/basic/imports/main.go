package main

import (
	"fmt"
	"math/rand"

	"github.com/arneph/toph/tests/basic/imports/a"
	z "github.com/arneph/toph/tests/basic/imports/b"
	. "github.com/arneph/toph/tests/basic/imports/c"
	_ "github.com/docker/docker/api/server/backend/build"
	"hooli.com/logger"
)

func main() {
	chA := make(chan int)
	chB := make(chan int)
	chC := make(chan int)
	chD := make(chan string)

	go func() {
		for i := 0; i < 10; i++ {
			chA <- i
		}
		close(chA)
	}()

	go a.Relay(chA, chB)

	var oddFilter z.FilterFunc
	oddFilter = func(x int) bool {
		return x%2 == 1
	}

	go z.Filter(chB, chC, oddFilter)

	z.DefaultMapFunc = func(x int) (string, error) {
		return fmt.Sprintf("%d", x), nil
	}

	if rand.Int()%2 == 0 {
		go z.DefaultMap(chC, chD)
	} else {
		go z.Map(chC, chD, func(x int) (string, error) {
			return fmt.Sprintf("0x%x", x), nil
		})
	}

	select {
	case err := <-z.ErrorChan:
		fmt.Println(err)
	default:
	}

	r := Reduce(chD, "", func(a, b string) string {
		if a == "" {
			return b
		}
		return a + ", " + b
	})
	logger.Log(r)
}
