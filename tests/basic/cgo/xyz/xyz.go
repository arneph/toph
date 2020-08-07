// +build cgo,darwin,linux
package xyz

// int multiply(int x, int y) {
//     return x * y;
// }
import "C"

func Multiply(x, y int32) int {
	return int(C.multiply(C.int(x), C.int(y)))
}
