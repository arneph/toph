package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateStmt(stmt ir.Stmt, ctx *context) {
	switch stmt := stmt.(type) {
	case *ir.AssignStmt:
		t.translateAssignStmt(stmt, ctx)
	case *ir.CallStmt:
		t.translateCallStmt(stmt, ctx)
	case *ir.ReturnStmt:
		t.translateReturnStmt(stmt, ctx)
	case *ir.RecoverStmt:
		t.translateRecoverStmt(stmt, ctx)
	case *ir.IfStmt:
		t.translateIfStmt(stmt, ctx)
	case *ir.SwitchStmt:
		t.translateSwitchStmt(stmt, ctx)
	case *ir.ForStmt:
		t.translateForStmt(stmt, ctx)
	case *ir.RangeStmt:
		t.translateRangeStmt(stmt, ctx)
	case *ir.BranchStmt:
		t.translateBranchStmt(stmt, ctx)
	case *ir.MakeStructStmt:
		t.translateMakeStructStmt(stmt, ctx)
	case *ir.MakeChanStmt:
		t.translateMakeChanStmt(stmt, ctx)
	case *ir.ChanCommOpStmt:
		t.translateChanCommOpStmt(stmt, ctx)
	case *ir.CloseChanStmt:
		t.translateCloseChanStmt(stmt, ctx)
	case *ir.SelectStmt:
		t.translateSelectStmt(stmt, ctx)
	case *ir.DeadEndStmt:
		t.translateDeadEndStmt(stmt, ctx)
	case *ir.MakeMutexStmt:
		t.translateMakeMutexStmt(stmt, ctx)
	case *ir.MutexOpStmt:
		t.translateMutexOpStmt(stmt, ctx)
	case *ir.MakeWaitGroupStmt:
		t.translateMakeWaitGroupStmt(stmt, ctx)
	case *ir.WaitGroupOpStmt:
		t.translateWaitGroupOpSmt(stmt, ctx)
	default:
		t.addWarning(fmt.Errorf("ignoring %T statement", stmt))
	}
}

func (t *translator) translateDeadEndStmt(stmt *ir.DeadEndStmt, ctx *context) {
	dead := ctx.proc.AddState("dead_end_", uppaal.Renaming)
	dead.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	dead.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))

	ctx.proc.AddTrans(ctx.currentState, dead)

	ctx.addLocation(dead.Location())

	ctx.currentState = ctx.exitFuncState
}
