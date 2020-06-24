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

	irType, initialValue, ok := typesTypeToIrType(typesType)
	if !ok {
		return nil
	}
	irVar := b.program.NewVariable(ident.Name, irType, initialValue)
	scope.AddVariable(irVar)
	b.vars[typesVar] = irVar

	if irType == ir.MutexType {
		makeMutexStmt := ir.NewMakeMutexStmt(irVar, ident.Pos(), ident.End())
		ctx.body.AddStmt(makeMutexStmt)
	} else if irType == ir.WaitGroupType {
		makeWaitGroupStmt := ir.NewMakeWaitGroupStmt(irVar, ident.Pos(), ident.End())
		ctx.body.AddStmt(makeWaitGroupStmt)
	}

	return irVar
}
