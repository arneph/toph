package b

import "hooli.com/logger"

// MapFunc is the type of a function mapping from integers to strings.
type MapFunc func(x int) (string, error)

// ErrorChan is used to report errors.
var ErrorChan chan error = make(chan error)

// DefaultMapFunc is the map function used by DefaultMap.
var DefaultMapFunc MapFunc

// Map applies the map function to all values from the input channel and sends
// the results to the output channel. It closes the output channel when the
// input channel gets closed or an error gets encountered. Any error gets sent
// via ErrorChan.
func Map(inCh chan int, outCh chan string, mapFunc MapFunc) {
	mapImpl(inCh, outCh, mapFunc)
}

// DefaultMap applies the default map function to all values from the input
// channel and sends the results to the output channel. It closes the output
// channel when the input channel gets closed or an error gets encountered. Any
// error gets sent via ErrorChan.
func DefaultMap(inCh chan int, outCh chan string) {
	mapImpl(inCh, outCh, DefaultMapFunc)
}

func mapImpl(inCh chan int, outCh chan string, mapFunc MapFunc) {
	for x := range inCh {
		y, err := mapFunc(x)
		if err != nil {
			ErrorChan <- err
			break
		}
		outCh <- y
	}
	close(outCh)
	logger.Log("completed map")
}
