package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateWaitGroupOpSmt(stmt *ir.WaitGroupOpStmt, ctx *context) {
	var rvs randomVariableSupplier
	handle, _ := t.translateLValue(stmt.WaitGroup(), &rvs, ctx)
	name := stmt.WaitGroup().Name()
	var isWait bool
	var registeredName, completedName, registerUpdate, sync, completeUpdate string

	waitGroupVar := "op_wait_group"
	assign := waitGroupVar + " = " + handle
	ctx.proc.Declarations().AddVariable(waitGroupVar, "int", "0")

	switch stmt.Op() {
	case ir.Add:
		delta, _ := t.translateRValue(stmt.Delta(), nil, ctx)
		completedName = "added_to_wait_group"
		sync = fmt.Sprintf("add[%s]!", handle)
		completeUpdate = fmt.Sprintf("wait_group_counter[%s] += %s", handle, delta)
	case ir.Wait:
		isWait = true
		registeredName = "awaiting_wait_group"
		completedName = "awaited_wait_group"
		registerUpdate = fmt.Sprintf("wait_group_waiters[%s]++", waitGroupVar)
		sync = fmt.Sprintf("wait[%s]?", waitGroupVar)
		completeUpdate = fmt.Sprintf("wait_group_waiters[%s]--", waitGroupVar)
	default:
		t.addWarning(fmt.Errorf("unsupported WaitGroupOp: %v", stmt.Op()))
	}

	if isWait {
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
	if !isWait {
		rvs.addToTrans(complete)
	}
	complete.SetSync(sync)
	complete.AddUpdate(completeUpdate, true)
	complete.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	complete.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	complete.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))
	complete.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 96}))

	ctx.currentState = completed
	ctx.addLocation(completed.Location())
}
