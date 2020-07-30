package builder

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/arneph/toph/ir"
)

func (b *builder) processAssignStmt(stmt *ast.AssignStmt, ctx *context) {
	// Create newly defined variables:
	if stmt.Tok == token.DEFINE {
		for _, expr := range stmt.Lhs {
			ident, ok := expr.(*ast.Ident)
			if !ok {
				continue
			}
			b.processVarDefinition(ident, false, ctx)
		}
	}

	b.processAssignments(stmt.Lhs, stmt.Rhs, ctx)
}

func (b *builder) processAssignments(lhsExprs []ast.Expr, rhsExprs []ast.Expr, ctx *context) {
	// Handle Rhs expressions:
	rhs := b.processExprs(rhsExprs, ctx)

	// Handle Lhs expressions:
	lhs := make(map[int]ir.LValue)
	requiresCopy := make(map[int]bool)
	for i, expr := range lhsExprs {
		irVal := b.processExpr(expr, ctx)
		if irVal == nil {
			continue
		} else if irContainerAccess, ok := irVal.(*ir.ContainerAccess); ok {
			irContainerAccess.SetKind(ir.Write)
		}
		irVar := irVal.(ir.LValue)
		irType := irVar.Type()
		if irType == ir.MutexType {
			p := b.fset.Position(expr.Pos())
			b.addWarning(fmt.Errorf("%v: can not assign sync.Mutex or sync.RWMutex", p))
			continue
		} else if irType == ir.WaitGroupType {
			p := b.fset.Position(expr.Pos())
			b.addWarning(fmt.Errorf("%v: can not assign sync.WaitGroup", p))
			continue
		}
		lhs[i] = irVar
		typesType := ctx.typesInfo.TypeOf(expr)
		if _, ok := irType.(*ir.StructType); ok {
			requiresCopy[i] = !b.isPointer(typesType)
		} else if irContainerType, ok := irType.(*ir.ContainerType); ok && irContainerType.Kind() == ir.Array {
			requiresCopy[i] = !b.isPointer(typesType)
		} else {
			requiresCopy[i] = false
		}
	}

	// Create assignment statements:
	for i := 0; i < len(lhsExprs); i++ {
		var lhsExpr, rhsExpr ast.Expr
		lhsExpr = lhsExprs[i]
		if len(rhsExprs) == 1 {
			rhsExpr = rhsExprs[0]
		} else {
			rhsExpr = rhsExprs[i]
		}
		l := lhs[i]
		r := rhs[i]
		if l == nil && r == nil {
			continue
		} else if l == nil && r == ir.Value(-1) {
			continue
		} else if l == nil {
			if ident, ok := lhsExpr.(*ast.Ident); ok && ident.Name == "_" {
				continue
			}
			p := b.fset.Position(lhsExpr.Pos())
			lhsExprStr := b.nodeToString(lhsExpr)
			b.addWarning(fmt.Errorf("%v: could not handle lhs of assignment: %s", p, lhsExprStr))
			continue
		} else if r == nil {
			p := b.fset.Position(rhsExpr.Pos())
			rhsExprStr := b.nodeToString(rhsExpr)
			b.addWarning(
				fmt.Errorf("%v: could not handle rhs of assignment: %s", p, rhsExprStr))
			continue
		}
		c := requiresCopy[i]
		assignStmt := ir.NewAssignStmt(r, l, c, lhsExpr.Pos(), lhsExpr.End())
		ctx.body.AddStmt(assignStmt)
	}
}
