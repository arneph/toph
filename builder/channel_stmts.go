package builder

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/arneph/toph/ir"
)

func (b *builder) findChannel(chanExpr ast.Expr, ctx *context) ir.LValue {
	rv := b.processExpr(chanExpr, ctx)
	lv, ok := rv.(ir.LValue)
	if !ok || lv == nil {
		p := b.fset.Position(chanExpr.Pos())
		chanExprStr := b.nodeToString(chanExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve channel expr: %s", p, chanExprStr))
		return nil
	}
	return lv
}

func (b *builder) processMakeChanExpr(callExpr *ast.CallExpr, ctx *context) *ir.Variable {
	var bufferSize ir.RValue = ir.MakeValue(0, ir.IntType)
	if len(callExpr.Args) > 1 {
		bufferSizeExpr := callExpr.Args[1]

		if res := b.processContainerLength(bufferSizeExpr, ctx); res != nil {
			bufferSize = res
		} else if res := b.staticIntEval(bufferSizeExpr, ctx); res != nil {
			bufferSize = res
		} else {
			p := b.fset.Position(bufferSizeExpr.Pos())
			aStr := b.nodeToString(bufferSizeExpr)
			b.addWarning(fmt.Errorf("%v: can not process buffer size: %s", p, aStr))
		}
	}

	result := b.program.NewVariable("", ir.ChanType.InitializedValue())
	ctx.body.Scope().AddVariable(result)
	makeStmt := ir.NewMakeChanStmt(result, bufferSize, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(makeStmt)

	return result
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
			if stmt.Tok == token.DEFINE {
				for _, expr := range stmt.Lhs {
					ident, ok := expr.(*ast.Ident)
					if !ok {
						continue
					}
					b.processVarDefinition(ident, true, subCtx)
				}
			}

			// Handle Lhs expressions:
			lhs := b.processExprs(stmt.Lhs, subCtx)
			for i, expr := range stmt.Lhs {
				if _, ok := lhs[i]; !ok {
					continue
				}
				p := b.fset.Position(expr.Pos())
				b.addWarning(
					fmt.Errorf("%v: can not model value passing via channel", p))
				continue
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
