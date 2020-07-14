package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateMakeWaitGroupStmt(stmt *ir.MakeWaitGroupStmt, ctx *context) {
	handle, usesGlobals := t.translateLValue(stmt.WaitGroup(), ctx)
	name := stmt.WaitGroup().Name()

	made := ctx.proc.AddState("made_"+name+"_", uppaal.Renaming)
	made.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	made.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	make := ctx.proc.AddTransition(ctx.currentState, made)
	make.AddUpdate(fmt.Sprintf("%s = make_wait_group()", handle), usesGlobals)
	make.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))
	ctx.currentState = made
	ctx.addLocation(made.Location())
}

func (t *translator) translateWaitGroupOpSmt(stmt *ir.WaitGroupOpStmt, ctx *context) {
	handle, _ := t.translateLValue(stmt.WaitGroup(), ctx)
	name := stmt.WaitGroup().Name()
	var isWait bool
	var registeredName, completedName, registerUpdate, sync, completeUpdate string

	switch stmt.Op() {
	case ir.Add:
		delta, _ := t.translateRValue(stmt.Delta(), ir.IntType, ctx)
		completedName = "added_to_wait_group"
		sync = fmt.Sprintf("add[%s]!", handle)
		completeUpdate = fmt.Sprintf("wait_group_counter[%s] += %s", handle, delta)
	case ir.Wait:
		isWait = true
		registeredName = "awaiting_wait_group"
		completedName = "awaited_wait_group"
		registerUpdate = fmt.Sprintf("wait_group_waiters[%s]++", handle)
		sync = fmt.Sprintf("wait[%s]?", handle)
		completeUpdate = fmt.Sprintf("wait_group_waiters[%s]--", handle)
	default:
		t.addWarning(fmt.Errorf("unsupported WaitGroupOp: %v", stmt.Op()))
	}

	if isWait {
		registered := ctx.proc.AddState(registeredName+"_"+name+"_", uppaal.Renaming)
		registered.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		registered.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))

		register := ctx.proc.AddTransition(ctx.currentState, registered)
		register.AddUpdate(registerUpdate, true)
		register.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))

		ctx.proc.AddQuery(uppaal.NewQuery(
			"A[] (not out_of_resources) imply (not (deadlock and $."+registered.Name()+"))",
			"check deadlock with pending wait group operation unreachable",
			t.program.FileSet().Position(stmt.Pos()).String(),
			uppaal.NoWaitGroupRelatedDeadlocks))

		ctx.currentState = registered
		ctx.addLocation(registered.Location())
	}

	completed := ctx.proc.AddState(completedName+"_"+name+"_", uppaal.Renaming)
	completed.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	completed.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	complete := ctx.proc.AddTransition(ctx.currentState, completed)
	complete.SetSync(sync)
	complete.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	complete.AddUpdate(completeUpdate, true)
	complete.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))

	ctx.currentState = completed
	ctx.addLocation(completed.Location())
}
