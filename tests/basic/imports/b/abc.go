package b

// FilterFunc is the type of a function deciding whether to exclude a given
// integer.
type FilterFunc func(x int) bool

// Filter receives values via the input channel and forwards them to the output
// channel if they are not filtered by the filter function.
func Filter(inCh, outCh chan int, filter FilterFunc) {
	for x := range inCh {
		if filter(x) {
			continue
		}
		outCh <- x
	}
	close(outCh)
}
