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
		t, ok := typesTypeToIrType(b.info.TypeOf(field.Type).Underlying())
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
		t, ok := typesTypeToIrType(b.info.TypeOf(field.Type).(types.Type).Underlying())
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
	sig := b.info.Types[funcLit].Type.(*types.Signature)
	f := b.program.AddInnerFunc(sig, ctx.currentFunc(), ctx.body.Scope(), funcLit.Pos(), funcLit.End())
	subCtx := ctx.subContextForFunc(f)
	b.processFuncType(funcLit.Type, subCtx)
	b.processFuncBody(funcLit.Body, subCtx)
	return f
}

func (b *builder) processFuncBody(body *ast.BlockStmt, ctx *context) {
	b.processBlockStmt(body, ctx)
	if len(body.List) < 1 {
		return
	}
	lastStmt := body.List[len(body.List)-1]
	if _, ok := lastStmt.(*ast.ReturnStmt); !ok {
		b.processDeferredCalls(ctx)
	}
}

func (b *builder) canIgnoreCall(funcExpr ast.Expr, ctx *context) bool {
	switch funcExpr := funcExpr.(type) {
	case *ast.Ident:
		if funcExpr.Name == "append" ||
			funcExpr.Name == "cap" ||
			funcExpr.Name == "complex" ||
			funcExpr.Name == "copy" ||
			funcExpr.Name == "delete" ||
			funcExpr.Name == "imag" ||
			funcExpr.Name == "len" ||
			funcExpr.Name == "new" ||
			funcExpr.Name == "print" ||
			funcExpr.Name == "println" ||
			funcExpr.Name == "real" {
			return true
		}
		if _, ok := b.info.Uses[funcExpr].(*types.TypeName); ok {
			return true
		}
	case *ast.SelectorExpr:
		switch funcType := b.info.Uses[funcExpr.Sel].(type) {
		case *types.Func:
			if funcType.String() == "func (error).Error() string" {
				return true
			}
			switch funcType.Pkg().Name() {
			case "md5", "errors", "flag", "fmt", "math", "rand", "sort", "strconv", "ioutil", "strings":
				return true
			case "time":
				if funcType.Name() == "Now" ||
					funcType.Name() == "Sleep" ||
					funcType.Name() == "UnixNano" ||
					funcType.Name() == "Since" {
					return true
				}
			case "os":
				if funcType.Name() == "Getenv" ||
					funcType.Name() == "Geteuid" ||
					funcType.Name() == "IsNotExist" ||
					funcType.Name() == "Hostname" ||
					funcType.Name() == "Lstat" ||
					funcType.Name() == "Stat" {
					return true
				}
				if funcType.FullName() == "(os.FileInfo).IsDir" ||
					funcType.FullName() == "(os.FileInfo).IsRegular" ||
					funcType.FullName() == "(os.FileInfo).Mode" ||
					funcType.FullName() == "(os.FileInfo).Name" ||
					funcType.FullName() == "(os.FileMode).Mode" ||
					funcType.FullName() == "(os.FileMode).IsRegular" {
					return true
				}
			case "filepath":
				if funcType.Name() == "Join" {
					return true
				}
			}
		case types.Object:
			if funcType.Pkg().Name() == "time" &&
				funcType.Name() == "Duration" {
				return true
			}
		}
	}
	return false
}

func (b *builder) findCallee(funcExpr ast.Expr, ctx *context) (callee ir.Callable, calleeSignature *types.Signature) {
	if b.canIgnoreCall(funcExpr, ctx) {
		return nil, nil
	}

	switch funcExpr := funcExpr.(type) {
	case *ast.Ident:
		switch used := b.info.Uses[funcExpr].(type) {
		case *types.Func:
			f := b.funcTypes[used]
			if f == nil {
				p := b.fset.Position(funcExpr.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve callee: %q", p, funcExpr.Name))
				return nil, nil
			}
			return f, f.Signature()

		case *types.Var:
			v := b.varTypes[used]
			if v == nil {
				p := b.fset.Position(funcExpr.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve callee: %q", p, funcExpr.Name))
				return nil, nil
			}
			return v, used.Type().Underlying().(*types.Signature)

		default:
			p := b.fset.Position(funcExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve %T callee: %q", p, used, funcExpr.Name))
			return nil, nil
		}

	case *ast.FuncLit:
		f := b.processFuncLit(funcExpr, ctx)
		return f, f.Signature()

	case *ast.SelectorExpr:
		switch funcType := b.info.Uses[funcExpr.Sel].(type) {
		case *types.Func:
			f := b.funcTypes[funcType]
			if f != nil {
				return f, f.Signature()
			}

			funcSub := b.getSubstitute(funcType)
			if funcSub != nil {
				return funcSub, funcSub.Signature()
			}

			b.processExpr(funcExpr.X, ctx)

			p := b.fset.Position(funcExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve callee: %v", p, funcType))
			return nil, nil

		default:
			b.processExpr(funcExpr.X, ctx)

			p := b.fset.Position(funcExpr.Pos())
			b.addWarning(fmt.Errorf("%v: could not resolve callee: %v", p, funcExpr))
			return nil, nil
		}

	default:
		p := b.fset.Position(funcExpr.Pos())
		b.addWarning(fmt.Errorf("%v: could not resolve %T callee", p, funcExpr))
		return nil, nil
	}
}
