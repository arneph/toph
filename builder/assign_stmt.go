package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) getDefinedVarsInAssignStmt(stmt *ast.AssignStmt) map[int]*ir.Variable {
	definedVars := make(map[int]*ir.Variable)
	for i, expr := range stmt.Lhs {
		nameIdent, ok := expr.(*ast.Ident)
		if !ok || nameIdent.Name == "_" {
			continue
		}
		obj, ok := b.info.Defs[nameIdent]
		if !ok {
			continue
		}

		varType := obj.(*types.Var)
		t, ok := typesTypeToIrType(varType.Type())
		if !ok {
			continue
		}

		v := ir.NewVariable(nameIdent.Name, t, -1)
		definedVars[i] = v
		b.varTypes[varType] = v
	}
	return definedVars
}

func (b *builder) getAssignedVarsInAssignStmt(stmt *ast.AssignStmt, definedVars map[int]*ir.Variable, ctx *context) map[int]*ir.Variable {
	lhs := make(map[int]*ir.Variable)
	for i, expr := range stmt.Lhs {
		definedVar, ok := definedVars[i]
		if ok {
			lhs[i] = definedVar
			continue
		}

		ident, ok := expr.(*ast.Ident)
		if !ok {
			continue
		}

		v := b.processIdent(ident, ctx)
		if v == nil {
			continue
		}
		lhs[i] = v.(*ir.Variable)
	}
	return lhs
}

func (b *builder) processAssignStmt(stmt *ast.AssignStmt, ctx *context) {
	// Create newly defined variables:
	definedVars := b.getDefinedVarsInAssignStmt(stmt)

	// Resolve all assigned variables:
	lhs := b.getAssignedVarsInAssignStmt(stmt, definedVars, ctx)

	defer func() {
		// Handle lhs expressions
		for i, expr := range stmt.Lhs {
			if v, ok := definedVars[i]; ok {
				ctx.body.Scope().AddVariable(v)
				continue
			}
			if _, ok := lhs[i]; ok {
				continue
			}

			b.processExpr(expr, ctx)
		}
	}()

	// Handle single call expression:
	callExpr, ok := stmt.Rhs[0].(*ast.CallExpr)
	if ok && len(stmt.Rhs) == 1 {
		b.processCallExprWithResultVars(callExpr, ir.Call, lhs, ctx)
		return
	}

	// Handle Rhs expressions:
	for i, expr := range stmt.Rhs {
		l := lhs[i]
		r := b.processExpr(expr, ctx)
		if l == nil && r == nil {
			continue
		} else if l == nil {
			p := b.fset.Position(stmt.Lhs[i].Pos())
			b.addWarning(fmt.Errorf("%v: could not handle lhs of assignment", p))
			continue
		} else if r == nil {
			p := b.fset.Position(stmt.Rhs[i].Pos())
			b.addWarning(
				fmt.Errorf("%v: could not handle rhs of assignment", p))
			continue
		}

		assignStmt := ir.NewAssignStmt(r, l, stmt.Pos(), stmt.End())
		ctx.body.AddStmt(assignStmt)
	}
}
