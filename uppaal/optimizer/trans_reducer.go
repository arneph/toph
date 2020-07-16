package optimizer

import (
	"github.com/arneph/toph/uppaal"
)

// ReduceTransitions removes duplicate transitions in the given Uppaal system.
func ReduceTransitions(system *uppaal.System) {
	for _, process := range system.Processes() {
		reduceTransitionsInProcess(process)
	}
}

func reduceTransitionsInProcess(process *uppaal.Process) {
	removalSet := make(map[*uppaal.Trans]bool)
	for _, x := range process.TransitionLookup() {
		for _, transitions := range x {
			for i, t1 := range transitions {
				if removalSet[t1] {
					continue
				}
				for _, t2 := range transitions[i+1:] {
					if t1.Select() == t2.Select() &&
						t1.Guard() == t2.Guard() &&
						t1.Sync() == t2.Sync() &&
						t1.Update() == t2.Update() {
						removalSet[t2] = true
					}
				}
			}
		}
	}
	for trans := range removalSet {
		process.RemoveTransition(trans)
	}
}
