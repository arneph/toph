package builder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processExprs(exprs []ast.Expr, ctx *context) map[int]ir.RValue {
	results := make(map[int]ir.RValue)

	for i, expr := range exprs {
		v := b.processExpr(expr, ctx)
		if v != nil {
			results[i] = v
		}
	}

	return results
}

func (b *builder) processExpr(expr ast.Expr, ctx *context) ir.RValue {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return nil
	case *ast.BinaryExpr:
		b.processExprs([]ast.Expr{e.X, e.Y}, ctx)
		return nil
	case *ast.CallExpr:
		v := b.processCallExpr(e, ctx)
		if v != nil {
			return ir.RValue(v)
		}
		return nil
	case *ast.CompositeLit:
		if e.Type != nil {
			b.processExpr(e.Type, ctx)
		}
		b.processExprs(e.Elts, ctx)
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
		b.processExpr(e.X, ctx)
		if e.Low != nil {
			b.processExpr(e.Low, ctx)
		}
		if e.High != nil {
			b.processExpr(e.High, ctx)
		}
		if e.Max != nil {
			b.processExpr(e.Max, ctx)
		}
		return nil
	case *ast.StarExpr:
		b.processExpr(e.X, ctx)
		return nil
	case *ast.TypeAssertExpr:
		b.processExprs([]ast.Expr{e.X, e.Type}, ctx)
		return nil
	case *ast.UnaryExpr:
		if e.Op == token.ARROW {
			b.processReceiveExpr(e, true, ctx)
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
		if e == nil {
			panic("found nil expression in AST")
		}
		p := b.fset.Position(e.Pos())
		b.addWarning(fmt.Errorf("%v: ignoring %T expression", p, e))
		return nil
	}
}

func (b *builder) processIdent(ident *ast.Ident, ctx *context) ir.RValue {
	if ident.Name == "nil" {
		return ir.Value(-1)
	}

	v, s := ctx.body.Scope().GetVariable(ident.Name)
	if v == nil {
		return nil
	}

	obj := b.info.ObjectOf(ident)
	if obj == nil {
		p := b.fset.Position(ident.Pos())
		panic(fmt.Errorf("%v: types.Object for identifier is nil: %s", p, ident.Name))
	}
	switch obj := obj.(type) {
	case *types.Var:
		u := b.varTypes[obj]
		if u == nil {
			return nil
		} else if u != v {
			p := b.fset.Position(ident.Pos())
			b.addWarning(
				fmt.Errorf("%v: identifier does not refer to known variable with name: %s",
					p, ident.Name))
			return nil
		}

		if s != b.program.Scope() && s.IsSuperScopeOf(ctx.currentFunc().Scope()) {
			v.SetCaptured(true)
		}
		return v

	case *types.Func:
		f := b.funcTypes[obj]
		if f.FuncValue() != v.InitialValue() {
			p := b.fset.Position(ident.Pos())
			b.addWarning(
				fmt.Errorf("%v: identifier does not refer to known variable with name: %s",
					p, ident.Name))
			return nil
		}
		return v
	case *types.Const, *types.PkgName, *types.TypeName:
		return nil
	default:
		p := b.fset.Position(ident.Pos())
		panic(fmt.Errorf("%v: unexpected types.Object type: %T", p, obj))
	}
}
