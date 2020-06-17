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
	case *ir.Variable:
		handle := t.translateVariable(callee, ctx)

		nilState := ctx.proc.AddState(callee.Handle()+"_is_nil_", uppaal.Renaming)
		nilState.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		nilState.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))
		ctx.proc.AddQuery(uppaal.MakeQuery(
			"A[] (not out_of_resources) imply (not $."+nilState.Name()+")",
			"check function variable not nil"))
		nilTrans := ctx.proc.AddTrans(ctx.currentState, nilState)
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
			startState := ctx.proc.AddState(callee.Handle()+"_is_"+calleeFunc.Name()+"_", uppaal.Renaming)
			startState.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
			startState.SetLocationAndResetNameAndCommentLocation(
				ctx.currentState.Location().Add(uppaal.Location{(i + 1) * 136, 136}))
			startTrans := ctx.proc.AddTrans(ctx.currentState, startState)
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
	created.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	created.SetLocationAndResetNameAndCommentLocation(
		info.startState.Location().Add(uppaal.Location{0, 136}))
	create := ctx.proc.AddTrans(info.startState, created)
	if calleeFunc.EnclosingFunc() != nil {
		create.AddUpdate("p = make_" + calleeProc.Name() + "(" + info.parPid + ")")
	} else {
		create.AddUpdate("p = make_" + calleeProc.Name() + "()")
	}
	create.SetUpdateLocation(info.startState.Location().Add(uppaal.Location{4, 60}))

	for i, calleeArg := range calleeFunc.Args() {
		calleeArgStr := t.translateArg(calleeArg, "p")
		callerArg := stmt.Args()[i]
		callerArgStr := t.translateRValue(callerArg, ctx)
		if _, ok := callerArg.(ir.Value); ok && calleeArg.Type() == ir.FuncType {
			callerArgStr = "make_fid(" + calleeArgStr + ", pid)"
		}
		create.AddUpdate(
			fmt.Sprintf("%s = %s", calleeArgStr, callerArgStr))
	}

	if stmt.CallKind() == ir.Call || stmt.CallKind() == ir.Go {
		started := ctx.proc.AddState("started_"+calleeProc.Name()+"_", uppaal.Renaming)
		started.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		started.SetLocationAndResetNameAndCommentLocation(
			created.Location().Add(uppaal.Location{0, 136}))
		start := ctx.proc.AddTrans(created, started)

		if stmt.CallKind() == ir.Call {
			start.SetSync(fmt.Sprintf("sync_%s[p]!", calleeProc.Name()))
			start.SetSyncLocation(
				created.Location().Add(uppaal.Location{4, 60}))

			awaited := ctx.proc.AddState("awaited_"+calleeProc.Name()+"_", uppaal.Renaming)
			awaited.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
			awaited.SetLocationAndResetNameAndCommentLocation(
				started.Location().Add(uppaal.Location{0, 136}))
			wait := ctx.proc.AddTrans(started, awaited)
			wait.SetSync(fmt.Sprintf("sync_%s[p]?", calleeProc.Name()))
			wait.SetSyncLocation(started.Location().Add(uppaal.Location{4, 48}))

			for i := range calleeFunc.ResultTypes() {
				calleeRes := t.translateResult(calleeFunc, i, "p")
				callerRes := t.translateVariable(stmt.Results()[i], ctx)
				wait.AddUpdate(
					fmt.Sprintf("%s = %s", callerRes, calleeRes))
			}
			wait.SetUpdateLocation(
				started.Location().Add(uppaal.Location{4, 64}))

			if info.endState == nil {
				ctx.currentState = awaited
			} else {
				ctx.proc.AddTrans(awaited, info.endState)
			}
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
				ctx.proc.AddTrans(started, info.endState)
			}
			ctx.addLocation(created.Location())
			ctx.addLocation(started.Location())
		}
	} else if stmt.CallKind() == ir.Defer {
		deferred := ctx.proc.AddState("deferred_"+calleeProc.Name()+"_", uppaal.Renaming)
		deferred.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		deferred.SetLocationAndResetNameAndCommentLocation(
			created.Location().Add(uppaal.Location{0, 136}))
		xdefer := ctx.proc.AddTrans(created, deferred)
		xdefer.AddUpdate("deferred_action[deferred_count] = kDeferredCall")
		xdefer.AddUpdate("deferred_fid[deferred_count] = " + calleeFunc.FuncValue().String())
		xdefer.AddUpdate("deferred_pid[deferred_count] = p")
		xdefer.AddUpdate("deferred_count++")
		xdefer.SetUpdateLocation(created.Location().Add(uppaal.Location{4, 60}))

		if info.endState == nil {
			ctx.currentState = deferred
		} else {
			ctx.proc.AddTrans(deferred, info.endState)
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

		start := proc.AddTrans(deferred, started)
		start.SetGuard("deferred_count > 0 && deferred_action[deferred_count-1] == kDeferredCall && deferred_fid[deferred_count-1] == " + calleeFunc.FuncValue().String())
		start.SetGuardLocation(
			deferred.Location().Add(uppaal.Location{136 * (i + 1), 16 + 32*(i+1)}))
		start.SetSync(fmt.Sprintf("sync_%s[deferred_pid[deferred_count-1]]!", calleeProc.Name()))
		start.SetSyncLocation(
			deferred.Location().Add(uppaal.Location{136 * (i + 1), 32 + 32*(i+1)}))

		wait := proc.AddTrans(started, deferred)
		wait.SetSync(fmt.Sprintf("sync_%s[deferred_pid[deferred_count-1]]?", calleeProc.Name()))
		wait.SetSyncLocation(started.Location().Add(uppaal.Location{4, 32 + 16*i}))
		wait.AddUpdate("deferred_count--")
		wait.SetUpdateLocation(deferred.Location().Add(uppaal.Location{44, 182}))
		wait.AddNail(started.Location().Add(uppaal.Location{0, 68}))
		wait.AddNail(deferred.Location().Add(uppaal.Location{68, 204}))
	}
}

func (t *translator) translateReturnStmt(stmt *ir.ReturnStmt, ctx *context) {
	ret := ctx.proc.AddTrans(ctx.currentState, ctx.exitFuncState)
	for i, resType := range ctx.f.ResultTypes() {
		resVar, ok := stmt.Results()[i]
		if !ok {
			resVar = ctx.f.Results()[i]
		}
		resStr := t.translateRValue(resVar, ctx)
		if _, ok := resVar.(ir.Value); ok && resType == ir.FuncType {
			resStr = "make_fid(" + resStr + ", pid)"
		}

		ret.AddUpdate(fmt.Sprintf("res_%s_%d_%v[pid] = %s",
			ctx.proc.Name(), i, resType, resStr))
	}
	ret.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = ctx.exitFuncState
}
