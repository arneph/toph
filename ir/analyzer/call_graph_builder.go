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
	addDynamicCalleesToFuncCallGraph(program, fcg)
	addCallCountsToFuncCallGraph(program, callKinds, fcg)
	if callKinds == ir.Call|ir.Defer|ir.Go {
		removeCallsToClosuresInsideUncalledFunctionsFromFuncCallGraph(program, fcg)
	}
	analyzePanics(program, fcg)

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
				fcg.addDynamicCaller(caller, calleeSig)
			default:
				panic(fmt.Errorf("unexpected callee type: %T", callee))
			}
		})
	}
}

func addDynamicCalleesToFuncCallGraph(program *ir.Program, fcg *FuncCallGraph) {
	queue := []*ir.Scope{program.Scope()}
	for len(queue) > 0 {
		scope := queue[0]
		queue = queue[1:]
		queue = append(queue, scope.Children()...)
		for _, v := range scope.Variables() {
			if v.Type() == ir.FuncType {
				addDynamicCalleeToFuncCallGraph(program, v.InitialValue(), fcg)
			}
		}
	}
	for _, f := range program.Funcs() {
		f.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
			switch stmt := stmt.(type) {
			case *ir.AssignStmt:
				addDynamicCalleeToFuncCallGraph(program, stmt.Source(), fcg)
			case *ir.CallStmt:
				for _, arg := range stmt.Args() {
					addDynamicCalleeToFuncCallGraph(program, arg, fcg)
				}
			case *ir.ReturnStmt:
				for _, result := range stmt.Results() {
					addDynamicCalleeToFuncCallGraph(program, result, fcg)
				}
			}
		})
	}
}

func addDynamicCalleeToFuncCallGraph(program *ir.Program, rvalue ir.RValue, fcg *FuncCallGraph) {
	calleeVal, ok := rvalue.(ir.Value)
	if !ok || calleeVal.Type() != ir.FuncType || calleeVal.Value() < 0 {
		return
	}
	callee := program.Func(ir.FuncIndex(calleeVal.Value()))
	fcg.addDynamicCallee(callee)
}

func addCallCountsToFuncCallGraph(program *ir.Program, callKinds ir.CallKind, fcg *FuncCallGraph) {
	// Find calleeInfos for each function independently.
	callerToCalleesInfos := make(map[*ir.Func]callsInfo, len(program.Funcs()))
	for _, caller := range program.Funcs() {
		res := findCalleesInfoForBody(caller.Body(), callKinds, fcg)
		if caller == program.InitFunc() {
			for _, v := range program.Scope().Variables() {
				if v.InitialValue() == v.Type().InitializedValue() {
					res.add(findCalleesForInitializedType(v.Type()))
				}
			}
		}
		callerToCalleesInfos[caller] = res
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
				fcg.addTotalSpecialOpCount(op, count*sccCallCounts[currentSCC])
			}
			for irType, count := range info.typeAllocations {
				fcg.addTypeAllocations(caller, irType, count)
				fcg.addTotalTypeAllocations(irType, count*sccCallCounts[currentSCC])
			}
		}
	}
}

func removeCallsToClosuresInsideUncalledFunctionsFromFuncCallGraph(program *ir.Program, fcg *FuncCallGraph) {
	for _, f := range program.Funcs() {
		g := f.EnclosingFunc()
		if g == nil || fcg.CalleeCount(g) > 0 {
			continue
		}
		fcg.zeroCalleeCounts(f)
	}
}

func analyzePanics(program *ir.Program, fcg *FuncCallGraph) {
	for _, f := range program.Funcs() {
		fcg.canPanicInternally[f] = canPanicInternally(f)
		fcg.canRecover[f] = canRecover(f)
	}
	for i := 1; i < fcg.SCCCount(); i++ {
		sccFuncs := fcg.FuncsInSCC(SCC(i))
		canPanicInSCC := false
		for _, caller := range sccFuncs {
			canPanicExternally := false
			for _, callee := range fcg.AllCallees(caller) {
				if fcg.canPanicInternally[callee] ||
					fcg.canPanicExternally[callee] {
					canPanicExternally = true
					break
				}
			}
			fcg.canPanicExternally[caller] = canPanicExternally
			if fcg.canPanicInternally[caller] ||
				fcg.canPanicExternally[caller] {
				canPanicInSCC = true
			}
		}
		if len(sccFuncs) > 1 && canPanicInSCC {
			for _, f := range sccFuncs {
				fcg.canPanicExternally[f] = true
			}
		}
	}
}

func canPanicInternally(f *ir.Func) (canPanic bool) {
	f.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
		returnStmt, ok := stmt.(*ir.ReturnStmt)
		if ok && returnStmt.IsPanic() {
			canPanic = true
		}
	})
	return
}

func canRecover(f *ir.Func) (canRecover bool) {
	f.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
		if _, ok := stmt.(*ir.RecoverStmt); ok {
			canRecover = true
		}
	})
	return
}

type callsInfo struct {
	callCount       int
	calleeCounts    map[*ir.Func]int
	specialOpCounts map[ir.SpecialOp]int
	typeAllocations map[ir.Type]int
}

func (info *callsInfo) init() {
	info.callCount = 0
	info.calleeCounts = make(map[*ir.Func]int)
	info.specialOpCounts = make(map[ir.SpecialOp]int)
	info.typeAllocations = make(map[ir.Type]int)
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
	for irType := range info.typeAllocations {
		if info.typeAllocations[irType] > MaxCallCounts {
			info.typeAllocations[irType] = MaxCallCounts
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

func (info *callsInfo) addTypeAllocations(irType ir.Type, count int) {
	info.typeAllocations[irType] += count
	if info.typeAllocations[irType] > MaxCallCounts {
		info.typeAllocations[irType] = MaxCallCounts
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
	for irType := range info.typeAllocations {
		info.typeAllocations[irType] *= factor
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
	for irType, count := range other.typeAllocations {
		info.typeAllocations[irType] += count
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
	for irType, count := range other.typeAllocations {
		if info.typeAllocations[irType] < count {
			info.typeAllocations[irType] = count
		}
	}
}

func findCalleesInfoForBody(body *ir.Body, callKinds ir.CallKind, fcg *FuncCallGraph) (res callsInfo) {
	res.init()

	for _, v := range body.Scope().Variables() {
		if v.InitialValue() == v.Type().InitializedValue() {
			res.add(findCalleesForInitializedType(v.Type()))
		}
	}

	for _, stmt := range body.Stmts() {
		switch stmt := stmt.(type) {
		case *ir.AssignStmt:
			res.add(findCalleesInfoForAssignStmt(stmt))
		case *ir.CallStmt:
			res.add(findCalleesInfoForCallStmt(stmt, callKinds, fcg))
		case ir.SpecialOpStmt:
			res.addSpecialOpCount(stmt.SpecialOp(), 1)
		case *ir.MakeStructStmt:
			res.addTypeAllocations(stmt.StructType(), 1)
		case *ir.MakeContainerStmt:
			res.addTypeAllocations(stmt.ContainerType(), 1)
		case *ir.IfStmt:
			res.add(findCalleesInfoForIfStmt(stmt, callKinds, fcg))
		case *ir.SwitchStmt:
			res.add(findCalleesInfoForSwitchStmt(stmt, callKinds, fcg))
		case *ir.SelectStmt:
			res.add(findCalleesInfoForSelectStmt(stmt, callKinds, fcg))
		case *ir.ForStmt:
			res.add(findCalleesInfoForForStmt(stmt, callKinds, fcg))
		case *ir.ChanRangeStmt:
			res.add(findCalleesInfoForChanRangeStmt(stmt, callKinds, fcg))
		case *ir.ContainerRangeStmt:
			res.add(findCalleesInfoForContainerRangeStmt(stmt, callKinds, fcg))
		case *ir.BranchStmt, *ir.ChanCommOpStmt, *ir.ReturnStmt, *ir.RecoverStmt:
			continue
		default:
			panic(fmt.Errorf("unexpected ir.Stmt type: %T", stmt))
		}
	}

	return
}

func findCalleesInfoForAssignStmt(assignStmt *ir.AssignStmt) (res callsInfo) {
	containerAccess, ok := assignStmt.Source().(*ir.ContainerAccess)
	if ok && containerAccess.IsMapRead() && containerAccess.Index() == ir.RandomIndex {
		return findCalleesForInitializedType(containerAccess.ContainerType().ElementType())
	}
	if !assignStmt.RequiresCopy() {
		return
	}
	return findCalleesForTypeCopy(assignStmt.Destination().Type())
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
		if !callStmt.ArgRequiresCopy(i) {
			continue
		}
		res.add(findCalleesForTypeCopy(argLV.Type()))
	}
	for i, resLV := range callStmt.Results() {
		if !callStmt.ResultRequiresCopy(i) {
			continue
		}
		res.add(findCalleesForTypeCopy(resLV.Type()))
	}
	return
}

func findCalleesForInitializedType(irType ir.Type) (res callsInfo) {
	res.init()
	switch irType := irType.(type) {
	case ir.BasicType:
		if irType == ir.MutexType || irType == ir.WaitGroupType {
			res.addTypeAllocations(irType, 1)
		}
	case *ir.StructType:
		res.addTypeAllocations(irType, 1)
		for _, field := range irType.Fields() {
			if field.IsPointer() {
				continue
			}
			res.add(findCalleesForInitializedType(field.Type()))
		}
	case *ir.ContainerType:
		if irType.Kind() != ir.Array {
			break
		}
		res.addTypeAllocations(irType, 1)
		if irType.HoldsPointers() {
			elemRes := findCalleesForInitializedType(irType.ElementType())
			elemRes.multiply(irType.Len())

			res.add(elemRes)
		}
	}
	return res
}

func findCalleesForTypeCopy(irType ir.Type) (res callsInfo) {
	switch irType := irType.(type) {
	case *ir.StructType:
		return findCalleesForStructTypeCopy(irType)
	case *ir.ContainerType:
		return findCalleesForContainerTypeCopy(irType)
	default:
		return
	}
}

func findCalleesForStructTypeCopy(structType *ir.StructType) (res callsInfo) {
	res.init()
	res.addTypeAllocations(structType, 1)
	for _, field := range structType.Fields() {
		if field.RequiresDeepCopy() {
			res.add(findCalleesForTypeCopy(field.Type()))
		}
	}
	return res
}

func findCalleesForContainerTypeCopy(containerType *ir.ContainerType) (res callsInfo) {
	res.init()
	res.addTypeAllocations(containerType, 1)
	if containerType.RequiresDeepCopies() {
		subRes := findCalleesForTypeCopy(containerType.ElementType())
		if containerType.Kind() == ir.Array {
			subRes.multiply(containerType.Len())
		} else {
			subRes.multiply(MaxCallCounts)
		}
		res.add(subRes)
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

func findCalleesInfoForChanRangeStmt(rangeStmt *ir.ChanRangeStmt, callKinds ir.CallKind, fcg *FuncCallGraph) callsInfo {
	res := findCalleesInfoForBody(rangeStmt.Body(), callKinds, fcg)
	res.multiply(MaxCallCounts)
	return res
}

func findCalleesInfoForContainerRangeStmt(rangeStmt *ir.ContainerRangeStmt, callKinds ir.CallKind, fcg *FuncCallGraph) callsInfo {
	res := findCalleesInfoForBody(rangeStmt.Body(), callKinds, fcg)
	containerType := rangeStmt.Container().Type().(*ir.ContainerType)
	switch containerType.Kind() {
	case ir.Array:
		res.multiply(containerType.Len())
	case ir.Slice, ir.Map:
		res.multiply(MaxCallCounts)
	default:
		panic("unexpected container kind")
	}
	return res
}
