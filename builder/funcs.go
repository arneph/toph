package builder

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/arneph/toph/ir"
)

func (b *builder) processFuncReceiver(recv *ast.FieldList, ctx *context) {
	if recv == nil || len(recv.List) != 1 {
		return
	}
	irFunc := ctx.currentFunc()
	field := recv.List[0]
	if len(field.Names) != 1 {
		return
	}
	fieldNameIdent := field.Names[0]
	typesType := b.typesInfo.TypeOf(field.Type)
	irType := b.typesTypeToIrType(typesType)
	if irType == nil {
		return
	}
	initialValue := b.initialValueForIrType(irType)
	typesVar := b.typesInfo.ObjectOf(fieldNameIdent).(*types.Var)
	irVar := b.program.NewVariable(fieldNameIdent.Name, irType, initialValue)
	irFunc.AddArg(-1, irVar)
	b.vars[typesVar] = irVar
}

func (b *builder) processFuncType(funcType *ast.FuncType, ctx *context) {
	irFunc := ctx.currentFunc()

	argIndex := 0
	for _, field := range funcType.Params.List {
		typesType := b.typesInfo.TypeOf(field.Type)
		irType := b.typesTypeToIrType(typesType)
		if irType == nil {
			argIndex += len(field.Names)
			continue
		}
		initialValue := b.initialValueForIrType(irType)
		for _, fieldNameIdent := range field.Names {
			typesVar := b.typesInfo.Defs[fieldNameIdent].(*types.Var)
			irVar := b.program.NewVariable(fieldNameIdent.Name, irType, initialValue)
			irFunc.AddArg(argIndex, irVar)
			b.vars[typesVar] = irVar
			argIndex++
		}
	}

	if funcType.Results == nil {
		return
	}
	resultIndex := 0
	for _, field := range funcType.Results.List {
		typesType := b.typesInfo.TypeOf(field.Type)
		irType := b.typesTypeToIrType(typesType)
		if irType == nil {
			if len(field.Names) > 0 {
				resultIndex += len(field.Names)
			} else {
				resultIndex++
			}
			continue
		}
		initialValue := b.initialValueForIrType(irType)

		if len(field.Names) == 0 {
			irFunc.AddResultType(resultIndex, irType)
			resultIndex++

		} else {
			for _, fieldNameIdent := range field.Names {
				typesVar := b.typesInfo.Defs[fieldNameIdent].(*types.Var)
				irVar := b.program.NewVariable(fieldNameIdent.Name, irType, initialValue)
				irFunc.AddResult(resultIndex, irVar)
				b.vars[typesVar] = irVar
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
					"print",
					"println",
					"real":
					return true
				case "make":
					astType := callExpr.Args[0]
					typesType := b.typesInfo.TypeOf(astType)
					irType := b.typesTypeToIrType(typesType)
					return irType != ir.ChanType
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
			case "bytes",
				"errors",
				"flag",
				"fmt",
				"fnv",
				"ioutil",
				"math",
				"md5",
				"rand",
				"sort",
				"strconv",
				"strings":
				return true
			case "filepath":
				if funcType.Name() == "Join" {
					return true
				}
			case "log":
				// Note: this covers both the methods of *Logger and the
				// convenience functions for the standard logger.
				switch funcType.Name() {
				case "Flags", "Output", "Prefix", "Print", "Printf", "Println",
					"SetFlags", "SetPrefix":
					return true
				}
			case "os":
				if strings.HasPrefix(funcType.FullName(), "(os.FileInfo") ||
					strings.HasPrefix(funcType.FullName(), "(os.FileMode") ||
					strings.HasPrefix(funcType.FullName(), "(*os.File") {
					return true
				}
				switch funcType.Name() {
				case "Chdir", "Chmod", "Chown", "Chtimes", "Clearenv",
					"Environ", "Executable",
					"Getegid", "Getenv", "Geteuid", "Getgid", "Getgroups",
					"Getpagesize", "Getpid", "Getppid", "Getuid", "Getwd",
					"Hostname", "IsExist", "IsNotExist", "IsPathSeparator",
					"IsPermission", "IsTimeout", "Lchown", "Link",
					"LookupEnv", "Mkdir", "MkdirAll", "NewSyscallError",
					"Readlink", "Remove", "RemoveAll", "Rename",
					"SameFile", "Setenv", "Symlink", "TempDir", "Truncate",
					"Unsetenv", "UserCacheDir", "UserConfigDir",
					"UserHomeDir",
					"Create", "NewFile", "Open", "OpenFile":
					return true
				}
			case "reflect":
				switch funcType.Name() {
				case "DeepEqual":
					return true
				}
			case "testing":
				if strings.HasPrefix(funcType.FullName(), "(*testing.T)") ||
					strings.HasPrefix(funcType.FullName(), "(*testing.B)") ||
					strings.HasPrefix(funcType.FullName(), "(*testing.common)") {
					switch funcType.Name() {
					case "Error", "Errorf", "Fail", "Failed", "Helper",
						"Log", "Logf", "Name", "Parallel", "Skipped",
						"ReportAllocs", "ReportMetric", "ResetTimer",
						"SetBytes", "SetParallelism",
						"StartTimer", "StopTimer":
						return true
					}
				}
			case "time":
				if strings.HasPrefix(funcType.FullName(), "(time.Duration)") ||
					strings.HasPrefix(funcType.FullName(), "(time.Time)") {
					return true
				}
				switch funcType.Name() {
				case "Now", "Sleep", "Since", "Until":
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
			irType := b.typesTypeToIrType(typesType)
			if irType != ir.ChanType {
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

func (b *builder) isNewCall(callExpr *ast.CallExpr) bool {
	funcExpr, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	builtinTypesObj, ok := b.typesInfo.ObjectOf(funcExpr).(*types.Builtin)
	if !ok {
		return false
	}
	return builtinTypesObj.Name() == "new"
}
