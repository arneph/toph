package builder

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/arneph/toph/ir"
)

func (b *builder) processIfStmt(stmt *ast.IfStmt, ctx *context) {
	if stmt.Init != nil {
		b.processStmt(stmt.Init, ctx)
	}

	b.processExpr(stmt.Cond, ctx)

	elsePos := stmt.End()
	if stmt.Else != nil {
		elsePos = stmt.Else.Pos()
	}
	ifStmt := ir.NewIfStmt(ctx.body.Scope(), stmt.Pos(), stmt.End(), stmt.Pos(), elsePos)
	ctx.body.AddStmt(ifStmt)

	b.processStmt(stmt.Body, ctx.subContextForBody(ifStmt, "", ifStmt.IfBranch()))

	if stmt.Else != nil {
		b.processStmt(stmt.Else, ctx.subContextForBody(ifStmt, "", ifStmt.ElseBranch()))
	}
}

func (b *builder) processSwitchStmt(stmt *ast.SwitchStmt, label string, ctx *context) {
	if stmt.Init != nil {
		b.processStmt(stmt.Init, ctx)
	}
	if stmt.Tag != nil {
		b.processExpr(stmt.Tag, ctx)
	}

	switchStmt := ir.NewSwitchStmt(ctx.body.Scope(), stmt.Pos(), stmt.End())
	ctx.body.AddStmt(switchStmt)

	for _, s := range stmt.Body.List {
		caseClause := s.(*ast.CaseClause)
		switchCase := switchStmt.AddCase(caseClause.Case)

		if caseClause.List == nil {
			switchCase.SetIsDefault(true)
		}
		for _, condExpr := range caseClause.List {
			condBody := switchCase.AddCond(condExpr.Pos(), condExpr.End()+1)
			condCtx := ctx.subContextForBody(switchStmt, label, condBody)
			b.processExpr(condExpr, condCtx)
		}

		subCtx := ctx.subContextForBody(switchStmt, label, switchCase.Body())
		for _, s := range caseClause.Body {
			if branchStmt, ok := s.(*ast.BranchStmt); ok && branchStmt.Tok == token.FALLTHROUGH {
				switchCase.SetHasFallthrough(true)
				break
			}
			b.processStmt(s, subCtx)
		}
	}
}

func (b *builder) processLabeledStmt(labeledStmt *ast.LabeledStmt, ctx *context) {
	label := labeledStmt.Label.Name
	switch stmt := labeledStmt.Stmt.(type) {
	case *ast.ForStmt:
		b.processForStmt(stmt, label, ctx)
	case *ast.RangeStmt:
		b.processRangeStmt(stmt, label, ctx)
	case *ast.SelectStmt:
		b.processSelectStmt(stmt, label, ctx)
	case *ast.SwitchStmt:
		b.processSwitchStmt(stmt, label, ctx)
	default:
		p := b.fset.Position(labeledStmt.Pos())
		b.addWarning(
			fmt.Errorf("%v: ignoring label: %q", p, label))

		b.processStmt(stmt, ctx)
	}
}

func (b *builder) processForStmt(stmt *ast.ForStmt, label string, ctx *context) {
	if stmt.Init != nil {
		b.processStmt(stmt.Init, ctx)
	}

	forStmt := ir.NewForStmt(ctx.body.Scope(), stmt.Pos(), stmt.End())
	ctx.body.AddStmt(forStmt)

	if stmt.Cond != nil {
		b.processExpr(stmt.Cond, ctx.subContextForBody(forStmt, "", forStmt.Cond()))
	}
	forStmt.SetIsInfinite(stmt.Cond == nil)

	minAnn, maxAnn := b.findIterationBoundsFromAnnotation(stmt, ctx)
	iters := b.staticForLoopBoundsEval(stmt, ctx)
	if minAnn != -1 || maxAnn != -1 {
		if iters != -1 {
			p := b.fset.Position(stmt.Pos())
			b.addWarning(
				fmt.Errorf("%v: unnecessary loop iter annotation", p))
		}
		forStmt.SetMinIterations(minAnn)
		forStmt.SetMaxIterations(maxAnn)
	} else if iters != -1 {
		forStmt.SetMinIterations(iters)
		forStmt.SetMaxIterations(iters)
	}

	b.processStmt(stmt.Body, ctx.subContextForBody(forStmt, label, forStmt.Body()))
	if stmt.Post != nil {
		b.processStmt(stmt.Post, ctx.subContextForBody(forStmt, "", forStmt.Body()))
	}
}

func (b *builder) processRangeStmt(stmt *ast.RangeStmt, label string, ctx *context) {
	typeAndValue, ok := b.typesInfo.Types[stmt.X]
	if !ok {
		p := b.fset.Position(stmt.X.Pos())
		b.addWarning(
			fmt.Errorf("%v: could not determine type of value to range over", p))
	}

	irType := b.typesTypeToIrType(typeAndValue.Type)
	if irType == ir.ChanType {
		chanVar := b.findChannel(stmt.X, ctx)
		if chanVar != nil {
			// Range over channel:
			rangeStmt := ir.NewRangeStmt(chanVar, ctx.body.Scope(), stmt.Pos(), stmt.End())
			ctx.body.AddStmt(rangeStmt)

			b.processStmt(stmt.Body, ctx.subContextForBody(rangeStmt, label, rangeStmt.Body()))
			return
		}
	}

	// Fallback: for statement
	b.processExpr(stmt.X, ctx)

	forStmt := ir.NewForStmt(ctx.body.Scope(), stmt.Pos(), stmt.End())
	ctx.body.AddStmt(forStmt)

	min, max := b.findIterationBoundsFromAnnotation(stmt, ctx)
	forStmt.SetMinIterations(min)
	forStmt.SetMaxIterations(max)

	b.processStmt(stmt.Body, ctx.subContextForBody(forStmt, label, forStmt.Body()))
}

func (b *builder) processBranchStmt(stmt *ast.BranchStmt, ctx *context) {
	label := ""
	if stmt.Label != nil {
		label = stmt.Label.Name
	}
	var kind ir.BranchKind
	var targetStmt ir.Stmt
	switch stmt.Tok {
	case token.BREAK:
		kind = ir.Break
		targetStmt = ctx.findBreakable(label)
	case token.CONTINUE:
		kind = ir.Continue
		targetStmt = ctx.findContinuable(label)
	default:
		p := b.fset.Position(stmt.Pos())
		b.addWarning(
			fmt.Errorf("%v: unsuported branch statement: %s", p, stmt.Tok))
		return
	}
	branchStmt := ir.NewBranchStmt(targetStmt, kind, stmt.Pos(), stmt.End())
	ctx.body.AddStmt(branchStmt)
}
