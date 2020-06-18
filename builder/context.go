package builder

import (
	"go/ast"

	"github.com/arneph/toph/ir"
)

type context struct {
	pkg  string
	cmap ast.CommentMap

	body           *ir.Body
	enclosingFuncs []*ir.Func

	enclosingStmts      []ir.Stmt
	enclosingStmtLabels map[string]ir.Stmt
}

func newContext(pkg string, cmap ast.CommentMap, f *ir.Func) *context {
	ctx := new(context)
	ctx.pkg = pkg
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

func (c *context) findBreakable(label string) ir.Stmt {
	if label != "" {
		stmt := c.enclosingStmtLabels[label]
		switch stmt.(type) {
		case *ir.ForStmt, *ir.RangeStmt, *ir.SwitchStmt:
			return stmt
		default:
			return nil
		}
	}
	if len(c.enclosingStmts) == 0 {
		return nil
	}
	for i := len(c.enclosingStmts) - 1; i >= 0; i-- {
		stmt := c.enclosingStmts[i]
		switch stmt.(type) {
		case *ir.ForStmt, *ir.RangeStmt, *ir.SwitchStmt:
			return stmt
		}
	}
	return nil
}

func (c *context) findContinuable(label string) ir.Stmt {
	if label != "" {
		stmt := c.enclosingStmtLabels[label]
		switch stmt.(type) {
		case *ir.ForStmt, *ir.RangeStmt:
			return stmt
		default:
			return nil
		}
	}
	if len(c.enclosingStmts) == 0 {
		return nil
	}
	for i := len(c.enclosingStmts) - 1; i >= 0; i-- {
		stmt := c.enclosingStmts[i]
		switch stmt.(type) {
		case *ir.ForStmt, *ir.RangeStmt:
			return stmt
		}
	}
	return nil
}

func (c *context) subContextForBody(stmt ir.Stmt, label string, containedBody *ir.Body) *context {
	ctx := new(context)
	ctx.pkg = c.pkg
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
	ctx.pkg = c.pkg
	ctx.cmap = c.cmap
	ctx.body = containedFunc.Body()
	ctx.enclosingFuncs = append(c.enclosingFuncs, containedFunc)
	ctx.enclosingStmts = []ir.Stmt{}
	ctx.enclosingStmtLabels = make(map[string]ir.Stmt)

	return ctx
}
