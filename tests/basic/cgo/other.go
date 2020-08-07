// +build cgo,darwin,linux
package main

/*
#include <unistd.h>
*/
import "C"

func getOtherClockTicks() int {
	return int(C.sysconf(C._SC_CLK_TCK))
}
