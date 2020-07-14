package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateCallStmt(stmt *ir.CallStmt, ctx *context) {
	switch callee := stmt.Callee().(type) {
	case *ir.Func:
		t.translateCall(stmt, calleeInfo{
			f:          callee,
			parPid:     "pid",
			startState: ctx.currentState,
			endState:   nil,
		}, ctx)
	case ir.LValue:
		handle, _ := t.translateLValue(callee, ctx)

		nilState := ctx.proc.AddState(callee.Handle()+"_is_nil_", uppaal.Renaming)
		nilState.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		nilState.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))
		ctx.proc.AddQuery(uppaal.NewQuery(
			"A[] (not out_of_resources) imply (not $."+nilState.Name()+")",
			"check function variable not nil",
			t.program.FileSet().Position(stmt.Pos()).String(),
			uppaal.NoFunctionCallsWithNilVariable))
		nilTrans := ctx.proc.AddTransition(ctx.currentState, nilState)
		nilTrans.SetGuard(handle + ".id == -1")
		nilTrans.SetGuardLocation(
			ctx.currentState.Location().Add(uppaal.Location{4, 28}))

		endState := ctx.proc.AddState("called_"+callee.Handle()+"_", uppaal.Renaming)
		endState.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		switch stmt.CallKind() {
		case ir.Call:
			endState.SetLocationAndResetNameAndCommentLocation(
				ctx.currentState.Location().Add(uppaal.Location{0, 680}))
		case ir.Defer:
			endState.SetLocationAndResetNameAndCommentLocation(
				ctx.currentState.Location().Add(uppaal.Location{0, 544}))
		case ir.Go:
			endState.SetLocationAndResetNameAndCommentLocation(
				ctx.currentState.Location().Add(uppaal.Location{0, 544}))
		default:
			panic(fmt.Errorf("unsupported CallKind: %v", stmt.CallKind()))
		}

		calleeSig := stmt.CalleeSignature()
		for i, calleeFunc := range t.completeFCG.DynamicCallees(calleeSig) {
			startState := ctx.proc.AddState(callee.Handle()+"_is_"+calleeFunc.Handle()+"_", uppaal.Renaming)
			startState.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
			startState.SetLocationAndResetNameAndCommentLocation(
				ctx.currentState.Location().Add(uppaal.Location{(i + 1) * 136, 136}))
			startTrans := ctx.proc.AddTransition(ctx.currentState, startState)
			startTrans.SetGuard(handle + ".id == " + calleeFunc.FuncValue().String())
			startTrans.SetGuardLocation(
				ctx.currentState.Location().Add(uppaal.Location{(i+1)*136 + 4, 44 + i*16}))

			t.translateCall(stmt, calleeInfo{
				f:          calleeFunc,
				parPid:     handle + ".par_pid",
				startState: startState,
				endState:   endState,
			}, ctx)
		}

		ctx.currentState = endState
		ctx.addLocation(endState.Location())
	default:
		panic(fmt.Errorf("unexpected callee type: %T", callee))
	}
}

type calleeInfo struct {
	f          *ir.Func
	parPid     string
	startState *uppaal.State
	endState   *uppaal.State
}

func (t *translator) translateCall(stmt *ir.CallStmt, info calleeInfo, ctx *context) {
	calleeFunc := info.f
	calleeProc := t.funcToProcess[calleeFunc]

	created := ctx.proc.AddState("created_"+calleeProc.Name()+"_", uppaal.Renaming)
	created.SetType(uppaal.Committed)
	created.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	created.SetLocationAndResetNameAndCommentLocation(
		info.startState.Location().Add(uppaal.Location{0, 136}))
	create := ctx.proc.AddTransition(info.startState, created)
	if calleeFunc.EnclosingFunc() != nil {
		create.AddUpdate("p = make_"+calleeProc.Name()+"("+info.parPid+")", false)
	} else {
		create.AddUpdate("p = make_"+calleeProc.Name()+"()", false)
	}
	create.SetUpdateLocation(info.startState.Location().Add(uppaal.Location{4, 60}))

	for i, calleeArg := range calleeFunc.Args() {
		calleeArgStr := t.translateArg(calleeArg, "p")
		callerArg := stmt.Args()[i]
		callerArgStr, usesGlobals := t.translateRValue(callerArg, calleeArg.Type(), ctx)
		if stmt.ArgRequiresCopy(i) {
			callerArgStr = t.translateCopyOfRValue(callerArgStr, calleeArg.Type())
		}
		create.AddUpdate(
			fmt.Sprintf("%s = %s", calleeArgStr, callerArgStr),
			usesGlobals)
	}

	if stmt.CallKind() == ir.Call || stmt.CallKind() == ir.Go {
		started := ctx.proc.AddState("started_"+calleeProc.Name()+"_", uppaal.Renaming)
		started.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		started.SetLocationAndResetNameAndCommentLocation(
			created.Location().Add(uppaal.Location{0, 136}))
		start := ctx.proc.AddTransition(created, started)

		if stmt.CallKind() == ir.Call {
			start.SetSync(fmt.Sprintf("sync_%s[p]!", calleeProc.Name()))
			start.SetSyncLocation(
				created.Location().Add(uppaal.Location{4, 60}))

			awaited := ctx.proc.AddState("awaited_"+calleeProc.Name()+"_", uppaal.Renaming)
			awaited.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
			awaited.SetLocationAndResetNameAndCommentLocation(
				started.Location().Add(uppaal.Location{0, 136}))
			waitForRegularReturn := ctx.proc.AddTransition(started, awaited)
			waitForRegularReturn.SetGuard(fmt.Sprintf("!external_panic_%s[p]", calleeProc.Name()))
			waitForRegularReturn.SetGuardLocation(started.Location().Add(uppaal.Location{4, 48}))
			waitForRegularReturn.SetSync(fmt.Sprintf("sync_%s[p]?", calleeProc.Name()))
			waitForRegularReturn.SetSyncLocation(started.Location().Add(uppaal.Location{4, 64}))

			for i, resType := range calleeFunc.ResultTypes() {
				calleeRes := t.translateResult(calleeFunc, i, "p")
				callerRes, usesGlobals := t.translateLValue(stmt.Results()[i], ctx)
				if stmt.ResultRequiresCopy(i) {
					calleeRes = t.translateCopyOfRValue(calleeRes, resType)
				}
				waitForRegularReturn.AddUpdate(
					fmt.Sprintf("%s = %s", callerRes, calleeRes),
					usesGlobals)
			}
			waitForRegularReturn.SetUpdateLocation(started.Location().Add(uppaal.Location{4, 96}))

			waitForPanic := ctx.proc.AddTransition(started, ctx.exitFuncState)
			waitForPanic.SetGuard(fmt.Sprintf("external_panic_%s[p]", calleeProc.Name()))
			waitForPanic.SetGuardLocation(started.Location().Add(uppaal.Location{4, 32}))
			waitForPanic.SetSync(fmt.Sprintf("sync_%s[p]?", calleeProc.Name()))
			waitForPanic.SetSyncLocation(started.Location().Add(uppaal.Location{4, 64}))
			waitForPanic.AddUpdate(fmt.Sprintf("internal_panic = true"), false)
			waitForPanic.SetUpdateLocation(started.Location().Add(uppaal.Location{4, 80}))
			waitForPanic.AddNail(started.Location().Add(uppaal.Location{0, 68}))
			waitForPanic.AddNail(uppaal.Location{-68, started.Location().Y() + 68})

			if info.endState == nil {
				ctx.currentState = awaited
			} else {
				ctx.proc.AddTransition(awaited, info.endState)
			}
			ctx.returnTransitions[waitForPanic] = struct{}{}
			ctx.addLocation(created.Location())
			ctx.addLocation(started.Location())
			ctx.addLocation(awaited.Location())
		} else if stmt.CallKind() == ir.Go {
			start.SetSync(fmt.Sprintf("async_%s[p]!", calleeProc.Name()))
			start.SetSyncLocation(
				created.Location().Add(uppaal.Location{4, 60}))

			if info.endState == nil {
				ctx.currentState = started
			} else {
				ctx.proc.AddTransition(started, info.endState)
			}
			ctx.addLocation(created.Location())
			ctx.addLocation(started.Location())
		}
	} else if stmt.CallKind() == ir.Defer {
		deferred := ctx.proc.AddState("deferred_"+calleeProc.Name()+"_", uppaal.Renaming)
		deferred.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		deferred.SetLocationAndResetNameAndCommentLocation(
			created.Location().Add(uppaal.Location{0, 136}))
		xdefer := ctx.proc.AddTransition(created, deferred)
		xdefer.AddUpdate("deferred_fid[deferred_count] = "+calleeFunc.FuncValue().String(), false)
		xdefer.AddUpdate("deferred_pid[deferred_count] = p", false)
		xdefer.AddUpdate("deferred_count++", false)
		xdefer.SetUpdateLocation(created.Location().Add(uppaal.Location{4, 60}))

		if info.endState == nil {
			ctx.currentState = deferred
		} else {
			ctx.proc.AddTransition(deferred, info.endState)
		}
		ctx.addLocation(created.Location())
		ctx.addLocation(deferred.Location())
	}
}

func (t *translator) translateDeferredCalls(proc *uppaal.Process, deferred *uppaal.State, f *ir.Func) {
	for i, calleeFunc := range t.deferFCG.AllCallees(f) {
		calleeProc := t.funcToProcess[calleeFunc]

		started := proc.AddState("started_"+calleeProc.Name()+"_", uppaal.Renaming)
		started.SetLocationAndResetNameAndCommentLocation(
			deferred.Location().Add(uppaal.Location{136 * (i + 1), 136}))

		start := proc.AddTransition(deferred, started)
		start.SetGuard("deferred_count > 0 && deferred_fid[deferred_count-1] == " + calleeFunc.FuncValue().String())
		start.SetGuardLocation(
			deferred.Location().Add(uppaal.Location{136 * (i + 1), 0 + 48*(i+1)}))
		start.SetSync(fmt.Sprintf("sync_%s[deferred_pid[deferred_count-1]]!", calleeProc.Name()))
		start.SetSyncLocation(
			deferred.Location().Add(uppaal.Location{136 * (i + 1), 16 + 48*(i+1)}))
		start.AddUpdate(fmt.Sprintf("external_panic_%s[deferred_pid[deferred_count-1]] = internal_panic", calleeProc.Name()), false)
		start.SetUpdateLocation(
			deferred.Location().Add(uppaal.Location{136 * (i + 1), 32 + 48*(i+1)}))

		wait := proc.AddTransition(started, deferred)
		wait.SetSync(fmt.Sprintf("sync_%s[deferred_pid[deferred_count-1]]?", calleeProc.Name()))
		wait.SetSyncLocation(started.Location().Add(uppaal.Location{4, 32 + 32*i}))
		wait.AddUpdate(fmt.Sprintf("internal_panic = external_panic_%s[deferred_pid[deferred_count-1]]", calleeProc.Name()), false)
		wait.AddUpdate("deferred_count--", false)
		wait.SetUpdateLocation(started.Location().Add(uppaal.Location{4, 48 + 32*i}))
		wait.AddNail(started.Location().Add(uppaal.Location{0, 68}))
		wait.AddNail(deferred.Location().Add(uppaal.Location{68, 204}))
	}
}

func (t *translator) translateReturnStmt(stmt *ir.ReturnStmt, ctx *context) {
	ret := ctx.proc.AddTransition(ctx.currentState, ctx.exitFuncState)
	for i, resType := range ctx.f.ResultTypes() {
		resVal, ok := stmt.Results()[i]
		if !ok {
			continue
		}
		resStr, usesGlobals := t.translateRValue(resVal, resType, ctx)

		ret.AddUpdate(fmt.Sprintf("%s = %s",
			t.translateResult(ctx.f, i, "pid"), resStr), usesGlobals)
	}
	if stmt.IsPanic() {
		ret.AddUpdate("internal_panic = true", false)
	}
	ret.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))
	ret.AddNail(ctx.currentState.Location().Add(uppaal.Location{0, 68}))
	ret.AddNail(uppaal.Location{-68, ctx.currentState.Location().Y() + 68})

	ctx.currentState = ctx.exitFuncState
	ctx.returnTransitions[ret] = struct{}{}
}

func (t *translator) translateRecoverStmt(stmt *ir.RecoverStmt, ctx *context) {
	recovered := ctx.proc.AddState("attempted_recover_", uppaal.Renaming)
	recovered.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	recovered.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	recover := ctx.proc.AddTransition(ctx.currentState, recovered)
	recover.AddUpdate(fmt.Sprintf("external_panic_%s[pid] = false", ctx.proc.Name()), false)
	recover.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = recovered
	ctx.addLocation(recovered.Location())
}
