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

const mode = parser.ParseComments |
	parser.DeclarationErrors |
	parser.AllErrors

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

	b.addedSubstitutes = make(map[*types.Func]*ir.Func)

	for _, file := range files {
		b.processFile(file)
	}

	return b.program, b.warnings
}

type builder struct {
	fset *token.FileSet
	info *types.Info

	program   *ir.Program
	funcTypes map[*types.Func]*ir.Func
	varTypes  map[*types.Var]*ir.Variable

	addedSubstitutes map[*types.Func]*ir.Func

	warnings []error
}

func (b *builder) addWarning(err error) {
	b.warnings = append(b.warnings, err)
}

type context struct {
	cmap ast.CommentMap

	body           *ir.Body
	enclosingFuncs []*ir.Func

	enclosingStmts      []ir.Stmt
	enclosingStmtLabels map[string]ir.Stmt
}

func newContext(cmap ast.CommentMap, f *ir.Func) *context {
	ctx := new(context)
	ctx.cmap = cmap
	ctx.body = f.Body()
	ctx.enclosingFuncs = []*ir.Func{f}
	ctx.enclosingStmts = []ir.Stmt{}
	ctx.enclosingStmtLabels = make(map[string]ir.Stmt)

	return ctx
}

func (c *context) currentFunc() *ir.Func {
	n := len(c.enclosingFuncs)
	return c.enclosingFuncs[n-1]
}

func (c *context) currentLoop() ir.Loop {
	if len(c.enclosingStmts) == 0 {
		return nil
	}
	for i := len(c.enclosingStmts) - 1; i >= 0; i-- {
		loop, ok := c.enclosingStmts[i].(ir.Loop)
		if ok {
			return loop
		}
	}
	return nil
}

func (c *context) currentLabeledLoop(label string) ir.Loop {
	stmt, ok := c.enclosingStmtLabels[label]
	if !ok {
		return nil
	}
	loop, ok := stmt.(ir.Loop)
	if !ok {
		return nil
	}
	return loop
}

func (c *context) subContextForBody(stmt ir.Stmt, label string, containedBody *ir.Body) *context {
	ctx := new(context)
	ctx.cmap = c.cmap
	ctx.body = containedBody
	ctx.enclosingFuncs = c.enclosingFuncs
	ctx.enclosingStmts = append(c.enclosingStmts, stmt)
	ctx.enclosingStmtLabels = make(map[string]ir.Stmt)
	for l, s := range c.enclosingStmtLabels {
		ctx.enclosingStmtLabels[l] = s
	}
	if label != "" {
		ctx.enclosingStmtLabels[label] = stmt
	}

	return ctx
}

func (c *context) subContextForFunc(containedFunc *ir.Func) *context {
	ctx := new(context)
	ctx.cmap = c.cmap
	ctx.body = containedFunc.Body()
	ctx.enclosingFuncs = append(c.enclosingFuncs, containedFunc)
	ctx.enclosingStmts = []ir.Stmt{}
	ctx.enclosingStmtLabels = make(map[string]ir.Stmt)

	return ctx
}

func (b *builder) processFile(file *ast.File) {
	mainFunc := ir.NewOuterFunc("main", b.program.Scope())
	cmap := ast.NewCommentMap(b.fset, file, file.Comments)
	mainCtx := newContext(cmap, mainFunc)

	// Process declarations:
	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			b.processGenDecl(decl, b.program.Scope(), mainCtx)

		case *ast.FuncDecl:
			name := decl.Name.Name
			funcType := b.info.Defs[decl.Name].(*types.Func)
			var f *ir.Func
			if name == "main" {
				f = mainFunc
			} else {
				f = ir.NewOuterFunc(name, b.program.Scope())
			}
			v := ir.NewVariable(name, ir.FuncType, f.FuncValue())

			b.processFuncType(decl.Type, newContext(cmap, f))

			b.program.AddFunc(f)
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
		f := b.program.GetFunc(name)
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
			return
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
			} else if r == nil {
				p := b.fset.Position(valueSpec.Values[i].Pos())
				b.addWarning(
					fmt.Errorf("%v: could not handle rhs of assignment", p))
			}

			assignStmt := ir.NewAssignStmt(r, l)
			ctx.body.AddStmt(assignStmt)
		}
	}
}
