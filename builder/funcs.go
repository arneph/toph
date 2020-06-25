package builder

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processFuncReceiver(recv *ast.FieldList, ctx *context) {
	if recv == nil || len(recv.List) != 1 {
		return
	}
	f := ctx.currentFunc()
	field := recv.List[0]
	if len(field.Names) != 1 {
		return
	}
	fieldNameIdent := field.Names[0]
	fieldType, initialValue, ok := typesTypeToIrType(b.typesInfo.TypeOf(field.Type).Underlying())
	if !ok {
		return
	}
	typesVar := b.typesInfo.ObjectOf(fieldNameIdent).(*types.Var)
	v := b.program.NewVariable(fieldNameIdent.Name, fieldType, initialValue)
	f.AddArg(-1, v)
	b.vars[typesVar] = v
}

func (b *builder) processFuncType(funcType *ast.FuncType, ctx *context) {
	f := ctx.currentFunc()

	argIndex := 0
	for _, field := range funcType.Params.List {
		t, initialValue, ok := typesTypeToIrType(b.typesInfo.TypeOf(field.Type).Underlying())
		if !ok {
			argIndex += len(field.Names)
			continue
		}
		for _, fieldNameIdent := range field.Names {
			varType := b.typesInfo.Defs[fieldNameIdent].(*types.Var)
			v := b.program.NewVariable(fieldNameIdent.Name, t, initialValue)
			f.AddArg(argIndex, v)
			b.vars[varType] = v
			argIndex++
		}
	}

	if funcType.Results == nil {
		return
	}
	resultIndex := 0
	for _, field := range funcType.Results.List {
		t, initialValue, ok := typesTypeToIrType(b.typesInfo.TypeOf(field.Type).(types.Type).Underlying())
		if !ok {
			resultIndex += len(field.Names)
			continue
		}

		if len(field.Names) == 0 {
			f.AddResultType(resultIndex, t)
			resultIndex++

		} else {
			for _, fieldNameIdent := range field.Names {
				varType := b.typesInfo.Defs[fieldNameIdent].(*types.Var)
				v := b.program.NewVariable(fieldNameIdent.Name, t, initialValue)
				f.AddResult(resultIndex, v)
				b.vars[varType] = v
				resultIndex++
			}
		}
	}
}

func (b *builder) processFuncLit(funcLit *ast.FuncLit, ctx *context) *ir.Func {
	sig := b.typesInfo.Types[funcLit].Type.(*types.Signature)
	f := b.program.AddInnerFunc(sig, ctx.currentFunc(), ctx.body.Scope(), funcLit.Pos(), funcLit.End())
	subCtx := ctx.subContextForFunc(f)
	b.processFuncType(funcLit.Type, subCtx)
	b.processFuncBody(funcLit.Body, subCtx)
	return f
}

func (b *builder) processFuncBody(body *ast.BlockStmt, ctx *context) {
	if body == nil {
		f := ctx.currentFunc()
		p := b.fset.Position(f.Pos())
		b.addWarning(fmt.Errorf("%v: function is not defined: %s", p, f.Name()))
		return
	}
	b.processBlockStmt(body, ctx)
}

func (b *builder) canIgnoreCall(callExpr *ast.CallExpr) bool {
	switch funcExpr := callExpr.Fun.(type) {
	case *ast.Ident:
		if used, ok := b.typesInfo.Uses[funcExpr]; ok {
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
				case "make":
					astType := callExpr.Args[0]
					typesType := b.typesInfo.TypeOf(astType)
					irType, _, ok := typesTypeToIrType(typesType)
					return !ok || irType != ir.ChanType
				}
			case *types.TypeName:
				return true
			}
		}
	case *ast.SelectorExpr:
		switch funcType := b.typesInfo.Uses[funcExpr.Sel].(type) {
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

func (b *builder) specialOpForCall(callExpr *ast.CallExpr) (ir.SpecialOp, bool) {
	var usedTypesObj types.Object

	switch funcExpr := callExpr.Fun.(type) {
	case *ast.Ident:
		usedTypesObj = b.typesInfo.ObjectOf(funcExpr)
	case *ast.SelectorExpr:
		usedTypesObj = b.typesInfo.ObjectOf(funcExpr.Sel)
	default:
		return nil, false
	}

	switch usedTypesObj := usedTypesObj.(type) {
	case *types.Builtin:
		switch usedTypesObj.Name() {
		case "make":
			astType := callExpr.Args[0]
			typesType := b.typesInfo.TypeOf(astType)
			irType, _, ok := typesTypeToIrType(typesType)
			if !ok || irType != ir.ChanType {
				return nil, false
			}
			return ir.MakeChan, true
		case "close":
			return ir.Close, true
		}
	case *types.Func:
		switch usedTypesObj.FullName() {
		case "(*sync.Mutex).Lock", "(*sync.RWMutex).Lock":
			return ir.Lock, true
		case "(*sync.Mutex).Unlock", "(*sync.RWMutex).Unlock":
			return ir.Unlock, true
		case "(*sync.RWMutex).RLock":
			return ir.RLock, true
		case "(*sync.RWMutex).RUnlock":
			return ir.RUnlock, true
		case "(*sync.WaitGroup).Add", "(*sync.WaitGroup).Done":
			return ir.Add, true
		case "(*sync.WaitGroup).Wait":
			return ir.Wait, true
		case "os.Exit":
			return ir.DeadEnd, true
		}
	}

	return nil, false
}
