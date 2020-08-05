package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processVarDefinitions(idents []*ast.Ident, initialize bool, ctx *context) map[int]*ir.Variable {
	return b.processVarDefinitionsInScope(idents, ctx.body.Scope(), initialize, ctx)
}

func (b *builder) processVarDefinitionsInScope(idents []*ast.Ident, scope *ir.Scope, initialize bool, ctx *context) map[int]*ir.Variable {
	irVars := make(map[int]*ir.Variable)
	for i, ident := range idents {
		irVar := b.processVarDefinitionInScope(ident, scope, initialize, ctx)
		if irVar != nil {
			irVars[i] = irVar
		}
	}
	return irVars
}

func (b *builder) processVarDefinition(ident *ast.Ident, initialize bool, ctx *context) *ir.Variable {
	return b.processVarDefinitionInScope(ident, ctx.body.Scope(), initialize, ctx)
}

func (b *builder) processVarDefinitionInScope(ident *ast.Ident, scope *ir.Scope, initialize bool, ctx *context) *ir.Variable {
	typesObj, ok := ctx.typesInfo.Defs[ident]
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
	initialValue := irType.UninitializedValue()
	if initialize {
		initialValue = irType.InitializedValue()
	}
	irVar := b.program.NewVariable(ident.Name, initialValue)
	scope.AddVariable(irVar)
	b.vars[typesVar] = irVar

	return irVar
}
