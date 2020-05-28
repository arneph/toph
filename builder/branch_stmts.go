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

func (b *builder) processLabeledStmt(labeledStmt *ast.LabeledStmt, ctx *context) {
	label := labeledStmt.Label.Name
	switch stmt := labeledStmt.Stmt.(type) {
	case *ast.ForStmt:
		b.processForStmt(stmt, label, ctx)
	case *ast.RangeStmt:
		b.processRangeStmt(stmt, label, ctx)
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
	iters := b.findIterationBoundThroughAnalysis(stmt, ctx)
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
	typeAndValue, ok := b.info.Types[stmt.X]
	if !ok {
		p := b.fset.Position(stmt.X.Pos())
		b.addWarning(
			fmt.Errorf("%v: could not determine type of value to range over", p))
	}

	t, ok := typesTypeToIrType(typeAndValue.Type)
	if ok && t == ir.ChanType {
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

func (b *builder) processBranchStmt(branchStmt *ast.BranchStmt, ctx *context) {
	switch branchStmt.Tok {
	case token.BREAK, token.CONTINUE:
		var kind ir.BranchKind
		var loop ir.Loop
		if branchStmt.Tok == token.CONTINUE {
			kind = ir.Continue
		} else {
			kind = ir.Break
		}
		if branchStmt.Label == nil {
			loop = ctx.currentLoop()
		} else {
			loop = ctx.currentLabeledLoop(branchStmt.Label.Name)
		}

		branchStmt := ir.NewBranchStmt(loop, kind, branchStmt.Pos(), branchStmt.End())
		ctx.body.AddStmt(branchStmt)

	default:
		p := b.fset.Position(branchStmt.Pos())
		b.addWarning(
			fmt.Errorf("%v: unsuported branch statement: %s", p, branchStmt.Tok))
		return
	}
}
