package builder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/arneph/toph/ir"
	"golang.org/x/tools/go/ast/astutil"
)

func (b *builder) findCallee(funcExpr ast.Expr, ctx *context) (callee ir.Callable, calleeSignature *types.Signature) {
	funcExpr = astutil.Unparen(funcExpr)
	var calleeValue ir.RValue
	var calleeTypesType types.Type
	if selExpr, ok := funcExpr.(*ast.SelectorExpr); ok {
		typesSelection, ok := ctx.typesInfo.Selections[selExpr]
		if ok && typesSelection.Kind() == types.MethodVal {
			calleeValue = b.processIdent(selExpr.Sel, ctx)
			calleeTypesType = ctx.typesInfo.TypeOf(selExpr.Sel)
		} else {
			calleeValue = b.processSelectorExpr(selExpr, ctx)
			calleeTypesType = ctx.typesInfo.TypeOf(selExpr)
		}
	} else {
		calleeValue = b.processExpr(funcExpr, ctx)
		calleeTypesType = ctx.typesInfo.TypeOf(funcExpr)
	}
	if calleeValue == nil {
		funcExprTypesType := ctx.typesInfo.TypeOf(funcExpr).Underlying()
		switch funcExprTypesType.(type) {
		case *types.Array, *types.Basic, *types.Chan, *types.Interface, *types.Map, *types.Pointer, *types.Slice:
		default:
			p := b.fset.Position(funcExpr.Pos())
			funcExprStr := b.nodeToString(funcExpr)
			b.addWarning(fmt.Errorf("%v: could not resolve func expr: %v", p, funcExprStr))
		}
		return nil, nil
	}

	switch calleeValue := calleeValue.(type) {
	case *ir.Variable:
		callee = calleeValue
	case *ir.FieldSelection:
		callee = calleeValue
	case *ir.ContainerAccess:
		callee = calleeValue
	case ir.Value:
		callee = b.program.Func(ir.FuncIndex(calleeValue.Value()))
		if callee == nil {
			p := b.fset.Position(funcExpr.Pos())
			funcExprStr := b.nodeToString(funcExpr)
			b.addWarning(fmt.Errorf("%v: could not resolve func index in ir.Program: %v", p, funcExprStr))
			return nil, nil
		}
	}

	calleeSignature = calleeTypesType.Underlying().(*types.Signature)

	return callee, calleeSignature
}

func (b *builder) processCallExpr(callExpr *ast.CallExpr, ctx *context) map[int]*ir.Variable {
	return b.processCallExprWithCallKind(callExpr, ir.Call, ctx)
}

func (b *builder) processDeferStmt(stmt *ast.DeferStmt, ctx *context) {
	b.processCallExprWithCallKind(stmt.Call, ir.Defer, ctx)
}

func (b *builder) processGoStmt(stmt *ast.GoStmt, ctx *context) {
	b.processCallExprWithCallKind(stmt.Call, ir.Go, ctx)
}

func (b *builder) processReturnStmt(stmt *ast.ReturnStmt, ctx *context) {
	resultVals := b.processExprs(stmt.Results, ctx)

	returnStmt := ir.NewReturnStmt(false, stmt.Pos(), stmt.End())
	ctx.body.AddStmt(returnStmt)

	if len(stmt.Results) > 0 {
		// return stmt returns specified values
		for i, v := range resultVals {
			if t, ok := ctx.currentFunc().ResultTypes()[i]; ok && v == ir.Nil {
				v = t.UninitializedValue()
			}
			returnStmt.AddResult(i, v)
		}
		for i, t := range ctx.currentFunc().ResultTypes() {
			if resultVals[i] == nil {
				returnStmt.AddResult(i, t.UninitializedValue())

				p := b.fset.Position(stmt.Pos())
				if i >= len(stmt.Results) {
					i = 0
				}
				resultExpr := stmt.Results[i]
				resultExprStr := b.nodeToString(resultExpr)
				b.addWarning(fmt.Errorf("%v: could not resolve return value: %s", p, resultExprStr))
			}
		}
	} else {
		// return stmt returns function result variables (if any)
		for i, v := range ctx.currentFunc().Results() {
			returnStmt.AddResult(i, v)
		}
	}
}

func (b *builder) processPanicCall(callExpr *ast.CallExpr, ctx *context) {
	returnStmt := ir.NewReturnStmt(true, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(returnStmt)

	for i, t := range ctx.currentFunc().ResultTypes() {
		v, ok := ctx.currentFunc().Results()[i]
		if ok {
			returnStmt.AddResult(i, v)
		} else {
			returnStmt.AddResult(i, t.UninitializedValue())
		}
	}
}

func (b *builder) processRecoverCall(callExpr *ast.CallExpr, ctx *context) {
	recoverStmt := ir.NewRecoverStmt(callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(recoverStmt)
}

func (b *builder) processCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) map[int]*ir.Variable {
	if b.canIgnoreCall(callExpr, ctx) {
		b.processExprs(callExpr.Args, ctx)
		return map[int]*ir.Variable{}
	}

	if name, ok := b.isKnownBuiltin(callExpr, ctx); ok {
		if callKind != ir.Call {
			p := b.fset.Position(callExpr.Pos())
			b.addWarning(fmt.Errorf("%v: only direct calls to %s are supported", p, name))
			return nil
		}
		switch name {
		case "append":
			resultVar := b.processSliceAppendExpr(callExpr, ctx)
			if resultVar != nil {
				return map[int]*ir.Variable{0: resultVar}
			}
		case "delete":
			b.processDeleteExpr(callExpr, ctx)
		case "make":
			resultVar := b.processMakeContainerExpr(callExpr, ctx)
			if resultVar != nil {
				return map[int]*ir.Variable{0: resultVar}
			}
		case "new":
			resultVar := b.processNewExpr(callExpr, ctx)
			if resultVar != nil {
				return map[int]*ir.Variable{0: resultVar}
			}
		case "panic":
			b.processPanicCall(callExpr, ctx)
		case "recover":
			b.processRecoverCall(callExpr, ctx)
		}
		return map[int]*ir.Variable{}
	}

	if specialOp, ok := b.specialOpForCall(callExpr, ctx); ok {
		resultVar := b.processSpecialOpCallExprWithCallKind(callExpr, callKind, specialOp, ctx)
		if resultVar != nil {
			return map[int]*ir.Variable{0: resultVar}
		}
		return map[int]*ir.Variable{}
	}

	return b.processRegularCallExprWithCallKind(callExpr, callKind, ctx)
}

func (b *builder) processCallReceiverVal(callExpr *ast.CallExpr, calleeSignature *types.Signature, ctx *context) (recvVal ir.RValue, requiresCopy, ok bool) {
	recv := calleeSignature.Recv()
	if recv == nil {
		return nil, false, true
	}
	recvExpr := callExpr.Fun.(*ast.SelectorExpr).X
	recvVal = b.processExpr(recvExpr, ctx)
	recvTypesType := recv.Type()
	irType := b.typesTypeToIrType(recvTypesType)
	if irType == nil {
		return nil, false, true
	}
	if recvVal == nil {
		recvExprStr := b.nodeToString(recvExpr)
		p := b.fset.Position(recvExpr.Pos())
		b.addWarning(fmt.Errorf("%v: could not resolve receiver: %s", p, recvExprStr))
		return nil, false, false
	}
	if recvLV, ok := recvVal.(ir.LValue); ok && recvLV.Type() != irType {
		structType := recvLV.Type().(*ir.StructType)
		embeddedFields, ok := structType.FindEmbeddedFieldOfType(irType)
		if !ok {
			recvExprStr := b.nodeToString(recvExpr)
			p := b.fset.Position(recvExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve receiver: %s", p, recvExprStr))
			return nil, false, false
		}
		for _, field := range embeddedFields {
			recvLV = ir.NewFieldSelection(recvLV, field)
		}
		recvVal = recvLV.(ir.RValue)
	}
	if _, ok := irType.(*ir.StructType); ok {
		requiresCopy = !b.isPointer(recvTypesType)
	} else if irContainerType, ok := irType.(*ir.ContainerType); ok && irContainerType.Kind() == ir.Array {
		requiresCopy = !b.isPointer(recvTypesType)
	} else {
		requiresCopy = false
	}
	return recvVal, requiresCopy, true
}

func (b *builder) processCallArgVals(callExpr *ast.CallExpr, calleeSignature *types.Signature, ctx *context) (argVals map[int]ir.RValue, requiresCopy map[int]bool, ok bool) {
	argVals = b.processExprs(callExpr.Args, ctx)
	requiresCopy = make(map[int]bool)
	regularParamN := calleeSignature.Params().Len()
	if calleeSignature.Variadic() {
		regularParamN--
	}
	for i := 0; i < regularParamN; i++ {
		param := calleeSignature.Params().At(i)
		paramTypesType := param.Type()
		irType := b.typesTypeToIrType(paramTypesType)
		if irType == nil {
			continue
		}
		argVal, ok := argVals[i]
		if !ok {
			if i >= len(callExpr.Args) {
				i = 0
			}
			argExpr := callExpr.Args[i]
			argExprStr := b.nodeToString(argExpr)
			p := b.fset.Position(argExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve argument: %s", p, argExprStr))
			return nil, nil, false
		} else if argVal == ir.Nil {
			argVal = irType.UninitializedValue()
			argVals[i] = argVal
		}
		if _, ok := irType.(*ir.StructType); ok {
			requiresCopy[i] = !b.isPointer(paramTypesType)
		} else if irContainerType, ok := irType.(*ir.ContainerType); ok && irContainerType.Kind() == ir.Array {
			requiresCopy[i] = !b.isPointer(paramTypesType)
		} else {
			requiresCopy[i] = false
		}
	}
	if calleeSignature.Variadic() {
		typesSlice := calleeSignature.Params().At(regularParamN).Type().(*types.Slice)
		irType := b.typesTypeToIrType(typesSlice)
		if irType != nil {
			length := len(callExpr.Args) - regularParamN
			irSliceType := irType.(*ir.ContainerType)
			irElementType := irSliceType.ElementType()
			irElemVals := make([]ir.RValue, length)
			for i := regularParamN; i < len(callExpr.Args); i++ {
				val, ok := argVals[i]
				if !ok {
					argExpr := callExpr.Args[i]
					argExprStr := b.nodeToString(argExpr)
					p := b.fset.Position(argExpr.Pos())
					b.addWarning(fmt.Errorf("%v: could not resolve argument: %s", p, argExprStr))
					return nil, nil, false
				} else if val == ir.Nil {
					val = irElementType.UninitializedValue()
				}
				delete(argVals, i)
				irElemVals[i-regularParamN] = val
			}

			irSlice := b.program.NewVariable("", irSliceType.UninitializedValue())
			ctx.body.Scope().AddVariable(irSlice)

			makeContainerStmt := ir.NewMakeContainerStmt(irSlice, length, false, callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(makeContainerStmt)

			for i := 0; i < length; i++ {
				astElemExpr := callExpr.Args[regularParamN+i]
				irElemVal := irElemVals[i]
				requiresCopy := irSliceType.RequiresDeepCopies()
				irContainerAccess := ir.NewContainerAccess(irSlice, ir.MakeValue(int64(i), ir.IntType))
				irContainerAccess.SetKind(ir.Write)
				assignStmt := ir.NewAssignStmt(irElemVal, irContainerAccess, requiresCopy, astElemExpr.Pos(), astElemExpr.End())
				ctx.body.AddStmt(assignStmt)
			}

			argVals[regularParamN] = irSlice
			requiresCopy[regularParamN] = false
		}
	}
	return argVals, requiresCopy, true
}

func (b *builder) processCallResultVars(calleeSignature *types.Signature, ctx *context) (resultVars map[int]*ir.Variable, requiresCopy map[int]bool) {
	resultVars = make(map[int]*ir.Variable)
	requiresCopy = make(map[int]bool)
	for i := 0; i < calleeSignature.Results().Len(); i++ {
		result := calleeSignature.Results().At(i)
		resultTypesType := result.Type()
		irType := b.typesTypeToIrType(resultTypesType)
		if irType == nil {
			continue
		}
		irVar := b.program.NewVariable("", irType.UninitializedValue())
		ctx.body.Scope().AddVariable(irVar)
		resultVars[i] = irVar
		if _, ok := irType.(*ir.StructType); ok {
			requiresCopy[i] = !b.isPointer(resultTypesType)
		} else if irContainerType, ok := irType.(*ir.ContainerType); ok && irContainerType.Kind() == ir.Array {
			requiresCopy[i] = !b.isPointer(resultTypesType)
		} else {
			requiresCopy[i] = false
		}
	}
	return resultVars, requiresCopy
}

func (b *builder) processRegularCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) map[int]*ir.Variable {
	callee, calleeSignature := b.findCallee(callExpr.Fun, ctx)
	if callee == nil {
		return map[int]*ir.Variable{}
	}
	recvVal, recvRequiresCopy, ok := b.processCallReceiverVal(callExpr, calleeSignature, ctx)
	if !ok {
		return map[int]*ir.Variable{}
	}
	argVals, argRequiresCopy, ok := b.processCallArgVals(callExpr, calleeSignature, ctx)
	if !ok {
		return map[int]*ir.Variable{}
	}

	callStmt := ir.NewCallStmt(callee, calleeSignature, callKind, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(callStmt)

	if recvVal != nil {
		callStmt.AddArg(-1, recvVal, recvRequiresCopy)
	}
	for i, v := range argVals {
		callStmt.AddArg(i, v, argRequiresCopy[i])
	}

	if callKind != ir.Call {
		return map[int]*ir.Variable{}
	}

	resultVars, resultRequiresCopy := b.processCallResultVars(calleeSignature, ctx)
	for i, v := range resultVars {
		callStmt.AddResult(i, v, resultRequiresCopy[i])
	}

	return resultVars
}

func (b *builder) liftedSpecialOpFunc(specialOp ir.SpecialOp) *ir.Func {
	irFunc, ok := b.liftedSpecialOpFuncs[specialOp]
	if ok {
		return irFunc
	}

	irFunc = b.program.AddOuterFunc("lifted_"+specialOp.String(), nil, token.NoPos, token.NoPos)
	subCtx := newContext(nil, nil, irFunc)
	switch specialOp {
	case ir.Close:
		chanVar := b.program.NewVariable("ch", ir.ChanType.UninitializedValue())
		irFunc.AddArg(0, chanVar)
		closeStmt := ir.NewCloseChanStmt(chanVar, token.NoPos, token.NoPos)
		subCtx.body.AddStmt(closeStmt)

	case ir.Lock, ir.Unlock, ir.RLock, ir.RUnlock:
		mutexVar := b.program.NewVariable("mu", ir.MutexType.UninitializedValue())
		irFunc.AddArg(0, mutexVar)
		mutexOpStmt := ir.NewMutexOpStmt(mutexVar, specialOp.(ir.MutexOp), token.NoPos, token.NoPos)
		subCtx.body.AddStmt(mutexOpStmt)

	case ir.Add, ir.Wait:
		waitGroupVar := b.program.NewVariable("wg", ir.WaitGroupType.UninitializedValue())
		irFunc.AddArg(0, waitGroupVar)
		var delta *ir.Variable
		if specialOp == ir.Add {
			delta = b.program.NewVariable("delta", ir.IntType.UninitializedValue())
			irFunc.AddArg(1, delta)
		}
		waitGroupOpStmt := ir.NewWaitGroupOpStmt(waitGroupVar, specialOp.(ir.WaitGroupOp), delta, token.NoPos, token.NoPos)
		subCtx.body.AddStmt(waitGroupOpStmt)

	case ir.Do:
		onceVar := b.program.NewVariable("oc", ir.OnceType.UninitializedValue())
		irFunc.AddArg(0, onceVar)
		fVar := b.program.NewVariable("f", ir.FuncType.UninitializedValue())
		irFunc.AddArg(1, fVar)
		onceDoStmt := ir.NewOnceDoStmt(onceVar, fVar, token.NoPos, token.NoPos)
		subCtx.body.AddStmt(onceDoStmt)

	case ir.DeadEnd:
		deadEndStmt := ir.NewDeadEndStmt(token.NoPos, token.NoPos)
		subCtx.body.AddStmt(deadEndStmt)

	default:
		panic("unexpected special op")
	}

	b.liftedSpecialOpFuncs[specialOp] = irFunc
	return irFunc
}

func (b *builder) processSpecialOpCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, specialOp ir.SpecialOp, ctx *context) *ir.Variable {
	var liftedFuncArgs []ir.RValue

	switch specialOp {
	case ir.MakeChan:
		if callKind != ir.Call {
			return nil
		}
		return b.processMakeChanExpr(callExpr, ctx)

	case ir.Close:
		chanVal := b.findChannel(callExpr.Args[0], ctx)
		if chanVal == nil {
			return nil
		}

		if callKind == ir.Call {
			closeStmt := ir.NewCloseChanStmt(chanVal, callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(closeStmt)
			return nil
		}
		liftedFuncArgs = []ir.RValue{chanVal.(ir.RValue)}

	case ir.Lock, ir.Unlock, ir.RLock, ir.RUnlock:
		selExpr := callExpr.Fun.(*ast.SelectorExpr)
		mutexVal := b.findMutex(selExpr.X, ctx)
		if mutexVal == nil {
			return nil
		}

		if callKind == ir.Call {
			mutexOpStmt := ir.NewMutexOpStmt(mutexVal, specialOp.(ir.MutexOp), callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(mutexOpStmt)
			return nil
		}
		liftedFuncArgs = []ir.RValue{mutexVal.(ir.RValue)}

	case ir.Add, ir.Wait:
		selExpr := callExpr.Fun.(*ast.SelectorExpr)
		waitGroupVal := b.findWaitGroup(selExpr.X, ctx)
		if waitGroupVal == nil {
			return nil
		}
		var delta ir.RValue = ir.MakeValue(-1, ir.IntType)
		if specialOp == ir.Add && selExpr.Sel.Name == "Add" {
			a := callExpr.Args[0]
			res, ok := b.staticIntEval(a, ctx)
			if !ok {
				p := b.fset.Position(a.Pos())
				aStr := b.nodeToString(a)
				b.addWarning(fmt.Errorf("%v: can not process sync.WaitGroup.Add argument: %s", p, aStr))
			} else {
				delta = ir.MakeValue(int64(res), ir.IntType)
			}
		}

		if callKind == ir.Call {
			waitGroupOpStmt := ir.NewWaitGroupOpStmt(waitGroupVal, specialOp.(ir.WaitGroupOp), delta, callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(waitGroupOpStmt)
			return nil
		}
		if specialOp == ir.Add {
			liftedFuncArgs = []ir.RValue{waitGroupVal.(ir.RValue), delta}
		} else {
			liftedFuncArgs = []ir.RValue{waitGroupVal.(ir.RValue)}
		}

	case ir.Do:
		selExpr := callExpr.Fun.(*ast.SelectorExpr)
		onceVal := b.findOnce(selExpr.X, ctx)
		if onceVal == nil {
			return nil
		}
		f := b.processExpr(callExpr.Args[0], ctx)
		if f == nil {
			p := b.fset.Position(callExpr.Args[0].Pos())
			fStr := b.nodeToString(callExpr.Args[0])
			b.addWarning(fmt.Errorf("%v: can not process sync.Once.Do argument: %s", p, fStr))
			return nil
		}

		if callKind == ir.Call {
			onceDoStmt := ir.NewOnceDoStmt(onceVal, f, callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(onceDoStmt)
			return nil
		}
		liftedFuncArgs = []ir.RValue{onceVal.(ir.RValue), f}

	case ir.DeadEnd:
		if callKind == ir.Call {
			deadEndStmt := ir.NewDeadEndStmt(callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(deadEndStmt)
			return nil
		}

	default:
		panic("unexpected special op")
	}

	liftedFunc := b.liftedSpecialOpFunc(specialOp)
	callStmt := ir.NewCallStmt(liftedFunc, nil, callKind, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(callStmt)

	for i, liftedFuncArg := range liftedFuncArgs {
		callStmt.AddArg(i, liftedFuncArg, false)
	}

	return nil
}
