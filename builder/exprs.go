package builder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processExprs(exprs []ast.Expr, ctx *context) map[int]ir.RValue {
	if len(exprs) == 1 {
		callExpr, ok := exprs[0].(*ast.CallExpr)
		if ok {
			resultVals := make(map[int]ir.RValue)
			resultVars := b.processCallExprWithCallKind(callExpr, ir.Call, ctx)
			for i, resultVar := range resultVars {
				resultVals[i] = resultVar
			}
			return resultVals
		}
	}

	resultVals := make(map[int]ir.RValue)
	for i, expr := range exprs {
		v := b.processExpr(expr, ctx)
		if v != nil {
			resultVals[i] = v
		}
	}
	return resultVals
}

func (b *builder) processExpr(expr ast.Expr, ctx *context) ir.RValue {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return nil
	case *ast.BinaryExpr:
		b.processExpr(e.X, ctx)
		b.processExpr(e.Y, ctx)
		return nil
	case *ast.CallExpr:
		result := b.processCallExpr(e, ctx)[0]
		if result == nil {
			return ir.RValue(nil)
		}
		return result
	case *ast.CompositeLit:
		result := b.processCompositeLit(e, ctx)
		if result == nil {
			return ir.RValue(nil)
		}
		return result
	case *ast.Ellipsis:
		b.processExpr(e.Elt, ctx)
		return nil
	case *ast.FuncLit:
		return b.processFuncLit(e, ctx).FuncValue()
	case *ast.Ident:
		return b.processIdent(e, ctx)
	case *ast.IndexExpr:
		return b.procesIndexExpr(e, ctx)
	case *ast.KeyValueExpr:
		b.processExpr(e.Key, ctx)
		b.processExpr(e.Value, ctx)
		return nil
	case *ast.ParenExpr:
		b.processExpr(e.X, ctx)
		return nil
	case *ast.SelectorExpr:
		return b.processSelectorExpr(e, ctx)
	case *ast.SliceExpr:
		if e.Low != nil {
			b.processExpr(e.Low, ctx)
		}
		if e.High != nil {
			b.processExpr(e.High, ctx)
		}
		if e.Max != nil {
			b.processExpr(e.Max, ctx)
		}
		result := b.processExpr(e.X, ctx)
		if result != nil {
			if e.Low != nil || e.High != nil || e.Max != nil {
				p := b.fset.Position(e.Pos())
				eStr := b.nodeToString(e)
				b.addWarning(fmt.Errorf("%v: ignoring indices of slice expression: %s", p, eStr))
			}
		}
		return result
	case *ast.StarExpr:
		return b.processExpr(e.X, ctx)
	case *ast.TypeAssertExpr:
		b.processExpr(e.X, ctx)
		b.processExpr(e.Type, ctx)
		return nil
	case *ast.UnaryExpr:
		switch e.Op {
		case token.ARROW:
			b.processReceiveExpr(e, true, ctx)
			return nil
		case token.AND:
			return b.processExpr(e.X, ctx)
		default:
			b.processExpr(e.X, ctx)
			return nil
		}
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
		eStr := b.nodeToString(e)
		b.addWarning(fmt.Errorf("%v: ignoring %T expression: %s", p, e, eStr))
		return nil
	}
}

func (b *builder) processIdent(ident *ast.Ident, ctx *context) ir.RValue {
	if ident.Name == "_" {
		return nil
	} else if ident.Name == "nil" {
		return ir.Nil
	}

	usedTypesObj := ctx.typesInfo.ObjectOf(ident)
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
	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return nil
	}

	substitute := b.getSubstitute(usedTypesObj)
	if substitute != nil {
		return substitute
	}

	switch usedTypesObj := usedTypesObj.(type) {
	case *types.Var:
		v := b.vars[usedTypesObj]
		if v == nil {
			return nil
		}
		s := v.Scope()
		if s != b.program.Scope() && s.IsParentOf(ctx.currentFunc().Scope()) {
			v.SetCaptured(true)
		}
		return v
	case *types.Func:
		f := b.funcs[usedTypesObj]
		if f == nil {
			return nil
		}
		return f.FuncValue()
	case *types.Const, *types.TypeName:
		return nil
	default:
		p := b.fset.Position(usedTypesObj.Pos())
		b.addWarning(fmt.Errorf("%v: unexpected types.Object type: %T", p, usedTypesObj))
		return nil
	}
}

func (b *builder) processSelectorExpr(selExpr *ast.SelectorExpr, ctx *context) ir.RValue {
	typesSelection, ok := ctx.typesInfo.Selections[selExpr]
	if !ok {
		// Assume *ast.SelectorExpr is for qualified identifier (package.identifier)
		xIdent := selExpr.X.(*ast.Ident)
		xTypesObj := ctx.typesInfo.ObjectOf(xIdent)
		_ = xTypesObj.(*types.PkgName)

		return b.processIdent(selExpr.Sel, ctx)
	}

	if typesSelection.Kind() != types.FieldVal {
		p := b.fset.Position(selExpr.Pos())
		selExprStr := b.nodeToString(selExpr)
		b.addWarning(fmt.Errorf("%v: method expressions or method values are not supported: %s", p, selExprStr))
		return nil
	}

	xVal := b.processExpr(selExpr.X, ctx)
	xTypesType := ctx.typesInfo.TypeOf(selExpr.X)
	xIrType := b.typesTypeToIrType(xTypesType)
	if xIrType == nil {
		return nil
	}
	irStructVal, ok := xVal.(ir.LValue)
	if xVal == nil || !ok {
		p := b.fset.Position(selExpr.X.Pos())
		xStr := b.nodeToString(selExpr.X)
		b.addWarning(fmt.Errorf("%v: could not resolve struct variable expression: %s", p, xStr))
		return nil
	}
	irStructType := irStructVal.Type().(*ir.StructType)

	fieldTypesVar := typesSelection.Obj().(*types.Var)
	fieldTypesType := fieldTypesVar.Type()
	fieldIrType := b.typesTypeToIrType(fieldTypesType)
	if fieldIrType == nil {
		return nil
	}
	irField, ok := b.fields[fieldTypesVar]
	if !ok {
		p := b.fset.Position(selExpr.Sel.Pos())
		selStr := b.nodeToString(selExpr.Sel)
		b.addWarning(fmt.Errorf("%v: could not resolve field expression: %s", p, selStr))
		return nil
	}

	neededStructType := irField.StructType()
	if neededStructType != irStructType {
		embeddedFields, ok := irStructType.FindEmbeddedFieldOfType(neededStructType)
		if !ok {
			p := b.fset.Position(selExpr.Sel.Pos())
			selStr := b.nodeToString(selExpr.Sel)
			b.addWarning(fmt.Errorf("%v: could not resolve field expression: %s", p, selStr))
			return nil
		}
		for _, irField := range embeddedFields {
			irStructVal = ir.NewFieldSelection(irStructVal, irField)
		}
	}

	return ir.NewFieldSelection(irStructVal, irField)
}

func (b *builder) procesIndexExpr(indexExpr *ast.IndexExpr, ctx *context) ir.RValue {
	xExpr := indexExpr.X
	iExpr := indexExpr.Index
	defer b.processExpr(iExpr, ctx)
	xTypesType := ctx.typesInfo.TypeOf(xExpr)
	iTypesType := ctx.typesInfo.TypeOf(iExpr)
	xIrType := b.typesTypeToIrType(xTypesType)
	iIrType := b.typesTypeToIrType(iTypesType)
	if xIrType == nil {
		return nil
	}

	irContainerVal := b.findContainer(xExpr, ctx)
	if irContainerVal == nil {
		return nil
	}
	if iIrType != nil {
		p := b.fset.Position(iExpr.Pos())
		iStr := b.nodeToString(iExpr)
		b.addWarning(fmt.Errorf("%v: ignoring index value: %s", p, iStr))
	}
	irContainerType := xIrType.(*ir.ContainerType)
	irContainerIndex := ir.RValue(ir.RandomIndex)
	if irContainerType.Kind() == ir.Array ||
		irContainerType.Kind() == ir.Slice {
		if res, ok := b.staticIntEval(iExpr, ctx); ok {
			irContainerIndex = ir.MakeValue(int64(res), ir.IntType)
		} else if iIdent, ok := iExpr.(*ast.Ident); ok {
			typesVar, ok := ctx.typesInfo.Uses[iIdent].(*types.Var)
			if ok {
				irVar, ok := b.vars[typesVar]
				if ok && irVar.Type() == ir.IntType {
					irContainerIndex = irVar
				}
			}
		}
	}

	return ir.NewContainerAccess(irContainerVal, irContainerIndex)
}
