package builder

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

type staticVarInfo struct {
	obj  *types.Var
	name string
	val  constant.Value
}

func (b *builder) staticIntEval(expr ast.Expr, ctx *context) (int, bool) {
	val, ok := b.staticExprEval(expr, nil, ctx)
	if !ok {
		return 0, false
	}
	val = constant.ToInt(val)
	if val.Kind() == constant.Unknown {
		return 0, false
	}
	x, _ := constant.Int64Val(val)
	return int(x), true
}

func (b *builder) staticExprEval(expr ast.Expr, vars []staticVarInfo, ctx *context) (val constant.Value, ok bool) {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		return constant.MakeFromLiteral(expr.Value, expr.Kind, 0), true

	case *ast.Ident:
		for _, inf := range vars {
			if inf.obj == ctx.typesInfo.Uses[expr] {
				return inf.val, true
			}
		}
		typeAndValue, ok := ctx.typesInfo.Types[expr]
		if !ok || typeAndValue.Value == nil {
			return nil, false
		}
		return typeAndValue.Value, true

	case *ast.UnaryExpr:
		x, ok := b.staticExprEval(expr.X, vars, ctx)
		if !ok {
			return nil, false
		}
		return constant.UnaryOp(expr.Op, x, 0), true

	case *ast.BinaryExpr:
		x, ok := b.staticExprEval(expr.X, vars, ctx)
		if !ok {
			return nil, false
		}
		y, ok := b.staticExprEval(expr.Y, vars, ctx)
		if !ok {
			return nil, false
		}
		switch expr.Op {
		case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
			return constant.MakeBool(constant.Compare(x, expr.Op, y)), true
		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.AND, token.OR, token.XOR, token.AND_NOT:
			return constant.BinaryOp(x, expr.Op, y), true
		default:
			return nil, false
		}

	default:
		return nil, false
	}
}

func (b *builder) staticStmtEval(stmt ast.Stmt, vars []staticVarInfo, ctx *context) (ok bool) {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		if stmt.Tok == token.DEFINE || len(stmt.Lhs) != len(stmt.Rhs) {
			return false
		}
		for i, lhsExpr := range stmt.Lhs {
			vIdent, ok := lhsExpr.(*ast.Ident)
			if !ok {
				return false
			}
			loopVarIndex := -1
			for j, inf := range vars {
				if inf.obj == ctx.typesInfo.Uses[vIdent] {
					loopVarIndex = j
				}
			}
			if loopVarIndex == -1 {
				return false
			}
			rhsExpr := stmt.Rhs[i]
			var val constant.Value
			if stmt.Tok != token.ASSIGN {
				binExpr := ast.BinaryExpr{
					X:  stmt.Lhs[0],
					Op: stmt.Tok - (token.ADD_ASSIGN - token.ADD),
					Y:  stmt.Rhs[0],
				}
				binExpr.X = stmt.Lhs[0]
				val, ok = b.staticExprEval(&binExpr, vars, ctx)
			} else {
				val, ok = b.staticExprEval(rhsExpr, vars, ctx)
			}
			if !ok {
				return false
			}
			vars[loopVarIndex].val = val
		}
		return true

	case *ast.IncDecStmt:
		vIdent, ok := stmt.X.(*ast.Ident)
		if !ok {
			return false
		}
		loopVarIndex := -1
		for i, inf := range vars {
			if inf.obj == ctx.typesInfo.Uses[vIdent] {
				loopVarIndex = i
			}
		}
		if loopVarIndex == -1 {
			return false
		}
		x := vars[loopVarIndex].val
		if stmt.Tok == token.INC {
			x = constant.BinaryOp(x, token.ADD, constant.MakeInt64(1))
		} else if stmt.Tok == token.DEC {
			x = constant.BinaryOp(x, token.SUB, constant.MakeInt64(1))
		} else {
			return false
		}
		vars[loopVarIndex].val = x
		return true
	default:
		return false
	}
}

func (b *builder) basicVarIsReadOnlyInBody(astBody *ast.BlockStmt, typesVar *types.Var, ctx *context) bool {
	isReadOnly := true
	ast.Inspect(astBody, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.UnaryExpr:
			if node.Op != token.AND {
				break
			}
			xIdent, ok := node.X.(*ast.Ident)
			if !ok || ctx.typesInfo.Uses[xIdent] != typesVar {
				break
			}
			isReadOnly = false
			return false
		case *ast.AssignStmt:
			for _, expr := range node.Lhs {
				vIdent, ok := expr.(*ast.Ident)
				if !ok || ctx.typesInfo.Uses[vIdent] != typesVar {
					continue
				}
				isReadOnly = false
				return false
			}
		case *ast.IncDecStmt:
			xIdent, ok := node.X.(*ast.Ident)
			if !ok || ctx.typesInfo.Uses[xIdent] != typesVar {
				break
			}
			isReadOnly = false
			return false
		}
		return true
	})
	return isReadOnly
}

func (b *builder) staticForLoopBoundsEval(forStmt *ast.ForStmt, ctx *context) int {
	if forStmt.Init == nil || forStmt.Cond == nil || forStmt.Post == nil {
		return -1
	}

	initAssignStmt, ok := forStmt.Init.(*ast.AssignStmt)
	if !ok ||
		initAssignStmt.Tok != token.DEFINE ||
		len(initAssignStmt.Lhs) != len(initAssignStmt.Rhs) {
		return -1
	}

	var loopVars []staticVarInfo
	for i, lhsExpr := range initAssignStmt.Lhs {
		rhsExpr := initAssignStmt.Rhs[i]
		vIdent, ok := lhsExpr.(*ast.Ident)
		if !ok {
			return -1
		}
		v, ok := ctx.typesInfo.Defs[vIdent].(*types.Var)
		if !ok {
			return -1
		}
		valLit, ok := rhsExpr.(*ast.BasicLit)
		if !ok {
			return -1
		}
		val := constant.MakeFromLiteral(valLit.Value, valLit.Kind, 0)

		loopVars = append(loopVars, staticVarInfo{
			v,
			vIdent.Name,
			val,
		})
	}

	for _, info := range loopVars {
		if !b.basicVarIsReadOnlyInBody(forStmt.Body, info.obj, ctx) {
			return -1
		}
	}

	const MaxIterCount int = 1000000
	iterCount := 0
	for ; iterCount < MaxIterCount; iterCount++ {
		condVal, ok := b.staticExprEval(forStmt.Cond, loopVars, ctx)
		if !ok {
			return -1
		}
		if constant.BoolVal(condVal) == false {
			break
		}
		ok = b.staticStmtEval(forStmt.Post, loopVars, ctx)
		if !ok {
			return -1
		}
	}
	if iterCount == MaxIterCount {
		return -1
	}
	return iterCount
}
