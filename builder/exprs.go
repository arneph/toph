package builder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/arneph/toph/ir"
)

func (b *builder) processExprs(exprs []ast.Expr, ctx context) map[int]*ir.Variable {
	results := make(map[int]*ir.Variable)

	for i, expr := range exprs {
		v := b.processExpr(expr, ctx)
		if v != nil {
			results[i] = v
		}
	}

	return results
}

func (b *builder) processExpr(expr ast.Expr, ctx context) *ir.Variable {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return nil
	case *ast.BinaryExpr:
		b.processExprs([]ast.Expr{e.X, e.Y}, ctx)
		return nil
	case *ast.CallExpr:
		results := make(map[int]*ir.Variable)
		b.processCallExpr(e, results, ctx)
		if len(results) == 0 {
			return nil
		} else if len(results) == 1 {
			return results[0]
		} else {
			panic("attempted to use call expr as single expr")
		}
	case *ast.CompositeLit:
		b.processExprs(append([]ast.Expr{e.Type}, e.Elts...), ctx)
		return nil
	case *ast.Ellipsis:
		b.processExpr(e.Elt, ctx)
		return nil
	case *ast.FuncLit:
		f := b.processFuncLit(e, ctx)
		v := ir.NewVariable("", ir.FuncType, f.FuncValue())
		ctx.body.Scope().AddVariable(v)
		return v
	case *ast.Ident:
		return b.processIdent(e, ctx)
	case *ast.IndexExpr:
		b.processExprs([]ast.Expr{e.X, e.Index}, ctx)
		return nil
	case *ast.KeyValueExpr:
		b.processExprs([]ast.Expr{e.Key, e.Value}, ctx)
		return nil
	case *ast.ParenExpr:
		b.processExpr(e.X, ctx)
		return nil
	case *ast.SelectorExpr:
		b.processExpr(e.X, ctx)
		return nil
	case *ast.SliceExpr:
		b.processExprs([]ast.Expr{e.X, e.Low, e.High, e.Max}, ctx)
		return nil
	case *ast.StarExpr:
		b.processExpr(e.X, ctx)
		return nil
	case *ast.TypeAssertExpr:
		b.processExprs([]ast.Expr{e.X, e.Type}, ctx)
		return nil
	case *ast.UnaryExpr:
		if e.Op == token.ARROW {
			v := b.processExpr(e.X, ctx)
			if v == nil {
				p := b.fset.Position(e.X.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve channel expr: %v", p, e.X))
			}

			ctx.body.AddStmt(ir.NewChanOpStmt(v, ir.Receive))
		} else {
			b.processExpr(e.X, ctx)
		}
		return nil

	case
		*ast.ArrayType,
		*ast.ChanType,
		*ast.FuncType,
		*ast.InterfaceType,
		*ast.MapType,
		*ast.StructType:
		return nil
	default:
		p := b.fset.Position(e.Pos())
		b.addWarning(fmt.Errorf("%v: ignoring %T expression", p, e))
		return nil
	}
}

func (b *builder) processIdent(ident *ast.Ident, ctx context) *ir.Variable {
	v, s := ctx.body.Scope().GetVariable(ident.Name)
	if v == nil {
		// Name not found in scope
		return nil
	}

	varType, ok := b.info.Uses[ident].(*types.Var)
	if !ok {
		varType, ok = b.info.Defs[ident].(*types.Var)
	}
	if !ok {
		p := b.fset.Position(ident.Pos())
		b.addWarning(
			fmt.Errorf("%v: identifier does not refer to known variable with name: %s",
				p, ident.Name))

		return nil
	}
	u, ok := b.varTypes[varType]
	if !ok || v != u {
		p := b.fset.Position(ident.Pos())
		b.addWarning(
			fmt.Errorf("%v: identifier does not refer to known variable with name: %s",
				p, ident.Name))

		return nil
	}

	if s == b.program.Scope() {
		return v
	}

	w := v
	for _, f := range ctx.enclosingFuncs {
		fScope := f.Scope()
		if !s.IsSuperScopeOf(fScope) {
			continue
		}

		w = f.GetCapturer(v)
		if w == nil {
			w = ir.NewVariable(v.Name(), v.Type(), -1)
			f.AddCapture(v, w)
		}
		v = w
	}

	return w
}

func (b *builder) processMakeExpr(callExpr *ast.CallExpr, result *ir.Variable, ctx context) {
	_, ok := callExpr.Args[0].(*ast.ChanType)
	if !ok {
		return
	}

	var bufferSize int

	if len(callExpr.Args) > 1 {
		a := callExpr.Args[1]
		l, ok := a.(*ast.BasicLit)
		if !ok {
			p := b.fset.Position(a.Pos())
			b.addWarning(fmt.Errorf("%v: can not process buffer size: %s", p, a))

		} else {
			n, err := strconv.Atoi(l.Value)
			if err != nil {
				p := b.fset.Position(l.Pos())
				b.addWarning(fmt.Errorf("%v: can not process buffer size: %s", p, l.Value))
			} else {
				bufferSize = n
			}
		}
	}

	makeStmt := ir.NewMakeChanStmt(result, bufferSize)
	ctx.body.AddStmt(makeStmt)
}

func (b *builder) processCloseExpr(callExpr *ast.CallExpr, ctx context) {
	v := b.processExpr(callExpr.Args[0], ctx)
	if v == nil {
		return
	}

	closeStmt := ir.NewChanOpStmt(v, ir.Close)
	ctx.body.AddStmt(closeStmt)
}
