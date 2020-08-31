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

	c "github.com/arneph/toph/config"
	"github.com/arneph/toph/ir"

	"github.com/arneph/toph/builder/packages"
	"golang.org/x/tools/go/types/typeutil"
)

const loadMode packages.LoadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo
const parserMode parser.Mode = parser.ParseComments |
	parser.DeclarationErrors |
	parser.AllErrors

// BuildProgram parses the Go files at the given path and builds an ir.Program.
func BuildProgram(paths []string, config *c.Config) (program *ir.Program, entryFuncs []*ir.Func, errs []error) {
	b := new(builder)
	b.config = config

	absPaths := make([]string, len(paths))
	for i, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			b.addWarning(fmt.Errorf("could not find absolute path for %q: %v", path, err))
		} else {
			absPaths[i] = absPath
		}
	}

	// Parse program:
	b.fset = token.NewFileSet()
	b.typesPkgs = make(map[*types.Package]struct{})

	oldDefault := build.Default
	build.Default = *config.BuildContext
	defer func() {
		build.Default = oldDefault
	}()

	packagesConfig := &packages.Config{
		Mode: loadMode,
		Env: append(os.Environ(),
			"GOOS="+config.BuildContext.GOOS,
			"GOARCH="+config.BuildContext.GOARCH,
		),
		Fset:  b.fset,
		Tests: true,
	}
	rootPackages, err := packages.Load(packagesConfig, absPaths...)
	if err != nil {
		b.addWarning(err)
		return
	}
	packages.Visit(rootPackages, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			b.addWarning(err)
		}
	})
	packages.Visit(rootPackages, func(pkg *packages.Package) bool {
		if pkg.Name == "unsafe" {
			return false
		} else if len(pkg.GoFiles) == 0 {
			return false
		} else if strings.HasPrefix(pkg.GoFiles[0], config.BuildContext.GOROOT) {
			return false
		} else if config.ShouldExcludeEntirePackage(pkg.PkgPath) {
			return false
		}
		b.typesPkgs[pkg.Types] = struct{}{}
		return true
	}, func(pkg *packages.Package) {
		if pkg.Name == "unsafe" {
			return
		} else if len(pkg.GoFiles) == 0 {
			b.addWarning(fmt.Errorf("no files in package: %s", pkg.PkgPath))
			return
		} else if pkg.IllTyped {
			b.addWarning(fmt.Errorf("skipped due to incomplete type information: %s", pkg.PkgPath))
			return
		} else if strings.HasPrefix(pkg.GoFiles[0], config.BuildContext.GOROOT) {
			return
		} else if config.ShouldExcludeEntirePackage(pkg.PkgPath) {
			return
		}
		if config.Debug {
			b.addWarning(fmt.Errorf("translating package: %s", pkg.PkgPath))
		}
		b.pkgs = append(b.pkgs, pkg)
	})

	subsFile, err := parser.ParseFile(b.fset, "substitutes.go", substitutesCode, parserMode)
	if err != nil {
		b.addWarning(fmt.Errorf("parsing substitutes failed: %v", err))
		return nil, nil, b.warnings
	}
	subsTypesInfo := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	subsTypesConfig := &types.Config{
		Importer: importer.ForCompiler(b.fset, "source", nil),
	}
	_, err = subsTypesConfig.Check("subs", b.fset, []*ast.File{subsFile}, subsTypesInfo)
	if err != nil {
		b.addWarning(fmt.Errorf("type checking substitutes failed: %v", err))
		return nil, nil, b.warnings
	}

	// Types:
	b.funcs = make(map[*types.Func]*ir.Func)
	b.vars = make(map[*types.Var]*ir.Variable)
	b.fields = make(map[*types.Var]*ir.Field)

	// Comment maps:
	b.cmaps = make(map[*ast.File]ast.CommentMap)
	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.Syntax {
			b.cmaps[astFile] = ast.NewCommentMap(b.fset, astFile, astFile.Comments)
		}
	}

	// IR setup:
	b.program = ir.NewProgram(b.fset)
	b.liftedSpecialOpFuncs = make(map[ir.SpecialOp]*ir.Func)

	// Substitures processing:
	b.processFuncDeclsInFile(subsFile, subsTypesInfo)
	b.processGenDeclsInFile(subsFile, subsTypesInfo)
	b.processFuncDefsInFile(subsFile, subsTypesInfo)

	// AST processing:
	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.Syntax {
			b.processFuncDeclsInFile(astFile, pkg.TypesInfo)
		}
	}

	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.Syntax {
			b.processGenDeclsInFile(astFile, pkg.TypesInfo)
		}
	}

	for _, pkg := range b.pkgs {
		for _, init := range pkg.TypesInfo.InitOrder {
			b.processInitializer(pkg.TypesInfo, init)
		}
		for _, astFile := range pkg.Syntax {
			for _, decl := range astFile.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || funcDecl.Name.Name != "init" {
					continue
				}
				typesFunc := pkg.TypesInfo.Defs[funcDecl.Name].(*types.Func)
				irFunc := b.funcs[typesFunc]
				if irFunc.Signature() != nil &&
					irFunc.Signature().Recv() == nil &&
					irFunc.Signature().Params() == nil &&
					irFunc.Signature().Results() == nil {
					callStmt := ir.NewCallStmt(irFunc, irFunc.Signature(), ir.Call, funcDecl.Pos(), funcDecl.End())
					b.program.InitFunc().Body().AddStmt(callStmt)
				}
			}
		}
	}

	for _, pkg := range b.pkgs {
		for _, astFile := range pkg.Syntax {
			b.processFuncDefsInFile(astFile, pkg.TypesInfo)
		}
	}

	// Entry funcs:
	for _, irFunc := range b.program.Funcs() {
		if irFunc.Name() == "main" &&
			irFunc.Signature() != nil &&
			irFunc.Signature().String() == "func()" {
			entryFuncs = append(entryFuncs, irFunc)
		} else if strings.HasPrefix(irFunc.Name(), "Test") &&
			irFunc.Signature() != nil &&
			irFunc.Signature().String() == "func(t *testing.T)" {
			entryFuncs = append(entryFuncs, irFunc)
		}
	}

	return b.program, entryFuncs, b.warnings
}

type builder struct {
	fset      *token.FileSet
	pkgs      []*packages.Package
	typesPkgs map[*types.Package]struct{}
	funcs     map[*types.Func]*ir.Func
	vars      map[*types.Var]*ir.Variable
	fields    map[*types.Var]*ir.Field
	types     typeutil.Map
	cmaps     map[*ast.File]ast.CommentMap

	program              *ir.Program
	liftedSpecialOpFuncs map[ir.SpecialOp]*ir.Func

	config *c.Config

	warnings []error
}

func (b *builder) addWarning(err error) {
	b.warnings = append(b.warnings, err)
}

func (b *builder) nodeToString(node ast.Node) string {
	var bob strings.Builder
	format.Node(&bob, b.fset, node)
	str := bob.String()
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\t", " ")
	return str
}

func (b *builder) processFuncDeclsInFile(file *ast.File, typesInfo *types.Info) {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := funcDecl.Name.Name
		typesFunc := typesInfo.Defs[funcDecl.Name].(*types.Func)
		typesSig := typesFunc.Type().(*types.Signature)
		irFunc := b.program.AddOuterFunc(name, typesSig, decl.Pos(), decl.End())
		ctx := newContext(b.cmaps[file], typesInfo, irFunc)
		b.processFuncReceiver(funcDecl.Recv, ctx)
		b.processFuncType(funcDecl.Type, ctx)
		b.funcs[typesFunc] = irFunc
	}
}

func (b *builder) processFuncDefsInFile(file *ast.File, typesInfo *types.Info) {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		typesFunc := typesInfo.Defs[funcDecl.Name].(*types.Func)
		irFunc := b.funcs[typesFunc]
		ctx := newContext(b.cmaps[file], typesInfo, irFunc)
		b.processFuncBody(funcDecl.Body, ctx)
	}
}

func (b *builder) processGenDeclsInFile(file *ast.File, typesInfo *types.Info) {
	initCtx := newContext(b.cmaps[file], typesInfo, b.program.InitFunc())
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		b.processGenDecl(genDecl, false, b.program.Scope(), initCtx)
	}
}

func (b *builder) processGenDecl(genDecl *ast.GenDecl, processInitializations bool, scope *ir.Scope, ctx *context) {
	if genDecl.Tok != token.CONST && genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		valueSpec := spec.(*ast.ValueSpec)

		b.processVarDefinitionsInScope(valueSpec.Names, scope, len(valueSpec.Values) == 0, ctx)

		if !processInitializations || len(valueSpec.Values) == 0 {
			continue
		}
		lhs := make([]ast.Expr, len(valueSpec.Names))
		rhs := valueSpec.Values
		for i, ident := range valueSpec.Names {
			lhs[i] = ident
		}
		b.processAssignments(lhs, rhs, ctx)
	}
}

func (b *builder) processInitializer(typesInfo *types.Info, init *types.Initializer) {
	initCtx := newContext(nil, typesInfo, b.program.InitFunc())

	lhs := make(map[int]*ir.Variable)
	rhs := make(map[int]ir.RValue)
	for i, typesVar := range init.Lhs {
		irVar, ok := b.vars[typesVar]
		if ok {
			lhs[i] = irVar
		}
	}
	callExpr, ok := init.Rhs.(*ast.CallExpr)
	if ok {
		rhsVars := b.processCallExpr(callExpr, initCtx)
		for i, v := range rhsVars {
			rhs[i] = v
		}
	} else {
		rhs[0] = b.processExpr(init.Rhs, initCtx)
	}

	// Create assignment statements:
	for i := 0; i < len(init.Lhs); i++ {
		rhsExpr := init.Rhs
		l := lhs[i]
		r := rhs[i]
		if l == nil && r == nil {
			continue
		} else if l == nil && r == ir.Nil {
			continue
		} else if l == nil {
			p := b.fset.Position(init.Lhs[i].Pos())
			rhsExprStr := b.nodeToString(rhsExpr)
			b.addWarning(fmt.Errorf("%v: could not handle lhs of assignment: %s", p, rhsExprStr))
			continue
		} else if r == nil {
			p := b.fset.Position(rhsExpr.Pos())
			rhsExprStr := b.nodeToString(rhsExpr)
			b.addWarning(
				fmt.Errorf("%v: could not handle rhs of assignment: %s", p, rhsExprStr))
			continue
		}
		requiresCopy := false
		typesType := typesInfo.TypeOf(rhsExpr)
		if _, ok := l.Type().(*ir.StructType); ok {
			requiresCopy = !b.isPointer(typesType)
		} else if irContainerType, ok := l.Type().(*ir.ContainerType); ok && irContainerType.Kind() == ir.Array {
			requiresCopy = !b.isPointer(typesType)
		}
		assignStmt := ir.NewAssignStmt(r, l, requiresCopy, rhsExpr.Pos(), rhsExpr.End())
		initCtx.body.AddStmt(assignStmt)
	}
}
