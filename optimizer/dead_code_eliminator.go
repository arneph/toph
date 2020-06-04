package optimizer

import (
	"fmt"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/ir"
)

// EliminateDeadCode removes statements that are not necessary for
// verification.
func EliminateDeadCode(program *ir.Program) {
	entryFunc := program.EntryFunc()
	if entryFunc == nil {
		return
	}
	fcg := analyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go)
	emptyFuncs := make(map[*ir.Func]bool)
	for i := 1; i < fcg.SCCCount(); i++ {
		scc := analyzer.SCC(i)
		for _, f := range fcg.FuncsInSCC(scc) {
			eliminateDeadCodeInBody(f.Body())
			eliminateFuncCallsInBody(f.Body(), emptyFuncs, fcg)
		}
		for _, f := range fcg.FuncsInSCC(scc) {
			if isBodyEmpty(f.Body()) {
				emptyFuncs[f] = true
			}
		}
	}
}

// EliminateUnusedFunctions removes functions that are never called.
func EliminateUnusedFunctions(program *ir.Program) {
	fcg := analyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go)
	oldFuncs := make(map[*ir.Func]bool)
	for _, f := range program.Funcs() {
		if fcg.CalleeCount(f) > 0 {
			continue
		}
		oldFuncs[f] = true
	}
	program.RemoveFuncs(oldFuncs)
}

func eliminateDeadCodeInBody(body *ir.Body) {
	stmts := make([]ir.Stmt, 0, len(body.Stmts()))

stmtsLoop:
	for _, stmt := range body.Stmts() {
		switch stmt := stmt.(type) {
		case *ir.ForStmt:
			eliminateDeadCodeInBody(stmt.Cond())
			eliminateDeadCodeInBody(stmt.Body())

			if !stmt.IsInfinite() &&
				isBodyEmpty(stmt.Cond()) && isBodyEmpty(stmt.Body()) {
				continue stmtsLoop
			}
		case *ir.IfStmt:
			eliminateDeadCodeInBody(stmt.IfBranch())
			eliminateDeadCodeInBody(stmt.ElseBranch())

			if isBodyEmpty(stmt.IfBranch()) && isBodyEmpty(stmt.ElseBranch()) {
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
			switch callee := callStmt.Callee().(type) {
			case *ir.Func:
				if calleesToEliminate[callee] {
					continue stmtsLoop
				}
			case *ir.Variable:
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
				panic(fmt.Errorf("unexpected callee type: %T", callee))
			}
		}
		stmts = append(stmts, stmt)
	}
	body.SetStmts(stmts)
}

func isBodyEmpty(body *ir.Body) bool {
	return len(body.Stmts()) == 0
}
