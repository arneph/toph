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
		typesSelection, ok := b.typesInfo.Selections[selExpr]
		if ok && typesSelection.Kind() == types.MethodVal {
			calleeValue = b.processIdent(selExpr.Sel, ctx)
			calleeTypesType = b.typesInfo.TypeOf(selExpr.Sel)
		} else {
			calleeValue = b.processSelectorExpr(selExpr, ctx)
			calleeTypesType = b.typesInfo.TypeOf(selExpr)
		}
	} else {
		calleeValue = b.processExpr(funcExpr, ctx)
		calleeTypesType = b.typesInfo.TypeOf(funcExpr)
	}
	if calleeValue == nil {
		p := b.fset.Position(funcExpr.Pos())
		funcExprStr := b.nodeToString(funcExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve func expr: %v", p, funcExprStr))
		return nil, nil
	}

	switch calleeValue := calleeValue.(type) {
	case *ir.Variable:
		callee = calleeValue
	case *ir.FieldSelection:
		callee = calleeValue
	case ir.Value:
		callee = b.program.Func(ir.FuncIndex(calleeValue))
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

func (b *builder) processCallExpr(callExpr *ast.CallExpr, ctx *context) map[int]ir.LValue {
	return b.processCallExprWithCallKind(callExpr, ir.Call, ctx)
}

func (b *builder) processDeferStmt(stmt *ast.DeferStmt, ctx *context) {
	b.processCallExprWithCallKind(stmt.Call, ir.Defer, ctx)
}

func (b *builder) processGoStmt(stmt *ast.GoStmt, ctx *context) {
	b.processCallExprWithCallKind(stmt.Call, ir.Go, ctx)
}

func (b *builder) processReturnStmt(stmt *ast.ReturnStmt, ctx *context) {
	resultVars := b.processExprs(stmt.Results, ctx)
	returnStmt := ir.NewReturnStmt(stmt.Pos(), stmt.End())
	ctx.body.AddStmt(returnStmt)

	for i, v := range resultVars {
		returnStmt.AddResult(i, v)
	}
	for i := range ctx.currentFunc().ResultTypes() {
		if resultVars[i] != nil {
			continue
		}
		resultVar, ok := ctx.currentFunc().Results()[i]
		if !ok {
			p := b.fset.Position(stmt.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve return value at index %d", p, i))
			continue
		}
		resultVars[i] = resultVar
	}
}

func (b *builder) processCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) map[int]ir.LValue {
	if b.canIgnoreCall(callExpr) {
		b.processExprs(callExpr.Args, ctx)
		return map[int]ir.LValue{}
	}

	specialOp, ok := b.specialOpForCall(callExpr)
	if ok {
		resultVar := b.processSpecialOpCallExprWithCallKind(callExpr, callKind, specialOp, ctx)
		if resultVar != nil {
			return map[int]ir.LValue{0: resultVar}
		}
		return map[int]ir.LValue{}
	}

	if b.isNewCall(callExpr) {
		if callKind != ir.Call {
			return nil
		}

		resultVar := b.processNewExpr(callExpr, ctx)
		if resultVar != nil {
			return map[int]ir.LValue{0: resultVar}
		}
		return map[int]ir.LValue{}
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
	} else {
		requiresCopy = false
	}
	return recvVal, requiresCopy, true
}

func (b *builder) processCallArgVals(callExpr *ast.CallExpr, calleeSignature *types.Signature, ctx *context) (argVals map[int]ir.RValue, requiresCopy map[int]bool, ok bool) {
	argVals = b.processExprs(callExpr.Args, ctx)
	requiresCopy = make(map[int]bool)
	for i := 0; i < calleeSignature.Params().Len(); i++ {
		param := calleeSignature.Params().At(i)
		paramTypesType := param.Type()
		irType := b.typesTypeToIrType(paramTypesType)
		if irType == nil {
			continue
		}
		if _, ok := argVals[i]; !ok {
			argExpr := callExpr.Args[i]
			argExprStr := b.nodeToString(argExpr)
			p := b.fset.Position(argExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve argument: %s", p, argExprStr))
			return nil, nil, false
		}
		if _, ok := irType.(*ir.StructType); ok {
			requiresCopy[i] = !b.isPointer(paramTypesType)
		} else {
			requiresCopy[i] = false
		}
	}
	return argVals, requiresCopy, true
}

func (b *builder) processCallResultVars(calleeSignature *types.Signature, ctx *context) (resultVals map[int]ir.LValue, requiresCopy map[int]bool) {
	resultVals = make(map[int]ir.LValue)
	requiresCopy = make(map[int]bool)
	for i := 0; i < calleeSignature.Results().Len(); i++ {
		result := calleeSignature.Results().At(i)
		resultTypesType := result.Type()
		irType := b.typesTypeToIrType(resultTypesType)
		if irType == nil {
			continue
		}
		initialValue := b.initialValueForIrType(irType)
		irVar := b.program.NewVariable("", irType, initialValue)
		ctx.body.Scope().AddVariable(irVar)
		resultVals[i] = irVar
		if _, ok := irType.(*ir.StructType); ok {
			requiresCopy[i] = !b.isPointer(resultTypesType)
		} else {
			requiresCopy[i] = false
		}
	}
	return resultVals, requiresCopy
}

func (b *builder) processRegularCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) map[int]ir.LValue {
	callee, calleeSignature := b.findCallee(callExpr.Fun, ctx)
	if callee == nil {
		return map[int]ir.LValue{}
	}
	recvVal, recvRequiresCopy, ok := b.processCallReceiverVal(callExpr, calleeSignature, ctx)
	if !ok {
		return map[int]ir.LValue{}
	}
	argVals, argRequiresCopy, ok := b.processCallArgVals(callExpr, calleeSignature, ctx)
	if !ok {
		return map[int]ir.LValue{}
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
		return map[int]ir.LValue{}
	}

	resultVals, resultRequiresCopy := b.processCallResultVars(calleeSignature, ctx)
	for i, v := range resultVals {
		callStmt.AddResult(i, v, resultRequiresCopy[i])
	}

	return resultVals
}

func (b *builder) liftedSpecialOpFunc(specialOp ir.SpecialOp) *ir.Func {
	irFunc, ok := b.liftedSpecialOpFuncs[specialOp]
	if ok {
		return irFunc
	}

	irFunc = b.program.AddOuterFunc("lifted_"+specialOp.String(), nil, token.NoPos, token.NoPos)
	subCtx := newContext(nil, irFunc)
	switch specialOp {
	case ir.Close:
		chanVar := b.program.NewVariable("ch", ir.ChanType, -1)
		irFunc.AddArg(0, chanVar)
		closeStmt := ir.NewCloseChanStmt(chanVar, token.NoPos, token.NoPos)
		subCtx.body.AddStmt(closeStmt)

	case ir.Lock, ir.Unlock, ir.RLock, ir.RUnlock:
		mutexVar := b.program.NewVariable("mu", ir.MutexType, -1)
		irFunc.AddArg(0, mutexVar)
		mutexOpStmt := ir.NewMutexOpStmt(mutexVar, specialOp.(ir.MutexOp), token.NoPos, token.NoPos)
		subCtx.body.AddStmt(mutexOpStmt)

	case ir.Add, ir.Wait:
		waitGroupVar := b.program.NewVariable("wg", ir.WaitGroupType, -1)
		irFunc.AddArg(0, waitGroupVar)
		var delta *ir.Variable
		if specialOp == ir.Add {
			delta = b.program.NewVariable("delta", ir.IntType, 0)
			irFunc.AddArg(1, delta)
		}
		waitGroupOpStmt := ir.NewWaitGroupOpStmt(waitGroupVar, specialOp.(ir.WaitGroupOp), delta, token.NoPos, token.NoPos)
		subCtx.body.AddStmt(waitGroupOpStmt)

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

		return b.processMakeExpr(callExpr, ctx)

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
		var delta ir.RValue = ir.Value(-1)
		if specialOp == ir.Add && selExpr.Sel.Name == "Add" {
			a := callExpr.Args[0]
			res, ok := b.staticIntEval(a, ctx)
			if !ok {
				p := b.fset.Position(a.Pos())
				aStr := b.nodeToString(a)
				b.addWarning(fmt.Errorf("%v: can not process sync.WaitGroup.Add argument: %s", p, aStr))
			} else {
				delta = ir.Value(res)
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
