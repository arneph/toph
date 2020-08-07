package optimizer

import (
	"fmt"

	c "github.com/arneph/toph/config"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/ir/analyzer"
)

// EliminateDeadCode removes statements that are not necessary for
// verification.
func EliminateDeadCode(program *ir.Program, config *c.Config) {
	fcg := analyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go, config)
	emptyFuncs := make(map[*ir.Func]bool)
	for i := 1; i < fcg.SCCCount(); i++ {
		scc := analyzer.SCC(i)
		for _, f := range fcg.FuncsInSCC(scc) {
			eliminateDeadCodeInBody(f.Body(), true)
			eliminateFuncCallsInBody(f.Body(), emptyFuncs, fcg)
		}
		for _, f := range fcg.FuncsInSCC(scc) {
			if isBodyEmpty(f.Body()) {
				emptyFuncs[f] = true
			}
		}
	}
}

func eliminateDeadCodeInBody(body *ir.Body, isFuncBody bool) {
	stmts := make([]ir.Stmt, 0, len(body.Stmts()))

stmtsLoop:
	for _, stmt := range body.Stmts() {
		switch stmt := stmt.(type) {
		case *ir.IfStmt:
			eliminateDeadCodeInBody(stmt.IfBranch(), false)
			eliminateDeadCodeInBody(stmt.ElseBranch(), false)

			if isBodyEmpty(stmt.IfBranch()) && isBodyEmpty(stmt.ElseBranch()) {
				continue stmtsLoop
			}
		case *ir.ForStmt:
			eliminateDeadCodeInBody(stmt.Cond(), false)
			eliminateDeadCodeInBody(stmt.Body(), false)

			if !stmt.IsInfinite() &&
				isBodyEmpty(stmt.Cond()) && isBodyEmpty(stmt.Body()) {
				continue stmtsLoop
			}
		case *ir.ReturnStmt:
			if isFuncBody && !stmt.IsPanic() && len(stmt.Results()) == 0 {
				continue stmtsLoop
			}
		}
		stmts = append(stmts, stmt)
	}

	body.SetStmts(stmts)
}

func eliminateFuncCallsInBody(body *ir.Body, calleesToEliminate map[*ir.Func]bool, fcg *analyzer.FuncCallGraph) {
	stmts := make([]ir.Stmt, 0, len(body.Stmts()))

stmtsLoop:
	for _, stmt := range body.Stmts() {
		if callStmt, ok := stmt.(*ir.CallStmt); ok {
			switch callStmt.Callee().(type) {
			case *ir.Func:
				callee := callStmt.Callee().(*ir.Func)
				if calleesToEliminate[callee] {
					continue stmtsLoop
				}
			case *ir.Variable, *ir.FieldSelection, *ir.ContainerAccess:
				canEliminiate := true
				calleeSig := callStmt.CalleeSignature()
				for _, dynCallee := range fcg.DynamicCallees(calleeSig) {
					if !calleesToEliminate[dynCallee] {
						canEliminiate = false
						break
					}
				}
				if canEliminiate {
					continue stmtsLoop
				}
			default:
				panic(fmt.Errorf("unexpected callee type: %T", callStmt.Callee()))
			}
		}
		stmts = append(stmts, stmt)
	}
	body.SetStmts(stmts)
}

func isBodyEmpty(body *ir.Body) bool {
	return len(body.Stmts()) == 0
}
