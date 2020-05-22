package analyzer

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

// BuildFuncCallGraph returns a new function call graph for the given program,
// program entry function, and call kind. Only calls of the given call kinds
// are contained in the graph.
func BuildFuncCallGraph(program *ir.Program, callKinds ir.CallKind) *FuncCallGraph {
	fcg := newFuncCallGraph(program.EntryFunc())
	if program.EntryFunc() == nil {
		return fcg
	}

	addFuncsToFuncCallGraph(program, fcg)
	addCallsToFuncCallGraph(program, callKinds, fcg)
	addCallCountsToFuncCallGraph(program, callKinds, fcg)

	return fcg
}

func addFuncsToFuncCallGraph(program *ir.Program, fcg *FuncCallGraph) {
	for _, f := range program.Funcs() {
		fcg.addFunc(f)
	}
}

func addCallsToFuncCallGraph(program *ir.Program, callKinds ir.CallKind, fcg *FuncCallGraph) {
	for _, caller := range program.Funcs() {
		caller.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
			callStmt, ok := stmt.(*ir.CallStmt)
			if !ok || callStmt.Kind()&callKinds == 0 {
				return
			}
			switch callee := callStmt.Callee().(type) {
			case *ir.Func:
				fcg.addStaticCall(caller, callee)
			case *ir.Variable:
				calleeSig := callStmt.CalleeSignature()
				fcg.addDynamicCall(caller, calleeSig)
			default:
				panic(fmt.Errorf("unexpected callee type: %T", callee))
			}
		})
	}
}

func addCallCountsToFuncCallGraph(program *ir.Program, callKinds ir.CallKind, fcg *FuncCallGraph) {
	// Find calleeInfos for each function independently.
	callerToCalleesInfos := make(map[*ir.Func]calleesInfo, len(program.Funcs()))
	for _, caller := range program.Funcs() {
		callerToCalleesInfos[caller] = findCalleesInfoForBody(caller.Body(), callKinds, fcg)
	}

	// Process all SCCs in topological order (starting from entry):
	entrySCC := fcg.SCCOfFunc(program.EntryFunc())
	sccCallCounts := make(map[SCC]int)
	sccCallCounts[entrySCC] = 1
	for i := fcg.SCCCount() - 1; i > 0; i-- {
		currentSCC := SCC(i)
		currentSCCFuncs := fcg.FuncsInSCC(currentSCC)

		hasCallCylce := false
		if len(currentSCCFuncs) > 1 {
			hasCallCylce = true
		} else if f := currentSCCFuncs[0]; callerToCalleesInfos[f].funcCallCounts[f] > 0 {
			hasCallCylce = true
		}
		if hasCallCylce {
			sccCallCounts[currentSCC] = MaxCallCounts
		}

		for _, caller := range currentSCCFuncs {
			info := callerToCalleesInfos[caller]
			for callee, funcCallCount := range info.funcCallCounts {
				calleeSCC := fcg.SCCOfFunc(callee)
				sccCallCounts[calleeSCC] += funcCallCount * sccCallCounts[currentSCC]
				if sccCallCounts[calleeSCC] > MaxCallCounts {
					sccCallCounts[calleeSCC] = MaxCallCounts
				}
			}
			fcg.addCallCount(caller, sccCallCounts[currentSCC])
			fcg.addMakeChanCallCount(sccCallCounts[currentSCC] * info.makeChanCount)
		}
	}
}

type calleesInfo struct {
	funcCallCounts map[*ir.Func]int
	makeChanCount  int
}

func (info *calleesInfo) addFuncCallCount(f *ir.Func, count int) {
	info.funcCallCounts[f] += count
	if info.funcCallCounts[f] > MaxCallCounts {
		info.funcCallCounts[f] = MaxCallCounts
	}
}

func (info *calleesInfo) addMakeChanCallCount(count int) {
	info.makeChanCount += count
	if info.makeChanCount > MaxCallCounts {
		info.makeChanCount = MaxCallCounts
	}
}

func (info *calleesInfo) multiplyAllByFactor(factor int) {
	for f := range info.funcCallCounts {
		info.funcCallCounts[f] *= factor
		if info.funcCallCounts[f] > MaxCallCounts {
			info.funcCallCounts[f] = MaxCallCounts
		}
	}
	info.makeChanCount *= factor
	if info.makeChanCount > MaxCallCounts {
		info.makeChanCount = MaxCallCounts
	}
}

func (info *calleesInfo) addCalleesInfo(other calleesInfo) {
	for f, c := range other.funcCallCounts {
		info.addFuncCallCount(f, c)
	}
	info.addMakeChanCallCount(other.makeChanCount)
}

func (info *calleesInfo) mergeFromCalleesInfo(other calleesInfo) {
	for f, c := range other.funcCallCounts {
		if info.funcCallCounts[f] < c {
			info.funcCallCounts[f] = c
		}
	}
	if info.makeChanCount < other.makeChanCount {
		info.makeChanCount = other.makeChanCount
	}
}

func findCalleesInfoForBody(body *ir.Body, callKinds ir.CallKind, fcg *FuncCallGraph) (res calleesInfo) {
	res.funcCallCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0

	for _, stmt := range body.Stmts() {
		switch stmt := stmt.(type) {
		case *ir.CallStmt:
			if stmt.Kind()&callKinds == 0 {
				continue
			}
			switch callee := stmt.Callee().(type) {
			case *ir.Func:
				res.addFuncCallCount(callee, 1)
			case *ir.Variable:
				calleeSig := stmt.CalleeSignature()
				for _, dynCallee := range fcg.DynamicCallees(calleeSig) {
					res.addFuncCallCount(dynCallee, 1)
				}
			default:
				panic(fmt.Errorf("unexpected callee type: %T", callee))
			}
		case *ir.MakeChanStmt:
			res.addMakeChanCallCount(1)
		case *ir.IfStmt:
			res.addCalleesInfo(findCalleesInfoForIfStmt(stmt, callKinds, fcg))
		case *ir.SelectStmt:
			res.addCalleesInfo(findCalleesInfoForSelectStmt(stmt, callKinds, fcg))
		case *ir.ForStmt:
			res.addCalleesInfo(findCalleesInfoForForStmt(stmt, callKinds, fcg))
		case *ir.RangeStmt:
			res.addCalleesInfo(findCalleesInfoForRangeStmt(stmt, callKinds, fcg))
		case *ir.InlinedCallStmt:
			res.addCalleesInfo(findCalleesInfoForBody(stmt.Body(), callKinds, fcg))
		case *ir.AssignStmt, *ir.BranchStmt, *ir.ChanOpStmt, *ir.ReturnStmt:
			continue
		default:
			panic(fmt.Errorf("unexpected ir.Stmt type: %T", stmt))
		}
	}

	return
}

func findCalleesInfoForIfStmt(ifStmt *ir.IfStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res calleesInfo) {
	res.funcCallCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0
	res.mergeFromCalleesInfo(findCalleesInfoForBody(ifStmt.IfBranch(), callKinds, fcg))
	res.mergeFromCalleesInfo(findCalleesInfoForBody(ifStmt.ElseBranch(), callKinds, fcg))
	return
}

func findCalleesInfoForSelectStmt(selectStmt *ir.SelectStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res calleesInfo) {
	res.funcCallCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0
	for _, c := range selectStmt.Cases() {
		res.mergeFromCalleesInfo(findCalleesInfoForBody(c.Body(), callKinds, fcg))
	}
	if selectStmt.HasDefault() {
		res.mergeFromCalleesInfo(findCalleesInfoForBody(selectStmt.DefaultBody(), callKinds, fcg))
	}
	return
}

func findCalleesInfoForForStmt(forStmt *ir.ForStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res calleesInfo) {
	res.funcCallCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0
	f := MaxCallCounts
	if forStmt.HasMaxIterations() {
		f = forStmt.MaxIterations()
	}
	condRes := findCalleesInfoForBody(forStmt.Cond(), callKinds, fcg)
	condRes.multiplyAllByFactor(f + 1)
	bodyRes := findCalleesInfoForBody(forStmt.Body(), callKinds, fcg)
	bodyRes.multiplyAllByFactor(f)
	res.addCalleesInfo(condRes)
	res.addCalleesInfo(bodyRes)
	return
}

func findCalleesInfoForRangeStmt(rangeStmt *ir.RangeStmt, callKinds ir.CallKind, fcg *FuncCallGraph) calleesInfo {
	res := findCalleesInfoForBody(rangeStmt.Body(), callKinds, fcg)
	res.multiplyAllByFactor(MaxCallCounts)
	return res
}
