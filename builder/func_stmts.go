package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processCallExpr(callExpr *ast.CallExpr, ctx *context) *ir.Variable {
	results := make(map[int]*ir.Variable)
	b.processCallExprWithResultVars(callExpr, ir.Call, results, ctx)
	if len(results) == 0 {
		return nil
	} else if _, ok := results[0]; len(results) > 1 || !ok {
		panic("attempted to use call expr as single expr")
	}
	return results[0]
}

func (b *builder) processDeferStmt(stmt *ast.DeferStmt, ctx *context) {
	b.processCallExprWithResultVars(stmt.Call, ir.Defer, nil, ctx)
}

func (b *builder) processGoStmt(stmt *ast.GoStmt, ctx *context) {
	b.processCallExprWithResultVars(stmt.Call, ir.Go, nil, ctx)
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

func (b *builder) processCallResultVars(calleeSignature *types.Signature, results map[int]*ir.Variable, ctx *context) {
	for i := 0; i < calleeSignature.Results().Len(); i++ {
		res := calleeSignature.Results().At(i)
		t, ok := typesTypeToIrType(res.Type())
		if !ok {
			delete(results, i)
			continue
		}
		v, ok := results[i]
		if !ok {
			v = ir.NewVariable("", t, -1)
			ctx.body.Scope().AddVariable(v)
			results[i] = v
		}
	}
}

func (b *builder) processCallExprWithResultVars(callExpr *ast.CallExpr, callKind ir.CallKind, resVars map[int]*ir.Variable, ctx *context) {
	if b.canIgnoreCall(callExpr.Fun, ctx) {
		b.processExprs(callExpr.Args, ctx)
		return
	}
	if fIdent, ok := callExpr.Fun.(*ast.Ident); ok {
		if fIdent.Name == "make" {
			v, ok := resVars[0]
			if !ok {
				v = ir.NewVariable("", ir.ChanType, -1)
				ctx.body.Scope().AddVariable(v)
				resVars[0] = v
			}
			b.processMakeExpr(callExpr, v, ctx)
			return

		} else if fIdent.Name == "close" {
			b.processCloseExpr(callExpr, callKind, ctx)
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
	if callKind == ir.Call {
		b.processCallResultVars(calleeSignature, resVars, ctx)
		for i, v := range resVars {
			callStmt.AddResult(i, v)
		}
	}
}
