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

	enter := ctx.proc.AddState(name+"_enter_", uppaal.Renaming)
	enter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	enter.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))

	exit := ctx.proc.AddState(name+"_exit_", uppaal.Renaming)
	exit.SetComment(t.program.FileSet().Position(stmt.Pos()).String())

	do := ctx.proc.AddState(name+"_do_", uppaal.Renaming)
	do.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	do.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{136, 272}))

	trans1 := ctx.proc.AddTransition(ctx.currentState, enter)
	rvs.addToTrans(trans1)
	trans1.AddUpdate(fmt.Sprintf("oid = %s", handle), true)
	trans1.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	trans1.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	trans1.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))

	dontExecute := ctx.proc.AddTransition(enter, exit)
	dontExecute.SetGuard("once_values[oid] == 2", true)
	dontExecute.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 184}))

	doExecute := ctx.proc.AddTransition(enter, do)
	doExecute.SetGuard("once_values[oid] == 0", true)
	doExecute.AddUpdate("once_values[oid] = 1", true)
	doExecute.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{140, 200}))
	doExecute.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{140, 216}))

	ctx.currentState = do

	var callee ir.Callable
	switch f := stmt.F().(type) {
	case ir.Value:
		callee = t.program.Func(ir.FuncIndex(f.Value()))
	case ir.LValue:
		callee = f.(ir.Callable)
	default:
		panic("unexpected rvalue type")
	}

	t.translateCallStmt(
		ir.NewCallStmt(
			callee,
			types.NewSignature(nil, nil, nil, false),
			ir.Call, stmt.Pos(), stmt.End()),
		ctx)

	doExit := ctx.proc.AddTransition(ctx.currentState, exit)
	doExit.AddUpdate("once_values[oid] = 2", true)
	doExit.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))

	exit.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{-136, 136}))

	ctx.proc.AddQuery(uppaal.NewQuery(
		"A[] (not out_of_resources) imply (not (deadlock and $."+enter.Name()+"))",
		"check deadlock with pending once operation unreachable",
		t.program.FileSet().Position(stmt.Pos()).String(),
		uppaal.NoOnceRelatedDeadlocks))

	ctx.currentState = exit
	ctx.addLocation(do.Location())
	ctx.addLocation(exit.Location())
}
