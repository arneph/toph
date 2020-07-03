package analyzer

import (
	"fmt"

	"github.com/arneph/toph/ir"
)

// BuildFuncCallGraph returns a new function call graph for the given program
// and call kind. Only calls of the given call kinds are contained in the
// graph.
func BuildFuncCallGraph(program *ir.Program, callKinds ir.CallKind) *FuncCallGraph {
	fcg := newFuncCallGraph(program.InitFunc())

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
			case ir.LValue:
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
	initSCC := fcg.SCCOfFunc(program.InitFunc())
	sccCallCounts := make(map[SCC]int)
	sccCallCounts[initSCC] = 1
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
			fcg.addCallerCount(caller, info.callCount)
			fcg.addCalleeCount(caller, sccCallCounts[currentSCC])
			for op, count := range info.specialOpCounts {
				fcg.addSpecialOpCount(caller, op, count)
			}
			for structType, count := range info.structAllocations {
				fcg.addStructAllocations(caller, structType, count)
			}
		}
	}
}

type callsInfo struct {
	callCount         int
	calleeCounts      map[*ir.Func]int
	specialOpCounts   map[ir.SpecialOp]int
	structAllocations map[*ir.StructType]int
}

func (info *callsInfo) init() {
	info.callCount = 0
	info.calleeCounts = make(map[*ir.Func]int)
	info.specialOpCounts = make(map[ir.SpecialOp]int)
	info.structAllocations = make(map[*ir.StructType]int)
}

func (info *callsInfo) enforceMaxCallCounts() {
	if info.callCount > MaxCallCounts {
		info.callCount = MaxCallCounts
	}
	for callee := range info.calleeCounts {
		if info.calleeCounts[callee] > MaxCallCounts {
			info.calleeCounts[callee] = MaxCallCounts
		}
	}
	for op := range info.specialOpCounts {
		if info.specialOpCounts[op] > MaxCallCounts {
			info.specialOpCounts[op] = MaxCallCounts
		}
	}
	for structType := range info.structAllocations {
		if info.structAllocations[structType] > MaxCallCounts {
			info.structAllocations[structType] = MaxCallCounts
		}
	}
}

func (info *callsInfo) addCallCount(count int) {
	info.callCount += count
	if info.callCount > MaxCallCounts {
		info.callCount = MaxCallCounts
	}
}

func (info *callsInfo) addCalleeCount(callee *ir.Func, count int) {
	info.calleeCounts[callee] += count
	if info.calleeCounts[callee] > MaxCallCounts {
		info.calleeCounts[callee] = MaxCallCounts
	}
}

func (info *callsInfo) addSpecialOpCount(op ir.SpecialOp, count int) {
	info.specialOpCounts[op] += count
	if info.specialOpCounts[op] > MaxCallCounts {
		info.specialOpCounts[op] = MaxCallCounts
	}
}

func (info *callsInfo) addStructAllocations(structType *ir.StructType, count int) {
	info.structAllocations[structType] += count
	if info.structAllocations[structType] > MaxCallCounts {
		info.structAllocations[structType] = MaxCallCounts
	}
}

func (info *callsInfo) multiply(factor int) {
	info.callCount *= factor
	for callee := range info.calleeCounts {
		info.calleeCounts[callee] *= factor
	}
	for op := range info.specialOpCounts {
		info.specialOpCounts[op] *= factor
	}
	for structType := range info.structAllocations {
		info.structAllocations[structType] *= factor
	}
	info.enforceMaxCallCounts()
}

func (info *callsInfo) add(other callsInfo) {
	info.callCount += other.callCount
	for callee, count := range other.calleeCounts {
		info.calleeCounts[callee] += count
	}
	for op, count := range other.specialOpCounts {
		info.specialOpCounts[op] += count
	}
	for structType, count := range other.structAllocations {
		info.structAllocations[structType] += count
	}
	info.enforceMaxCallCounts()
}

func (info *callsInfo) mergeFrom(other callsInfo) {
	if info.callCount < other.callCount {
		info.callCount = other.callCount
	}
	for f, count := range other.calleeCounts {
		if info.calleeCounts[f] < count {
			info.calleeCounts[f] = count
		}
	}
	for op, count := range other.specialOpCounts {
		if info.specialOpCounts[op] < count {
			info.specialOpCounts[op] = count
		}
	}
	for structType, count := range other.structAllocations {
		if info.structAllocations[structType] < count {
			info.structAllocations[structType] = count
		}
	}
}

func findCalleesInfoForBody(body *ir.Body, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.init()

	for _, stmt := range body.Stmts() {
		switch stmt := stmt.(type) {
		case *ir.AssignStmt:
			res.add(findCalleesInfoForAssignStmt(stmt))
		case *ir.CallStmt:
			res.add(findCalleesInfoForCallStmt(stmt, callKinds, fcg))
		case ir.SpecialOpStmt:
			res.addSpecialOpCount(stmt.SpecialOp(), 1)
		case *ir.MakeStructStmt:
			res.addStructAllocations(stmt.StructType(), 1)
		case *ir.IfStmt:
			res.add(findCalleesInfoForIfStmt(stmt, callKinds, fcg))
		case *ir.SwitchStmt:
			res.add(findCalleesInfoForSwitchStmt(stmt, callKinds, fcg))
		case *ir.SelectStmt:
			res.add(findCalleesInfoForSelectStmt(stmt, callKinds, fcg))
		case *ir.ForStmt:
			res.add(findCalleesInfoForForStmt(stmt, callKinds, fcg))
		case *ir.RangeStmt:
			res.add(findCalleesInfoForRangeStmt(stmt, callKinds, fcg))
		case *ir.BranchStmt, *ir.ChanCommOpStmt, *ir.ReturnStmt:
			continue
		default:
			panic(fmt.Errorf("unexpected ir.Stmt type: %T", stmt))
		}
	}

	return
}

func findCalleesInfoForAssignStmt(assignStmt *ir.AssignStmt) (res callsInfo) {
	structType, ok := assignStmt.Destination().Type().(*ir.StructType)
	if !ok {
		return
	}
	if !assignStmt.RequiresCopy() {
		return
	}
	return findCalleesForStructTypeCopy(structType)
}

func findCalleesInfoForCallStmt(callStmt *ir.CallStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.init()
	if callStmt.CallKind()&callKinds == 0 {
		return
	}
	res.addCallCount(1)
	switch callee := callStmt.Callee().(type) {
	case *ir.Func:
		res.addCalleeCount(callee, 1)
	case ir.LValue:
		calleeSig := callStmt.CalleeSignature()
		for _, dynCallee := range fcg.DynamicCallees(calleeSig) {
			res.addCalleeCount(dynCallee, 1)
		}
	default:
		panic(fmt.Errorf("unexpected callee type: %T", callee))
	}
	for i, argRV := range callStmt.Args() {
		argLV, ok := argRV.(ir.LValue)
		if !ok {
			continue
		}
		structType, ok := argLV.Type().(*ir.StructType)
		if !ok {
			continue
		}
		if !callStmt.ArgRequiresCopy(i) {
			continue
		}
		res.add(findCalleesForStructTypeCopy(structType))
	}
	for i, resLV := range callStmt.Results() {
		structType, ok := resLV.Type().(*ir.StructType)
		if !ok {
			continue
		}
		if !callStmt.ResultRequiresCopy(i) {
			continue
		}
		res.add(findCalleesForStructTypeCopy(structType))
	}
	return
}

func findCalleesForStructTypeCopy(structType *ir.StructType) (res callsInfo) {
	res.init()
	res.addStructAllocations(structType, 1)
	queue := []*ir.Field{}
	for _, field := range structType.Fields() {
		queue = append(queue, field)
	}
	for len(queue) > 0 {
		field := queue[0]
		queue = queue[1:]
		structType, ok := field.Type().(*ir.StructType)
		if !ok || field.IsPointer() {
			continue
		}
		res.addStructAllocations(structType, 1)
		for _, field := range structType.Fields() {
			queue = append(queue, field)
		}
	}
	return res
}

func findCalleesInfoForIfStmt(ifStmt *ir.IfStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.init()
	res.mergeFrom(findCalleesInfoForBody(ifStmt.IfBranch(), callKinds, fcg))
	res.mergeFrom(findCalleesInfoForBody(ifStmt.ElseBranch(), callKinds, fcg))
	return
}

func findCalleesInfoForSwitchStmt(switchStmt *ir.SwitchStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.init()

	bodyInfos := make([]callsInfo, len(switchStmt.Cases()))
	for i, switchCase := range switchStmt.Cases() {
		bodyInfos[i] = findCalleesInfoForBody(switchCase.Body(), callKinds, fcg)
	}

	var condInfo callsInfo
	condInfo.init()

	var defaultCase *ir.SwitchCase
	var defaultCaseIndex int

	for i, switchCase := range switchStmt.Cases() {
		if switchCase.IsDefault() {
			defaultCase = switchCase
			defaultCaseIndex = i
			continue
		}

		for _, cond := range switchCase.Conds() {
			condInfo.add(findCalleesInfoForBody(cond, callKinds, fcg))
		}

		var executeCaseInfo callsInfo
		executeCaseInfo.init()
		executeCaseInfo.add(condInfo)
		executeCaseInfo.add(bodyInfos[i])

		j := i
		for switchStmt.Cases()[j].HasFallthrough() {
			j++
			executeCaseInfo.add(bodyInfos[j])
		}

		res.mergeFrom(executeCaseInfo)
	}

	if defaultCase != nil {
		var executeDefaultInfo callsInfo
		executeDefaultInfo.init()
		executeDefaultInfo.add(condInfo)
		executeDefaultInfo.add(bodyInfos[defaultCaseIndex])

		j := defaultCaseIndex
		for switchStmt.Cases()[j].HasFallthrough() {
			j++
			executeDefaultInfo.add(bodyInfos[j])
		}

		res.mergeFrom(executeDefaultInfo)
	}

	return
}

func findCalleesInfoForSelectStmt(selectStmt *ir.SelectStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.init()
	for _, selectCase := range selectStmt.Cases() {
		res.mergeFrom(findCalleesInfoForBody(selectCase.Body(), callKinds, fcg))
	}
	if selectStmt.HasDefault() {
		res.mergeFrom(findCalleesInfoForBody(selectStmt.DefaultBody(), callKinds, fcg))
	}
	return
}

func findCalleesInfoForForStmt(forStmt *ir.ForStmt, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.init()
	f := MaxCallCounts
	if forStmt.HasMaxIterations() {
		f = forStmt.MaxIterations()
	}
	condRes := findCalleesInfoForBody(forStmt.Cond(), callKinds, fcg)
	condRes.multiply(f + 1)
	bodyRes := findCalleesInfoForBody(forStmt.Body(), callKinds, fcg)
	bodyRes.multiply(f)
	res.add(condRes)
	res.add(bodyRes)
	return
}

func findCalleesInfoForRangeStmt(rangeStmt *ir.RangeStmt, callKinds ir.CallKind, fcg *FuncCallGraph) callsInfo {
	res := findCalleesInfoForBody(rangeStmt.Body(), callKinds, fcg)
	res.multiply(MaxCallCounts)
	return res
}
