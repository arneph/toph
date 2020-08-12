package analyzer

import (
	"fmt"
	"go/types"

	c "github.com/arneph/toph/config"
	"github.com/arneph/toph/ir"
)

// BuildFuncCallGraph returns a new function call graph for the given program
// and call kind. Only calls of the given call kinds are contained in the
// graph.
func BuildFuncCallGraph(program *ir.Program, callKinds ir.CallKind, config *c.Config) *FuncCallGraph {
	b := new(callGraphBuilder)
	b.program = program
	b.callKinds = callKinds
	b.fcg = newFuncCallGraph(program.InitFunc())
	b.config = config

	b.addFuncsToFuncCallGraph()
	b.addCallsToFuncCallGraph()
	b.addDynamicCalleesToFuncCallGraph()
	b.addCallCountsToFuncCallGraph()
	if callKinds == ir.Call|ir.Defer|ir.Go {
		b.removeCallsToClosuresInsideUncalledFunctionsFromFuncCallGraph()
	}
	b.analyzePanics()

	return b.fcg
}

type callGraphBuilder struct {
	program   *ir.Program
	callKinds ir.CallKind
	fcg       *FuncCallGraph
	config    *c.Config
}

func (b *callGraphBuilder) addFuncsToFuncCallGraph() {
	for _, f := range b.program.Funcs() {
		b.fcg.addFunc(f)
	}
}

func (b *callGraphBuilder) addCallsToFuncCallGraph() {
	for _, caller := range b.program.Funcs() {
		caller.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
			switch stmt := stmt.(type) {
			case *ir.CallStmt:
				if stmt.CallKind()&b.callKinds == 0 {
					return
				}
				switch callee := stmt.Callee().(type) {
				case *ir.Func:
					b.fcg.addStaticCall(caller, callee)
				case ir.LValue:
					calleeSig := stmt.CalleeSignature()
					b.fcg.addDynamicCaller(caller, calleeSig)
				default:
					panic(fmt.Errorf("unexpected callee type: %T", callee))
				}
			case *ir.OnceDoStmt:
				if ir.Call&b.callKinds == 0 {
					return
				}
				switch callee := stmt.F().(type) {
				case ir.Value:
					b.fcg.addStaticCall(caller, b.program.Func(ir.FuncIndex(callee.Value())))
				case ir.LValue:
					calleeSig := types.NewSignature(nil, nil, nil, false)
					b.fcg.addDynamicCaller(caller, calleeSig)
				default:
					panic(fmt.Errorf("unexpected callee type: %T", callee))
				}
			}
		})
	}
}

func (b *callGraphBuilder) addDynamicCalleesToFuncCallGraph() {
	queue := []*ir.Scope{b.program.Scope()}
	for len(queue) > 0 {
		scope := queue[0]
		queue = queue[1:]
		queue = append(queue, scope.Children()...)
		for _, v := range scope.Variables() {
			if v.Type() == ir.FuncType {
				b.addDynamicCalleeToFuncCallGraph(v.InitialValue())
			}
		}
	}
	for _, f := range b.program.Funcs() {
		f.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
			switch stmt := stmt.(type) {
			case *ir.AssignStmt:
				b.addDynamicCalleeToFuncCallGraph(stmt.Source())
			case *ir.CallStmt:
				for _, arg := range stmt.Args() {
					b.addDynamicCalleeToFuncCallGraph(arg)
				}
			case *ir.ReturnStmt:
				for _, result := range stmt.Results() {
					b.addDynamicCalleeToFuncCallGraph(result)
				}
			}
		})
	}
}

func (b *callGraphBuilder) addDynamicCalleeToFuncCallGraph(rvalue ir.RValue) {
	calleeVal, ok := rvalue.(ir.Value)
	if !ok || calleeVal.Type() != ir.FuncType || calleeVal.Value() < 0 {
		return
	}
	callee := b.program.Func(ir.FuncIndex(calleeVal.Value()))
	b.fcg.addDynamicCallee(callee)
}

func (b *callGraphBuilder) addCallCountsToFuncCallGraph() {
	// Find calleeInfos for each function independently.
	callerToCalleesInfos := make(map[*ir.Func]callsInfo, len(b.program.Funcs()))
	for _, caller := range b.program.Funcs() {
		res := b.findCalleesInfoForBody(caller.Body())
		if caller == b.program.InitFunc() {
			for _, v := range b.program.Scope().Variables() {
				if v.InitialValue() == v.Type().InitializedValue() {
					res.add(b.findCalleesForInitializedType(v.Type()))
				}
			}
		}
		callerToCalleesInfos[caller] = res
	}

	// Process all SCCs in topological order (starting from entry):
	initSCC := b.fcg.SCCOfFunc(b.program.InitFunc())
	sccCallCounts := make(map[SCC]int)
	sccCallCounts[initSCC] = 1
	for i := b.fcg.SCCCount() - 1; i > 0; i-- {
		currentSCC := SCC(i)
		currentSCCFuncs := b.fcg.FuncsInSCC(currentSCC)

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
				calleeSCC := b.fcg.SCCOfFunc(callee)
				sccCallCounts[calleeSCC] += calleeCount * sccCallCounts[currentSCC]
				if sccCallCounts[calleeSCC] > MaxCallCounts {
					sccCallCounts[calleeSCC] = MaxCallCounts
				}
			}
			b.fcg.addCallerCount(caller, info.callCount)
			b.fcg.addCalleeCount(caller, sccCallCounts[currentSCC])
			for op, count := range info.specialOpCounts {
				b.fcg.addSpecialOpCount(caller, op, count)
				b.fcg.addTotalSpecialOpCount(op, count*sccCallCounts[currentSCC])
			}
			for irType, count := range info.typeAllocations {
				b.fcg.addTypeAllocations(caller, irType, count)
				b.fcg.addTotalTypeAllocations(irType, count*sccCallCounts[currentSCC])
			}
		}
	}
}

func (b *callGraphBuilder) removeCallsToClosuresInsideUncalledFunctionsFromFuncCallGraph() {
	for _, f := range b.program.Funcs() {
		g := f.EnclosingFunc()
		if g == nil || b.fcg.CalleeCount(g) > 0 {
			continue
		}
		b.fcg.zeroCalleeCounts(f)
	}
}

func (b *callGraphBuilder) analyzePanics() {
	for _, f := range b.program.Funcs() {
		b.fcg.canPanicInternally[f] = b.canPanicInternally(f)
		b.fcg.canRecover[f] = b.canRecover(f)
	}
	for i := 1; i < b.fcg.SCCCount(); i++ {
		sccFuncs := b.fcg.FuncsInSCC(SCC(i))
		canPanicInSCC := false
		for _, caller := range sccFuncs {
			canPanicExternally := false
			for _, callee := range b.fcg.AllCallees(caller) {
				if b.fcg.canPanicInternally[callee] ||
					b.fcg.canPanicExternally[callee] {
					canPanicExternally = true
					break
				}
			}
			b.fcg.canPanicExternally[caller] = canPanicExternally
			if b.fcg.canPanicInternally[caller] ||
				b.fcg.canPanicExternally[caller] {
				canPanicInSCC = true
			}
		}
		if len(sccFuncs) > 1 && canPanicInSCC {
			for _, f := range sccFuncs {
				b.fcg.canPanicExternally[f] = true
			}
		}
	}
}

func (b *callGraphBuilder) canPanicInternally(f *ir.Func) (canPanic bool) {
	f.Body().WalkStmts(func(stmt ir.Stmt, scope *ir.Scope) {
		returnStmt, ok := stmt.(*ir.ReturnStmt)
		if ok && returnStmt.IsPanic() {
			canPanic = true
		}
	})
	return
}

func (b *callGraphBuilder) canRecover(f *ir.Func) (canRecover bool) {
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

func (b *callGraphBuilder) findCalleesInfoForBody(body *ir.Body) (res callsInfo) {
	res.init()

	for _, v := range body.Scope().Variables() {
		if v.InitialValue() == v.Type().InitializedValue() {
			res.add(b.findCalleesForInitializedType(v.Type()))
		}
	}

	for _, stmt := range body.Stmts() {
		switch stmt := stmt.(type) {
		case *ir.AssignStmt:
			res.add(b.findCalleesInfoForAssignStmt(stmt))
		case *ir.CallStmt:
			res.add(b.findCalleesInfoForCallStmt(stmt))
		case *ir.OnceDoStmt:
			res.addSpecialOpCount(ir.Do, 1)
			if ir.Call&b.callKinds == 0 {
				continue
			}
			res.addCallCount(1)
			switch callee := stmt.F().(type) {
			case ir.Value:
				res.addCalleeCount(b.program.Func(ir.FuncIndex(callee.Value())), 1)
			case ir.LValue:
				calleeSig := types.NewSignature(nil, nil, nil, false)
				for _, dynCallee := range b.fcg.DynamicCallees(calleeSig) {
					res.addCalleeCount(dynCallee, 1)
				}
			default:
				panic(fmt.Errorf("unexpected callee type: %T", callee))
			}
		case ir.SpecialOpStmt:
			res.addSpecialOpCount(stmt.SpecialOp(), 1)
		case *ir.MakeStructStmt:
			if stmt.InitialzeFields() {
				res.add(b.findCalleesForInitializedType(stmt.StructType()))
			} else {
				res.addTypeAllocations(stmt.StructType(), 1)
			}
		case *ir.MakeContainerStmt:
			if stmt.InitializeElements() {
				res.add(b.findCalleesForInitializedType(stmt.ContainerType()))
			} else {
				res.addTypeAllocations(stmt.ContainerType(), 1)
			}
		case *ir.IfStmt:
			res.add(b.findCalleesInfoForIfStmt(stmt))
		case *ir.SwitchStmt:
			res.add(b.findCalleesInfoForSwitchStmt(stmt))
		case *ir.SelectStmt:
			res.add(b.findCalleesInfoForSelectStmt(stmt))
		case *ir.ForStmt:
			res.add(b.findCalleesInfoForForStmt(stmt))
		case *ir.ChanRangeStmt:
			res.add(b.findCalleesInfoForChanRangeStmt(stmt))
		case *ir.ContainerRangeStmt:
			res.add(b.findCalleesInfoForContainerRangeStmt(stmt))
		case *ir.BranchStmt, *ir.ChanCommOpStmt, *ir.DeleteMapEntryStmt, *ir.ReturnStmt, *ir.RecoverStmt:
			continue
		default:
			panic(fmt.Errorf("unexpected ir.Stmt type: %T", stmt))
		}
	}

	return
}

func (b *callGraphBuilder) findCalleesInfoForAssignStmt(assignStmt *ir.AssignStmt) (res callsInfo) {
	res.init()
	containerAccess, ok := assignStmt.Source().(*ir.ContainerAccess)
	if ok && containerAccess.IsMapRead() && containerAccess.Index() == ir.RandomIndex {
		res.add(b.findCalleesForInitializedType(containerAccess.ContainerType().ElementType()))
		return
	}
	v, ok := assignStmt.Source().(ir.Value)
	if ok && v.IsInitializedStruct() {
		structType := v.Type().(*ir.StructType)
		res.add(b.findCalleesForInitializedType(structType))
	} else if ok && v.IsInitializedArray() {
		arrayType := v.Type().(*ir.ContainerType)
		res.add(b.findCalleesForInitializedType(arrayType))
	} else if ok && v == ir.InitializedMutex {
		res.addTypeAllocations(ir.MutexType, 1)
	} else if ok && v == ir.InitializedWaitGroup {
		res.addTypeAllocations(ir.WaitGroupType, 1)
	}
	if assignStmt.RequiresCopy() {
		res.add(b.findCalleesForTypeCopy(assignStmt.Destination().Type()))
	}
	return
}

func (b *callGraphBuilder) findCalleesInfoForCallStmt(callStmt *ir.CallStmt) (res callsInfo) {
	res.init()
	if callStmt.CallKind()&b.callKinds == 0 {
		return
	}
	res.addCallCount(1)
	switch callee := callStmt.Callee().(type) {
	case *ir.Func:
		res.addCalleeCount(callee, 1)
	case ir.LValue:
		calleeSig := callStmt.CalleeSignature()
		for _, dynCallee := range b.fcg.DynamicCallees(calleeSig) {
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
		res.add(b.findCalleesForTypeCopy(argLV.Type()))
	}
	for i, resLV := range callStmt.Results() {
		if !callStmt.ResultRequiresCopy(i) {
			continue
		}
		res.add(b.findCalleesForTypeCopy(resLV.Type()))
	}
	return
}

func (b *callGraphBuilder) findCalleesForInitializedType(irType ir.Type) (res callsInfo) {
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
			res.add(b.findCalleesForInitializedType(field.Type()))
		}
	case *ir.ContainerType:
		if irType.Kind() != ir.Array {
			break
		}
		res.addTypeAllocations(irType, 1)
		if !irType.HoldsPointers() {
			elemRes := b.findCalleesForInitializedType(irType.ElementType())
			elemRes.multiply(irType.Len())

			res.add(elemRes)
		}
	}
	return res
}

func (b *callGraphBuilder) findCalleesForTypeCopy(irType ir.Type) (res callsInfo) {
	switch irType := irType.(type) {
	case *ir.StructType:
		return b.findCalleesForStructTypeCopy(irType)
	case *ir.ContainerType:
		return b.findCalleesForContainerTypeCopy(irType)
	default:
		return
	}
}

func (b *callGraphBuilder) findCalleesForStructTypeCopy(structType *ir.StructType) (res callsInfo) {
	res.init()
	res.addTypeAllocations(structType, 1)
	for _, field := range structType.Fields() {
		if field.RequiresDeepCopy() {
			res.add(b.findCalleesForTypeCopy(field.Type()))
		}
	}
	return res
}

func (b *callGraphBuilder) findCalleesForContainerTypeCopy(containerType *ir.ContainerType) (res callsInfo) {
	res.init()
	res.addTypeAllocations(containerType, 1)
	if containerType.RequiresDeepCopies() {
		subRes := b.findCalleesForTypeCopy(containerType.ElementType())
		if containerType.Kind() == ir.Array {
			subRes.multiply(containerType.Len())
		} else {
			subRes.multiply(b.config.ContainerCapacity)
		}
		res.add(subRes)
	}
	return res
}

func (b *callGraphBuilder) findCalleesInfoForIfStmt(ifStmt *ir.IfStmt) (res callsInfo) {
	res.init()
	res.mergeFrom(b.findCalleesInfoForBody(ifStmt.IfBranch()))
	res.mergeFrom(b.findCalleesInfoForBody(ifStmt.ElseBranch()))
	return
}

func (b *callGraphBuilder) findCalleesInfoForSwitchStmt(switchStmt *ir.SwitchStmt) (res callsInfo) {
	res.init()

	bodyInfos := make([]callsInfo, len(switchStmt.Cases()))
	for i, switchCase := range switchStmt.Cases() {
		bodyInfos[i] = b.findCalleesInfoForBody(switchCase.Body())
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
			condInfo.add(b.findCalleesInfoForBody(cond))
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

func (b *callGraphBuilder) findCalleesInfoForSelectStmt(selectStmt *ir.SelectStmt) (res callsInfo) {
	res.init()
	for _, selectCase := range selectStmt.Cases() {
		res.mergeFrom(b.findCalleesInfoForBody(selectCase.Body()))
	}
	if selectStmt.HasDefault() {
		res.mergeFrom(b.findCalleesInfoForBody(selectStmt.DefaultBody()))
	}
	return
}

func (b *callGraphBuilder) findCalleesInfoForForStmt(forStmt *ir.ForStmt) (res callsInfo) {
	res.init()
	f := MaxCallCounts
	if forStmt.HasMaxIterations() {
		f = forStmt.MaxIterations()
	}
	condRes := b.findCalleesInfoForBody(forStmt.Cond())
	condRes.multiply(f + 1)
	bodyRes := b.findCalleesInfoForBody(forStmt.Body())
	bodyRes.multiply(f)
	res.add(condRes)
	res.add(bodyRes)
	return
}

func (b *callGraphBuilder) findCalleesInfoForChanRangeStmt(rangeStmt *ir.ChanRangeStmt) callsInfo {
	res := b.findCalleesInfoForBody(rangeStmt.Body())
	res.multiply(MaxCallCounts)
	return res
}

func (b *callGraphBuilder) findCalleesInfoForContainerRangeStmt(rangeStmt *ir.ContainerRangeStmt) callsInfo {
	res := b.findCalleesInfoForBody(rangeStmt.Body())
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
