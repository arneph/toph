package builder

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/arneph/toph/ir"
)

func (b *builder) processStmt(stmt ast.Stmt, ctx *context) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		b.processAssignStmt(s, ctx)
	case *ast.BlockStmt:
		b.processBlockStmt(s, ctx)
	case *ast.BranchStmt:
		b.processBranchStmt(s, ctx)
	case *ast.DeclStmt:
		b.processGenDecl(s.Decl.(*ast.GenDecl), ctx.body.Scope(), ctx)
	case *ast.ExprStmt:
		b.processExpr(s.X, ctx)
	case *ast.ForStmt:
		b.processForStmt(s, "", ctx)
	case *ast.GoStmt:
		b.processGoStmt(s, ctx)
	case *ast.IfStmt:
		b.processIfStmt(s, ctx)
	case *ast.IncDecStmt:
		b.processExpr(s.X, ctx)
	case *ast.LabeledStmt:
		b.processLabeledStmt(s, ctx)
	case *ast.RangeStmt:
		b.processRangeStmt(s, "", ctx)
	case *ast.ReturnStmt:
		b.processReturnStmt(s, ctx)
	case *ast.SelectStmt:
		b.processSelectStmt(s, ctx)
	case *ast.SendStmt:
		b.processSendStmt(s, ctx)
	default:
		p := b.fset.Position(stmt.Pos())
		b.addWarning(fmt.Errorf("%v: ignoring %T statement", p, s))
	}
}

func (b *builder) getDefinedVarsInAssignStmt(stmt *ast.AssignStmt) map[int]*ir.Variable {
	definedVars := make(map[int]*ir.Variable)
	for i, expr := range stmt.Lhs {
		nameIdent, ok := expr.(*ast.Ident)
		if !ok {
			continue
		}
		obj, ok := b.info.Defs[nameIdent]
		if !ok {
			continue
		}

		varType := obj.(*types.Var)
		t, ok := typesTypeToIrType(varType.Type())
		if !ok {
			continue
		}

		v := ir.NewVariable(nameIdent.Name, t, -1)
		definedVars[i] = v
		b.varTypes[varType] = v
	}
	return definedVars
}

func (b *builder) getAssignedVarsInAssignStmt(stmt *ast.AssignStmt, definedVars map[int]*ir.Variable, ctx *context) map[int]*ir.Variable {
	lhs := make(map[int]*ir.Variable)
	for i, expr := range stmt.Lhs {
		definedVar, ok := definedVars[i]
		if ok {
			lhs[i] = definedVar
			continue
		}

		ident, ok := expr.(*ast.Ident)
		if !ok {
			continue
		}

		v := b.processIdent(ident, ctx)
		if v == nil {
			continue
		}
		lhs[i] = v
	}
	return lhs
}

func (b *builder) processAssignStmt(stmt *ast.AssignStmt, ctx *context) {
	// Create newly defined variables:
	definedVars := b.getDefinedVarsInAssignStmt(stmt)

	// Resolve assigned variables:
	lhs := b.getAssignedVarsInAssignStmt(stmt, definedVars, ctx)

	defer func() {
		// Handle Lhs expressions
		for i, expr := range stmt.Lhs {
			if v, ok := definedVars[i]; ok {
				ctx.body.Scope().AddVariable(v)
				continue
			}
			if _, ok := lhs[i]; ok {
				continue
			}

			b.processExpr(expr, ctx)
		}
	}()

	// Handle single call expression:
	callExpr, ok := stmt.Rhs[0].(*ast.CallExpr)
	if ok && len(stmt.Rhs) == 1 {
		b.processCallExprWithResultVars(callExpr, ir.Call, lhs, ctx)
		return
	}

	// Handle Rhs expressions:
	for i, expr := range stmt.Rhs {
		l := lhs[i]
		r := b.processExpr(expr, ctx)
		if l == nil && r == nil {
			continue
		} else if l == nil {
			p := b.fset.Position(stmt.Lhs[i].Pos())
			b.addWarning(fmt.Errorf("%v: could not handle lhs of assignment", p))
			continue
		} else if r == nil {
			p := b.fset.Position(stmt.Rhs[i].Pos())
			b.addWarning(
				fmt.Errorf("%v: could not handle rhs of assignment", p))
			continue
		}

		assignStmt := ir.NewAssignStmt(r, l)
		ctx.body.AddStmt(assignStmt)
	}
}

func (b *builder) processBlockStmt(stmt *ast.BlockStmt, ctx *context) {
	for _, s := range stmt.List {
		b.processStmt(s, ctx)
	}
}

func (b *builder) processIfStmt(stmt *ast.IfStmt, ctx *context) {
	if stmt.Init != nil {
		b.processStmt(stmt.Init, ctx)
	}

	b.processExpr(stmt.Cond, ctx)

	ifStmt := ir.NewIfStmt(ctx.body.Scope())
	ctx.body.AddStmt(ifStmt)

	b.processStmt(stmt.Body, ctx.subContextForBody(ifStmt, "", ifStmt.IfBranch()))

	if stmt.Else != nil {
		b.processStmt(stmt.Else, ctx.subContextForBody(ifStmt, "", ifStmt.ElseBranch()))
	}
}

func (b *builder) processForStmt(stmt *ast.ForStmt, label string, ctx *context) {
	if stmt.Init != nil {
		b.processStmt(stmt.Init, ctx)
	}

	forStmt := ir.NewForStmt(ctx.body.Scope())
	ctx.body.AddStmt(forStmt)

	if stmt.Cond != nil {
		b.processExpr(stmt.Cond, ctx.subContextForBody(forStmt, "", forStmt.Cond()))
	}
	min, max := b.findIterationBounds(stmt, ctx)
	forStmt.SetIsInfinite(stmt.Cond == nil)
	forStmt.SetMinIterations(min)
	forStmt.SetMaxIterations(max)

	b.processStmt(stmt.Body, ctx.subContextForBody(forStmt, label, forStmt.Body()))
	if stmt.Post != nil {
		b.processStmt(stmt.Post, ctx.subContextForBody(forStmt, "", forStmt.Body()))
	}
}

func (b *builder) processRangeStmt(stmt *ast.RangeStmt, label string, ctx *context) {
	rv := b.processExpr(stmt.X, ctx)
	v, ok := rv.(*ir.Variable)
	if ok && v != nil && v.Type() == ir.ChanType {
		// Range over channel:
		rangeStmt := ir.NewRangeStmt(v, ctx.body.Scope())
		ctx.body.AddStmt(rangeStmt)

		b.processStmt(stmt.Body, ctx.subContextForBody(rangeStmt, label, rangeStmt.Body()))

	} else {
		// Fallback: for statement
		b.processExpr(stmt.X, ctx)

		forStmt := ir.NewForStmt(ctx.body.Scope())
		ctx.body.AddStmt(forStmt)

		min, max := b.findIterationBounds(stmt, ctx)
		forStmt.SetMinIterations(min)
		forStmt.SetMaxIterations(max)

		b.processStmt(stmt.Body, ctx.subContextForBody(forStmt, label, forStmt.Body()))
	}
}

func (b *builder) processLabeledStmt(labeledStmt *ast.LabeledStmt, ctx *context) {
	label := labeledStmt.Label.Name
	switch stmt := labeledStmt.Stmt.(type) {
	case *ast.ForStmt:
		b.processForStmt(stmt, label, ctx)
	case *ast.RangeStmt:
		b.processRangeStmt(stmt, label, ctx)
	default:
		p := b.fset.Position(labeledStmt.Pos())
		b.addWarning(
			fmt.Errorf("%v: ignoring label: %q", p, label))

		b.processStmt(stmt, ctx)
	}
}

func (b *builder) processBranchStmt(branchStmt *ast.BranchStmt, ctx *context) {
	switch branchStmt.Tok {
	case token.BREAK, token.CONTINUE:
		var kind ir.BranchKind
		var loop ir.Loop
		if branchStmt.Tok == token.CONTINUE {
			kind = ir.Continue
		} else {
			kind = ir.Break
		}
		if branchStmt.Label == nil {
			loop = ctx.currentLoop()
		} else {
			loop = ctx.currentLabeledLoop(branchStmt.Label.Name)
		}

		branchStmt := ir.NewBranchStmt(loop, kind)
		ctx.body.AddStmt(branchStmt)

	default:
		p := b.fset.Position(branchStmt.Pos())
		b.addWarning(
			fmt.Errorf("%v: unsuported branch statement: %s", p, branchStmt.Tok))
		return
	}
}

func (b *builder) processReturnStmt(stmt *ast.ReturnStmt, ctx *context) {
	resultVars := b.processExprs(stmt.Results, ctx)
	returnStmt := ir.NewReturnStmt()
	ctx.body.AddStmt(returnStmt)

	for i, v := range resultVars {
		returnStmt.AddResult(i, v)
	}
}

func (b *builder) processSelectStmt(stmt *ast.SelectStmt, ctx *context) {
	selectStmt := ir.NewSelectStmt(ctx.body.Scope())

	for _, stmt := range stmt.Body.List {
		commClause := stmt.(*ast.CommClause)
		reachReq := b.findReachabilityRequirement(commClause, ctx)

		var body *ir.Body
		switch stmt := commClause.Comm.(type) {
		case *ast.SendStmt:
			rv := b.processExpr(stmt.Chan, ctx)
			b.processExpr(stmt.Value, ctx)

			v, ok := rv.(*ir.Variable)
			if !ok || v == nil {
				p := b.fset.Position(stmt.Chan.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve channel expr: %v", p, stmt.Chan))
				continue
			}

			selectCase := selectStmt.AddCase(ir.NewChanOpStmt(v, ir.Send))
			selectCase.SetReachReq(reachReq)
			body = selectCase.Body()

		case *ast.ExprStmt:
			expr := stmt.X.(*ast.UnaryExpr)
			rv := b.processExpr(expr.X, ctx)

			v, ok := rv.(*ir.Variable)
			if !ok || v == nil {
				p := b.fset.Position(expr.X.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve channel expr: %v", p, expr.X))
				continue
			}

			selectCase := selectStmt.AddCase(ir.NewChanOpStmt(v, ir.Receive))
			selectCase.SetReachReq(reachReq)
			body = selectCase.Body()

		case *ast.AssignStmt:
			expr := stmt.Rhs[0].(*ast.UnaryExpr)
			rv := b.processExpr(expr.X, ctx)

			v, ok := rv.(*ir.Variable)
			if !ok || v == nil {
				p := b.fset.Position(expr.X.Pos())
				b.addWarning(fmt.Errorf("%v: could not resolve channel expr: %v", p, expr.X))
				continue
			}

			selectCase := selectStmt.AddCase(ir.NewChanOpStmt(v, ir.Receive))
			selectCase.SetReachReq(reachReq)
			body = selectCase.Body()

			// Create newly defined variable:
			ident, ok := stmt.Lhs[0].(*ast.Ident)
			if ok {
				varType, ok := b.info.Defs[ident].(*types.Var)
				if ok {
					t, ok := typesTypeToIrType(varType.Type())
					if ok {
						a := ir.NewVariable(ident.Name, t, -1)
						body.Scope().AddVariable(a)
						b.varTypes[varType] = a
					}
				}

			} else {
				b.processExpr(expr, ctx.subContextForBody(selectStmt, "", body))
			}

		default:
			if stmt != nil {
				p := b.fset.Position(stmt.Pos())
				b.addWarning(fmt.Errorf("%v: unexpected %T communcation clause", p, stmt))
			}

			selectStmt.SetHasDefault(true)
			body = selectStmt.DefaultBody()
		}

		subCtx := ctx.subContextForBody(selectStmt, "", body)

		for _, stmt := range commClause.Body {
			b.processStmt(stmt, subCtx)
		}
	}

	ctx.body.AddStmt(selectStmt)
}

func (b *builder) processSendStmt(stmt *ast.SendStmt, ctx *context) {
	b.processExpr(stmt.Value, ctx)

	rv := b.processExpr(stmt.Chan, ctx)
	v, ok := rv.(*ir.Variable)
	if !ok || v == nil {
		p := b.fset.Position(stmt.Chan.Pos())
		b.addWarning(fmt.Errorf("%v: could not resolve channel expr: %v", p, stmt.Chan))
		return
	}

	sendStmt := ir.NewChanOpStmt(v, ir.Send)
	ctx.body.AddStmt(sendStmt)
}
