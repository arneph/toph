package optimizer

import (
	"github.com/arneph/toph/uppaal"
)

// ReduceStates attempts to remove states in the given Uppaal system by
// combining incoming and outgoing transitions for a state, if possible.
func ReduceStates(system *uppaal.System) {
	for _, process := range system.Processes() {
		reduceStatesInProcess(process)
	}
}

func reduceStatesInProcess(process *uppaal.Process) {
	sortedStates := make([]*uppaal.State, 0, len(process.States()))
	seen := map[*uppaal.State]bool{process.InitialState(): true}
	queue := []*uppaal.State{process.InitialState()}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		sortedStates = append(sortedStates, current)

		for _, trans := range current.OutgoingTransitions() {
			child := trans.End()
			if seen[child] {
				continue
			}
			seen[child] = true
			queue = append(queue, child)
		}
	}

	for _, state := range sortedStates {
		removeStateIfPossible(process, state)
	}
}

func removeStateIfPossible(process *uppaal.Process, state *uppaal.State) {
	// Check if state can be removed:
	if state.IsInitialState() || state.Type() != uppaal.Normal {
		return
	}

	oldTrans := append([]*uppaal.Trans{}, state.Transitions()...)
	for _, trans := range oldTrans {
		if trans.Start() == trans.End() ||
			trans.Start().Type() != uppaal.Normal ||
			trans.End().Type() != uppaal.Normal ||
			trans.Select() != "" {
			return
		}
	}

	oldInTrans := state.IncomingTransitions()
	oldOutTrans := state.OutgoingTransitions()
	if len(oldOutTrans) == 0 {
		return
	}

	type transProps struct {
		hasGuard    bool
		hasSync     bool
		hasUpdate   bool
		usesGlobals bool
	}
	analyze := func(set []*uppaal.Trans) (props transProps) {
		for _, trans := range set {
			if trans.Guard() != "" {
				props.hasGuard = true
			}
			if trans.Sync() != "" {
				props.hasSync = true
			}
			if trans.Update() != "" {
				props.hasUpdate = true
			}
			if trans.UsesGlobals() {
				props.usesGlobals = true
			}
		}
		return
	}
	oldInTransProps := analyze(oldInTrans)
	oldOutTransProps := analyze(oldOutTrans)

	if oldInTransProps.usesGlobals && oldOutTransProps.usesGlobals {
		return
	}

	if (oldInTransProps.hasSync || oldInTransProps.hasUpdate) &&
		oldOutTransProps.hasGuard {
		// Committed would be possible
		return
	} else if oldInTransProps.hasUpdate &&
		oldOutTransProps.hasSync {
		// Committed would be possible
		return
	}

	// Remove state and replace transitions:
	for _, trans := range oldTrans {
		process.RemoveTransition(trans)
	}
	process.RemoveState(state)

	for _, oldIn := range oldInTrans {
		for _, oldOut := range oldOutTrans {
			newTrans := process.AddTransition(oldIn.Start(), oldOut.End())

			if oldIn.Guard() != "" && oldOut.Guard() != "" {
				newTrans.SetGuard("("+oldIn.Guard()+") && \n("+oldOut.Guard()+")",
					oldIn.GuardUsesGlobals() || oldOut.GuardUsesGlobals())
				newTrans.SetGuardLocation(oldIn.GuardLocation())
			} else if oldIn.Guard() != "" {
				newTrans.SetGuard(oldIn.Guard(), oldIn.GuardUsesGlobals())
				newTrans.SetGuardLocation(oldIn.GuardLocation())
			} else if oldOut.Guard() != "" {
				newTrans.SetGuard(oldOut.Guard(), oldOut.GuardUsesGlobals())
				newTrans.SetGuardLocation(oldOut.GuardLocation())
			}

			if oldIn.Sync() != "" {
				newTrans.SetSync(oldIn.Sync())
				newTrans.SetSyncLocation(oldIn.SyncLocation())
			} else if oldOut.Sync() != "" {
				newTrans.SetSync(oldOut.Sync())
				newTrans.SetSyncLocation(oldOut.SyncLocation())
			}

			if oldIn.Update() != "" && oldOut.Update() != "" {
				newTrans.AddUpdate(oldIn.Update(), oldIn.UpdateUsesGlobals())
				newTrans.AddUpdate("\n"+oldOut.Update(), oldOut.UpdateUsesGlobals())
				newTrans.SetUpdateLocation(oldIn.UpdateLocation())
			} else if oldIn.Update() != "" {
				newTrans.AddUpdate(oldIn.Update(), oldIn.UpdateUsesGlobals())
				newTrans.SetUpdateLocation(oldIn.UpdateLocation())
			} else if oldOut.Update() != "" {
				newTrans.AddUpdate(oldOut.Update(), oldOut.UpdateUsesGlobals())
				newTrans.SetUpdateLocation(oldOut.UpdateLocation())
			}

			for _, nail := range oldIn.NailLocations() {
				newTrans.AddNail(nail)
			}
			newTrans.AddNail(state.Location())
			for _, nail := range oldOut.NailLocations() {
				newTrans.AddNail(nail)
			}
		}
	}
}
