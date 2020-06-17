package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) getDefinedVarsInAssignStmt(stmt *ast.AssignStmt, ctx *context) map[int]*ir.Variable {
	definedVars := make(map[int]*ir.Variable)
	for i, expr := range stmt.Lhs {
		nameIdent, ok := expr.(*ast.Ident)
		if !ok || nameIdent.Name == "_" {
			continue
		}
		obj, ok := b.pkgTypesInfos[ctx.pkg].Defs[nameIdent]
		if !ok {
			continue
		}

		varType := obj.(*types.Var)
		t, initialValue, ok := typesTypeToIrType(varType.Type())
		if !ok {
			continue
		} else if t == ir.MutexType {
			p := b.fset.Position(expr.Pos())
			b.addWarning(fmt.Errorf("%v: can not declare sync.Mutex or sync.RWMutex in assignment", p))
			continue
		} else if t == ir.WaitGroupType {
			p := b.fset.Position(expr.Pos())
			b.addWarning(fmt.Errorf("%v: can not declare sync.WaitGroup in assignment", p))
			continue
		}

		v := b.program.NewVariable(nameIdent.Name, t, initialValue)
		definedVars[i] = v
		b.pkgVarTypes[ctx.pkg][varType] = v
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

		switch expr := expr.(type) {
		case *ast.Ident:
			if expr.Name == "_" {
				continue
			}
			v := b.processIdent(expr, ctx)
			if v == nil {
				continue
			}
			lhs[i] = v.(*ir.Variable)

		case *ast.SelectorExpr:
			ident, ok := expr.X.(*ast.Ident)
			if !ok {
				continue
			}
			typesPkgName, ok := b.pkgTypesInfos[ctx.pkg].Uses[ident].(*types.PkgName)
			if !ok {
				continue
			}
			pkg := typesPkgName.Imported().Path()
			typesPkg := b.pkgTypesPackages[pkg]
			typesPkgScope := typesPkg.Scope()
			varType, ok := typesPkgScope.Lookup(expr.Sel.Name).(*types.Var)
			if !ok {
				continue
			}
			v, ok := b.pkgVarTypes[pkg][varType]
			if !ok {
				continue
			}
			lhs[i] = v

		default:
			continue
		}
	}
	return lhs
}

func (b *builder) processAssignStmt(stmt *ast.AssignStmt, ctx *context) {
	// Create newly defined variables:
	definedVars := b.getDefinedVarsInAssignStmt(stmt, ctx)

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
			if nameIdent, ok := expr.(*ast.Ident); ok && nameIdent.Name == "_" {
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
