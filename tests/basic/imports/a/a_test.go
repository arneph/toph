package a

import "testing"

func TestRelay(t *testing.T) {
	chA := make(chan int)
	chB := make(chan int)
	go Relay(chA, chB)
	go func() {
		chA <- 1
		chA <- 2
		chA <- 3
		close(chA)
	}()
	for _, want := range []int{1, 2, 3} {
		got, ok := <-chB
		if !ok {
			t.Errorf("expected: %d, but channel was closed", want)
		} else if got != want {
			t.Errorf("expected: %d, got: %d", want, got)
		}
	}
	_, ok := <-chB
	if ok {
		t.Errorf("expected outCh to be closed")
	}
}
