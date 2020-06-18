package a

// Relay forwards every value on the input channel to the output channel.
// If the input channel gets closed, Relay closes the output channel.
func Relay(inCh, outCh chan int) {
	for x := range inCh {
		outCh <- x
	}
	close(outCh)
}
