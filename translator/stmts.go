package translator

import (
	"fmt"
	"strings"

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
	s := stmt.Source().Handle()
	d := stmt.Destination().Handle()

	assigned := ctx.proc.AddState("assigned_"+d+"_", uppaal.Renaming)
	assigned.SetLocationAndResetNameLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	assign := ctx.proc.AddTrans(ctx.currentState, assigned)
	assign.AddUpdate(fmt.Sprintf("%s = %s", d, s))
	assign.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = assigned
	ctx.addLocation(assigned.Location())
}

func (t *translator) translateCallStmt(stmt *ir.CallStmt, ctx *context) {
	calleeFunc := stmt.Callee()
	calleeProc := t.funcToProcess[calleeFunc]

	createdInst := ctx.proc.AddState("created_"+calleeProc.Name()+"_", uppaal.Renaming)
	createdInst.SetLocationAndResetNameLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	create := ctx.proc.AddTrans(ctx.currentState, createdInst)
	create.AddUpdate("p = make_" + calleeProc.Name() + "()")
	create.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	for i, calleeArg := range calleeFunc.Args() {
		callerArg := stmt.Args()[i]
		create.AddUpdate(
			fmt.Sprintf("arg_%s[p] = %s",
				calleeArg.Handle(), callerArg.Handle()))
	}
	for capturing, calleeCap := range calleeFunc.Captures() {
		callerArg := stmt.GetCaptured(capturing)
		create.AddUpdate(
			fmt.Sprintf("cap_%s[p] = %s",
				calleeCap.Handle(), callerArg.Handle()))
		t.addWarning(fmt.Errorf("treating captured as regular arg"))
	}

	startedInst := ctx.proc.AddState("started_"+calleeProc.Name()+"_", uppaal.Renaming)
	startedInst.SetLocationAndResetNameLocation(
		createdInst.Location().Add(uppaal.Location{0, 136}))
	start := ctx.proc.AddTrans(createdInst, startedInst)

	switch stmt.Kind() {
	case ir.Go:
		start.SetSync(fmt.Sprintf("async_%s[p]!", calleeProc.Name()))
		start.SetSyncLocation(
			createdInst.Location().Add(uppaal.Location{4, 60}))

		ctx.currentState = startedInst
		ctx.addLocation(createdInst.Location())
		ctx.addLocation(startedInst.Location())

	case ir.Call:
		start.SetSync(fmt.Sprintf("sync_%s[p]!", calleeProc.Name()))
		start.SetSyncLocation(
			createdInst.Location().Add(uppaal.Location{4, 60}))

		awaitedInst := ctx.proc.AddState("awaited_"+calleeProc.Name()+"_", uppaal.Renaming)
		awaitedInst.SetLocationAndResetNameLocation(
			startedInst.Location().Add(uppaal.Location{0, 136}))
		wait := ctx.proc.AddTrans(startedInst, awaitedInst)
		wait.SetSync(fmt.Sprintf("sync_%s[p]?", calleeProc.Name()))
		wait.SetSyncLocation(startedInst.Location().Add(uppaal.Location{4, 48}))

		for i, calleeRes := range calleeFunc.ResultTypes() {
			callerRes := stmt.Results()[i]
			wait.AddUpdate(
				fmt.Sprintf("%s = res_%s_%d_%v[p]",
					callerRes.Handle(), calleeProc.Name(), i, calleeRes))
		}
		wait.SetUpdateLocation(
			startedInst.Location().Add(uppaal.Location{4, 64}))

		ctx.currentState = awaitedInst
		ctx.addLocation(createdInst.Location())
		ctx.addLocation(startedInst.Location())
		ctx.addLocation(awaitedInst.Location())

	default:
		panic(fmt.Errorf("unsupported CallKind: %v", stmt.Kind()))
	}
}

func (t *translator) translateInlinedCallStmt(stmt *ir.InlinedCallStmt, ctx *context) {
	inlinedEnter := ctx.proc.AddState("enter_inlined_"+stmt.CalleeName()+"_", uppaal.Renaming)
	inlinedEnter.SetLocationAndResetNameLocation(
		ctx.currentState.Location().Add(uppaal.Location{136, 136}))

	inlinedExit := ctx.proc.AddState("exit_inlined_"+stmt.CalleeName()+"_", uppaal.Renaming)

	inlinedSubCtx := ctx.subContextForInlinedCallBody(inlinedEnter, inlinedExit)
	t.translateBody(stmt.Body(), inlinedSubCtx)

	inlinedExit.SetLocationAndResetNameLocation(
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

		ret.AddUpdate(fmt.Sprintf("res_%s_%d_%v[pid] = %s",
			ctx.proc.Name(), i, resType, resVar.Handle()))
	}
	ret.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	ctx.currentState = ctx.exitFuncState
}

func (t *translator) translateIfStmt(stmt *ir.IfStmt, ctx *context) {
	ifBody := stmt.IfBranch()
	elseBody := stmt.ElseBranch()

	ifEnter := ctx.proc.AddState("enter_if_", uppaal.Renaming)
	ifEnter.SetLocationAndResetNameLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))

	ifExit := ctx.proc.AddState("exit_if_", uppaal.Renaming)

	ifSubCtx := ctx.subContextForBody(ifEnter, ifExit)
	t.translateBody(ifBody, ifSubCtx)

	elseEnter := ctx.proc.AddState("enter_else_", uppaal.Renaming)
	elseEnter.SetLocationAndResetNameLocation(
		uppaal.Location{ifSubCtx.maxLoc[0] + 136, ctx.currentState.Location()[1] + 136})

	elseSubCtx := ctx.subContextForBody(elseEnter, ifExit)
	t.translateBody(elseBody, elseSubCtx)

	var maxY int
	if ifSubCtx.maxLoc[1] > elseSubCtx.maxLoc[1] {
		maxY = ifSubCtx.maxLoc[1]
	} else {
		maxY = elseSubCtx.maxLoc[1]
	}

	ifExit.SetLocationAndResetNameLocation(
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
	condEnter.SetLocationAndResetNameLocation(
		ctx.currentState.Location().Add(uppaal.Location{136, 136}))
	condExit := ctx.proc.AddState("exit_loop_cond_", uppaal.Renaming)

	condSubCtx := ctx.subContextForBody(condEnter, condExit)
	t.translateBody(cond, condSubCtx)

	condExitY := condSubCtx.maxLoc[1] + 136
	condExit.SetLocationAndResetNameLocation(
		uppaal.Location{condEnter.Location()[0], condExitY})

	bodyEnter := ctx.proc.AddState("enter_loop_body_", uppaal.Renaming)
	bodyEnter.SetLocationAndResetNameLocation(
		condExit.Location().Add(uppaal.Location{0, 136}))
	bodyExit := ctx.proc.AddState("exit_loop_body_", uppaal.Renaming)
	loopExit := ctx.proc.AddState("exit_loop_", uppaal.Renaming)

	bodySubCtx := ctx.subContextForLoopBody(stmt, bodyEnter, bodyExit, loopExit)
	t.translateBody(body, bodySubCtx)

	bodyExitY := bodySubCtx.maxLoc[1] + 136
	bodyExit.SetLocationAndResetNameLocation(
		uppaal.Location{bodyEnter.Location()[0], bodyExitY})
	loopExit.SetLocationAndResetNameLocation(
		uppaal.Location{ctx.currentState.Location()[0], bodyExitY})

	var counterVar string
	if stmt.HasMinIterations() || stmt.HasMaxIterations() {
		loopCount := len(ctx.continueLoopStates)
		counterVar = fmt.Sprintf("i%d", loopCount)
		ctx.proc.Declarations().AddVariableDeclaration("int " + counterVar + " = 0;")
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
	h := stmt.Channel().Handle()
	body := stmt.Body()

	rangeEnter := ctx.proc.AddState("range_enter_", uppaal.Renaming)
	rangeEnter.SetLocationAndResetNameLocation(ctx.currentState.Location().Add(uppaal.Location{136, 136}))

	receiving := ctx.proc.AddState("range_receiving_"+h+"_", uppaal.Renaming)
	receiving.SetLocationAndResetNameLocation(
		rangeEnter.Location().Add(uppaal.Location{0, 136}))
	request := ctx.proc.AddTrans(rangeEnter, receiving)
	request.SetSync("receiver_trigger[" + h + "]!")
	request.AddUpdate("chan_counter[" + h + "]--")
	request.SetSyncLocation(rangeEnter.Location().Add(uppaal.Location{4, 48}))
	request.SetUpdateLocation(rangeEnter.Location().Add(uppaal.Location{4, 64}))
	received := ctx.proc.AddState("range_received_"+h+"_", uppaal.Renaming)
	received.SetType(uppaal.Committed)
	received.SetLocationAndResetNameLocation(
		receiving.Location().Add(uppaal.Location{0, 136}))
	confirm := ctx.proc.AddTrans(receiving, received)
	confirm.SetSync("receiver_confirm[" + h + "]?")
	confirm.SetSyncLocation(
		receiving.Location().Add(uppaal.Location{4, 60}))

	bodyEnter := ctx.proc.AddState("enter_loop_body_", uppaal.Renaming)
	bodyEnter.SetLocationAndResetNameLocation(
		received.Location().Add(uppaal.Location{0, 136}))
	bodyExit := ctx.proc.AddState("exit_loop_body_", uppaal.Renaming)
	loopExit := ctx.proc.AddState("exit_loop_", uppaal.Renaming)

	bodySubCtx := ctx.subContextForLoopBody(stmt, bodyEnter, bodyExit, loopExit)
	t.translateBody(body, bodySubCtx)

	bodyExitY := bodySubCtx.maxLoc[1] + 136
	bodyExit.SetLocationAndResetNameLocation(
		uppaal.Location{bodyEnter.Location()[0], bodyExitY})
	loopExit.SetLocationAndResetNameLocation(
		uppaal.Location{ctx.currentState.Location()[0], bodyExitY})

	trans1 := ctx.proc.AddTrans(ctx.currentState, rangeEnter)
	trans1.AddNail(ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	trans2 := ctx.proc.AddTrans(received, bodyEnter)
	trans2.SetGuard("chan_buffer[" + h + "] >= 0")
	trans2.SetGuardLocation(
		received.Location().Add(uppaal.Location{4, 48}))
	trans3 := ctx.proc.AddTrans(received, loopExit)
	trans3.SetGuard("chan_buffer[" + h + "] < 0")
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

func (t *translator) translateMakeChanStmt(stmt *ir.MakeChanStmt, ctx *context) {
	h := stmt.Channel().Handle()
	b := stmt.BufferSize()

	made := ctx.proc.AddState("made_"+h+"_", uppaal.Renaming)
	made.SetLocationAndResetNameLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	make := ctx.proc.AddTrans(ctx.currentState, made)
	make.AddUpdate(fmt.Sprintf("%s = make_chan(%d)", h, b))
	make.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))
	ctx.currentState = made
	ctx.addLocation(made.Location())
}

func (t *translator) translateChanOpStmt(stmt *ir.ChanOpStmt, ctx *context) {
	h := stmt.Channel().Handle()

	switch stmt.Op() {
	case ir.Send:
		sending := ctx.proc.AddState("sending_"+h+"_", uppaal.Renaming)
		sending.SetLocationAndResetNameLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))
		request := ctx.proc.AddTrans(ctx.currentState, sending)
		request.SetSync("sender_trigger[" + h + "]!")
		request.AddUpdate("chan_counter[" + h + "]++")
		request.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
		request.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
		sent := ctx.proc.AddState("sent_"+h+"_", uppaal.Renaming)
		sent.SetLocationAndResetNameLocation(
			sending.Location().Add(uppaal.Location{0, 136}))
		confirm := ctx.proc.AddTrans(sending, sent)
		confirm.SetSync("sender_confirm[" + h + "]?")
		confirm.SetSyncLocation(
			sending.Location().Add(uppaal.Location{4, 60}))
		ctx.currentState = sent
		ctx.addLocation(sending.Location())
		ctx.addLocation(sent.Location())

	case ir.Receive:
		receiving := ctx.proc.AddState("receiving_"+h+"_", uppaal.Renaming)
		receiving.SetLocationAndResetNameLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))
		request := ctx.proc.AddTrans(ctx.currentState, receiving)
		request.SetSync("receiver_trigger[" + h + "]!")
		request.AddUpdate("chan_counter[" + h + "]--")
		request.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
		request.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
		received := ctx.proc.AddState("received_"+h+"_", uppaal.Renaming)
		received.SetLocationAndResetNameLocation(
			receiving.Location().Add(uppaal.Location{0, 136}))
		confirm := ctx.proc.AddTrans(receiving, received)
		confirm.SetSync("receiver_confirm[" + h + "]?")
		confirm.SetSyncLocation(
			receiving.Location().Add(uppaal.Location{4, 60}))
		ctx.currentState = received
		ctx.addLocation(receiving.Location())
		ctx.addLocation(received.Location())

	case ir.Close:
		closing := ctx.proc.AddState("closed_"+h+"_", uppaal.Renaming)
		closing.SetLocationAndResetNameLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))
		close := ctx.proc.AddTrans(ctx.currentState, closing)
		close.SetSync("close[" + h + "]!")
		close.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))
		ctx.currentState = closing
		ctx.addLocation(closing.Location())

	default:
		t.addWarning(fmt.Errorf("unsupported ChanOp: %v", stmt.Op()))
	}
}

func (t *translator) translateSelectStmt(stmt *ir.SelectStmt, ctx *context) {
	handles := make([]string, len(stmt.Cases()))
	for i, c := range stmt.Cases() {
		handles[i] = c.OpStmt().Channel().Handle()
	}

	exitSelect := ctx.proc.AddState("select_end_", uppaal.Renaming)

	// Generate case bodies
	caseXs := make([]int, len(stmt.Cases()))
	maxY := ctx.currentState.Location()[1] + 408

	var exitPass1 *uppaal.State
	if stmt.HasDefault() {
		defaultEnter := ctx.proc.AddState("select_default_enter_", uppaal.Renaming)
		defaultEnter.SetLocationAndResetNameLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 408}))

		bodySubCtx := ctx.subContextForBody(defaultEnter, exitSelect)
		t.translateBody(stmt.DefaultBody(), bodySubCtx)

		if len(stmt.Cases()) > 0 {
			caseXs[0] = bodySubCtx.maxLoc[0] + 136
		}
		if maxY < bodySubCtx.maxLoc[1] {
			maxY = bodySubCtx.maxLoc[1]
		}

		ctx.addLocation(defaultEnter.Location())
		ctx.addLocationsFromSubContext(bodySubCtx)

		exitPass1 = defaultEnter

	} else {
		if len(stmt.Cases()) > 0 {
			caseXs[0] = ctx.currentState.Location()[0] + 136
		}

		exitPass1 = ctx.proc.AddState("select_pass_2_", uppaal.Renaming)
		exitPass1.SetLocationAndResetNameLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 272}))

		ctx.addLocation(exitPass1.Location())
	}

	caseEnters := make([]*uppaal.State, len(stmt.Cases()))
	for i, c := range stmt.Cases() {
		caseEnter := ctx.proc.AddState(fmt.Sprintf("select_case_%d_enter_", i+1), uppaal.Renaming)
		caseEnter.SetLocationAndResetNameLocation(uppaal.Location{caseXs[i], ctx.currentState.Location()[1] + 408})
		caseEnters[i] = caseEnter

		if c.ReachReq() == ir.Reachable {
			ctx.proc.AddQuery(uppaal.MakeQuery(
				"E<> $."+caseEnter.Name(),
				"check reachable: "+ctx.proc.Name()+"."+caseEnter.Name()))
		} else if c.ReachReq() == ir.Unreachable {
			ctx.proc.AddQuery(uppaal.MakeQuery(
				"A[] not $."+caseEnter.Name(),
				"check unreachable: "+ctx.proc.Name()+"."+caseEnter.Name()))
		}

		bodySubCtx := ctx.subContextForBody(caseEnter, exitSelect)
		t.translateBody(c.Body(), bodySubCtx)

		if i < len(stmt.Cases())-1 {
			caseXs[i+1] = bodySubCtx.maxLoc[0] + 136
		}
		if maxY < bodySubCtx.maxLoc[1] {
			maxY = bodySubCtx.maxLoc[1]
		}

		ctx.addLocation(caseEnter.Location())
		ctx.addLocationsFromSubContext(bodySubCtx)
	}

	exitSelect.SetLocationAndResetNameLocation(
		uppaal.Location{ctx.currentState.Location()[0], maxY + 136})

	// Update counters:
	pass1 := ctx.proc.AddState("select_pass_1_", uppaal.Renaming)
	pass1.SetType(uppaal.Committed)
	pass1.SetLocationAndResetNameLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	enteringPass1 := ctx.proc.AddTrans(ctx.currentState, pass1)
	for i, c := range stmt.Cases() {
		switch c.OpStmt().Op() {
		case ir.Send:
			enteringPass1.AddUpdate("chan_counter[" + handles[i] + "]++")
		case ir.Receive:
			enteringPass1.AddUpdate("chan_counter[" + handles[i] + "]--")
		default:
			panic("unexpected select case channel op")
		}
	}
	enteringPass1.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	// Poll channels (pass 1):
	inverseGuards := make([]string, len(stmt.Cases()))
	for i, c := range stmt.Cases() {
		closedGuard := "chan_buffer[" + handles[i] + "] < 0"
		var opPossibleGuard string
		switch c.OpStmt().Op() {
		case ir.Send:
			opPossibleGuard = "chan_counter[" + handles[i] + "] <= chan_buffer[" + handles[i] + "]"
		case ir.Receive:
			opPossibleGuard = "chan_counter[" + handles[i] + "] >= 0"
		default:
			panic("unexpected select case channel op")
		}
		guard := closedGuard + " || " + opPossibleGuard

		triggeredCase := ctx.proc.AddState(fmt.Sprintf("select_case_%d_trigger_", i+1), uppaal.Renaming)
		triggeredCase.SetLocationAndResetNameLocation(
			uppaal.Location{caseXs[i], ctx.currentState.Location()[1] + 272})
		triggeringCase := ctx.proc.AddTrans(pass1, triggeredCase)
		triggeringCase.SetGuard(guard)
		triggeringCase.SetGuardLocation(
			triggeredCase.Location().Sub(uppaal.Location{
				32 * (len(stmt.Cases()) - i), 32 * (len(stmt.Cases()) - i)}))
		enteringCase := ctx.proc.AddTrans(triggeredCase, caseEnters[i])
		switch c.OpStmt().Op() {
		case ir.Send:
			triggeringCase.SetSync("sender_trigger[" + handles[i] + "]!")
			enteringCase.SetSync("sender_confirm[" + handles[i] + "]?")
		case ir.Receive:
			triggeringCase.SetSync("receiver_trigger[" + handles[i] + "]!")
			enteringCase.SetSync("receiver_confirm[" + handles[i] + "]?")
		default:
			panic("unexpected select case channel op")
		}
		triggeringCase.SetSyncLocation(
			triggeredCase.Location().Sub(uppaal.Location{
				32 * (len(stmt.Cases()) - i), 32*(len(stmt.Cases())-i) - 16}))
		enteringCase.SetSyncLocation(
			caseEnters[i].Location().Sub(uppaal.Location{
				-4, 32 * (len(stmt.Cases()) - i)}))

		// Undo all other counters when entering case:
		for j, d := range stmt.Cases() {
			if i == j {
				continue
			}

			switch d.OpStmt().Op() {
			case ir.Send:
				enteringCase.AddUpdate("chan_counter[" + handles[j] + "]--")
			case ir.Receive:
				enteringCase.AddUpdate("chan_counter[" + handles[j] + "]++")
			default:
				panic("unexpected select case channel op")
			}
		}
		enteringCase.SetUpdateLocation(
			caseEnters[i].Location().Sub(uppaal.Location{
				-4, 32*(len(stmt.Cases())-i) - 16}))

		inverseGuards[i] = "!(" + guard + ")"
	}

	exitingPass1 := ctx.proc.AddTrans(pass1, exitPass1)
	exitingPass1.SetGuard(strings.Join(inverseGuards, " && "))
	exitingPass1.SetGuardLocation(
		exitPass1.Location().Sub(uppaal.Location{-4, len(stmt.Cases())*32 + 32}))
	if stmt.HasDefault() {
		// Undo all counters when entering default case:
		for i, c := range stmt.Cases() {
			switch c.OpStmt().Op() {
			case ir.Send:
				exitingPass1.AddUpdate("chan_counter[" + handles[i] + "]--")
			case ir.Receive:
				exitingPass1.AddUpdate("chan_counter[" + handles[i] + "]++")
			default:
				panic("unexpected select case channel op")
			}
		}
		exitingPass1.SetUpdateLocation(
			exitPass1.Location().Sub(uppaal.Location{-4, len(stmt.Cases())*32 + 16}))

	} else {
		// Wait for channel (pass 2):
		pass2 := exitPass1
		for i, c := range stmt.Cases() {
			enteringCase := ctx.proc.AddTrans(pass2, caseEnters[i])
			switch c.OpStmt().Op() {
			case ir.Send:
				enteringCase.SetSync("sender_confirm[" + handles[i] + "]?")
			case ir.Receive:
				enteringCase.SetSync("receiver_confirm[" + handles[i] + "]?")
			default:
				panic("unexpected select case channel op")
			}
			enteringCase.SetSyncLocation(
				caseEnters[i].Location().Sub(uppaal.Location{
					-4, 32 * (len(stmt.Cases()) - i)}))

			// Undo all other counters when entering case:
			for j, d := range stmt.Cases() {
				if i == j {
					continue
				}

				switch d.OpStmt().Op() {
				case ir.Send:
					enteringCase.AddUpdate("chan_counter[" + handles[j] + "]--")
				case ir.Receive:
					enteringCase.AddUpdate("chan_counter[" + handles[j] + "]++")
				default:
					panic("unexpected select case channel op")
				}
			}
			enteringCase.SetUpdateLocation(
				caseEnters[i].Location().Sub(uppaal.Location{
					-4, 32*(len(stmt.Cases())-i) - 16}))
		}
	}

	ctx.currentState = exitSelect
	ctx.addLocation(exitSelect.Location())
}
