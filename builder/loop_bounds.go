package builder

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

type loopVarInfo struct {
	obj  *types.Var
	name string
	val  constant.Value
}

func (b *builder) findIterationBoundThroughAnalysis(forStmt *ast.ForStmt, ctx *context) int {
	if forStmt.Init == nil || forStmt.Cond == nil || forStmt.Post == nil {
		return -1
	}

	initAssignStmt, ok := forStmt.Init.(*ast.AssignStmt)
	if !ok ||
		initAssignStmt.Tok != token.DEFINE ||
		len(initAssignStmt.Lhs) != len(initAssignStmt.Rhs) {
		return -1
	}

	var loopVars []loopVarInfo
	for i, lhsExpr := range initAssignStmt.Lhs {
		rhsExpr := initAssignStmt.Rhs[i]
		vIdent, ok := lhsExpr.(*ast.Ident)
		if !ok {
			return -1
		}
		v, ok := b.info.Defs[vIdent].(*types.Var)
		if !ok {
			return -1
		}
		valLit, ok := rhsExpr.(*ast.BasicLit)
		if !ok {
			return -1
		}
		val := constant.MakeFromLiteral(valLit.Value, valLit.Kind, 0)

		loopVars = append(loopVars, loopVarInfo{
			v,
			vIdent.Name,
			val,
		})
	}

	loopBodyOk := true
	ast.Inspect(forStmt.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				loopBodyOk = false
				return false
			}
		case *ast.AssignStmt:
			for _, expr := range node.Lhs {
				vIdent, ok := expr.(*ast.Ident)
				if !ok {
					continue
				}
				v, ok := b.info.Uses[vIdent]
				if !ok {
					continue
				}
				for _, inf := range loopVars {
					if v == inf.obj {
						loopBodyOk = false
						return false
					}
				}
			}
		}
		return true
	})
	if !loopBodyOk {
		return -1
	}

	const MaxIterCount int = 1000000
	iterCount := 0
	for ; iterCount < MaxIterCount; iterCount++ {
		condVal, ok := b.evalLoopVarExpr(forStmt.Cond, loopVars)
		if !ok {
			return -1
		}
		if constant.BoolVal(condVal) == false {
			break
		}
		ok = b.evalLoopVarStmt(forStmt.Post, loopVars)
		if !ok {
			return -1
		}
	}
	if iterCount == MaxIterCount {
		return -1
	}
	return iterCount
}

func (b *builder) evalLoopVarStmt(stmt ast.Stmt, loopVars []loopVarInfo) (ok bool) {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		if stmt.Tok != token.ASSIGN || len(stmt.Lhs) != len(stmt.Rhs) {
			return false
		}
		for i, lhsExpr := range stmt.Lhs {
			vIdent, ok := lhsExpr.(*ast.Ident)
			if !ok {
				return false
			}
			loopVarIndex := -1
			for j, inf := range loopVars {
				if inf.obj == b.info.Uses[vIdent] {
					loopVarIndex = j
				}
			}
			if loopVarIndex == -1 {
				return false
			}
			rhsExpr := stmt.Rhs[i]
			val, ok := b.evalLoopVarExpr(rhsExpr, loopVars)
			if !ok {
				return false
			}
			loopVars[loopVarIndex].val = val
		}
		return true
	case *ast.IncDecStmt:
		vIdent, ok := stmt.X.(*ast.Ident)
		if !ok {
			return false
		}
		loopVarIndex := -1
		for i, inf := range loopVars {
			if inf.obj == b.info.Uses[vIdent] {
				loopVarIndex = i
			}
		}
		if loopVarIndex == -1 {
			return false
		}
		x := loopVars[loopVarIndex].val
		if stmt.Tok == token.INC {
			x = constant.BinaryOp(x, token.ADD, constant.MakeInt64(1))
		} else if stmt.Tok == token.DEC {
			x = constant.BinaryOp(x, token.SUB, constant.MakeInt64(1))
		} else {
			return false
		}
		loopVars[loopVarIndex].val = x
		return true
	default:
		return false
	}
}

func (b *builder) evalLoopVarExpr(expr ast.Expr, loopVars []loopVarInfo) (val constant.Value, ok bool) {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		return constant.MakeFromLiteral(expr.Value, expr.Kind, 0), true
	case *ast.Ident:
		for _, inf := range loopVars {
			if inf.obj == b.info.Uses[expr] {
				return inf.val, true
			}
		}
		typeAndValue, ok := b.info.Types[expr]
		if !ok || typeAndValue.Value == nil {
			return nil, false
		}
		return typeAndValue.Value, true
	case *ast.BinaryExpr:
		x, ok := b.evalLoopVarExpr(expr.X, loopVars)
		if !ok {
			return nil, false
		}
		y, ok := b.evalLoopVarExpr(expr.Y, loopVars)
		if !ok {
			return nil, false
		}
		if expr.Op == token.EQL ||
			expr.Op == token.NEQ ||
			expr.Op == token.LSS ||
			expr.Op == token.GTR ||
			expr.Op == token.LEQ ||
			expr.Op == token.GEQ {
			return constant.MakeBool(constant.Compare(x, expr.Op, y)), true
		}
		return nil, false
	default:
		return nil, false
	}
}
