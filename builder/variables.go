package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processVarDefinitions(idents []*ast.Ident, ctx *context) map[int]*ir.Variable {
	return b.processVarDefinitionsInScope(idents, ctx.body.Scope(), ctx)
}

func (b *builder) processVarDefinitionsInScope(idents []*ast.Ident, scope *ir.Scope, ctx *context) map[int]*ir.Variable {
	irVars := make(map[int]*ir.Variable)
	for i, ident := range idents {
		irVar := b.processVarDefinitionInScope(ident, scope, ctx)
		if irVar != nil {
			irVars[i] = irVar
		}
	}
	return irVars
}

func (b *builder) processVarDefinition(ident *ast.Ident, ctx *context) *ir.Variable {
	return b.processVarDefinitionInScope(ident, ctx.body.Scope(), ctx)
}

func (b *builder) processVarDefinitionInScope(ident *ast.Ident, scope *ir.Scope, ctx *context) *ir.Variable {
	typesObj, ok := b.typesInfo.Defs[ident]
	if !ok {
		return nil
	}
	typesVar, ok := typesObj.(*types.Var)
	if !ok {
		return nil
	}
	typesType := typesVar.Type()
	if typesType == nil {
		p := b.fset.Position(ident.Pos())
		b.addWarning(fmt.Errorf("%v: types.Type for identifier is nil: %s", p, ident.Name))
		return nil
	}

	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return nil
	}
	initialValue := b.initialValueForIrType(irType)
	irVar := b.program.NewVariable(ident.Name, irType, initialValue)
	scope.AddVariable(irVar)
	b.vars[typesVar] = irVar

	b.addValInitializationStmts(irVar, b.isPointer(typesType), ident, ctx)

	return irVar
}

func (b *builder) addValInitializationStmts(irVal ir.LValue, isPointer bool, node ast.Node, ctx *context) {
	irType := irVal.Type()
	if irType == ir.MutexType {
		makeMutexStmt := ir.NewMakeMutexStmt(irVal, node.Pos(), node.End())
		ctx.body.AddStmt(makeMutexStmt)
	} else if irType == ir.WaitGroupType {
		makeWaitGroupStmt := ir.NewMakeWaitGroupStmt(irVal, node.Pos(), node.End())
		ctx.body.AddStmt(makeWaitGroupStmt)
	} else if irStructType, ok := irType.(*ir.StructType); ok {
		if isPointer {
			return
		}
		makeStructStmt := ir.NewMakeStructStmt(irVal, node.Pos(), node.End())
		ctx.body.AddStmt(makeStructStmt)

		for _, irField := range irStructType.Fields() {
			irFieldSelection := ir.NewFieldSelection(irVal, irField)
			b.addValInitializationStmts(irFieldSelection, irField.IsPointer(), node, ctx)
		}
	}
}
