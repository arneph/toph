package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) findContainer(containerExpr ast.Expr, ctx *context) ir.LValue {
	rv := b.processExpr(containerExpr, ctx)
	lv, ok := rv.(ir.LValue)
	if !ok || lv == nil {
		p := b.fset.Position(containerExpr.Pos())
		containerExprStr := b.nodeToString(containerExpr)
		b.addWarning(fmt.Errorf("%v: could not resolve container expr: %s", p, containerExprStr))
		return nil
	}
	return lv
}

func (b *builder) processNewExpr(callExpr *ast.CallExpr, ctx *context) *ir.Variable {
	astType := callExpr.Args[0]
	typesType := ctx.typesInfo.TypeOf(astType)
	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return nil
	}
	switch irType := irType.(type) {
	case *ir.StructType:
		irVar := b.program.NewVariable("", irType.UninitializedValue())
		ctx.body.Scope().AddVariable(irVar)

		makeStructStmt := ir.NewMakeStructStmt(irVar, true, callExpr.Pos(), callExpr.End())
		ctx.body.AddStmt(makeStructStmt)

		return irVar
	case *ir.ContainerType:
		if irType.Kind() != ir.Array {
			break
		}

		irVar := b.program.NewVariable("", irType.UninitializedValue())
		ctx.body.Scope().AddVariable(irVar)

		makeStructStmt := ir.NewMakeContainerStmt(irVar, -1, true, callExpr.Pos(), callExpr.End())
		ctx.body.AddStmt(makeStructStmt)

		return irVar
	}
	p := b.fset.Position(callExpr.Pos())
	callExprStr := b.nodeToString(callExpr)
	b.addWarning(fmt.Errorf("%v: new not supported for type: %s", p, callExprStr))
	return nil
}

func (b *builder) processMakeContainerExpr(callExpr *ast.CallExpr, ctx *context) *ir.Variable {
	astType := callExpr.Args[0]
	typesType := ctx.typesInfo.TypeOf(astType)
	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return nil
	}
	irContainerType, ok := irType.(*ir.ContainerType)
	if !ok {
		p := b.fset.Position(callExpr.Pos())
		callExprStr := b.nodeToString(callExpr)
		b.addWarning(fmt.Errorf("%v: unexpected make with non-container type: %s", p, callExprStr))
		return nil
	}
	var length int
	switch irContainerType.Kind() {
	case ir.Slice:
		if len(callExpr.Args) >= 2 {
			lengthExpr := callExpr.Args[1]
			res, ok := b.staticIntEval(lengthExpr, ctx)
			if !ok {
				p := b.fset.Position(lengthExpr.Pos())
				aStr := b.nodeToString(lengthExpr)
				b.addWarning(fmt.Errorf("%v: can not process slice legnth: %s", p, aStr))
			} else {
				length = res
			}
		}
	case ir.Map:
	default:
		panic("unexpected container kind")
	}
	irVar := b.program.NewVariable("", irContainerType.UninitializedValue())
	ctx.body.Scope().AddVariable(irVar)

	makeContainerStmt := ir.NewMakeContainerStmt(irVar, length, true, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(makeContainerStmt)

	return irVar
}

func (b *builder) processSliceAppendExpr(callExpr *ast.CallExpr, ctx *context) *ir.Variable {
	typesSliceType := ctx.typesInfo.TypeOf(callExpr.Args[0])
	irType := b.typesTypeToIrType(typesSliceType)
	if irType == nil {
		return nil
	}
	irSliceType := irType.(*ir.ContainerType)
	oldSlice := b.findContainer(callExpr.Args[0], ctx)
	if oldSlice == nil {
		return nil
	}
	if len(callExpr.Args) == 2 {
		valueTypesType := ctx.typesInfo.TypeOf(callExpr.Args[1])
		if _, ok := valueTypesType.(*types.Slice); ok {
			p := b.fset.Position(callExpr.Pos())
			callStr := b.nodeToString(callExpr)
			b.addWarning(fmt.Errorf("%v: appending slice to slice is unsupported: %s", p, callStr))
			return nil
		}
	}

	newSlice := b.program.NewVariable("", oldSlice.Type().UninitializedValue())
	ctx.body.Scope().AddVariable(newSlice)

	copyStmt := ir.NewAssignStmt(oldSlice.(ir.RValue), newSlice, true, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(copyStmt)

	for _, argExpr := range callExpr.Args[1:] {
		argVal := b.processExpr(argExpr, ctx)
		if argVal == nil {
			p := b.fset.Position(argExpr.Pos())
			argStr := b.nodeToString(argExpr)
			b.addWarning(fmt.Errorf("%v: can not process slice element: %s", p, argStr))
			continue
		}
		requiresCopy := irSliceType.RequiresDeepCopies()
		irSliceAccess := ir.NewContainerAccess(newSlice, ir.AppendIndex)
		irSliceAccess.SetKind(ir.Write)
		appendStmt := ir.NewAssignStmt(argVal, irSliceAccess, requiresCopy, callExpr.Pos(), callExpr.End())
		ctx.body.AddStmt(appendStmt)
	}

	return newSlice
}

func (b *builder) processCompositeLit(compositeLit *ast.CompositeLit, ctx *context) *ir.Variable {
	typesType := ctx.typesInfo.TypeOf(compositeLit)
	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return nil
	}
	switch irType := irType.(type) {
	case *ir.StructType:
		if typesPointer, ok := typesType.Underlying().(*types.Pointer); ok {
			typesType = typesPointer.Elem()
		}
		typesStruct := typesType.Underlying().(*types.Struct)
		return b.processStructCompositeLit(compositeLit, typesStruct, irType, ctx)
	case *ir.ContainerType:
		return b.processContainerCompositeLit(compositeLit, typesType, irType, ctx)
	default:
		return nil
	}
}

func (b *builder) processStructCompositeLit(compositeLit *ast.CompositeLit,
	typesStruct *types.Struct, irStructType *ir.StructType, ctx *context) *ir.Variable {
	irVar := b.program.NewVariable("", irStructType.UninitializedValue())
	ctx.body.Scope().AddVariable(irVar)

	makeStructStmt := ir.NewMakeStructStmt(irVar, false, compositeLit.Pos(), compositeLit.End())
	ctx.body.AddStmt(makeStructStmt)

	initializedFields := make(map[*ir.Field]bool)

	for i, valExpr := range compositeLit.Elts {
		var typesVar *types.Var
		if keyValueExpr, ok := valExpr.(*ast.KeyValueExpr); ok {
			keyExpr := keyValueExpr.Key.(*ast.Ident)
			valExpr = keyValueExpr.Value
			typesVar = ctx.typesInfo.ObjectOf(keyExpr).(*types.Var)
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
		requiresCopy := irField.RequiresDeepCopy()
		neededStructType := irField.StructType()
		irStructVal := ir.LValue(irVar)
		if neededStructType != irStructType {
			embeddedFields, ok := irStructType.FindEmbeddedFieldOfType(neededStructType)
			if !ok {
				p := b.fset.Position(valExpr.Pos())
				valExprStr := b.nodeToString(valExpr)
				b.addWarning(fmt.Errorf("%v: could not find field for value: %s", p, valExprStr))
				return nil
			}
			for _, irField := range embeddedFields {
				irStructVal = ir.NewFieldSelection(irStructVal, irField)
			}
		}
		irFieldSelection := ir.NewFieldSelection(irStructVal, irField)

		assignStmt := ir.NewAssignStmt(irFieldVal, irFieldSelection, requiresCopy, valExpr.Pos(), valExpr.End())
		ctx.body.AddStmt(assignStmt)

		initializedFields[irField] = true
	}
	for _, irField := range irStructType.Fields() {
		if initializedFields[irField] {
			continue
		}
		var irFieldVal ir.Value
		if !irField.IsPointer() {
			irFieldVal = irField.Type().InitializedValue()
		} else {
			irFieldVal = irField.Type().UninitializedValue()
		}
		irFieldSelection := ir.NewFieldSelection(irVar, irField)
		assignStmt := ir.NewAssignStmt(irFieldVal, irFieldSelection, false, compositeLit.Pos(), compositeLit.End())
		ctx.body.AddStmt(assignStmt)
	}

	return irVar
}

func (b *builder) processContainerCompositeLit(compositeLit *ast.CompositeLit,
	typesType types.Type, irContainerType *ir.ContainerType, ctx *context) *ir.Variable {
	irVar := b.program.NewVariable("", irContainerType.UninitializedValue())
	ctx.body.Scope().AddVariable(irVar)

	var arrayOrSliceEntries []arrayOrSliceCompositeLitEntry
	var length int
	switch irContainerType.Kind() {
	case ir.Array, ir.Slice:
		arrayOrSliceEntries, length = b.indicesAndValueExprsForArrayOrSliceCompositeLit(compositeLit, ctx)
		if irContainerType.Kind() == ir.Array {
			length = irContainerType.Len()
		}
	case ir.Map:
	default:
		panic("unexpected container kind")
	}

	makeContainerStmt := ir.NewMakeContainerStmt(irVar, length, false, compositeLit.Pos(), compositeLit.End())
	ctx.body.AddStmt(makeContainerStmt)

	switch irContainerType.Kind() {
	case ir.Array, ir.Slice:
		initializedIndices := make([]bool, length)
		for _, entry := range arrayOrSliceEntries {
			index := entry.index
			valExpr := entry.valueExpr
			irElemVal := b.processExpr(valExpr, ctx)
			if irElemVal == nil {
				p := b.fset.Position(valExpr.Pos())
				valExprStr := b.nodeToString(valExpr)
				b.addWarning(fmt.Errorf("%v: could not evaluate element value: %s", p, valExprStr))
				index++
				continue
			}
			requiresCopy := irContainerType.RequiresDeepCopies()
			irContainerAccess := ir.NewContainerAccess(irVar, ir.MakeValue(int64(index), ir.IntType))
			irContainerAccess.SetKind(ir.Write)
			assignStmt := ir.NewAssignStmt(irElemVal, irContainerAccess, requiresCopy, valExpr.Pos(), valExpr.End())
			ctx.body.AddStmt(assignStmt)

			initializedIndices[index] = true
			index++
		}
		for i := 0; i < length; i++ {
			if initializedIndices[i] {
				continue
			}
			var irElemVal ir.Value
			if !irContainerType.HoldsPointers() {
				irElemVal = irContainerType.ElementType().InitializedValue()
			} else {
				irElemVal = irContainerType.ElementType().UninitializedValue()
			}
			irContainerAccess := ir.NewContainerAccess(irVar, ir.MakeValue(int64(i), ir.IntType))
			irContainerAccess.SetKind(ir.Write)
			assignStmt := ir.NewAssignStmt(irElemVal, irContainerAccess, false, compositeLit.Pos(), compositeLit.End())
			ctx.body.AddStmt(assignStmt)
		}

	case ir.Map:
		for _, x := range compositeLit.Elts {
			keyValueExpr := x.(*ast.KeyValueExpr)
			keyExpr := keyValueExpr.Key
			valExpr := keyValueExpr.Value

			b.processExpr(keyExpr, ctx)
			irElemVal := b.processExpr(valExpr, ctx)
			if irElemVal == nil {
				p := b.fset.Position(valExpr.Pos())
				valExprStr := b.nodeToString(valExpr)
				b.addWarning(fmt.Errorf("%v: could not evaluate element value: %s", p, valExprStr))
				continue
			}
			requiresCopy := irContainerType.RequiresDeepCopies()
			irContainerAccess := ir.NewContainerAccess(irVar, ir.RandomIndex)
			irContainerAccess.SetKind(ir.Write)
			assignStmt := ir.NewAssignStmt(irElemVal, irContainerAccess, requiresCopy, valExpr.Pos(), valExpr.End())
			ctx.body.AddStmt(assignStmt)
		}

	default:
		panic("unexpected container kind")
	}

	return irVar
}

type arrayOrSliceCompositeLitEntry struct {
	index     int
	valueExpr ast.Expr
}

func (b *builder) indicesAndValueExprsForArrayOrSliceCompositeLit(compositeLit *ast.CompositeLit, ctx *context) (entries []arrayOrSliceCompositeLitEntry, length int) {
	index := 0
	for _, valExpr := range compositeLit.Elts {
		if keyValueExpr, ok := valExpr.(*ast.KeyValueExpr); ok {
			keyExpr := keyValueExpr.Key
			valExpr = keyValueExpr.Value
			res, ok := b.staticIntEval(keyExpr, ctx)
			if !ok {
				p := b.fset.Position(keyExpr.Pos())
				keyExprStr := b.nodeToString(keyExpr)
				b.addWarning(fmt.Errorf("%v: could not evaluate index value: %s", p, keyExprStr))
			} else {
				index = res
			}
		}
		if length <= index {
			length = index + 1
		}
		entries = append(entries, arrayOrSliceCompositeLitEntry{
			index:     index,
			valueExpr: valExpr,
		})
		index++
	}
	return
}

func (b *builder) processCopyExpr(callExpr *ast.CallExpr, ctx *context) {
	typesSliceType := ctx.typesInfo.TypeOf(callExpr.Args[0])
	irType := b.typesTypeToIrType(typesSliceType)
	if irType == nil {
		return
	}
	dstVal := b.findContainer(callExpr.Args[0], ctx)
	srcVal := b.findContainer(callExpr.Args[1], ctx)
	if dstVal == nil || srcVal == nil {
		return
	}
	copyStmt := ir.NewCopySliceStmt(dstVal, srcVal, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(copyStmt)
}

func (b *builder) processDeleteExpr(callExpr *ast.CallExpr, ctx *context) {
	typesMapType := ctx.typesInfo.TypeOf(callExpr.Args[0])
	irType := b.typesTypeToIrType(typesMapType)
	if irType == nil {
		return
	}
	mapVal := b.findContainer(callExpr.Args[0], ctx)
	b.processExpr(callExpr.Args[1], ctx)
	if mapVal == nil {
		return
	}
	deleteStmt := ir.NewDeleteMapEntryStmt(mapVal, callExpr.Pos(), callExpr.End())
	ctx.body.AddStmt(deleteStmt)
}
