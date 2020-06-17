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
		t, initialValue, ok := typesTypeToIrType(b.pkgTypesInfos[ctx.pkg].TypeOf(field.Type).Underlying())
		if !ok {
			argIndex += len(field.Names)
			continue
		}
		for _, fieldNameIdent := range field.Names {
			varType := b.pkgTypesInfos[ctx.pkg].Defs[fieldNameIdent].(*types.Var)
			v := b.program.NewVariable(fieldNameIdent.Name, t, initialValue)
			f.AddArg(argIndex, v)
			b.pkgVarTypes[ctx.pkg][varType] = v
			argIndex++
		}
	}

	if funcType.Results == nil {
		return
	}
	resultIndex := 0
	for _, field := range funcType.Results.List {
		t, initialValue, ok := typesTypeToIrType(b.pkgTypesInfos[ctx.pkg].TypeOf(field.Type).(types.Type).Underlying())
		if !ok {
			resultIndex += len(field.Names)
			continue
		}

		if len(field.Names) == 0 {
			f.AddResultType(resultIndex, t)
			resultIndex++

		} else {
			for _, fieldNameIdent := range field.Names {
				varType := b.pkgTypesInfos[ctx.pkg].Defs[fieldNameIdent].(*types.Var)
				v := b.program.NewVariable(fieldNameIdent.Name, t, initialValue)
				f.AddResult(resultIndex, v)
				b.pkgVarTypes[ctx.pkg][varType] = v
				resultIndex++
			}
		}
	}
}

func (b *builder) processFuncLit(funcLit *ast.FuncLit, ctx *context) *ir.Func {
	sig := b.pkgTypesInfos[ctx.pkg].Types[funcLit].Type.(*types.Signature)
	f := b.program.AddInnerFunc(sig, ctx.currentFunc(), ctx.body.Scope(), funcLit.Pos(), funcLit.End())
	subCtx := ctx.subContextForFunc(f)
	b.processFuncType(funcLit.Type, subCtx)
	b.processFuncBody(funcLit.Body, subCtx)
	return f
}

func (b *builder) processFuncBody(body *ast.BlockStmt, ctx *context) {
	b.processBlockStmt(body, ctx)
}

func (b *builder) canIgnoreCall(funcExpr ast.Expr, ctx *context) bool {
	switch funcExpr := funcExpr.(type) {
	case *ast.Ident:
		if used, ok := b.pkgTypesInfos[ctx.pkg].Uses[funcExpr]; ok {
			switch used := used.(type) {
			case *types.Builtin:
				switch used.Name() {
				case "append",
					"cap",
					"complex",
					"copy",
					"delete",
					"imag",
					"len",
					"new",
					"print",
					"println",
					"real":
					return true
				}
			case *types.TypeName:
				return true
			}
		}
	case *ast.SelectorExpr:
		switch funcType := b.pkgTypesInfos[ctx.pkg].Uses[funcExpr.Sel].(type) {
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
		case *types.TypeName:
			return true
		}
	}
	return false
}

func (b *builder) findCallee(funcExpr ast.Expr, ctx *context) (callee ir.Callable, calleeSignature *types.Signature) {
	if b.canIgnoreCall(funcExpr, ctx) {
		return nil, nil
	}

	switch funcExpr := funcExpr.(type) {
	case *ast.FuncLit:
		f := b.processFuncLit(funcExpr, ctx)
		return f, f.Signature()

	case *ast.Ident:
		switch used := b.pkgTypesInfos[ctx.pkg].Uses[funcExpr].(type) {
		case *types.Func:
			f := b.pkgFuncTypes[ctx.pkg][used]
			if f == nil {
				p := b.fset.Position(funcExpr.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve callee: %q", p, funcExpr.Name))
				return nil, nil
			}
			return f, f.Signature()

		case *types.Var:
			v := b.pkgVarTypes[ctx.pkg][used]
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

	case *ast.SelectorExpr:
		pkg := ctx.pkg
		funcType := b.pkgTypesInfos[pkg].Uses[funcExpr.Sel]

		if ident, ok := funcExpr.X.(*ast.Ident); ok {
			if typesPkg, ok := b.pkgTypesInfos[ctx.pkg].Uses[ident].(*types.PkgName); ok {
				_, ok := b.pkgAstFiles[typesPkg.Imported().Path()]
				if ok {
					pkg = typesPkg.Imported().Path()
					typesPkg := b.pkgTypesPackages[pkg]
					typesPkgScope := typesPkg.Scope()
					funcType = typesPkgScope.Lookup(funcExpr.Sel.Name)
				}
			}
		}

		switch funcType := funcType.(type) {
		case *types.Func:
			f := b.pkgFuncTypes[pkg][funcType]
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

		case *types.Var:
			v := b.pkgVarTypes[pkg][funcType]
			if v == nil {
				p := b.fset.Position(funcExpr.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve callee: %q", p, funcType))
				return nil, nil
			}
			return v, funcType.Type().Underlying().(*types.Signature)

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
