package translator

import (
	"fmt"
	"go/types"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateOnceDoStmt(stmt *ir.OnceDoStmt, ctx *context) {
	var rvs randomVariableSupplier
	handle, _ := t.translateLValue(stmt.Once(), &rvs, ctx)
	name := stmt.Once().Name()

	onceVar := "oid"
	ctx.proc.Declarations().AddVariable(onceVar, "int", "")

	start := ctx.currentState

	exit := ctx.proc.AddState(name+"_exit_", uppaal.Renaming)
	exit.SetComment(t.program.FileSet().Position(stmt.Pos()).String())

	do := ctx.proc.AddState(name+"_do_", uppaal.Renaming)
	do.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	do.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{136, 136}))

	dontEnter := ctx.proc.AddTransition(ctx.currentState, exit)
	dontEnter.SetGuard(fmt.Sprintf("once_values[%s] == 2", handle), true)
	rvs.addToTrans(dontEnter)
	dontEnter.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	dontEnter.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))

	doEnter := ctx.proc.AddTransition(ctx.currentState, do)
	doEnter.SetGuard(fmt.Sprintf("once_values[%s] == 0", handle), true)
	doEnter.AddUpdate(fmt.Sprintf("oid = %s", handle), true)
	doEnter.AddUpdate("\nonce_values[oid] = 1", true)
	rvs.addToTrans(dontEnter)
	doEnter.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	doEnter.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{140, 80}))
	doEnter.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{140, 96}))

	ctx.currentState = do

	t.translateCallStmt(
		ir.NewCallStmt(
			stmt.F(),
			types.NewSignature(nil, nil, nil, false),
			ir.Call, stmt.Pos(), stmt.End()),
		ctx)

	doExit := ctx.proc.AddTransition(ctx.currentState, exit)
	doExit.AddUpdate("once_values[oid] = 2", true)
	doExit.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))

	exit.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{-136, 136}))

	ctx.proc.AddQuery(uppaal.NewQuery(
		"A[] (not out_of_resources) imply (not (deadlock and $."+start.Name()+"))",
		"check deadlock with pending once operation unreachable",
		t.program.FileSet().Position(stmt.Pos()).String(),
		uppaal.NoOnceRelatedDeadlocks))

	ctx.currentState = exit
	ctx.addLocation(do.Location())
	ctx.addLocation(exit.Location())
}
