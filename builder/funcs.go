package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processFuncType(funcType *ast.FuncType, ctx *context) {
	f := ctx.currentFunc()

	argIndex := 0
	for _, field := range funcType.Params.List {
		t, ok := astTypeToIrType(field.Type)
		if !ok {
			argIndex += len(field.Names)
			continue
		}
		for _, fieldNameIdent := range field.Names {
			varType := b.info.Defs[fieldNameIdent].(*types.Var)
			v := ir.NewVariable(fieldNameIdent.Name, t, -1)
			f.AddArg(argIndex, v)
			b.varTypes[varType] = v
			argIndex++
		}
	}

	if funcType.Results == nil {
		return
	}
	resultIndex := 0
	for _, field := range funcType.Results.List {
		t, ok := astTypeToIrType(field.Type)
		if !ok {
			resultIndex += len(field.Names)
			continue
		}

		if len(field.Names) == 0 {
			f.AddResultType(resultIndex, t)
			resultIndex++

		} else {
			for _, fieldNameIdent := range field.Names {
				varType := b.info.Defs[fieldNameIdent].(*types.Var)
				v := ir.NewVariable(fieldNameIdent.Name, t, -1)
				f.AddResult(resultIndex, v)
				b.varTypes[varType] = v
				resultIndex++
			}
		}
	}
}

func (b *builder) processFuncLit(funcLit *ast.FuncLit, ctx *context) *ir.Func {
	f := ir.NewInnerFunc(ctx.currentFunc(), ctx.body.Scope())
	b.program.AddFunc(f)
	subCtx := ctx.subContextForFunc(f)
	b.processFuncType(funcLit.Type, subCtx)
	b.processStmt(funcLit.Body, subCtx)
	return f
}

func (b *builder) findCallee(funcExpr ast.Expr, ctx *context) *ir.Func {
	switch funcExpr := funcExpr.(type) {
	case *ast.Ident:
		switch used := b.info.Uses[funcExpr].(type) {
		case *types.Func:
			f := b.funcTypes[used]
			if f == nil {
				p := b.fset.Position(funcExpr.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve callee: %q", p, funcExpr.Name))
			}
			return f

		case *types.TypeName:
			return nil

		default:
			p := b.fset.Position(funcExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve %T callee: %q", p, used, funcExpr.Name))

			return nil
		}

	case *ast.FuncLit:
		return b.processFuncLit(funcExpr, ctx)

	case *ast.SelectorExpr:
		switch funcType := b.info.Uses[funcExpr.Sel].(type) {
		case *types.Func:
			switch funcType.Pkg().Name() {
			case "flag", "fmt", "math", "rand", "sort", "strconv":
				return nil
			case "time":
				if funcType.Name() == "Now" ||
					funcType.Name() == "Sleep" ||
					funcType.Name() == "UnixNano" ||
					funcType.Name() == "Since" {
					return nil
				}
			}

			funcSub := b.getSubstitute(funcType)
			if funcSub != nil {
				return funcSub
			}

			b.processExpr(funcExpr.X, ctx)

			p := b.fset.Position(funcExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve callee: %v", p, funcType))

			return nil

		case types.Object:
			if funcType.Pkg().Name() == "time" &&
				funcType.Name() == "Duration" {
				return nil
			}
			b.processExpr(funcExpr.X, ctx)

			p := b.fset.Position(funcExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve callee: %v", p, funcType))

			return nil

		default:
			b.processExpr(funcExpr.X, ctx)

			p := b.fset.Position(funcExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve callee: %v", p, funcExpr))

			return nil
		}

	default:
		p := b.fset.Position(funcExpr.Pos())
		b.addWarning(fmt.Errorf("%v: could not resolve %T callee", p, funcExpr))

		return nil
	}
}
