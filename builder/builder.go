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
	"path/filepath"
	"strings"

	"github.com/arneph/toph/ir"
)

const buildImportMode build.ImportMode = build.ImportComment
const parserMode parser.Mode = parser.ParseComments |
	parser.DeclarationErrors |
	parser.AllErrors

// BuildProgram parses the Go files at the given path and builds an ir.Program.
func BuildProgram(path string, entryFuncName string, buildContext *build.Context) (*ir.Program, []error) {
	b := new(builder)

	// Temporarily change build context (needed for source importer):
	oldBuildContext := build.Default
	build.Default = *buildContext
	defer func() {
		build.Default = oldBuildContext
	}()

	// Parse program:
	b.buildContext = buildContext
	b.fset = token.NewFileSet()
	b.pkgs = make(map[string]*pkg)

	absPath, err := filepath.Abs(path)
	if err != nil {
		b.addWarning(fmt.Errorf("could not find absolute path for %q: %v", path, err))
	} else {
		path = absPath
	}
	buildPackage, err := b.buildContext.Import(".", path, buildImportMode)
	if err != nil {
		b.addWarning(fmt.Errorf("import of %q failed: \n\t%v", path, err))
		return nil, b.warnings
	}
	queue := []*build.Package{buildPackage}
	queueSet := map[string]bool{buildPackage.Dir: true}
	for len(queue) > 0 {
		buildPackage := queue[0]
		queue = queue[1:]

		absPath := buildPackage.Dir
		b.pkgs[absPath] = new(pkg)
		b.pkgs[absPath].buildPackage = buildPackage

		for i, importPath := range append(buildPackage.Imports, buildPackage.TestImports...) {
			if importPath == "C" {
				var p token.Position
				if i < len(buildPackage.Imports) {
					p = buildPackage.ImportPos["C"][0]
				} else {
					p = buildPackage.TestImportPos["C"][0]
				}
				b.addWarning(fmt.Errorf("%v: cgo (import \"C\") not supported", p))
				return nil, b.warnings
			}

			importedBuildPackage, err := b.buildContext.Import(importPath, absPath, buildImportMode)
			if err != nil {
				b.addWarning(fmt.Errorf("import of %q from %s failed: \n\t%v", importPath, absPath, err))
				continue
			} else if importedBuildPackage.Goroot {
				continue
			}

			absImportPath := importedBuildPackage.Dir
			b.pkgs[absPath].absImportPaths = append(b.pkgs[absPath].absImportPaths, absImportPath)
			if queueSet[absImportPath] {
				continue
			} else {
				queue = append(queue, importedBuildPackage)
				queueSet[absImportPath] = true
			}
		}

		filter := func(info os.FileInfo) bool {
			ok, err := b.buildContext.MatchFile(absPath, info.Name())
			if err != nil {
				b.addWarning(fmt.Errorf("file matching failed: %v", err))
				return true
			}
			return ok
		}
		astPackages, err := parser.ParseDir(b.fset, absPath, filter, parserMode)
		if err != nil {
			b.addWarning(fmt.Errorf("parsing failed: %v", err))
			return nil, b.warnings
		}
		for _, astPackage := range astPackages {
			if astPackage.Name != buildPackage.Name {
				continue
			}
			b.pkgs[absPath].astPackage = astPackage
		}
	}
	if len(b.pkgs) < 1 {
		b.addWarning(fmt.Errorf("expected at least one package"))
	}
	subsFile, err := parser.ParseFile(b.fset, "substitutes.go", substitutesCode, parserMode)
	if err != nil {
		b.addWarning(fmt.Errorf("parsing substitutes failed: %v", err))
		return nil, b.warnings
	}
	subsAstPackage := new(ast.Package)
	subsAstPackage.Name = "subs"
	subsAstPackage.Files = map[string]*ast.File{"substitutes.go": subsFile}
	b.pkgs["subs"] = new(pkg)
	b.pkgs["subs"].astPackage = subsAstPackage

	// Type check:
	orderedPaths, err := b.packageProcessingOrder()
	if err != nil {
		b.addWarning(fmt.Errorf("type checking failed: %v", err))
		return nil, b.warnings
	}

	b.typesInfo = new(types.Info)
	b.typesInfo.Types = make(map[ast.Expr]types.TypeAndValue)
	b.typesInfo.Defs = make(map[*ast.Ident]types.Object)
	b.typesInfo.Uses = make(map[*ast.Ident]types.Object)
	b.typesInfo.Implicits = make(map[ast.Node]types.Object)
	b.typesInfo.Selections = make(map[*ast.SelectorExpr]*types.Selection)
	b.typesInfo.Scopes = make(map[ast.Node]*types.Scope)
	b.typesInfo.InitOrder = make([]*types.Initializer, 0)
	b.typesSrcImporter = importer.ForCompiler(b.fset, "source", nil)
	b.funcs = make(map[*types.Func]*ir.Func)
	b.vars = make(map[*types.Var]*ir.Variable)
	typesConfig := types.Config{
		Importer:                 b,
		DisableUnusedImportCheck: true,
	}

	for _, path := range orderedPaths {
		buildPackage := b.pkgs[path].buildPackage
		astPackage := b.pkgs[path].astPackage

		var importPath string
		var astFiles []*ast.File
		if path == "subs" {
			importPath = "subs"
		} else {
			importPath = buildPackage.ImportPath
		}
		for _, file := range astPackage.Files {
			astFiles = append(astFiles, file)
		}

		typesPackage, err := typesConfig.Check(importPath, b.fset, astFiles, b.typesInfo)
		if err != nil {
			b.addWarning(fmt.Errorf("type checking failed: %v", err))
			return nil, b.warnings
		}
		b.pkgs[path].typesPackage = typesPackage
	}

	// Comment maps:
	b.cmaps = make(map[*ast.File]ast.CommentMap)
	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.astPackage.Files {
			b.cmaps[astFile] = ast.NewCommentMap(b.fset, astFile, astFile.Comments)
		}
	}

	// IR setup:
	b.program = ir.NewProgram(b.fset)

	// File processing:
	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.astPackage.Files {
			b.processFuncDeclsInFile(astFile)
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

	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.astPackage.Files {
			b.processGenDeclsInFile(astFile)
		}
	}

	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.astPackage.Files {
			b.processFuncDefsInFile(astFile)
		}
	}

	return b.program, b.warnings
}

type pkg struct {
	astPackage   *ast.Package
	buildPackage *build.Package
	typesPackage *types.Package

	absImportPaths []string
}

type builder struct {
	buildContext     *build.Context
	fset             *token.FileSet
	typesInfo        *types.Info
	typesSrcImporter types.Importer
	pkgs             map[string]*pkg
	funcs            map[*types.Func]*ir.Func
	vars             map[*types.Var]*ir.Variable
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

func (b *builder) processFuncDeclsInFile(file *ast.File) {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := funcDecl.Name.Name
		typesFunc := b.typesInfo.Defs[funcDecl.Name].(*types.Func)
		typesSig := typesFunc.Type().(*types.Signature)
		irFunc := b.program.AddOuterFunc(name, typesSig, decl.Pos(), decl.End())
		ctx := newContext(b.cmaps[file], irFunc)
		b.processFuncReceiver(funcDecl.Recv, ctx)
		b.processFuncType(funcDecl.Type, ctx)
		b.funcs[typesFunc] = irFunc
	}
}

func (b *builder) processFuncDefsInFile(file *ast.File) {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		typesFunc := b.typesInfo.Defs[funcDecl.Name].(*types.Func)
		irFunc := b.funcs[typesFunc]
		ctx := newContext(b.cmaps[file], irFunc)
		b.processFuncBody(funcDecl.Body, ctx)
	}
}

func (b *builder) processGenDeclsInFile(file *ast.File) {
	entryCtx := newContext(b.cmaps[file], b.program.EntryFunc())
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
