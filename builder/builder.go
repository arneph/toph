package builder

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"

	"github.com/arneph/toph/ir"
)

const parserMode = parser.ParseComments |
	parser.DeclarationErrors |
	parser.AllErrors

// BuildProgram parses the Go files at the given path and builds an ir.Program.
func BuildProgram(path string, entryFuncName string) (*ir.Program, []error) {
	b := new(builder)

	// Parse program:
	b.fset = token.NewFileSet()
	filter := func(info os.FileInfo) bool {
		ok, err := build.Default.MatchFile(path, info.Name())
		if err != nil {
			b.addWarning(err)
			return false
		}
		return ok
	}
	pkgs, err := parser.ParseDir(b.fset, path, filter, parserMode)
	if err != nil {
		b.addWarning(fmt.Errorf("failed to parse input: %v", err))
		return b.program, b.warnings
	} else if len(pkgs) < 1 {
		b.addWarning(fmt.Errorf("expected at least one package, got: %d", len(pkgs)))
	}
	subsFile, err := parser.ParseFile(b.fset, "substitutes.go", substitutesCode, parserMode)
	if err != nil {
		b.addWarning(fmt.Errorf("failed to parse substitutes file: %v", err))
		return b.program, b.warnings
	}

	// Comment maps:
	b.cmaps = make(map[*ast.File]ast.CommentMap)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			b.cmaps[file] = ast.NewCommentMap(b.fset, file, file.Comments)
		}
	}

	// Type check:
	b.info = new(types.Info)
	b.info.Types = make(map[ast.Expr]types.TypeAndValue)
	b.info.Defs = make(map[*ast.Ident]types.Object)
	b.info.Uses = make(map[*ast.Ident]types.Object)
	b.info.Selections = make(map[*ast.SelectorExpr]*types.Selection)
	b.info.Scopes = make(map[ast.Node]*types.Scope)

	conf := types.Config{
		Importer: importer.ForCompiler(b.fset, "source", nil),
	}

	for pkgName, pkg := range pkgs {
		var pkgFiles []*ast.File
		for _, file := range pkg.Files {
			pkgFiles = append(pkgFiles, file)
		}
		_, err = conf.Check(pkgName, b.fset, pkgFiles, b.info)
		if err != nil {
			b.addWarning(fmt.Errorf("%v", err))
			return b.program, b.warnings
		}
	}

	_, err = conf.Check("subs", b.fset, []*ast.File{subsFile}, b.info)
	if err != nil {
		panic("type checker failed for substitutes")
	}

	// IR setup:
	b.program = ir.NewProgram(b.fset)
	b.funcTypes = make(map[*types.Func]*ir.Func)
	b.varTypes = make(map[*types.Var]*ir.Variable)

	// File processing:
	files := []*ast.File{subsFile}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			files = append(files, file)
		}
	}

	for _, file := range files {
		b.processFuncDeclsInFile(file)
	}
	for _, f := range b.program.Funcs() {
		if f.Name() == entryFuncName {
			b.program.SetEntryFunc(f)
			break
		}
	}
	if b.program.EntryFunc() == nil {
		entrySig := types.NewSignature(nil, nil, nil, false)
		entryFunc := b.program.AddOuterFunc(entryFuncName, entrySig, token.NoPos, token.NoPos)
		b.program.SetEntryFunc(entryFunc)
	}
	for _, file := range files {
		b.processGenDeclsInFile(file)
	}
	for _, file := range files {
		b.processFuncDefsInFile(file)
	}
	return b.program, b.warnings
}

type builder struct {
	fset  *token.FileSet
	cmaps map[*ast.File]ast.CommentMap
	info  *types.Info

	program   *ir.Program
	funcTypes map[*types.Func]*ir.Func
	varTypes  map[*types.Var]*ir.Variable

	warnings []error
}

func (b *builder) addWarning(err error) {
	b.warnings = append(b.warnings, err)
}

func (b *builder) processFuncDeclsInFile(file *ast.File) {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := funcDecl.Name.Name
		funcType := b.info.Defs[funcDecl.Name].(*types.Func)
		sig := funcType.Type().(*types.Signature)
		f := b.program.AddOuterFunc(name, sig, decl.Pos(), decl.End())
		v := ir.NewVariable(name, ir.FuncType, f.FuncValue())
		if funcDecl.Recv != nil && len(funcDecl.Recv.List) == 1 {
			field := funcDecl.Recv.List[0]
			fieldNameIdent := field.Names[0]
			if t, ok := typesTypeToIrType(b.info.TypeOf(field.Type).Underlying()); ok {
				varType := b.info.Defs[fieldNameIdent].(*types.Var)
				v := ir.NewVariable(fieldNameIdent.Name, t, -1)
				f.AddArg(-1, v)
				b.varTypes[varType] = v
			}
		}
		b.processFuncType(funcDecl.Type, newContext(b.cmaps[file], f))
		b.program.Scope().AddVariable(v)
		b.funcTypes[funcType] = f
	}
}

func (b *builder) processGenDeclsInFile(file *ast.File) {
	cmap := b.cmaps[file]
	entryCtx := newContext(cmap, b.program.EntryFunc())
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		b.processGenDecl(genDecl, b.program.Scope(), entryCtx)
	}
}

func (b *builder) processFuncDefsInFile(file *ast.File) {
	cmap := b.cmaps[file]
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		funcType, ok := b.info.Defs[funcDecl.Name].(*types.Func)
		if !ok {
			continue
		}
		f := b.funcTypes[funcType]
		b.processStmt(funcDecl.Body, newContext(cmap, f))
	}
}

func (b *builder) processGenDecl(genDecl *ast.GenDecl, scope *ir.Scope, ctx *context) {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		lhs := make(map[int]*ir.Variable)
		for i, nameIdent := range valueSpec.Names {
			t, ok := typesTypeToIrType(b.info.TypeOf(nameIdent))
			if !ok {
				continue
			}

			varType := b.info.Defs[nameIdent].(*types.Var)
			v := ir.NewVariable(nameIdent.Name, t, -1)
			lhs[i] = v
			scope.AddVariable(v)
			b.varTypes[varType] = v
		}

		if len(valueSpec.Values) == 0 {
			continue
		}

		// Handle single call expression:
		callExpr, ok := valueSpec.Values[0].(*ast.CallExpr)
		if ok && len(valueSpec.Values) == 1 {
			b.processCallExprWithResultVars(callExpr, ir.Call, lhs, ctx)
			continue
		}

		// Handle value expressions:
		for i, expr := range valueSpec.Values {
			l := lhs[i]
			r := b.processExpr(expr, ctx)
			if l == nil && r == nil {
				continue
			} else if l == nil {
				p := b.fset.Position(valueSpec.Names[i].Pos())
				b.addWarning(fmt.Errorf("%v: could not handle lhs of assignment", p))
				continue
			} else if r == nil {
				p := b.fset.Position(valueSpec.Values[i].Pos())
				b.addWarning(
					fmt.Errorf("%v: could not handle rhs of assignment", p))
				continue
			}

			assignStmt := ir.NewAssignStmt(r, l, valueSpec.Pos(), valueSpec.End())
			ctx.body.AddStmt(assignStmt)
		}
	}
}
