package builder

import (
	"go/ast"

	"github.com/arneph/toph/ir"
)

type context struct {
	cmap ast.CommentMap

	body           *ir.Body
	enclosingFuncs []*ir.Func

	enclosingStmts      []ir.Stmt
	enclosingStmtLabels map[string]ir.Stmt
}

func newContext(cmap ast.CommentMap, f *ir.Func) *context {
	ctx := new(context)
	ctx.cmap = cmap
	ctx.body = f.Body()
	ctx.enclosingFuncs = []*ir.Func{f}
	ctx.enclosingStmts = []ir.Stmt{}
	ctx.enclosingStmtLabels = make(map[string]ir.Stmt)

	return ctx
}

func (c *context) currentFunc() *ir.Func {
	n := len(c.enclosingFuncs)
	return c.enclosingFuncs[n-1]
}

func (c *context) currentLoop() ir.Loop {
	if len(c.enclosingStmts) == 0 {
		return nil
	}
	for i := len(c.enclosingStmts) - 1; i >= 0; i-- {
		loop, ok := c.enclosingStmts[i].(ir.Loop)
		if ok {
			return loop
		}
	}
	return nil
}

func (c *context) currentLabeledLoop(label string) ir.Loop {
	stmt, ok := c.enclosingStmtLabels[label]
	if !ok {
		return nil
	}
	loop, ok := stmt.(ir.Loop)
	if !ok {
		return nil
	}
	return loop
}

func (c *context) subContextForBody(stmt ir.Stmt, label string, containedBody *ir.Body) *context {
	ctx := new(context)
	ctx.cmap = c.cmap
	ctx.body = containedBody
	ctx.enclosingFuncs = c.enclosingFuncs
	ctx.enclosingStmts = append(c.enclosingStmts, stmt)
	ctx.enclosingStmtLabels = make(map[string]ir.Stmt)
	for l, s := range c.enclosingStmtLabels {
		ctx.enclosingStmtLabels[l] = s
	}
	if label != "" {
		ctx.enclosingStmtLabels[label] = stmt
	}

	return ctx
}

func (c *context) subContextForFunc(containedFunc *ir.Func) *context {
	ctx := new(context)
	ctx.cmap = c.cmap
	ctx.body = containedFunc.Body()
	ctx.enclosingFuncs = append(c.enclosingFuncs, containedFunc)
	ctx.enclosingStmts = []ir.Stmt{}
	ctx.enclosingStmtLabels = make(map[string]ir.Stmt)

	return ctx
}
