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
		b.addWarning(fmt.Errorf("%v: could not resolve mutex expr: %v", p, mutexExpr))
		return nil
	}
	return v
}

func (b *builder) findWaitGroup(waitGroupExpr ast.Expr, ctx *context) *ir.Variable {
	rv := b.processExpr(waitGroupExpr, ctx)
	v, ok := rv.(*ir.Variable)
	if !ok || v == nil {
		p := b.fset.Position(waitGroupExpr.Pos())
		b.addWarning(fmt.Errorf("%v: could not resolve wait group expr: %v", p, waitGroupExpr))
		return nil
	}
	return v
}

func (b *builder) processMutexOpExpr(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) {
	selExpr := callExpr.Fun.(*ast.SelectorExpr)
	mutexVar := b.findMutex(selExpr.X, ctx)
	if mutexVar == nil {
		return
	}

	var mutexOp ir.MutexOp
	switch selExpr.Sel.Name {
	case "Lock":
		mutexOp = ir.Lock
	case "Unlock":
		mutexOp = ir.Unlock
	case "RLock":
		mutexOp = ir.RLock
	case "RUnlock":
		mutexOp = ir.RUnlock
	default:
		b.addWarning(fmt.Errorf("unexpected mutex method: %s", selExpr.Sel.Name))
		return
	}

	mutexOpStmt := ir.NewMutexOpStmt(mutexVar, mutexOp, callKind, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(mutexOpStmt)
}

func (b *builder) processWaitGroupOpExpr(callExpr *ast.CallExpr, callKind ir.CallKind, ctx *context) {
	selExpr := callExpr.Fun.(*ast.SelectorExpr)
	waitGroupVar := b.findWaitGroup(selExpr.X, ctx)
	if waitGroupVar == nil {
		return
	}

	var waitGroupOp ir.WaitGroupOp
	delta := 1
	switch selExpr.Sel.Name {
	case "Add":
		waitGroupOp = ir.Add
		a := callExpr.Args[0]
		res, ok := b.evalIntExpr(a, ctx)
		if !ok {
			p := b.fset.Position(a.Pos())
			b.addWarning(fmt.Errorf("%v: can not process sync.WaitGroup.Add argument: %s", p, a))
		} else {
			delta = res
		}
	case "Done":
		waitGroupOp = ir.Add
		delta = -1
	case "Wait":
		waitGroupOp = ir.Wait
	default:
		b.addWarning(fmt.Errorf("unexpected wait group method: %s", selExpr.Sel.Name))
		return
	}

	waitGroupOpStmt := ir.NewWaitGroupOpStmt(waitGroupVar, waitGroupOp, delta, callKind, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(waitGroupOpStmt)
}
