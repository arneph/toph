package builder

import (
	"fmt"
	"go/ast"

	"github.com/arneph/toph/ir"
)

func (b *builder) findChannel(chanExpr ast.Expr, ctx *context) *ir.Variable {
	rv := b.processExpr(chanExpr, ctx)
	v, ok := rv.(*ir.Variable)
	if !ok || v == nil {
		p := b.fset.Position(chanExpr.Pos())
		b.addWarning(fmt.Errorf("%v: could not resolve channel expr: %v", p, chanExpr))
		return nil
	}
	return v
}

func (b *builder) processMakeExpr(callExpr *ast.CallExpr, result *ir.Variable, ctx *context) {
	_, ok := callExpr.Args[0].(*ast.ChanType)
	if !ok {
		return
	}

	bufferSize := 0
	if len(callExpr.Args) > 1 {
		a := callExpr.Args[1]

		res, ok := b.evalIntExpr(a, ctx)
		if !ok {
			p := b.fset.Position(a.Pos())
			b.addWarning(fmt.Errorf("%v: can not process buffer size: %s", p, a))
		} else {
			bufferSize = res
		}
	}

	makeStmt := ir.NewMakeChanStmt(result, bufferSize, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(makeStmt)
}

func (b *builder) processReceiveExpr(expr *ast.UnaryExpr, addToCtx bool, ctx *context) *ir.ChanCommOpStmt {
	chanVar := b.findChannel(expr.X, ctx)
	if chanVar == nil {
		return nil
	}

	receiveStmt := ir.NewChanCommOpStmt(chanVar, ir.Receive, expr.Pos(), expr.End())
	if addToCtx {
		ctx.body.AddStmt(receiveStmt)
	}
	return receiveStmt
}

func (b *builder) processSendStmt(stmt *ast.SendStmt, addToCtx bool, ctx *context) *ir.ChanCommOpStmt {
	b.processExpr(stmt.Value, ctx)

	chanVar := b.findChannel(stmt.Chan, ctx)
	if chanVar == nil {
		return nil
	}

	sendStmt := ir.NewChanCommOpStmt(chanVar, ir.Send, stmt.Pos(), stmt.End())
	if addToCtx {
		ctx.body.AddStmt(sendStmt)
	}
	return sendStmt
}

func (b *builder) processCloseExpr(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) {
	chanVar := b.findChannel(callExpr.Args[0], ctx)
	if chanVar == nil {
		return
	}

	closeStmt := ir.NewCloseChanStmt(chanVar, callKind, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(closeStmt)
}

func (b *builder) processSelectStmt(stmt *ast.SelectStmt, label string, ctx *context) {
	selectStmt := ir.NewSelectStmt(ctx.body.Scope(), stmt.Pos(), stmt.End())

	for _, stmt := range stmt.Body.List {
		commClause := stmt.(*ast.CommClause)
		reachReq := b.findReachabilityRequirementFromAnnotation(commClause, ctx)

		var body *ir.Body
		switch stmt := commClause.Comm.(type) {
		case *ast.SendStmt:
			sendStmt := b.processSendStmt(stmt, false, ctx)
			if sendStmt == nil {
				continue
			}

			selectCase := selectStmt.AddCase(sendStmt, commClause.Pos())
			selectCase.SetReachReq(reachReq)
			body = selectCase.Body()

		case *ast.ExprStmt:
			receiveStmt := b.processReceiveExpr(stmt.X.(*ast.UnaryExpr), false, ctx)
			if receiveStmt == nil {
				continue
			}

			selectCase := selectStmt.AddCase(receiveStmt, commClause.Pos())
			selectCase.SetReachReq(reachReq)
			body = selectCase.Body()

		case *ast.AssignStmt:
			receiveStmt := b.processReceiveExpr(stmt.Rhs[0].(*ast.UnaryExpr), false, ctx)
			if receiveStmt == nil {
				continue
			}

			selectCase := selectStmt.AddCase(receiveStmt, commClause.Pos())
			selectCase.SetReachReq(reachReq)
			body = selectCase.Body()
			subCtx := ctx.subContextForBody(selectStmt, "", body)

			// Create newly defined variables:
			definedVars := b.getDefinedVarsInAssignStmt(stmt, ctx)

			// Resolve all assigned variables:
			lhs := b.getAssignedVarsInAssignStmt(stmt, definedVars, subCtx)

			// Handle lhs expressions
			for i, expr := range stmt.Lhs {
				if v, ok := definedVars[i]; ok {
					ctx.body.Scope().AddVariable(v)
					p := b.fset.Position(expr.Pos())
					b.addWarning(
						fmt.Errorf("%v: can not model value passing via channel", p))
					continue
				}
				if _, ok := lhs[i]; ok {
					p := b.fset.Position(expr.Pos())
					b.addWarning(
						fmt.Errorf("%v: can not model value passing via channel", p))
					continue
				}
				b.processExpr(expr, ctx)
			}

		default:
			if stmt != nil {
				p := b.fset.Position(stmt.Pos())
				b.addWarning(fmt.Errorf("%v: unexpected %T communcation clause", p, stmt))
			}

			selectStmt.SetHasDefault(true)
			body = selectStmt.DefaultBody()
		}

		subCtx := ctx.subContextForBody(selectStmt, label, body)
		for _, stmt := range commClause.Body {
			b.processStmt(stmt, subCtx)
		}
	}

	ctx.body.AddStmt(selectStmt)
}
