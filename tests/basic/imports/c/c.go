package c

// ReduceFunc is the function type for a reducer function.
type ReduceFunc func(a, b string) string

// Reduce reads values from the input channel until it is closed and applies
// the reducer function.
func Reduce(inCh chan string, startVal string, reducer ReduceFunc) string {
	a := startVal
	for b := range inCh {
		a = reducer(a, b)
	}
	return a
}
