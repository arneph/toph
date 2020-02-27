package builder

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	"github.com/arneph/toph/ir"
)

// BuildProgram parses the Go files at the given path and builds an ir.Program.
func BuildProgram(path string) (*ir.Program, []error) {
	b := new(builder)

	// Parse program:
	b.fset = token.NewFileSet()
	pkgs, err := parser.ParseDir(b.fset, path, nil, mode)
	if err != nil {
		b.addWarning(fmt.Errorf("failed to parse input: %v", err))
		return b.program, b.warnings
	} else if len(pkgs) != 1 {
		b.addWarning(fmt.Errorf("expected exactly one package, got: %d", len(pkgs)))
	}

	pkgName := "main"
	var files []*ast.File
	for name, pkg := range pkgs {
		pkgName = name

		for _, file := range pkg.Files {
			files = append(files, file)
		}
	}

	// Type check:
	b.info = new(types.Info)
	b.info.Defs = make(map[*ast.Ident]types.Object)
	b.info.Uses = make(map[*ast.Ident]types.Object)
	b.info.Selections = make(map[*ast.SelectorExpr]*types.Selection)
	b.info.Scopes = make(map[ast.Node]*types.Scope)
	conf := types.Config{Importer: importer.Default()}
	_, err = conf.Check(pkgName, b.fset, files, b.info)
	if err != nil {
		b.addWarning(fmt.Errorf("failed to type check input: %v", err))
		return b.program, b.warnings
	}

	b.program = ir.NewProgram()
	b.funcTypes = make(map[*types.Func]*ir.Func)
	b.varTypes = make(map[*types.Var]*ir.Variable)
	for _, file := range files {
		b.processFile(file)
	}

	return b.program, b.warnings
}

const mode = parser.ParseComments |
	parser.DeclarationErrors |
	parser.AllErrors

type builder struct {
	fset *token.FileSet
	info *types.Info

	program   *ir.Program
	funcTypes map[*types.Func]*ir.Func
	varTypes  map[*types.Var]*ir.Variable

	warnings []error
}

func (b *builder) addWarning(err error) {
	b.warnings = append(b.warnings, err)
}

type context struct {
	body           *ir.Body
	enclosingFuncs []*ir.Func
}

func makeContext(f *ir.Func) context {
	return context{
		body:           f.Body(),
		enclosingFuncs: []*ir.Func{f},
	}
}

func (c context) currentFunc() *ir.Func {
	n := len(c.enclosingFuncs)
	return c.enclosingFuncs[n-1]
}

func (c context) subContextForBody(containedBody *ir.Body) context {
	return context{
		body:           containedBody,
		enclosingFuncs: c.enclosingFuncs,
	}
}

func (c context) subContextForFunc(containedFunc *ir.Func) context {
	return context{
		body:           containedFunc.Body(),
		enclosingFuncs: append(c.enclosingFuncs, containedFunc),
	}
}

func (b *builder) processFile(file *ast.File) {
	// Process function declarations:
	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			b.processGenDecl(decl, b.program.Scope())

		case *ast.FuncDecl:
			name := decl.Name.Name
			funcType := b.info.Defs[decl.Name].(*types.Func)
			f := ir.NewFunc(name, b.program.Scope())
			v := ir.NewVariable(name, ir.FuncType, f.FuncValue())
			b.program.AddNamedFunc(name, f)
			b.program.Scope().AddVariable(v)
			b.funcTypes[funcType] = f
		}
	}

	// Build all declared functions:
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		name := funcDecl.Name.Name
		f := b.program.GetNamedFunc(name)
		b.processFunc(funcDecl.Type, funcDecl.Body, makeContext(f))
	}
}

func (b *builder) processGenDecl(genDecl *ast.GenDecl, scope *ir.Scope) {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		t, ok := astTypeToIrType(valueSpec.Type)
		if !ok {
			continue
		}

		for _, nameIdent := range valueSpec.Names {
			name := nameIdent.Name
			varType := b.info.Defs[nameIdent].(*types.Var)
			v := ir.NewVariable(name, t, -1)
			scope.AddVariable(v)
			b.varTypes[varType] = v
		}
	}
}
