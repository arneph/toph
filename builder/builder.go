package builder

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"

	"github.com/arneph/toph/ir"
)

const parserMode = parser.ParseComments |
	parser.DeclarationErrors |
	parser.AllErrors

// BuildProgram parses the Go files at the given path and builds an ir.Program.
func BuildProgram(path string, entryFuncName string, buildContext build.Context) (*ir.Program, []error) {
	b := new(builder)

	// Parse program:
	b.fset = token.NewFileSet()
	type importTask struct {
		path   string
		srcDir string
	}
	pathsQueue := []importTask{importTask{".", path}}
	pathsQueueSet := make(map[string]bool)
	pathsQueueSet[path] = true
	astPkgs := make(map[string]*ast.Package)
	for len(pathsQueue) > 0 {
		path := pathsQueue[0].path
		srcDir := pathsQueue[0].srcDir
		pathsQueue = pathsQueue[1:]

		buildPkg, err := buildContext.Import(path, srcDir, build.ImportComment)
		if err != nil {
			b.addWarning(fmt.Errorf("import of %q from %s failed: \n\t%v", path, srcDir, err))
			continue
		}
		if buildPkg.Goroot {
			continue
		}
		absPath := buildPkg.Dir

		for _, importPath := range buildPkg.Imports {
			if importPath == "C" {
				continue
			}
			if pathsQueueSet[importPath] {
				continue
			} else {
				pathsQueue = append(pathsQueue, importTask{importPath, absPath})
				pathsQueueSet[importPath] = true
			}
		}

		filter := func(info os.FileInfo) bool {
			ok, err := buildContext.MatchFile(absPath, info.Name())
			if err != nil {
				b.addWarning(fmt.Errorf("matching file failed: %v", err))
				return true
			}
			return ok
		}
		parsedASTPkgs, err := parser.ParseDir(b.fset, absPath, filter, parserMode)
		if err != nil {
			b.addWarning(fmt.Errorf("parsing failed: %v", err))
			return nil, b.warnings
		}
		for _, pkg := range parsedASTPkgs {
			astPkgs[absPath] = pkg
		}
	}
	if len(astPkgs) < 1 {
		b.addWarning(fmt.Errorf("expected at least one package"))
	}
	subsFile, err := parser.ParseFile(b.fset, "substitutes.go", substitutesCode, parserMode)
	if err != nil {
		b.addWarning(fmt.Errorf("parsing substitutes failed: %v", err))
		return nil, b.warnings
	}
	subsPkg := new(ast.Package)
	subsPkg.Name = "subs"
	subsPkg.Files = map[string]*ast.File{"substitutes.go": subsFile}
	astPkgs["subs"] = subsPkg

	// Type check:
	b.pkgTypesInfos = make(map[string]*types.Info)
	b.pkgTypesPackages = make(map[string]*types.Package)
	b.pkgFuncTypes = make(map[string]map[*types.Func]*ir.Func)
	b.pkgVarTypes = make(map[string]map[*types.Var]*ir.Variable)
	paths := make(map[string]struct{})
	typesConfig := types.Config{
		Importer:                 importer.ForCompiler(b.fset, "source", nil),
		FakeImportC:              true,
		DisableUnusedImportCheck: true,
	}
	for pkgName, astPkg := range astPkgs {
		info := new(types.Info)
		info.Types = make(map[ast.Expr]types.TypeAndValue)
		info.Defs = make(map[*ast.Ident]types.Object)
		info.Uses = make(map[*ast.Ident]types.Object)
		info.Selections = make(map[*ast.SelectorExpr]*types.Selection)
		info.Scopes = make(map[ast.Node]*types.Scope)

		var astFiles []*ast.File
		for _, file := range astPkg.Files {
			astFiles = append(astFiles, file)
		}

		typesPkg, err := typesConfig.Check(pkgName, b.fset, astFiles, info)
		if err != nil {
			b.addWarning(fmt.Errorf("type checking failed: %v", err))
			return nil, b.warnings
		}
		path := typesPkg.Path()
		if _, ok := paths[path]; ok {
			b.addWarning(fmt.Errorf("type checking failed: found repeated package path: %s", path))
			return nil, b.warnings
		}
		paths[path] = struct{}{}
		b.pkgTypesInfos[path] = info
		b.pkgTypesPackages[path] = typesPkg
		b.pkgFuncTypes[path] = make(map[*types.Func]*ir.Func)
		b.pkgVarTypes[path] = make(map[*types.Var]*ir.Variable)
	}

	// Comment maps:
	b.cmaps = make(map[*ast.File]ast.CommentMap)
	for _, pkg := range astPkgs {
		for _, file := range pkg.Files {
			b.cmaps[file] = ast.NewCommentMap(b.fset, file, file.Comments)
		}
	}

	// IR setup:
	b.program = ir.NewProgram(b.fset)

	// File processing:
	for pkgName, astPkg := range astPkgs {
		for _, astFile := range astPkg.Files {
			b.processFuncDeclsInFile(pkgName, astFile)
		}
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

	for pkgName, astPkg := range astPkgs {
		for _, astFile := range astPkg.Files {
			b.processGenDeclsInFile(pkgName, astFile)
		}
	}

	for pkgName, astPkg := range astPkgs {
		for _, astFile := range astPkg.Files {
			b.processFuncDefsInFile(pkgName, astFile)
		}
	}

	return b.program, b.warnings
}

type builder struct {
	fset             *token.FileSet
	pkgTypesInfos    map[string]*types.Info
	pkgTypesPackages map[string]*types.Package
	pkgFuncTypes     map[string]map[*types.Func]*ir.Func
	pkgVarTypes      map[string]map[*types.Var]*ir.Variable
	cmaps            map[*ast.File]ast.CommentMap

	program *ir.Program

	warnings []error
}

func (b *builder) addWarning(err error) {
	b.warnings = append(b.warnings, err)
}

func (b *builder) nodeToString(node ast.Node) string {
	var bob strings.Builder
	format.Node(&bob, b.fset, node)
	return bob.String()
}

func (b *builder) processFuncDeclsInFile(pkg string, file *ast.File) {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := funcDecl.Name.Name
		typesFunc := b.pkgTypesInfos[pkg].ObjectOf(funcDecl.Name).(*types.Func)
		typesSig := typesFunc.Type().(*types.Signature)
		irFunc := b.program.AddOuterFunc(name, typesSig, decl.Pos(), decl.End())
		ctx := newContext(pkg, b.cmaps[file], irFunc)
		b.processFuncReceiver(funcDecl.Recv, ctx)
		b.processFuncType(funcDecl.Type, ctx)
		b.pkgFuncTypes[pkg][typesFunc] = irFunc
	}
}

func (b *builder) processFuncDefsInFile(pkg string, file *ast.File) {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		funcType, ok := b.pkgTypesInfos[pkg].Defs[funcDecl.Name].(*types.Func)
		if !ok {
			continue
		}
		irFunc := b.pkgFuncTypes[pkg][funcType]
		ctx := newContext(pkg, b.cmaps[file], irFunc)
		b.processFuncBody(funcDecl.Body, ctx)
	}
}

func (b *builder) processGenDeclsInFile(pkg string, file *ast.File) {
	entryCtx := newContext(pkg, b.cmaps[file], b.program.EntryFunc())
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		b.processGenDecl(genDecl, b.program.Scope(), entryCtx)
	}
}

func (b *builder) processGenDecl(genDecl *ast.GenDecl, scope *ir.Scope, ctx *context) {
	if genDecl.Tok != token.CONST && genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		valueSpec := spec.(*ast.ValueSpec)

		b.processVarDefinitionsInScope(valueSpec.Names, scope, ctx)

		if len(valueSpec.Values) == 0 {
			continue
		}

		lhsExprs := make([]ast.Expr, len(valueSpec.Names))
		for i, name := range valueSpec.Names {
			lhsExprs[i] = name
		}
		b.processAssignments(lhsExprs, valueSpec.Values, ctx)
	}
}
