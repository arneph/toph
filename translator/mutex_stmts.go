package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateMutexOpStmt(stmt *ir.MutexOpStmt, ctx *context) {
	var rvs randomVariableSupplier
	handle, _ := t.translateLValue(stmt.Mutex(), &rvs, ctx)
	name := stmt.Mutex().Name()
	var isLock bool
	var registeredName, completedName, registerUpdate, sync, completeUpdate string

	mutexVar := "op_mutex"
	assign := mutexVar + " = " + handle
	ctx.proc.Declarations().AddVariable(mutexVar, "int", "0")

	switch stmt.Op() {
	case ir.Lock:
		isLock = true
		registeredName = "awaiting_write_lock"
		completedName = "aquired_write_lock"
		registerUpdate = fmt.Sprintf("mutex_pending_writers[%s]++", mutexVar)
		sync = fmt.Sprintf("write_lock[%s]!", mutexVar)
		completeUpdate = fmt.Sprintf("mutex_pending_writers[%s]--", mutexVar)
	case ir.RLock:
		isLock = true
		registeredName = "awaiting_read_lock"
		completedName = "aquired_read_lock"
		registerUpdate = fmt.Sprintf("mutex_pending_readers[%s]++", mutexVar)
		sync = fmt.Sprintf("read_lock[%s]!", mutexVar)
		completeUpdate = fmt.Sprintf("mutex_pending_readers[%s]--", mutexVar)
	case ir.RUnlock:
		completedName = "released_read_lock"
		sync = fmt.Sprintf("read_unlock[%s]!", handle)
	case ir.Unlock:
		completedName = "released_write_lock"
		sync = fmt.Sprintf("write_unlock[%s]!", handle)
	default:
		t.addWarning(fmt.Errorf("unsupported MutexOp: %v", stmt.Op()))
	}

	if isLock {
		registered := ctx.proc.AddState(registeredName+"_"+name+"_", uppaal.Renaming)
		registered.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		registered.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))

		register := ctx.proc.AddTransition(ctx.currentState, registered)
		rvs.addToTrans(register)
		register.AddUpdate(assign, true)
		register.AddUpdate(registerUpdate, true)
		register.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
		register.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
		register.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))

		ctx.proc.AddQuery(uppaal.NewQuery(
			"A[] (not out_of_resources) imply (not (deadlock and $."+registered.Name()+"))",
			"check deadlock with pending mutex operation unreachable",
			t.program.FileSet().Position(stmt.Pos()).String(),
			uppaal.NoMutexRelatedDeadlocks))

		ctx.currentState = registered
		ctx.addLocation(registered.Location())
	}

	completed := ctx.proc.AddState(completedName+"_"+name+"_", uppaal.Renaming)
	completed.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	completed.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	complete := ctx.proc.AddTransition(ctx.currentState, completed)
	if !isLock {
		rvs.addToTrans(complete)
	}
	complete.SetSync(sync)
	if isLock {
		complete.AddUpdate(completeUpdate, true)
	}
	complete.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	complete.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	complete.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))
	complete.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 90}))

	ctx.currentState = completed
	ctx.addLocation(completed.Location())
}
