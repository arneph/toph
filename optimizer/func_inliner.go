package optimizer

import (
	"fmt"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/ir"
)

// InlineFuncCalls inlines every function call in the given program.
func InlineFuncCalls(prog *ir.Program) {
	origMainFunc := prog.GetFunc("main")
	fcg := analyzer.CalculateFuncCallGraph(prog, origMainFunc, ir.Call)

	in := inliner{
		fcg:     fcg,
		funcMap: make(map[*ir.Func]*ir.Func),
		scopeMap: map[*ir.Scope]*ir.Scope{
			prog.Scope(): prog.Scope(),
		},
		variableMap: make(map[*ir.Variable]*ir.Variable),
	}

	origFuncs := prog.Funcs()
	cloneFuncs := make([]*ir.Func, 0, len(origFuncs))

	for _, origFunc := range origFuncs {
		cloneFunc := in.cloneAndInlineCalls(origFunc)
		cloneFuncs = append(cloneFuncs, cloneFunc)

		in.funcMap[origFunc] = cloneFunc
	}

	for _, cloneFunc := range cloneFuncs {
		in.changeCallees(cloneFunc)
	}

	prog.RemoveFuncs()
	prog.AddFuncs(cloneFuncs)
}

type inliner struct {
	fcg *analyzer.FuncCallGraph

	funcMap     map[*ir.Func]*ir.Func
	scopeMap    map[*ir.Scope]*ir.Scope
	variableMap map[*ir.Variable]*ir.Variable
}

func (in *inliner) cloneAndInlineCalls(origFunc *ir.Func) *ir.Func {
	origEnclosingScope := origFunc.Scope().SuperScope()
	cloneEnclosingScope := in.scopeMap[origEnclosingScope]
	cloneFunc := ir.NewFunc(origFunc.Name(), cloneEnclosingScope)
	cloneCTX := cloneContext{
		isInline:    false,
		variableMap: make(map[*ir.Variable]*ir.Variable),
	}
	in.cloneBody(origFunc.Body(), cloneFunc.Body(), &cloneCTX)

	for i, origArg := range origFunc.Args() {
		cloneArg := in.variableMap[origArg]
		cloneFunc.DefineArg(i, cloneArg)
	}
	for captured, origCapturer := range origFunc.Captures() {
		cloneCapturer := in.variableMap[origCapturer]
		cloneFunc.DefineCapture(captured, cloneCapturer)
	}
	for i, resType := range origFunc.ResultTypes() {
		cloneFunc.AddResultType(i, resType)

		origResult, ok := origFunc.Results()[i]
		if !ok {
			continue
		}
		cloneResult := in.variableMap[origResult]
		cloneFunc.DefineResult(i, cloneResult)
	}

	for i := 0; i < 100; i++ {
		cloneFunc.Body().WalkStmts(func(stmt *ir.Stmt, scope *ir.Scope) {
			origCallStmt, ok := (*stmt).(*ir.CallStmt)
			if !ok || origCallStmt.Kind() != ir.Call {
				return
			}
			origCalleeName := origCallStmt.Callee().Name()
			inlinedCallStmt := ir.NewInlinedCallStmt(origCalleeName, scope)
			inlineCTX := cloneContext{
				isInline:        true,
				origCallStmt:    origCallStmt,
				origCallerScope: scope,
				variableMap:     make(map[*ir.Variable]*ir.Variable),
			}
			in.cloneBody(origCallStmt.Callee().Body(), inlinedCallStmt.Body(), &inlineCTX)

			*stmt = inlinedCallStmt
		})
	}

	return cloneFunc
}

type cloneContext struct {
	isInline        bool
	origCallStmt    *ir.CallStmt
	origCallerScope *ir.Scope

	variableMap map[*ir.Variable]*ir.Variable
}

func (in *inliner) cloneScope(origScope, cloneScope *ir.Scope, ctx *cloneContext) {
	if !ctx.isInline {
		in.scopeMap[origScope] = cloneScope
	}

outer:
	for _, origVar := range origScope.Variables() {
		if ctx.isInline {
			for i, arg := range ctx.origCallStmt.Callee().Args() {
				if origVar != arg {
					continue
				}

				ctx.variableMap[origVar] = ctx.origCallStmt.Args()[i]
				continue outer
			}
			for capturing, capturer := range ctx.origCallStmt.Callee().Captures() {
				if origVar != capturer {
					continue
				}

				ctx.variableMap[origVar] = ctx.origCallStmt.GetCaptured(capturing)
				continue outer
			}
		}

		cloneVar := ir.NewVariable(origVar.Name(), origVar.Type(), origVar.InitialValue())
		cloneScope.AddVariable(cloneVar)
		if !ctx.isInline {
			in.variableMap[origVar] = cloneVar
		}
		ctx.variableMap[origVar] = cloneVar
	}
}

func (in *inliner) cloneBody(origBody, cloneBody *ir.Body, ctx *cloneContext) {
	in.cloneScope(origBody.Scope(), cloneBody.Scope(), ctx)

	for _, origStmt := range origBody.Stmts() {
		switch origStmt := origStmt.(type) {
		case *ir.AssignStmt:
			cloneSource := ctx.variableMap[origStmt.Source()]
			cloneDestination := ctx.variableMap[origStmt.Destination()]
			cloneStmt := ir.NewAssignStmt(cloneSource, cloneDestination)
			cloneBody.AddStmt(cloneStmt)

		case *ir.MakeChanStmt:
			cloneChannel := ctx.variableMap[origStmt.Channel()]
			cloneStmt := ir.NewMakeChanStmt(cloneChannel, origStmt.BufferSize())
			cloneBody.AddStmt(cloneStmt)

		case *ir.ChanOpStmt:
			cloneChannel := ctx.variableMap[origStmt.Channel()]
			cloneStmt := ir.NewChanOpStmt(cloneChannel, origStmt.Op())
			cloneBody.AddStmt(cloneStmt)

		case *ir.SelectStmt:
			cloneStmt := ir.NewSelectStmt(cloneBody.Scope())
			cloneStmt.SetHasDefault(origStmt.HasDefault())

			for _, origCase := range origStmt.Cases() {
				cloneChannel := ctx.variableMap[origCase.OpStmt().Channel()]
				cloneChanOpStmt := ir.NewChanOpStmt(cloneChannel, origCase.OpStmt().Op())
				cloneCase := cloneStmt.AddCase(cloneChanOpStmt)
				cloneCase.SetReachReq(origCase.ReachReq())

				in.cloneBody(origCase.Body(), cloneCase.Body(), ctx)
			}

			in.cloneBody(origStmt.DefaultBody(), cloneStmt.DefaultBody(), ctx)
			cloneBody.AddStmt(cloneStmt)

		case *ir.CallStmt:
			cloneStmt := ir.NewCallStmt(origStmt.Callee(), origStmt.Kind())
			for i, origArg := range origStmt.Args() {
				cloneArg := ctx.variableMap[origArg]
				cloneStmt.AddArg(i, cloneArg)
			}
			for capturing, origCaptured := range origStmt.Captures() {
				cloneCaptured := ctx.variableMap[origCaptured]
				cloneStmt.AddCapture(capturing, cloneCaptured)
			}
			for i, origRes := range origStmt.Results() {
				cloneRes := ctx.variableMap[origRes]
				cloneStmt.AddResult(i, cloneRes)
			}
			cloneBody.AddStmt(cloneStmt)

		case *ir.ReturnStmt:
			if !ctx.isInline {
				cloneStmt := ir.NewReturnStmt()
				for i, origRes := range origStmt.Results() {
					cloneRes := ctx.variableMap[origRes]
					cloneStmt.AddResult(i, cloneRes)
				}
				cloneBody.AddStmt(cloneStmt)

			} else {
				for i, callerRes := range ctx.origCallStmt.Results() {
					origRes, ok := origStmt.Results()[i]
					if !ok {
						origRes, ok = ctx.origCallStmt.Callee().Results()[i]
					}
					var cloneRes *ir.Variable
					if !ok {
						tmpVar := ir.NewVariable("", callerRes.Type(), -1)
						cloneBody.Scope().AddVariable(tmpVar)
						cloneRes = tmpVar
					} else {
						cloneRes = ctx.variableMap[origRes]
					}

					assignStmt := ir.NewAssignStmt(cloneRes, callerRes)
					cloneBody.AddStmt(assignStmt)
				}

				cloneStmt := ir.NewReturnStmt()
				cloneBody.AddStmt(cloneStmt)
			}

		case *ir.IfStmt:
			cloneStmt := ir.NewIfStmt(cloneBody.Scope())

			in.cloneBody(origStmt.IfBranch(), cloneStmt.IfBranch(), ctx)
			in.cloneBody(origStmt.ElseBranch(), cloneStmt.ElseBranch(), ctx)

			cloneBody.AddStmt(cloneStmt)

		case *ir.ForStmt:
			cloneStmt := ir.NewForStmt(cloneBody.Scope())
			cloneStmt.SetIsInfinite(origStmt.IsInfinite())
			cloneStmt.SetMinIterations(origStmt.MinIterations())
			cloneStmt.SetMaxIterations(origStmt.MaxIterations())

			in.cloneBody(origStmt.Cond(), cloneStmt.Cond(), ctx)
			in.cloneBody(origStmt.Body(), cloneStmt.Body(), ctx)

			cloneBody.AddStmt(cloneStmt)

		case *ir.RangeStmt:
			cloneChannel := ctx.variableMap[origStmt.Channel()]
			cloneStmt := ir.NewRangeStmt(cloneChannel, cloneBody.Scope())

			in.cloneBody(origStmt.Body(), cloneStmt.Body(), ctx)

			cloneBody.AddStmt(cloneStmt)

		default:
			panic(fmt.Errorf("inlineBody encountered unknown Stmt: %T", origStmt))
		}
	}
}

func (in *inliner) changeCallees(f *ir.Func) {
	f.Body().WalkStmts(func(stmt *ir.Stmt, scope *ir.Scope) {
		callStmt, ok := (*stmt).(*ir.CallStmt)
		if !ok {
			return
		}
		origCallee := callStmt.Callee()
		cloneCallee := in.funcMap[origCallee]

		callStmt.SetCallee(cloneCallee)
	})
}
