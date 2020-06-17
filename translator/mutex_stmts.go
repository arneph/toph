package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateMakeMutexStmt(stmt *ir.MakeMutexStmt, ctx *context) {
	handle := t.translateVariable(stmt.Mutex(), ctx)
	name := stmt.Mutex().Name()

	made := ctx.proc.AddState("made_"+name+"_", uppaal.Renaming)
	made.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	made.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	make := ctx.proc.AddTrans(ctx.currentState, made)
	make.AddUpdate(fmt.Sprintf("%s = make_mutex()", handle))
	make.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))
	ctx.currentState = made
	ctx.addLocation(made.Location())
}

func (t *translator) translateMutexOpStmt(stmt *ir.MutexOpStmt, ctx *context) {
	handle := t.translateVariable(stmt.Mutex(), ctx)
	name := stmt.Mutex().Name()
	var isLock bool
	var registeredName, completedName, registerUpdate, sync, completeUpdate string

	switch stmt.Op() {
	case ir.Lock:
		isLock = true
		registeredName = "awaiting_write_lock"
		completedName = "aquired_write_lock"
		registerUpdate = fmt.Sprintf("mutex_pending_writers[%s]++", handle)
		sync = fmt.Sprintf("write_lock[%s]!", handle)
		completeUpdate = fmt.Sprintf("mutex_pending_writers[%s]--", handle)
	case ir.RLock:
		isLock = true
		registeredName = "awaiting_read_lock"
		completedName = "aquired_read_lock"
		registerUpdate = fmt.Sprintf("mutex_pending_readers[%s]++", handle)
		sync = fmt.Sprintf("read_lock[%s]!", handle)
		completeUpdate = fmt.Sprintf("mutex_pending_readers[%s]--", handle)
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

		register := ctx.proc.AddTrans(ctx.currentState, registered)
		register.AddUpdate(registerUpdate)
		register.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))

		ctx.proc.AddQuery(uppaal.MakeQuery(
			"A[] (not out_of_resources) imply (not (deadlock and $."+registered.Name()+"))",
			"check deadlock with pending mutex operation unreachable"))

		ctx.currentState = registered
		ctx.addLocation(registered.Location())
	}

	completed := ctx.proc.AddState(completedName+"_"+name+"_", uppaal.Renaming)
	completed.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	completed.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	complete := ctx.proc.AddTrans(ctx.currentState, completed)
	complete.SetSync(sync)
	complete.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	if isLock {
		complete.AddUpdate(completeUpdate)
		complete.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	}

	ctx.currentState = completed
	ctx.addLocation(completed.Location())
}
