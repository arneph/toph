package builder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) findCallee(funcExpr ast.Expr, ctx *context) (callee ir.Callable, calleeSignature *types.Signature, receiver ir.RValue) {
	var calleeValue ir.RValue
	var calleeTypesType types.Type
	if selExpr, ok := funcExpr.(*ast.SelectorExpr); ok {
		receiver, calleeValue = b.processSelectorExpr(selExpr, ctx)
		calleeTypesType = b.typesInfo.TypeOf(selExpr.Sel)
	} else {
		calleeValue = b.processExpr(funcExpr, ctx)
		calleeTypesType = b.typesInfo.TypeOf(funcExpr)
		receiver = nil
	}
	if calleeValue == nil {
		p := b.fset.Position(funcExpr.Pos())
		funcExprStr := b.nodeToString(funcExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve func expr: %v", p, funcExprStr))
		return nil, nil, nil
	}

	switch calleeValue := calleeValue.(type) {
	case *ir.Variable:
		callee = calleeValue
	case ir.Value:
		callee = b.program.Func(ir.FuncIndex(calleeValue))
		if callee == nil {
			p := b.fset.Position(funcExpr.Pos())
			funcExprStr := b.nodeToString(funcExpr)
			b.addWarning(fmt.Errorf("%v: could not resolve func index in ir.Program: %v", p, funcExprStr))
			return nil, nil, nil
		}
	}

	calleeSignature = calleeTypesType.Underlying().(*types.Signature)
	receiverTypesVar := calleeSignature.Recv()
	if receiverTypesVar == nil {
		receiver = nil
	}

	return callee, calleeSignature, receiver
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
	resultVars := b.processExprs(stmt.Results, ctx)
	returnStmt := ir.NewReturnStmt(stmt.Pos(), stmt.End())
	ctx.body.AddStmt(returnStmt)

	for i, v := range resultVars {
		returnStmt.AddResult(i, v)
	}
}

func (b *builder) processCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) map[int]*ir.Variable {
	if b.canIgnoreCall(callExpr) {
		b.processExprs(callExpr.Args, ctx)
		return map[int]*ir.Variable{}
	}

	specialOp, ok := b.specialOpForCall(callExpr)
	if ok {
		resultVar := b.processSpecialOpCallExprWithCallKind(callExpr, callKind, specialOp, ctx)
		if resultVar != nil {
			return map[int]*ir.Variable{0: resultVar}
		}
		return map[int]*ir.Variable{}
	}

	return b.processRegularCallExprWithCallKind(callExpr, callKind, ctx)
}

func (b *builder) processCallArgVals(callExpr *ast.CallExpr, calleeSignature *types.Signature, ctx *context) (argVals map[int]ir.RValue, ok bool) {
	argVals = b.processExprs(callExpr.Args, ctx)
	for i := 0; i < calleeSignature.Params().Len(); i++ {
		param := calleeSignature.Params().At(i)
		if _, _, ok := typesTypeToIrType(param.Type()); !ok {
			continue
		}
		if _, ok := argVals[i]; !ok {
			argExpr := callExpr.Args[i]
			p := b.fset.Position(argExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve argument: %v", p, argExpr))
			return nil, false
		}
	}
	return argVals, true
}

func (b *builder) processCallResultVars(calleeSignature *types.Signature, ctx *context) map[int]*ir.Variable {
	resultVars := make(map[int]*ir.Variable)
	for i := 0; i < calleeSignature.Results().Len(); i++ {
		res := calleeSignature.Results().At(i)
		irType, initialValue, ok := typesTypeToIrType(res.Type())
		if !ok {
			continue
		}
		irVar := b.program.NewVariable("", irType, initialValue)
		ctx.body.Scope().AddVariable(irVar)
		resultVars[i] = irVar
	}
	return resultVars
}

func (b *builder) processRegularCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) map[int]*ir.Variable {
	callee, calleeSignature, receiverVal := b.findCallee(callExpr.Fun, ctx)
	if callee == nil {
		return map[int]*ir.Variable{}
	}

	argVals, ok := b.processCallArgVals(callExpr, calleeSignature, ctx)
	if !ok {
		return map[int]*ir.Variable{}
	}

	callStmt := ir.NewCallStmt(callee, calleeSignature, callKind, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(callStmt)

	if receiverVal != nil {
		callStmt.AddArg(-1, receiverVal)
	}
	for i, v := range argVals {
		callStmt.AddArg(i, v)
	}

	if callKind != ir.Call {
		return map[int]*ir.Variable{}
	}

	resultVars := b.processCallResultVars(calleeSignature, ctx)
	for i, v := range resultVars {
		callStmt.AddResult(i, v)
	}

	return resultVars
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

		result := b.processMakeExpr(callExpr, ctx)
		return result

	case ir.Close:
		chanVar := b.findChannel(callExpr.Args[0], ctx)
		if chanVar == nil {
			return nil
		}

		if callKind == ir.Call {
			closeStmt := ir.NewCloseChanStmt(chanVar, callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(closeStmt)
			return nil
		}
		liftedFuncArgs = []ir.RValue{chanVar}

	case ir.Lock, ir.Unlock, ir.RLock, ir.RUnlock:
		selExpr := callExpr.Fun.(*ast.SelectorExpr)
		mutexVar := b.findMutex(selExpr.X, ctx)
		if mutexVar == nil {
			return nil
		}

		if callKind == ir.Call {
			mutexOpStmt := ir.NewMutexOpStmt(mutexVar, specialOp.(ir.MutexOp), callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(mutexOpStmt)
			return nil
		}
		liftedFuncArgs = []ir.RValue{mutexVar}

	case ir.Add, ir.Wait:
		selExpr := callExpr.Fun.(*ast.SelectorExpr)
		waitGroupVar := b.findWaitGroup(selExpr.X, ctx)
		if waitGroupVar == nil {
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
			waitGroupOpStmt := ir.NewWaitGroupOpStmt(waitGroupVar, specialOp.(ir.WaitGroupOp), delta, callExpr.Pos(), callExpr.End())
			ctx.body.AddStmt(waitGroupOpStmt)
			return nil
		}
		if specialOp == ir.Add {
			liftedFuncArgs = []ir.RValue{waitGroupVar, delta}
		} else {
			liftedFuncArgs = []ir.RValue{waitGroupVar}
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
		callStmt.AddArg(i, liftedFuncArg)
	}

	return nil
}
