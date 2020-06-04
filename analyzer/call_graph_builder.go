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
			if !ok || callStmt.CallKind()&callKinds == 0 {
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
	callerToCalleesInfos := make(map[*ir.Func]callsInfo, len(program.Funcs()))
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
		} else if f := currentSCCFuncs[0]; callerToCalleesInfos[f].calleeCounts[f] > 0 {
			hasCallCylce = true
		}
		if hasCallCylce {
			sccCallCounts[currentSCC] = MaxCallCounts
		}

		for _, caller := range currentSCCFuncs {
			info := callerToCalleesInfos[caller]
			for callee, calleeCount := range info.calleeCounts {
				calleeSCC := fcg.SCCOfFunc(callee)
				sccCallCounts[calleeSCC] += calleeCount * sccCallCounts[currentSCC]
				if sccCallCounts[calleeSCC] > MaxCallCounts {
					sccCallCounts[calleeSCC] = MaxCallCounts
				}
			}
			fcg.addCallerCount(caller, info.callerCount)
			fcg.addCalleeCount(caller, sccCallCounts[currentSCC])
			fcg.addMakeChanCallCount(caller, info.makeChanCount)
			fcg.addCloseChanCallCount(caller, info.closeChanCount)
			fcg.addTotalMakeChanCallCount(sccCallCounts[currentSCC] * info.makeChanCount)
			fcg.addTotalCloseChanCallCount(sccCallCounts[currentSCC] * info.closeChanCount)
		}
	}
}

type callsInfo struct {
	callerCount    int
	calleeCounts   map[*ir.Func]int
	makeChanCount  int
	closeChanCount int
}

func (info *callsInfo) addCallerCount(count int) {
	info.callerCount += count
	if info.callerCount > MaxCallCounts {
		info.callerCount = MaxCallCounts
	}
}

func (info *callsInfo) addCalleeCount(f *ir.Func, count int) {
	info.calleeCounts[f] += count
	if info.calleeCounts[f] > MaxCallCounts {
		info.calleeCounts[f] = MaxCallCounts
	}
}

func (info *callsInfo) addMakeChanCallCount(count int) {
	info.makeChanCount += count
	if info.makeChanCount > MaxCallCounts {
		info.makeChanCount = MaxCallCounts
	}
}

func (info *callsInfo) addCloseChanCallCount(count int) {
	info.closeChanCount += count
	if info.closeChanCount > MaxCallCounts {
		info.closeChanCount = MaxCallCounts
	}
}

func (info *callsInfo) multiplyAllByFactor(factor int) {
	info.callerCount *= factor
	if info.callerCount > MaxCallCounts {
		info.callerCount = MaxCallCounts
	}
	for f := range info.calleeCounts {
		info.calleeCounts[f] *= factor
		if info.calleeCounts[f] > MaxCallCounts {
			info.calleeCounts[f] = MaxCallCounts
		}
	}
	info.makeChanCount *= factor
	if info.makeChanCount > MaxCallCounts {
		info.makeChanCount = MaxCallCounts
	}
	info.closeChanCount *= factor
	if info.closeChanCount > MaxCallCounts {
		info.closeChanCount = MaxCallCounts
	}
}

func (info *callsInfo) addCalleesInfo(other callsInfo) {
	info.addCallerCount(other.callerCount)
	for f, c := range other.calleeCounts {
		info.addCalleeCount(f, c)
	}
	info.addMakeChanCallCount(other.makeChanCount)
	info.addCloseChanCallCount(other.closeChanCount)
}

func (info *callsInfo) mergeFromCalleesInfo(other callsInfo) {
	if info.callerCount < other.callerCount {
		info.callerCount = other.callerCount
	}
	for f, c := range other.calleeCounts {
		if info.calleeCounts[f] < c {
			info.calleeCounts[f] = c
		}
	}
	if info.makeChanCount < other.makeChanCount {
		info.makeChanCount = other.makeChanCount
	}
	if info.closeChanCount < other.closeChanCount {
		info.closeChanCount = other.closeChanCount
	}
}

func findCalleesInfoForBody(body *ir.Body, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.callerCount = 0
	res.calleeCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0
	res.closeChanCount = 0

	for _, stmt := range body.Stmts() {
		switch stmt := stmt.(type) {
		case *ir.CallStmt:
			if stmt.CallKind()&callKinds == 0 {
				continue
			}
			res.addCallerCount(1)
			switch callee := stmt.Callee().(type) {
			case *ir.Func:
				res.addCalleeCount(callee, 1)
			case *ir.Variable:
				calleeSig := stmt.CalleeSignature()
				for _, dynCallee := range fcg.DynamicCallees(calleeSig) {
					res.addCalleeCount(dynCallee, 1)
				}
			default:
				panic(fmt.Errorf("unexpected callee type: %T", callee))
			}
		case *ir.MakeChanStmt:
			res.addMakeChanCallCount(1)
		case *ir.CloseChanStmt:
			if stmt.CallKind()&callKinds == 0 {
				continue
			}
			res.addCloseChanCallCount(1)
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

func findCalleesInfoForIfStmt(ifStmt *ir.IfStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.callerCount = 0
	res.calleeCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0
	res.closeChanCount = 0
	res.mergeFromCalleesInfo(findCalleesInfoForBody(ifStmt.IfBranch(), callKinds, fcg))
	res.mergeFromCalleesInfo(findCalleesInfoForBody(ifStmt.ElseBranch(), callKinds, fcg))
	return
}

func findCalleesInfoForSelectStmt(selectStmt *ir.SelectStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.callerCount = 0
	res.calleeCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0
	res.closeChanCount = 0
	for _, c := range selectStmt.Cases() {
		res.mergeFromCalleesInfo(findCalleesInfoForBody(c.Body(), callKinds, fcg))
	}
	if selectStmt.HasDefault() {
		res.mergeFromCalleesInfo(findCalleesInfoForBody(selectStmt.DefaultBody(), callKinds, fcg))
	}
	return
}

func findCalleesInfoForForStmt(forStmt *ir.ForStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.callerCount = 0
	res.calleeCounts = make(map[*ir.Func]int)
	res.makeChanCount = 0
	res.closeChanCount = 0
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

func findCalleesInfoForRangeStmt(rangeStmt *ir.RangeStmt, callKinds ir.CallKind, fcg *FuncCallGraph) callsInfo {
	res := findCalleesInfoForBody(rangeStmt.Body(), callKinds, fcg)
	res.multiplyAllByFactor(MaxCallCounts)
	return res
}
