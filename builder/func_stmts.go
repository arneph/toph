package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) findCallee(funcExpr ast.Expr, ctx *context) (callee ir.Callable, calleeSignature *types.Signature, receiver ir.RValue) {
	if b.canIgnoreCall(funcExpr, ctx) {
		return nil, nil, nil
	}

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

func (b *builder) processCallExprWithCallKind(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) map[int]*ir.Variable {
	if b.canIgnoreCall(callExpr.Fun, ctx) {
		b.processExprs(callExpr.Args, ctx)
		return map[int]*ir.Variable{}
	}

	switch funcExpr := callExpr.Fun.(type) {
	case *ast.Ident:
		builtin, ok := b.typesInfo.Uses[funcExpr].(*types.Builtin)
		if !ok {
			break
		}
		switch builtin.Name() {
		case "make":
			result := b.processMakeExpr(callExpr, ctx)
			if result != nil {
				return map[int]*ir.Variable{0: result}
			}
			return map[int]*ir.Variable{}
		case "close":
			b.processCloseExpr(callExpr, callKind, ctx)
			return map[int]*ir.Variable{}
		}
	case *ast.SelectorExpr:
		used, ok := b.typesInfo.Uses[funcExpr.Sel]
		if !ok {
			break
		}
		switch used.String() {
		case "func (*sync.Mutex).Lock()",
			"func (*sync.Mutex).Unlock()",
			"func (*sync.RWMutex).Lock()",
			"func (*sync.RWMutex).RLock()",
			"func (*sync.RWMutex).RUnlock()",
			"func (*sync.RWMutex).Unlock()":
			b.processMutexOpExpr(callExpr, callKind, ctx)
			return map[int]*ir.Variable{}
		case "func (*sync.WaitGroup).Add(delta int)",
			"func (*sync.WaitGroup).Done()",
			"func (*sync.WaitGroup).Wait()":
			b.processWaitGroupOpExpr(callExpr, callKind, ctx)
			return map[int]*ir.Variable{}
		}
	}

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
