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
		result := b.processCallExpr(e, ctx)[0]
		if result == nil {
			return ir.RValue(nil)
		}
		return result
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
		return b.processFuncLit(e, ctx).FuncValue()
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
		_, sel := b.processSelectorExpr(e, ctx)
		return sel
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
	if ident.Name == "_" {
		return nil
	} else if ident.Name == "nil" {
		return ir.Value(-1)
	}

	usedTypesObj := b.typesInfo.ObjectOf(ident)
	if usedTypesObj == nil {
		p := b.fset.Position(ident.Pos())
		b.addWarning(fmt.Errorf("%v: types.Object for identifier is nil: %s", p, ident.Name))
		return nil
	}
	typesType := usedTypesObj.Type()
	if typesType == nil {
		p := b.fset.Position(ident.Pos())
		b.addWarning(fmt.Errorf("%v: types.Type for identifier is nil: %s", p, ident.Name))
		return nil
	}
	_, _, ok := typesTypeToIrType(typesType)
	if !ok {
		return nil
	}

	substitute := b.getSubstitute(usedTypesObj)
	if substitute != nil {
		return substitute
	}

	return b.processDefinedTypesObj(usedTypesObj, ctx)
}

func (b *builder) processSelectorExpr(selExpr *ast.SelectorExpr, ctx *context) (x ir.RValue, sel ir.RValue) {
	typesSelection, ok := b.typesInfo.Selections[selExpr]
	if !ok {
		// Assume *ast.SelectorExpr is for qualified identifier (package.identifier)
		xIdent := selExpr.X.(*ast.Ident)
		xTypesObj := b.typesInfo.ObjectOf(xIdent)
		_ = xTypesObj.(*types.PkgName)

		return nil, b.processIdent(selExpr.Sel, ctx)
	}

	xVal := b.processExpr(selExpr.X, ctx)

	usedTypesObj := typesSelection.Obj()

	selVal := b.processDefinedTypesObj(usedTypesObj, ctx)

	return xVal, selVal
}

func (b *builder) processDefinedTypesObj(definedTypesObj types.Object, ctx *context) ir.RValue {
	switch definedTypesObj := definedTypesObj.(type) {
	case *types.Var:
		v := b.vars[definedTypesObj]
		if v == nil {
			return nil
		}

		s := v.Scope()
		if s != b.program.Scope() && s.IsSuperScopeOf(ctx.currentFunc().Scope()) {
			v.SetCaptured(true)
		}

		return v

	case *types.Func:
		f := b.funcs[definedTypesObj]
		if f == nil {
			return nil
		}
		return f.FuncValue()

	case *types.Const, *types.TypeName:
		return nil

	default:
		p := b.fset.Position(definedTypesObj.Pos())
		b.addWarning(fmt.Errorf("%v: unexpected types.Object type: %T", p, definedTypesObj))
		return nil
	}
}
