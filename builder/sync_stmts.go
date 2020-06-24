package builder

import (
	"fmt"
	"go/ast"

	"github.com/arneph/toph/ir"
)

func (b *builder) findMutex(mutexExpr ast.Expr, ctx *context) *ir.Variable {
	rv := b.processExpr(mutexExpr, ctx)
	v, ok := rv.(*ir.Variable)
	if !ok || v == nil {
		p := b.fset.Position(mutexExpr.Pos())
		mutexExprStr := b.nodeToString(mutexExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve mutex expr: %v", p, mutexExprStr))
		return nil
	}
	return v
}

func (b *builder) findWaitGroup(waitGroupExpr ast.Expr, ctx *context) *ir.Variable {
	rv := b.processExpr(waitGroupExpr, ctx)
	v, ok := rv.(*ir.Variable)
	if !ok || v == nil {
		p := b.fset.Position(waitGroupExpr.Pos())
		waitGroupExprStr := b.nodeToString(waitGroupExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve wait group expr: %v", p, waitGroupExprStr))
		return nil
	}
	return v
}
