package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateStmt(stmt ir.Stmt, ctx *context) {
	switch stmt := stmt.(type) {
	case *ir.AssignStmt:
		t.translateAssignStmt(stmt, ctx)
	case *ir.CallStmt:
		t.translateCallStmt(stmt, ctx)
	case *ir.InlinedCallStmt:
		t.translateInlinedCallStmt(stmt, ctx)
	case *ir.ReturnStmt:
		t.translateReturnStmt(stmt, ctx)
	case *ir.IfStmt:
		t.translateIfStmt(stmt, ctx)
	case *ir.ForStmt:
		t.translateForStmt(stmt, ctx)
	case *ir.RangeStmt:
		t.translateRangeStmt(stmt, ctx)
	case *ir.BranchStmt:
		t.translateBranchStmt(stmt, ctx)
	case *ir.MakeChanStmt:
		t.translateMakeChanStmt(stmt, ctx)
	case *ir.ChanOpStmt:
		t.translateChanOpStmt(stmt, ctx)
	case *ir.SelectStmt:
		t.translateSelectStmt(stmt, ctx)
	default:
		t.addWarning(fmt.Errorf("ignoring %T statement", stmt))
	}
}

func (t *translator) translateAssignStmt(stmt *ir.AssignStmt, ctx *context) {
	s := t.translateRValue(stmt.Source(), ctx)
	if _, ok := stmt.Source().(ir.Value); ok && stmt.Destination().Type() == ir.FuncType {
		s = "make_fid(" + s + ", pid)"
	}
	d := t.translateVariable(stmt.Destination(), ctx)

	assigned := ctx.proc.AddState("assigned_"+stmt.Destination().Handle()+"_", uppaal.Renaming)
	assigned.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	assigned.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	assign := ctx.proc.AddTrans(ctx.currentState, assigned)
	assign.AddUpdate(fmt.Sprintf("%s = %s", d, s))
	assign.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = assigned
	ctx.addLocation(assigned.Location())
}

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
		switch stmt.Kind() {
		case ir.Go:
			endState.SetLocationAndResetNameAndCommentLocation(
				ctx.currentState.Location().Add(uppaal.Location{0, 544}))
		case ir.Call:
			endState.SetLocationAndResetNameAndCommentLocation(
				ctx.currentState.Location().Add(uppaal.Location{0, 680}))
		}

		calleeSig := stmt.CalleeSignature()
		for i, calleeFunc := range t.fcg.DynamicCallees(calleeSig) {
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

	createdInst := ctx.proc.AddState("created_"+calleeProc.Name()+"_", uppaal.Renaming)
	createdInst.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	createdInst.SetLocationAndResetNameAndCommentLocation(
		info.startState.Location().Add(uppaal.Location{0, 136}))
	create := ctx.proc.AddTrans(info.startState, createdInst)
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

	startedInst := ctx.proc.AddState("started_"+calleeProc.Name()+"_", uppaal.Renaming)
	startedInst.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	startedInst.SetLocationAndResetNameAndCommentLocation(
		createdInst.Location().Add(uppaal.Location{0, 136}))
	start := ctx.proc.AddTrans(createdInst, startedInst)

	switch stmt.Kind() {
	case ir.Go:
		start.SetSync(fmt.Sprintf("async_%s[p]!", calleeProc.Name()))
		start.SetSyncLocation(
			createdInst.Location().Add(uppaal.Location{4, 60}))

		if info.endState == nil {
			ctx.currentState = startedInst
		} else {
			ctx.proc.AddTrans(startedInst, info.endState)
		}
		ctx.addLocation(createdInst.Location())
		ctx.addLocation(startedInst.Location())

	case ir.Call:
		start.SetSync(fmt.Sprintf("sync_%s[p]!", calleeProc.Name()))
		start.SetSyncLocation(
			createdInst.Location().Add(uppaal.Location{4, 60}))

		awaitedInst := ctx.proc.AddState("awaited_"+calleeProc.Name()+"_", uppaal.Renaming)
		awaitedInst.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		awaitedInst.SetLocationAndResetNameAndCommentLocation(
			startedInst.Location().Add(uppaal.Location{0, 136}))
		wait := ctx.proc.AddTrans(startedInst, awaitedInst)
		wait.SetSync(fmt.Sprintf("sync_%s[p]?", calleeProc.Name()))
		wait.SetSyncLocation(startedInst.Location().Add(uppaal.Location{4, 48}))

		for i := range calleeFunc.ResultTypes() {
			calleeRes := t.translateResult(calleeFunc, i, "p")
			callerRes := t.translateVariable(stmt.Results()[i], ctx)
			wait.AddUpdate(
				fmt.Sprintf("%s = %s", callerRes, calleeRes))
		}
		wait.SetUpdateLocation(
			startedInst.Location().Add(uppaal.Location{4, 64}))

		if info.endState == nil {
			ctx.currentState = awaitedInst
		} else {
			ctx.proc.AddTrans(awaitedInst, info.endState)
		}
		ctx.addLocation(createdInst.Location())
		ctx.addLocation(startedInst.Location())
		ctx.addLocation(awaitedInst.Location())

	default:
		panic(fmt.Errorf("unsupported CallKind: %v", stmt.Kind()))
	}
}

func (t *translator) translateInlinedCallStmt(stmt *ir.InlinedCallStmt, ctx *context) {
	inlinedEnter := ctx.proc.AddState("enter_inlined_"+stmt.CalleeName()+"_", uppaal.Renaming)
	inlinedEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	inlinedEnter.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{136, 136}))

	inlinedExit := ctx.proc.AddState("exit_inlined_"+stmt.CalleeName()+"_", uppaal.Renaming)
	inlinedExit.SetComment(t.program.FileSet().Position(stmt.Pos()).String())

	inlinedSubCtx := ctx.subContextForInlinedCallBody(stmt.Body(), inlinedEnter, inlinedExit)
	t.translateBody(stmt.Body(), inlinedSubCtx)

	inlinedExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], inlinedSubCtx.maxLoc[1] + 136})

	ctx.proc.AddTrans(ctx.currentState, inlinedEnter)
	ctx.currentState = inlinedExit
	ctx.addLocation(inlinedEnter.Location())
	ctx.addLocation(inlinedExit.Location())
	ctx.addLocationsFromSubContext(inlinedSubCtx)
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

func (t *translator) translateIfStmt(stmt *ir.IfStmt, ctx *context) {
	ifBody := stmt.IfBranch()
	elseBody := stmt.ElseBranch()

	ifEnter := ctx.proc.AddState("enter_if_", uppaal.Renaming)
	ifEnter.SetComment(t.program.FileSet().Position(stmt.IfPos()).String())
	ifEnter.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))

	ifExit := ctx.proc.AddState("exit_if_", uppaal.Renaming)
	ifExit.SetComment(t.program.FileSet().Position(stmt.End()).String())

	ifSubCtx := ctx.subContextForBody(ifBody, ifEnter, ifExit)
	t.translateBody(ifBody, ifSubCtx)

	elseEnter := ctx.proc.AddState("enter_else_", uppaal.Renaming)
	elseEnter.SetComment(t.program.FileSet().Position(stmt.ElsePos()).String())
	elseEnter.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ifSubCtx.maxLoc[0] + 136, ctx.currentState.Location()[1] + 136})

	elseSubCtx := ctx.subContextForBody(elseBody, elseEnter, ifExit)
	t.translateBody(elseBody, elseSubCtx)

	var maxY int
	if ifSubCtx.maxLoc[1] > elseSubCtx.maxLoc[1] {
		maxY = ifSubCtx.maxLoc[1]
	} else {
		maxY = elseSubCtx.maxLoc[1]
	}

	ifExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], maxY + 136})

	ctx.proc.AddTrans(ctx.currentState, ifEnter)
	ctx.proc.AddTrans(ctx.currentState, elseEnter)

	ctx.currentState = ifExit
	ctx.addLocation(ifEnter.Location())
	ctx.addLocation(elseEnter.Location())
	ctx.addLocation(ifExit.Location())
	ctx.addLocationsFromSubContext(ifSubCtx)
	ctx.addLocationsFromSubContext(elseSubCtx)
}

func (t *translator) translateForStmt(stmt *ir.ForStmt, ctx *context) {
	cond := stmt.Cond()
	body := stmt.Body()

	condEnter := ctx.proc.AddState("enter_loop_cond_", uppaal.Renaming)
	condEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	condEnter.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{136, 136}))
	condExit := ctx.proc.AddState("exit_loop_cond_", uppaal.Renaming)
	condExit.SetComment(t.program.FileSet().Position(stmt.Pos()).String())

	condSubCtx := ctx.subContextForBody(cond, condEnter, condExit)
	t.translateBody(cond, condSubCtx)

	condExitY := condSubCtx.maxLoc[1] + 136
	condExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{condEnter.Location()[0], condExitY})

	bodyEnter := ctx.proc.AddState("enter_loop_body_", uppaal.Renaming)
	bodyEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	bodyEnter.SetLocationAndResetNameAndCommentLocation(
		condExit.Location().Add(uppaal.Location{0, 136}))
	bodyExit := ctx.proc.AddState("exit_loop_body_", uppaal.Renaming)
	bodyExit.SetComment(t.program.FileSet().Position(stmt.End()).String())
	loopExit := ctx.proc.AddState("exit_loop_", uppaal.Renaming)
	loopExit.SetComment(t.program.FileSet().Position(stmt.End()).String())

	bodySubCtx := ctx.subContextForLoopBody(stmt, bodyEnter, bodyExit, loopExit)
	t.translateBody(body, bodySubCtx)

	bodyExitY := bodySubCtx.maxLoc[1] + 136
	bodyExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{bodyEnter.Location()[0], bodyExitY})
	loopExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], bodyExitY})

	var counterVar string
	if stmt.HasMinIterations() || stmt.HasMaxIterations() {
		loopCount := len(ctx.continueLoopStates)
		counterVar = fmt.Sprintf("i%d", loopCount)
		ctx.proc.Declarations().AddVariable(counterVar, "int", "0")
	}

	trans1 := ctx.proc.AddTrans(ctx.currentState, condEnter)
	trans1.AddNail(ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	if stmt.HasMinIterations() || stmt.HasMaxIterations() {
		trans1.AddUpdate(counterVar + " = 0")
		trans1.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))
	}

	trans2 := ctx.proc.AddTrans(condExit, bodyEnter)
	if stmt.HasMaxIterations() {
		trans2.SetGuard(fmt.Sprintf("%s < %d", counterVar, stmt.MaxIterations()))
		trans2.SetGuardLocation(condExit.Location().Add(uppaal.Location{4, 60}))
	}

	if !stmt.IsInfinite() {
		trans3 := ctx.proc.AddTrans(condExit, loopExit)
		if stmt.HasMinIterations() {
			trans3.SetGuard(fmt.Sprintf("%s >= %d", counterVar, stmt.MinIterations()))
			trans3.SetGuardLocation(condExit.Location().Add(uppaal.Location{-132, 60}))
		}

		trans3.AddNail(condExit.Location().Add(uppaal.Location{-136, 0}))
	}

	trans4 := ctx.proc.AddTrans(bodyExit, condEnter)
	if stmt.HasMaxIterations() {
		trans4.AddUpdate(counterVar + "++")
		trans4.SetUpdateLocation(condEnter.Location().Add(uppaal.Location{-64, 60}))
	}
	trans4.AddNail(bodyExit.Location().Add(uppaal.Location{-68, 0}))
	trans4.AddNail(condEnter.Location().Add(uppaal.Location{-68, 0}))
	ctx.currentState = loopExit
	ctx.addLocation(condEnter.Location())
	ctx.addLocation(condExit.Location())
	ctx.addLocation(bodyEnter.Location())
	ctx.addLocation(bodyExit.Location())
	ctx.addLocation(loopExit.Location())
	ctx.addLocationsFromSubContext(condSubCtx)
	ctx.addLocationsFromSubContext(bodySubCtx)
}

func (t *translator) translateRangeStmt(stmt *ir.RangeStmt, ctx *context) {
	handle := t.translateVariable(stmt.Channel(), ctx)
	name := stmt.Channel().Handle()
	body := stmt.Body()

	rangeEnter := ctx.proc.AddState("range_enter_", uppaal.Renaming)
	rangeEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	rangeEnter.SetLocationAndResetNameAndCommentLocation(ctx.currentState.Location().Add(uppaal.Location{136, 136}))

	receiving := ctx.proc.AddState("range_receiving_"+name+"_", uppaal.Renaming)
	receiving.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	receiving.SetLocationAndResetNameAndCommentLocation(
		rangeEnter.Location().Add(uppaal.Location{0, 136}))
	trigger := ctx.proc.AddTrans(rangeEnter, receiving)
	trigger.SetSync("receiver_trigger[" + handle + "]!")
	trigger.AddUpdate("chan_counter[" + handle + "]--")
	trigger.AddUpdate("ok = chan_counter[" + handle + "] >= 0")
	trigger.SetSyncLocation(rangeEnter.Location().Add(uppaal.Location{4, 48}))
	trigger.SetUpdateLocation(rangeEnter.Location().Add(uppaal.Location{4, 64}))
	received := ctx.proc.AddState("range_received_"+name+"_", uppaal.Renaming)
	received.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	received.SetType(uppaal.Committed)
	received.SetLocationAndResetNameAndCommentLocation(
		receiving.Location().Add(uppaal.Location{0, 136}))
	confirm := ctx.proc.AddTrans(receiving, received)
	confirm.SetSync("receiver_confirm[" + handle + "]?")
	confirm.SetSyncLocation(
		receiving.Location().Add(uppaal.Location{4, 60}))

	ctx.proc.AddQuery(uppaal.MakeQuery(
		"A[] (not out_of_resources) imply (not (deadlock and $."+receiving.Name()+"))",
		"check deadlock with pending channel operation unreachable"))

	bodyEnter := ctx.proc.AddState("enter_loop_body_", uppaal.Renaming)
	bodyEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	bodyEnter.SetLocationAndResetNameAndCommentLocation(
		received.Location().Add(uppaal.Location{0, 136}))
	bodyExit := ctx.proc.AddState("exit_loop_body_", uppaal.Renaming)
	bodyExit.SetComment(t.program.FileSet().Position(stmt.End()).String())
	loopExit := ctx.proc.AddState("exit_loop_", uppaal.Renaming)
	loopExit.SetComment(t.program.FileSet().Position(stmt.End()).String())

	bodySubCtx := ctx.subContextForLoopBody(stmt, bodyEnter, bodyExit, loopExit)
	t.translateBody(body, bodySubCtx)

	bodyExitY := bodySubCtx.maxLoc[1] + 136
	bodyExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{bodyEnter.Location()[0], bodyExitY})
	loopExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], bodyExitY})

	trans1 := ctx.proc.AddTrans(ctx.currentState, rangeEnter)
	trans1.AddNail(ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	trans2 := ctx.proc.AddTrans(received, bodyEnter)
	trans2.SetGuard("chan_buffer[" + handle + "] >= 0 || ok")
	trans2.SetGuardLocation(
		received.Location().Add(uppaal.Location{4, 48}))
	trans3 := ctx.proc.AddTrans(received, loopExit)
	trans3.SetGuard("chan_buffer[" + handle + "] < 0 && !ok")
	trans3.AddNail(received.Location().Add(uppaal.Location{-136, 0}))
	trans3.SetGuardLocation(
		received.Location().Add(uppaal.Location{-132, 64}))
	trans4 := ctx.proc.AddTrans(bodyExit, rangeEnter)
	trans4.AddNail(bodyExit.Location().Add(uppaal.Location{-68, 0}))
	trans4.AddNail(rangeEnter.Location().Add(uppaal.Location{-68, 0}))

	ctx.currentState = loopExit
	ctx.addLocation(rangeEnter.Location())
	ctx.addLocation(receiving.Location())
	ctx.addLocation(received.Location())
	ctx.addLocation(bodyEnter.Location())
	ctx.addLocation(bodyExit.Location())
	ctx.addLocation(loopExit.Location())
	ctx.addLocationsFromSubContext(bodySubCtx)
}

func (t *translator) translateBranchStmt(stmt *ir.BranchStmt, ctx *context) {
	var next *uppaal.State
	var ok bool
	switch stmt.Kind() {
	case ir.Continue:
		next, ok = ctx.continueLoopStates[stmt.Loop()]
	case ir.Break:
		next, ok = ctx.breakLoopStates[stmt.Loop()]
	default:
		panic(fmt.Errorf("unexpected ir.BranchKind: %v", stmt.Kind()))
	}
	if !ok || next == nil {
		panic(fmt.Errorf("did not find next state for branch stmt: %v", stmt))
	}

	ctx.proc.AddTrans(ctx.currentState, next)
	ctx.currentState = next
}
