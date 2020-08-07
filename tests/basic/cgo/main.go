// +build cgo,darwin,linux
package main

/*
#include <unistd.h>
*/
import "C"
import (
	"fmt"
	"time"

	"github.com/arneph/toph/tests/basic/cgo/xyz"
)

func main() {
	fmt.Println(xyz.Multiply(24, 42))
	for {
		fmt.Println("ticks:", getClockTicks())
		fmt.Println("tocks:", getOtherClockTicks())
		time.Sleep(1 * time.Second)
	}
}

func getClockTicks() int {
	return int(C.sysconf(C._SC_CLK_TCK))
}
