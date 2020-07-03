package builder

import (
	"fmt"
	"go/ast"
)

func (b *builder) processStmt(stmt ast.Stmt, ctx *context) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		b.processAssignStmt(s, ctx)
	case *ast.BlockStmt:
		b.processBlockStmt(s, ctx)
	case *ast.BranchStmt:
		b.processBranchStmt(s, ctx)
	case *ast.DeferStmt:
		b.processDeferStmt(s, ctx)
	case *ast.DeclStmt:
		b.processGenDecl(s.Decl.(*ast.GenDecl), true, ctx.body.Scope(), ctx)
	case *ast.ExprStmt:
		b.processExpr(s.X, ctx)
	case *ast.ForStmt:
		b.processForStmt(s, "", ctx)
	case *ast.GoStmt:
		b.processGoStmt(s, ctx)
	case *ast.IfStmt:
		b.processIfStmt(s, ctx)
	case *ast.IncDecStmt:
		b.processExpr(s.X, ctx)
	case *ast.LabeledStmt:
		b.processLabeledStmt(s, ctx)
	case *ast.RangeStmt:
		b.processRangeStmt(s, "", ctx)
	case *ast.ReturnStmt:
		b.processReturnStmt(s, ctx)
	case *ast.SelectStmt:
		b.processSelectStmt(s, "", ctx)
	case *ast.SendStmt:
		b.processSendStmt(s, true, ctx)
	case *ast.SwitchStmt:
		b.processSwitchStmt(s, "", ctx)
	default:
		p := b.fset.Position(stmt.Pos())
		b.addWarning(fmt.Errorf("%v: ignoring %T statement", p, s))
	}
}

func (b *builder) processBlockStmt(stmt *ast.BlockStmt, ctx *context) {
	for _, s := range stmt.List {
		b.processStmt(s, ctx)
	}
}
