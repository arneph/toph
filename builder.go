package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/arneph/toph/ir"
)

const mode = parser.ParseComments |
	parser.DeclarationErrors |
	parser.AllErrors

func buildProg(path string) (*ir.Prog, error) {
	fset := token.NewFileSet()
	pkgs, first := parser.ParseDir(fset, path, nil, mode)
	if first != nil {
		return nil, first
	}

	prog := ir.NewProg(fset)

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			err := processFile(file, prog)

			if err != nil {
				return nil, err
			}
		}
	}

	return prog, nil
}

func processFile(file *ast.File, prog *ir.Prog) error {
	// Create all declared channels and functions:
	for _, decl := range file.Decls {
		processDecl(decl, prog.Scope())
	}

	// Build all declared functions:
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		f := prog.Scope().FindNamedFunc(funcDecl.Name.Name)
		err := processFunc(funcDecl.Type, funcDecl.Body, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func processDecl(decl ast.Decl, scope *ir.Scope) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		scope.AddNamedFunc(ir.NewFunc(d), d.Name.Name)

	case *ast.GenDecl:
		if d.Tok != token.VAR {
			return
		}
		for _, spec := range d.Specs {
			valueSpec := spec.(*ast.ValueSpec)
			_, ok := valueSpec.Type.(*ast.ChanType)
			if !ok {
				continue
			}
			for _, chanName := range valueSpec.Names {
				scope.AddNamedChan(ir.NewChan(chanName), chanName.Name)
			}
		}

	default:
		fmt.Printf("ignoring %T declaration\n", d)
	}
}

func processFunc(funcType *ast.FuncType, funcBody *ast.BlockStmt, f *ir.Func) error {
	for _, field := range funcType.Params.List {
		switch field.Type.(type) {
		case *ast.FuncType:
			for _, fieldNameIdent := range field.Names {
				f.AddArg(ir.NewFunc(fieldNameIdent), fieldNameIdent.Name)
			}
		case *ast.ChanType:
			for _, fieldNameIdent := range field.Names {
				f.AddArg(ir.NewChan(fieldNameIdent), fieldNameIdent.Name)
			}
		}
	}
	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			switch field.Type.(type) {
			case *ast.FuncType:
				for _, fieldNameIdent := range field.Names {
					f.AddResult(ir.NewFunc(fieldNameIdent), fieldNameIdent.Name)
				}
			case *ast.ChanType:
				for _, fieldNameIdent := range field.Names {
					f.AddResult(ir.NewChan(fieldNameIdent), fieldNameIdent.Name)
				}
			}
		}
	}

	err := processStmt(funcBody, f.Body(), f)
	if err != nil {
		return err
	}
	return nil
}

func processStmt(stmt ast.Stmt, body *ir.Body, f *ir.Func) error {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		// TODO: Handle declaration
		return processExprs(append(s.Rhs, s.Lhs...), body)

	case *ast.BlockStmt:
		for _, stmt := range s.List {
			err := processStmt(stmt, body, f)
			if err != nil {
				return err
			}
		}
		return nil

	case *ast.DeclStmt:
		processDecl(s.Decl, body.Scope())
		return nil

	case *ast.DeferStmt:
		if f.Body() != body {
			return fmt.Errorf("encountered nested defer statement")
		}

		return processCallExpr(s.Call, f.AddDeferred())

	case *ast.ExprStmt:
		return processExpr(s.X, body)

	case *ast.ForStmt:
		if s.Init != nil {
			err := processStmt(s.Init, body, f)
			if err != nil {
				return err
			}
		}

		forStmt := ir.NewForStmt(s, body.Scope())
		body.AddStmt(forStmt)

		if s.Cond != nil {
			err := processExpr(s.Cond, forStmt.Cond())
			if err != nil {
				return err
			}
		}

		err := processStmt(s.Body, forStmt.Body(), f)
		if err != nil {
			return err
		}
		if s.Post != nil {
			err := processStmt(s.Post, forStmt.Body(), f)
			if err != nil {
				return err
			}
		}
		return nil

	case *ast.GoStmt:
		err := processExprs(s.Call.Args, body)
		if err != nil {
			return err
		}

		callee, err := findCallee(s.Call.Fun, body.Scope(), true /*mustResolve*/)
		if err != nil {
			return err
		}
		body.AddStmt(ir.NewGoStmt(s, callee))
		return nil

	case *ast.IfStmt:
		if s.Init != nil {
			err := processStmt(s.Init, body, f)
			if err != nil {
				return err
			}
		}

		err := processExpr(s.Cond, body)
		if err != nil {
			return err
		}

		ifStmt := ir.NewIfStmt(s, body.Scope())
		body.AddStmt(ifStmt)

		err = processStmt(s.Body, ifStmt.IfBranch(), f)
		if err != nil {
			return err
		}

		if s.Else != nil {
			return processStmt(s.Else, ifStmt.ElseBranch(), f)
		}
		return nil

	case *ast.IncDecStmt:
		return processExpr(s.X, body)

	case *ast.RangeStmt:
		c, err := findChannel(s.X, body.Scope(), false /*mustResolve*/)
		if err != nil {
			return err
		}
		if c != nil {
			rangeStmt := ir.NewRangeStmt(s, c, body.Scope())
			body.AddStmt(rangeStmt)

			return processStmt(s.Body, rangeStmt.Body(), f)
		}

		err = processExpr(s.X, body)
		if err != nil {
			return err
		}

		forStmt := ir.NewForStmt(s, body.Scope())
		body.AddStmt(forStmt)

		return processStmt(s.Body, forStmt.Body(), f)

	case *ast.SendStmt:
		err := processExpr(s.Value, body)
		if err != nil {
			return err
		}

		c, err := findChannel(s.Chan, body.Scope(), true /*mustResolve*/)
		if err != nil {
			return err
		}
		body.AddStmt(ir.NewSendStmt(s, c))
		return nil

	default:
		fmt.Printf("ignoring %T statement\n", s)
		return nil
	}
}

func processExprs(exprs []ast.Expr, body *ir.Body) error {
	for _, expr := range exprs {
		err := processExpr(expr, body)
		if err != nil {
			return err
		}
	}
	return nil
}

func processExpr(expr ast.Expr, body *ir.Body) error {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return nil
	case *ast.BinaryExpr:
		return processExprs([]ast.Expr{e.X, e.Y}, body)
	case *ast.CallExpr:
		return processCallExpr(e, body)
	case *ast.CompositeLit:
		return processExprs(append([]ast.Expr{e.Type}, e.Elts...), body)
	case *ast.Ellipsis:
		return processExpr(e.Elt, body)
	case *ast.FuncLit:
		f, err := processFuncLit(e, body.Scope())
		fmt.Printf("orphaned function literal: %p\n", f)
		return err
	case *ast.Ident:
		return nil
	case *ast.IndexExpr:
		return processExprs([]ast.Expr{e.X, e.Index}, body)
	case *ast.KeyValueExpr:
		return processExprs([]ast.Expr{e.Key, e.Value}, body)
	case *ast.ParenExpr:
		return processExpr(e.X, body)
	case *ast.SelectorExpr:
		return processExpr(e.X, body)
	case *ast.SliceExpr:
		return processExprs([]ast.Expr{e.X, e.Low, e.High, e.Max}, body)
	case *ast.StarExpr:
		return processExpr(e.X, body)
	case *ast.TypeAssertExpr:
		return processExprs([]ast.Expr{e.X, e.Type}, body)
	case *ast.UnaryExpr:
		if e.Op == token.ARROW {
			c, err := findChannel(e.X, body.Scope(), true /*mustResolve*/)
			if err != nil {
				return err
			}

			body.AddStmt(ir.NewReceiveStmt(e, c))
			return nil
		}

		return processExpr(e.X, body)

	case
		*ast.ArrayType,
		*ast.ChanType,
		*ast.FuncType,
		*ast.InterfaceType,
		*ast.MapType,
		*ast.StructType:
		return nil
	default:
		fmt.Printf("ignoring %T expression\n", e)
		return nil
	}
}

func processFuncLit(funcLit *ast.FuncLit, scope *ir.Scope) (*ir.Func, error) {
	f := ir.NewFunc(funcLit)
	scope.AddFunc(f)
	err := processFunc(funcLit.Type, funcLit.Body, f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func findCallee(funcExpr ast.Expr, scope *ir.Scope, mustResolve bool) (*ir.Func, error) {
	switch funcExpr := funcExpr.(type) {
	case *ast.Ident:
		f := scope.FindNamedFunc(funcExpr.Name)
		if mustResolve && f == nil {
			return nil, fmt.Errorf("could not find named function: %q", funcExpr.Name)
		}
		return f, nil
	case *ast.FuncLit:
		f, err := processFuncLit(funcExpr, scope)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		if mustResolve {
			return nil, fmt.Errorf("can not process indirect function reference via %T", funcExpr)
		}
		return nil, nil
	}
}

func findChannel(chanExpr ast.Expr, scope *ir.Scope, mustResolve bool) (*ir.Chan, error) {
	switch chanExpr := chanExpr.(type) {
	case *ast.Ident:
		c := scope.FindNamedChan(chanExpr.Name)
		if mustResolve && c == nil {
			return nil, fmt.Errorf("could not find named channel: %q", chanExpr.Name)
		}
		return c, nil
	default:
		if mustResolve {
			return nil, fmt.Errorf("can not process indirect channel reference via %T", chanExpr)
		}
		return nil, nil
	}
}

func processCallExpr(callExpr *ast.CallExpr, body *ir.Body) error {
	err := processExprs(callExpr.Args, body)
	if err != nil {
		return err
	}

	f, err := findCallee(callExpr.Fun, body.Scope(), false /*mustResolve*/)
	if err != nil {
		return err
	} else if f != nil {
		body.AddStmt(ir.NewCallStmt(callExpr, f))
		return nil
	}

	fIdent, ok := callExpr.Fun.(*ast.Ident)
	if ok && fIdent.Name == "close" {
		c, err := findChannel(callExpr.Args[0], body.Scope(), true /*mustResolve*/)
		if err != nil {
			return err
		}
		body.AddStmt(ir.NewCloseStmt(callExpr, c))
		return nil
	}

	return processExpr(callExpr.Fun, body)
}
