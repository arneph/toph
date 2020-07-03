package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processNewExpr(callExpr *ast.CallExpr, ctx *context) *ir.Variable {
	astType := callExpr.Args[0]
	typesType := b.typesInfo.TypeOf(astType)
	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return nil
	}
	structType, ok := irType.(*ir.StructType)
	if !ok {
		p := b.fset.Position(callExpr.Pos())
		callExprStr := b.nodeToString(callExpr)
		b.addWarning(fmt.Errorf("%v: new with non-struct type not supported: %s", p, callExprStr))
		return nil
	}
	irVar := b.program.NewVariable("", structType, -1)
	ctx.body.Scope().AddVariable(irVar)

	b.addValInitializationStmts(irVar, false, callExpr, ctx)

	return irVar
}

func (b *builder) processCompositeLit(compositeLit *ast.CompositeLit, ctx *context) *ir.Variable {
	typesType := b.typesInfo.TypeOf(compositeLit)
	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return nil
	}
	typesStruct := typesType.Underlying().(*types.Struct)
	irStructType := irType.(*ir.StructType)
	irVar := b.program.NewVariable("", irStructType, -1)
	ctx.body.Scope().AddVariable(irVar)

	makeStructStmt := ir.NewMakeStructStmt(irVar, compositeLit.Pos(), compositeLit.End())
	ctx.body.AddStmt(makeStructStmt)

	initializedFields := make(map[*ir.Field]bool)

	for i, valExpr := range compositeLit.Elts {
		var typesVar *types.Var
		if keyValueExpr, ok := valExpr.(*ast.KeyValueExpr); ok {
			keyExpr := keyValueExpr.Key.(*ast.Ident)
			valExpr = keyValueExpr.Value
			typesVar = b.typesInfo.ObjectOf(keyExpr).(*types.Var)
		} else {
			typesVar = typesStruct.Field(i)
		}
		irField := b.fields[typesVar]
		irFieldVal := b.processExpr(valExpr, ctx)
		if irField == nil {
			continue
		}
		if irFieldVal == nil {
			p := b.fset.Position(valExpr.Pos())
			valExprStr := b.nodeToString(valExpr)
			b.addWarning(fmt.Errorf("%v: could not evaluate field value: %s", p, valExprStr))
			continue
		}
		requiresCopy := false
		if _, ok := irField.Type().(*ir.StructType); ok {
			requiresCopy = !irField.IsPointer()
		}
		irFieldSelection := ir.NewFieldSelection(irVar, irField)
		assignStmt := ir.NewAssignStmt(irFieldVal, irFieldSelection, requiresCopy, valExpr.Pos(), valExpr.End())
		ctx.body.AddStmt(assignStmt)

		initializedFields[irField] = true
	}
	for _, irField := range irStructType.Fields() {
		if initializedFields[irField] {
			continue
		}
		irFieldSelection := ir.NewFieldSelection(irVar, irField)
		isPointer := irField.IsPointer()
		b.addValInitializationStmts(irFieldSelection, isPointer, compositeLit, ctx)
	}

	return irVar
}
