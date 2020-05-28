package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processGoStmt(stmt *ast.GoStmt, ctx *context) {
	results := make(map[int]*ir.Variable)
	b.processCallExprWithResultVars(stmt.Call, ir.Go, results, ctx)
}

func (b *builder) processCallExpr(callExpr *ast.CallExpr, ctx *context) *ir.Variable {
	results := make(map[int]*ir.Variable)
	b.processCallExprWithResultVars(callExpr, ir.Call, results, ctx)
	if len(results) == 0 {
		return nil
	} else if len(results) > 1 {
		panic("attempted to use call expr as single expr")
	}
	result, ok := results[0]
	if !ok {
		panic("attempted to use call expr as single expr")
	}
	return result
}

func (b *builder) processDeferStmt(stmt *ast.DeferStmt, ctx *context) {
	if b.canIgnoreCall(stmt.Call.Fun, ctx) {
		b.processExprs(stmt.Call.Args, ctx)
		return
	} else if fIdent, ok := stmt.Call.Fun.(*ast.Ident); ok && fIdent.Name == "make" {
		b.processExprs(stmt.Call.Args, ctx)
		return
	}
	if len(ctx.enclosingStmts) > 0 {
		p := b.fset.Position(stmt.Pos())
		b.addWarning(fmt.Errorf("%v: could not handle defer statement inside other statement: %v", p, stmt.Call))
		return
	}

	deferIndex := len(ctx.deferredCallInfos)

	if fIdent, ok := stmt.Call.Fun.(*ast.Ident); ok && fIdent.Name == "close" {
		chanVar := b.findChannel(stmt.Call.Args[0], ctx)
		if chanVar == nil {
			return
		}
		deferredChanVar := ir.NewVariable(
			fmt.Sprintf("defer%d_arg0", deferIndex), ir.ChanType, -1)
		ctx.body.Scope().AddVariable(deferredChanVar)
		assignStmt := ir.NewAssignStmt(chanVar, deferredChanVar, stmt.Pos(), stmt.End())
		ctx.body.AddStmt(assignStmt)
		ctx.deferredCallInfos = append(ctx.deferredCallInfos,
			deferredCallInfo{
				deferStmt:   stmt,
				isChanClose: true,
				argVals:     map[int]ir.RValue{0: deferredChanVar},
			})
		return
	}

	callee, calleeSignature := b.findCallee(stmt.Call.Fun, ctx)
	if callee == nil {
		return
	}
	argVals, ok := b.processCallArgVals(stmt.Call, calleeSignature, ctx)
	if !ok {
		return
	}

	deferredCallee := callee
	if callee, ok := callee.(*ir.Variable); ok {
		v := ir.NewVariable(
			fmt.Sprintf("defer%d_callee", deferIndex), ir.FuncType, -1)
		ctx.body.Scope().AddVariable(v)

		assignStmt := ir.NewAssignStmt(callee, v, stmt.Pos(), stmt.End())
		ctx.body.AddStmt(assignStmt)

		deferredCallee = v
	}

	deferredArgVals := make(map[int]ir.RValue, len(argVals))
	for i, argVal := range argVals {
		param := calleeSignature.Params().At(i)
		t, _ := typesTypeToIrType(param.Type())

		deferredVal := ir.NewVariable(
			fmt.Sprintf("defer%d_arg%d", deferIndex, i), t, -1)
		ctx.body.Scope().AddVariable(deferredVal)

		assignStmt := ir.NewAssignStmt(argVal, deferredVal, stmt.Pos(), stmt.End())
		ctx.body.AddStmt(assignStmt)

		deferredArgVals[i] = deferredVal
	}

	ctx.deferredCallInfos = append(ctx.deferredCallInfos,
		deferredCallInfo{
			deferStmt:       stmt,
			isChanClose:     false,
			callee:          deferredCallee,
			calleeSignature: calleeSignature,
			argVals:         deferredArgVals,
		})
}

func (b *builder) processDeferredCalls(ctx *context) {
	for i := len(ctx.deferredCallInfos) - 1; i >= 0; i-- {
		b.processDeferredCall(i, &ctx.deferredCallInfos[i], ctx)
	}
}

func (b *builder) processDeferredCall(deferIndex int, info *deferredCallInfo, ctx *context) {
	if info.isChanClose {
		v := info.argVals[0].(*ir.Variable)
		closeStmt := ir.NewChanOpStmt(v, ir.Close, info.deferStmt.Call.Pos(), info.deferStmt.Call.End())
		ctx.body.AddStmt(closeStmt)
		return
	}

	callStmt := ir.NewCallStmt(info.callee, info.calleeSignature, ir.Call, info.deferStmt.Call.Pos(), info.deferStmt.Call.End())
	ctx.body.AddStmt(callStmt)

	for i, v := range info.argVals {
		callStmt.AddArg(i, v)
	}
	return
}

func (b *builder) processReturnStmt(stmt *ast.ReturnStmt, ctx *context) {
	resultVars := b.processExprs(stmt.Results, ctx)

	b.processDeferredCalls(ctx)

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
		if _, ok := typesTypeToIrType(param.Type()); !ok {
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

func (b *builder) processCallExprWithResultVars(callExpr *ast.CallExpr, callKind ir.CallKind, results map[int]*ir.Variable, ctx *context) {
	if b.canIgnoreCall(callExpr.Fun, ctx) {
		b.processExprs(callExpr.Args, ctx)
		return
	}
	if fIdent, ok := callExpr.Fun.(*ast.Ident); ok {
		if fIdent.Name == "make" {
			v, ok := results[0]
			if !ok {
				v = ir.NewVariable("", ir.ChanType, -1)
				ctx.body.Scope().AddVariable(v)
				results[0] = v
			}

			b.processMakeExpr(callExpr, v, ctx)
			return

		} else if fIdent.Name == "close" {
			b.processCloseExpr(callExpr, ctx)
			return
		}
	}

	callee, calleeSignature := b.findCallee(callExpr.Fun, ctx)
	if callee == nil {
		return
	}
	argVals, ok := b.processCallArgVals(callExpr, calleeSignature, ctx)
	if !ok {
		return
	}

	callStmt := ir.NewCallStmt(callee, calleeSignature, callKind, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(callStmt)

	for i, v := range argVals {
		callStmt.AddArg(i, v)
	}
	if callKind != ir.Go {
		for i := 0; i < calleeSignature.Results().Len(); i++ {
			res := calleeSignature.Results().At(i)
			t, ok := typesTypeToIrType(res.Type())
			if !ok {
				continue
			}
			v, ok := results[i]
			if !ok {
				v = ir.NewVariable("", t, -1)
				ctx.body.Scope().AddVariable(v)
				results[i] = v
			}
			callStmt.AddResult(i, v)
		}
	}
}
