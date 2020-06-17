package builder

import (
	"go/ast"
	"strconv"
)

func (b *builder) evalIntExpr(expr ast.Expr, ctx *context) (int, bool) {
	l, ok := expr.(*ast.BasicLit)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(l.Value)
	if err != nil {
		return 0, false
	}
	return n, true
}
