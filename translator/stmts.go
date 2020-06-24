package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

func (t *translator) translateStmt(stmt ir.Stmt, ctx *context) {
	switch stmt := stmt.(type) {
	case *ir.AssignStmt:
		t.translateAssignStmt(stmt, ctx)
	case *ir.CallStmt:
		t.translateCallStmt(stmt, ctx)
	case *ir.ReturnStmt:
		t.translateReturnStmt(stmt, ctx)
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
	case *ir.MakeChanStmt:
		t.translateMakeChanStmt(stmt, ctx)
	case *ir.ChanCommOpStmt:
		t.translateChanCommOpStmt(stmt, ctx)
	case *ir.CloseChanStmt:
		t.translateCloseChanStmt(stmt, ctx)
	case *ir.SelectStmt:
		t.translateSelectStmt(stmt, ctx)
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
