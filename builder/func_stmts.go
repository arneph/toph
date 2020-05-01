package builder

import (
	"fmt"
	"go/ast"

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

func (b *builder) processCallExprWithResultVars(callExpr *ast.CallExpr, callKind ir.CallKind, results map[int]*ir.Variable, ctx *context) {
	argVars := b.processExprs(callExpr.Args, ctx)

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
		} else if fIdent.Name == "append" ||
			fIdent.Name == "cap" ||
			fIdent.Name == "complex" ||
			fIdent.Name == "copy" ||
			fIdent.Name == "delete" ||
			fIdent.Name == "imag" ||
			fIdent.Name == "len" ||
			fIdent.Name == "new" ||
			fIdent.Name == "print" ||
			fIdent.Name == "println" ||
			fIdent.Name == "real" {
			return
		}
	}

	callee := b.findCallee(callExpr.Fun, ctx)
	if callee == nil {
		return
	}

	for i := range callee.Args() {
		_, ok := argVars[i]
		if !ok {
			argExpr := callExpr.Args[i]
			p := b.fset.Position(argExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve argument: %v", p, argExpr))
			return
		}
	}

	callStmt := ir.NewCallStmt(callee, callKind)
	ctx.body.AddStmt(callStmt)

	for i, v := range argVars {
		callStmt.AddArg(i, v)
	}
	if callKind != ir.Go {
		for i, t := range callee.ResultTypes() {
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
