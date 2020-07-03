package builder

import (
	"fmt"
	"go/ast"

	"github.com/arneph/toph/ir"
)

func (b *builder) findMutex(mutexExpr ast.Expr, ctx *context) ir.LValue {
	rv := b.processExpr(mutexExpr, ctx)
	lv, ok := rv.(ir.LValue)
	if !ok || lv == nil {
		p := b.fset.Position(mutexExpr.Pos())
		mutexExprStr := b.nodeToString(mutexExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve mutex expr: %v", p, mutexExprStr))
		return nil
	}
	if lv.Type() != ir.MutexType {
		structType := lv.Type().(*ir.StructType)
		embeddedFields, ok := structType.FindEmbeddedFieldOfType(ir.MutexType)
		if !ok {
			p := b.fset.Position(mutexExpr.Pos())
			mutexExprStr := b.nodeToString(mutexExpr)
			b.addWarning(fmt.Errorf("%v: could not resolve mutex expr: %v", p, mutexExprStr))
			return nil
		}
		for _, field := range embeddedFields {
			lv = ir.NewFieldSelection(lv, field)
		}
	}
	return lv
}

func (b *builder) findWaitGroup(waitGroupExpr ast.Expr, ctx *context) ir.LValue {
	rv := b.processExpr(waitGroupExpr, ctx)
	lv, ok := rv.(ir.LValue)
	if !ok || lv == nil {
		p := b.fset.Position(waitGroupExpr.Pos())
		waitGroupExprStr := b.nodeToString(waitGroupExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve wait group expr: %v", p, waitGroupExprStr))
		return nil
	}
	if lv.Type() != ir.WaitGroupType {
		structType := lv.Type().(*ir.StructType)
		embeddedFields, ok := structType.FindEmbeddedFieldOfType(ir.WaitGroupType)
		if !ok {
			p := b.fset.Position(waitGroupExpr.Pos())
			waitGroupExprStr := b.nodeToString(waitGroupExpr)
			b.addWarning(fmt.Errorf("%v: could not resolve wait group expr: %v", p, waitGroupExprStr))
			return nil
		}
		for _, field := range embeddedFields {
			lv = ir.NewFieldSelection(lv, field)
		}
	}
	return lv
}
