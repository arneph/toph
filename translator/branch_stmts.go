package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateIfStmt(stmt *ir.IfStmt, ctx *context) {
	ifBody := stmt.IfBranch()
	elseBody := stmt.ElseBranch()

	ifEnter := ctx.proc.AddState("enter_if_", uppaal.Renaming)
	ifEnter.SetComment(t.program.FileSet().Position(stmt.IfPos()).String())
	ifEnter.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))

	ifExit := ctx.proc.AddState("exit_if_", uppaal.Renaming)
	ifExit.SetComment(t.program.FileSet().Position(stmt.End()).String())

	ifSubCtx := ctx.subContextForStmt(stmt, ifBody, ifEnter, nil, nil, ifExit)
	t.translateBody(ifBody, ifSubCtx)

	elseEnter := ctx.proc.AddState("enter_else_", uppaal.Renaming)
	elseEnter.SetComment(t.program.FileSet().Position(stmt.ElsePos()).String())
	elseEnter.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ifSubCtx.maxLoc[0] + 136, ctx.currentState.Location()[1] + 136})

	elseSubCtx := ctx.subContextForStmt(stmt, elseBody, elseEnter, nil, nil, ifExit)
	t.translateBody(elseBody, elseSubCtx)

	var maxY int
	if ifSubCtx.maxLoc[1] > elseSubCtx.maxLoc[1] {
		maxY = ifSubCtx.maxLoc[1]
	} else {
		maxY = elseSubCtx.maxLoc[1]
	}

	ifExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], maxY + 136})

	ctx.proc.AddTransition(ctx.currentState, ifEnter)
	ctx.proc.AddTransition(ctx.currentState, elseEnter)

	ctx.currentState = ifExit
	ctx.addLocation(ifEnter.Location())
	ctx.addLocation(elseEnter.Location())
	ctx.addLocation(ifExit.Location())
	ctx.addLocationsFromSubContext(ifSubCtx)
	ctx.addLocationsFromSubContext(elseSubCtx)
}

func (t *translator) translateSwitchStmt(stmt *ir.SwitchStmt, ctx *context) {
	exitSwitch := ctx.proc.AddState("switch_end_", uppaal.Renaming)
	exitSwitch.SetComment(t.program.FileSet().Position(stmt.End()).String())

	caseCondStarts := make([][]*uppaal.State, len(stmt.Cases()))
	caseCondEnds := make([][]*uppaal.State, len(stmt.Cases()))
	caseBodyStarts := make([]*uppaal.State, len(stmt.Cases()))
	caseBodyEnds := make([]*uppaal.State, len(stmt.Cases()))
	y := ctx.currentState.Location()[1] + 136

	for i, switchCase := range stmt.Cases() {
		caseCondStarts[i] = make([]*uppaal.State, len(switchCase.Conds()))
		caseCondEnds[i] = make([]*uppaal.State, len(switchCase.Conds()))
		for j, cond := range switchCase.Conds() {
			condStart := ctx.proc.AddState(fmt.Sprintf("switch_case_%d_cond_%d_enter_", i+1, j+1), uppaal.Renaming)
			condStart.SetComment(t.program.FileSet().Position(switchCase.CondPos(j)).String())
			condStart.SetLocationAndResetNameAndCommentLocation(uppaal.Location{ctx.currentState.Location()[0] + 136, y})
			condEnd := ctx.proc.AddState(fmt.Sprintf("switch_case_%d_cond_%d_exit_", i+1, j+1), uppaal.Renaming)
			condEnd.SetComment(t.program.FileSet().Position(switchCase.CondEnd(j)).String())

			subCtx := ctx.subContextForStmt(stmt, cond, condStart, nil, nil, condEnd)
			t.translateBody(cond, subCtx)

			y = subCtx.maxLoc[1] + 136
			condEnd.SetLocationAndResetNameAndCommentLocation(uppaal.Location{ctx.currentState.Location()[0] + 136, y})
			y += 136

			caseCondStarts[i][j] = condStart
			caseCondEnds[i][j] = condEnd

			ctx.addLocation(condStart.Location())
			ctx.addLocation(condEnd.Location())
		}

		body := switchCase.Body()
		bodyStart := ctx.proc.AddState(fmt.Sprintf("switch_case_%d_body_enter_", i+1), uppaal.Renaming)
		bodyStart.SetComment(t.program.FileSet().Position(switchCase.Pos()).String())
		bodyStart.SetLocationAndResetNameAndCommentLocation(uppaal.Location{ctx.currentState.Location()[0] + 136, y})
		bodyEnd := ctx.proc.AddState(fmt.Sprintf("switch_case_%d_body_exit_", i+1), uppaal.Renaming)
		if i < len(stmt.Cases())-1 {
			bodyEnd.SetComment(t.program.FileSet().Position(stmt.Cases()[i+1].Pos()).String())
		} else {
			bodyEnd.SetComment(t.program.FileSet().Position(stmt.End()).String())
		}

		subCtx := ctx.subContextForStmt(stmt, body, bodyStart, exitSwitch, nil, bodyEnd)
		t.translateBody(body, subCtx)

		y = subCtx.maxLoc[1] + 136
		bodyEnd.SetLocationAndResetNameAndCommentLocation(uppaal.Location{ctx.currentState.Location()[0] + 136, y})
		y += 136

		caseBodyStarts[i] = bodyStart
		caseBodyEnds[i] = bodyEnd

		ctx.addLocation(bodyStart.Location())
		ctx.addLocation(bodyEnd.Location())
	}

	exitSwitch.SetLocationAndResetNameAndCommentLocation(uppaal.Location{ctx.currentState.Location()[0], y})

	lastCondState := ctx.currentState
	defaultCaseIndex := -1
	for i, switchCase := range stmt.Cases() {
		if !switchCase.HasFallthrough() {
			trans := ctx.proc.AddTransition(caseBodyEnds[i], exitSwitch)
			trans.AddNail(caseBodyEnds[i].Location().Add(uppaal.Location{-136, 0}))
		} else {
			trans := ctx.proc.AddTransition(caseBodyEnds[i], caseBodyStarts[i+1])
			trans.AddNail(caseBodyEnds[i].Location().Add(uppaal.Location{-34, 0}))
			trans.AddNail(caseBodyStarts[i+1].Location().Add(uppaal.Location{-34, 0}))
		}
		if switchCase.IsDefault() {
			defaultCaseIndex = i
			continue
		}

		for j := range switchCase.Conds() {
			trans1 := ctx.proc.AddTransition(lastCondState, caseCondStarts[i][j])
			if i > 0 && j == 0 {
				trans1.AddNail(lastCondState.Location().Add(uppaal.Location{-102, 0}))
				trans1.AddNail(caseCondStarts[i][j].Location().Add(uppaal.Location{-102, 0}))
			}
			trans2 := ctx.proc.AddTransition(caseCondEnds[i][j], caseBodyStarts[i])
			trans2.AddNail(caseCondEnds[i][j].Location().Add(uppaal.Location{-68, 0}))
			trans2.AddNail(caseBodyStarts[i].Location().Add(uppaal.Location{-68, 0}))
			lastCondState = caseCondEnds[i][j]
		}
	}
	if defaultCaseIndex != -1 {
		trans := ctx.proc.AddTransition(lastCondState, caseBodyStarts[defaultCaseIndex])
		if lastCondState != ctx.currentState {
			x := 68
			if defaultCaseIndex < len(stmt.Cases())-1 {
				x += 17
			}
			trans.AddNail(lastCondState.Location().Add(uppaal.Location{-x, 0}))
			trans.AddNail(caseBodyStarts[defaultCaseIndex].Location().Add(uppaal.Location{-x, 0}))
		}
	} else {
		trans := ctx.proc.AddTransition(lastCondState, exitSwitch)
		if lastCondState != ctx.currentState {
			trans.AddNail(lastCondState.Location().Add(uppaal.Location{-136, 0}))
		}
	}

	ctx.currentState = exitSwitch
	ctx.addLocation(exitSwitch.Location())
}

func (t *translator) translateForStmt(stmt *ir.ForStmt, ctx *context) {
	cond := stmt.Cond()
	body := stmt.Body()

	condEnter := ctx.proc.AddState("loop_cond_enter_", uppaal.Renaming)
	condEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	condEnter.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{136, 136}))
	condExit := ctx.proc.AddState("loop_cond_exit_", uppaal.Renaming)
	condExit.SetComment(t.program.FileSet().Position(stmt.Pos()).String())

	condSubCtx := ctx.subContextForStmt(stmt, cond, condEnter, nil, nil, condExit)
	t.translateBody(cond, condSubCtx)

	condExitY := condSubCtx.maxLoc[1] + 136
	condExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{condEnter.Location()[0], condExitY})

	bodyEnter, bodyExit, loopExit := t.translateLoopBody(stmt, body, condExit.Location().Add(uppaal.Location{0, 136}), ctx)

	var counterVar string
	if stmt.HasMinIterations() || stmt.HasMaxIterations() {
		loopCount := len(ctx.continueStates)
		counterVar = fmt.Sprintf("i%d", loopCount)
		ctx.proc.Declarations().AddVariable(counterVar, "int", "0")
	}

	trans1 := ctx.proc.AddTransition(ctx.currentState, condEnter)
	trans1.AddNail(ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	if stmt.HasMinIterations() || stmt.HasMaxIterations() {
		trans1.AddUpdate(counterVar+" = 0", false)
		trans1.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))
	}

	trans2 := ctx.proc.AddTransition(condExit, bodyEnter)
	if stmt.HasMaxIterations() {
		trans2.SetGuard(fmt.Sprintf("%s < %d", counterVar, stmt.MaxIterations()), false)
		trans2.SetGuardLocation(condExit.Location().Add(uppaal.Location{4, 60}))
	}

	if !stmt.IsInfinite() {
		trans3 := ctx.proc.AddTransition(condExit, loopExit)
		if stmt.HasMinIterations() {
			trans3.SetGuard(fmt.Sprintf("%s >= %d", counterVar, stmt.MinIterations()), false)
			trans3.SetGuardLocation(condExit.Location().Add(uppaal.Location{-132, 60}))
		}

		trans3.AddNail(condExit.Location().Add(uppaal.Location{-136, 0}))
	}

	trans4 := ctx.proc.AddTransition(bodyExit, condEnter)
	if stmt.HasMaxIterations() {
		trans4.AddUpdate(counterVar+"++", false)
		trans4.SetUpdateLocation(condEnter.Location().Add(uppaal.Location{-64, 60}))
	}
	trans4.AddNail(bodyExit.Location().Add(uppaal.Location{-68, 0}))
	trans4.AddNail(condEnter.Location().Add(uppaal.Location{-68, 0}))

	ctx.currentState = loopExit
	ctx.addLocation(condEnter.Location())
	ctx.addLocation(condExit.Location())
	ctx.addLocationsFromSubContext(condSubCtx)
}

func (t *translator) translateChanRangeStmt(stmt *ir.ChanRangeStmt, ctx *context) {
	var rvs randomVariableSupplier
	handle, usesGlobals := t.translateLValue(stmt.Channel(), &rvs, ctx)
	name := stmt.Channel().Handle()
	body := stmt.Body()

	loopCount := len(ctx.continueStates)
	channelVar := fmt.Sprintf("range_chan%d", loopCount)
	ctx.proc.Declarations().AddVariable(channelVar, "int", "0")

	rangeEnter := ctx.proc.AddState("range_enter_", uppaal.Renaming)
	rangeEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	rangeEnter.SetLocationAndResetNameAndCommentLocation(ctx.currentState.Location().Add(uppaal.Location{136, 136}))

	receiving := ctx.proc.AddState("range_receiving_"+name+"_", uppaal.Renaming)
	receiving.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	receiving.SetLocationAndResetNameAndCommentLocation(
		rangeEnter.Location().Add(uppaal.Location{0, 136}))
	trigger := ctx.proc.AddTransition(rangeEnter, receiving)
	trigger.SetSync("receiver_trigger[" + channelVar + "]!")
	trigger.AddUpdate("chan_counter["+channelVar+"]--", usesGlobals)
	trigger.AddUpdate("ok = chan_counter["+channelVar+"] >= 0", usesGlobals)
	trigger.SetSyncLocation(rangeEnter.Location().Add(uppaal.Location{4, 48}))
	trigger.SetUpdateLocation(rangeEnter.Location().Add(uppaal.Location{4, 64}))
	received := ctx.proc.AddState("range_received_"+name+"_", uppaal.Renaming)
	received.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	received.SetType(uppaal.Committed)
	received.SetLocationAndResetNameAndCommentLocation(
		receiving.Location().Add(uppaal.Location{0, 136}))
	confirm := ctx.proc.AddTransition(receiving, received)
	confirm.SetSync("receiver_confirm[" + channelVar + "]?")
	confirm.SetSyncLocation(
		receiving.Location().Add(uppaal.Location{4, 60}))

	ctx.proc.AddQuery(uppaal.NewQuery(
		"A[] (not out_of_resources) imply (not (deadlock and $."+receiving.Name()+"))",
		"check deadlock with pending channel operation unreachable",
		t.program.FileSet().Position(stmt.Pos()).String(),
		uppaal.NoChannelRelatedDeadlocks))

	bodyEnter, bodyExit, loopExit := t.translateLoopBody(stmt, body, received.Location().Add(uppaal.Location{0, 136}), ctx)

	trans1 := ctx.proc.AddTransition(ctx.currentState, rangeEnter)
	rvs.addToTrans(trans1)
	trans1.AddUpdate(fmt.Sprintf("%s = %s", channelVar, handle), usesGlobals)
	trans1.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	trans1.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	trans1.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))
	trans1.AddNail(ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	trans2 := ctx.proc.AddTransition(received, bodyEnter)
	trans2.SetGuard("chan_buffer["+channelVar+"] >= 0 || ok", true)
	trans2.SetGuardLocation(
		received.Location().Add(uppaal.Location{4, 48}))
	trans3 := ctx.proc.AddTransition(received, loopExit)
	trans3.SetGuard("chan_buffer["+channelVar+"] < 0 && !ok", true)
	trans3.AddNail(received.Location().Add(uppaal.Location{-136, 0}))
	trans3.SetGuardLocation(
		received.Location().Add(uppaal.Location{-132, 64}))
	trans4 := ctx.proc.AddTransition(bodyExit, rangeEnter)
	trans4.AddNail(bodyExit.Location().Add(uppaal.Location{-68, 0}))
	trans4.AddNail(rangeEnter.Location().Add(uppaal.Location{-68, 0}))

	ctx.currentState = loopExit
	ctx.addLocation(rangeEnter.Location())
	ctx.addLocation(receiving.Location())
	ctx.addLocation(received.Location())
}

func (t *translator) translateContainerRangeStmt(stmt *ir.ContainerRangeStmt, ctx *context) {
	var containerRVS randomVariableSupplier
	var valueValRVS randomVariableSupplier
	containerType := stmt.Container().Type().(*ir.ContainerType)
	containerHandle, containerUsesGlobals := t.translateLValue(stmt.Container(), &containerRVS, ctx)
	var counterVarHandle string
	if stmt.CounterVar() != nil {
		counterVarHandle, _ = t.translateVariable(stmt.CounterVar(), ctx)
	}
	var valueValHandle string
	var valueValUsesGlobals bool
	if stmt.ValueVal() != nil {
		valueValHandle, valueValUsesGlobals = t.translateLValue(stmt.ValueVal(), &valueValRVS, ctx)
	}
	usesGlobals := containerUsesGlobals || valueValUsesGlobals
	body := stmt.Body()

	var container string
	switch containerType.Kind() {
	case ir.Array:
		container = "array"
	case ir.Slice:
		container = "slice"
	case ir.Map:
		container = "map"
	default:
		panic("unexpected container kind")
	}

	loopCount := len(ctx.continueStates)
	counterVar := fmt.Sprintf("i%d", loopCount)
	containerVar := fmt.Sprintf("range_%s%d", container, loopCount)
	ctx.proc.Declarations().AddVariable(counterVar, "int", "0")
	ctx.proc.Declarations().AddVariable(containerVar, "int", "0")

	rangeEnter := ctx.proc.AddState("range_enter_", uppaal.Renaming)
	rangeEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	rangeEnter.SetLocationAndResetNameAndCommentLocation(ctx.currentState.Location().Add(uppaal.Location{136, 136}))

	assigning := ctx.proc.AddState("range_assigning_", uppaal.Renaming)
	assigning.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	assigning.SetLocationAndResetNameAndCommentLocation(ctx.currentState.Location().Add(uppaal.Location{136, 272}))

	bodyEnter, bodyExit, loopExit := t.translateLoopBody(stmt, body, assigning.Location().Add(uppaal.Location{0, 136}), ctx)

	trans1 := ctx.proc.AddTransition(ctx.currentState, rangeEnter)
	containerRVS.addToTrans(trans1)
	trans1.AddUpdate(fmt.Sprintf("%s = 0", counterVar), false)
	trans1.AddUpdate(fmt.Sprintf("%s = %s", containerVar, containerHandle), containerUsesGlobals)
	trans1.SetSelectLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	trans1.SetGuardLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))
	trans1.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 80}))
	trans1.AddNail(ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	trans2 := ctx.proc.AddTransition(rangeEnter, loopExit)
	trans2.SetGuard(fmt.Sprintf("%s >= %s_lengths[%s]",
		counterVar,
		containerType.VariablePrefix(),
		containerVar),
		containerUsesGlobals)
	trans2.SetGuardLocation(
		rangeEnter.Location().Add(uppaal.Location{4, 48}))
	trans2.AddNail(rangeEnter.Location().Add(uppaal.Location{0, 68}))
	trans2.AddNail(rangeEnter.Location().Add(uppaal.Location{-136, 136}))
	trans3 := ctx.proc.AddTransition(rangeEnter, assigning)
	trans3.SetGuard(fmt.Sprintf("%s < %s_lengths[%s]",
		counterVar,
		containerType.VariablePrefix(),
		containerVar),
		containerUsesGlobals)
	trans3.SetGuardLocation(
		rangeEnter.Location().Add(uppaal.Location{4, 64}))
	trans4 := ctx.proc.AddTransition(assigning, bodyEnter)
	if stmt.CounterVar() != nil {
		trans4.AddUpdate(fmt.Sprintf("%s = %s", counterVarHandle, counterVar), false)
	}
	if stmt.ValueVal() != nil {
		valueValRVS.addToTrans(trans4)
		if containerType.Kind() != ir.Map {
			trans4.AddUpdate(fmt.Sprintf("%s = %s_%ss[%s][%s]",
				valueValHandle,
				containerType.VariablePrefix(),
				container,
				containerVar,
				counterVar),
				usesGlobals)
		} else {
			trans4.AddUpdate(fmt.Sprintf("%s = read_%s(%s, %s)",
				valueValHandle,
				containerType.VariablePrefix(),
				containerVar,
				counterVar),
				usesGlobals)
		}
	}
	trans4.SetSelectLocation(assigning.Location().Add(uppaal.Location{4, 48}))
	trans4.SetGuardLocation(assigning.Location().Add(uppaal.Location{4, 64}))
	trans4.SetUpdateLocation(assigning.Location().Add(uppaal.Location{4, 80}))
	trans5 := ctx.proc.AddTransition(bodyExit, rangeEnter)
	trans5.AddUpdate(counterVar+"++", false)
	trans5.SetUpdateLocation(rangeEnter.Location().Add(uppaal.Location{-64, 60}))
	trans5.AddNail(bodyExit.Location().Add(uppaal.Location{-68, 0}))
	trans5.AddNail(rangeEnter.Location().Add(uppaal.Location{-68, 0}))

	ctx.currentState = loopExit
	ctx.addLocation(rangeEnter.Location())
	ctx.addLocation(assigning.Location())
}

func (t *translator) translateLoopBody(stmt ir.Stmt, body *ir.Body, loc uppaal.Location, ctx *context) (bodyEnter, bodyExit, loopExit *uppaal.State) {
	bodyEnter = ctx.proc.AddState("loop_body_enter_", uppaal.Renaming)
	bodyEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	bodyEnter.SetLocationAndResetNameAndCommentLocation(loc)
	bodyExit = ctx.proc.AddState("loop_body_exit_", uppaal.Renaming)
	bodyExit.SetComment(t.program.FileSet().Position(stmt.End()).String())
	loopExit = ctx.proc.AddState("loop_exit_", uppaal.Renaming)
	loopExit.SetComment(t.program.FileSet().Position(stmt.End()).String())

	bodySubCtx := ctx.subContextForStmt(stmt, body, bodyEnter, loopExit, bodyExit, bodyExit)
	t.translateBody(body, bodySubCtx)

	bodyExitY := bodySubCtx.maxLoc[1] + 136
	bodyExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{bodyEnter.Location()[0], bodyExitY})
	loopExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], bodyExitY})

	ctx.addLocation(bodyEnter.Location())
	ctx.addLocation(bodyExit.Location())
	ctx.addLocation(loopExit.Location())
	ctx.addLocationsFromSubContext(bodySubCtx)

	return
}

func (t *translator) translateBranchStmt(stmt *ir.BranchStmt, ctx *context) {
	var target *uppaal.State
	var ok bool
	switch stmt.Kind() {
	case ir.Continue:
		target, ok = ctx.continueStates[stmt.TargetStmt()]
	case ir.Break:
		target, ok = ctx.breakStates[stmt.TargetStmt()]
	default:
		panic(fmt.Errorf("unexpected ir.BranchKind: %v", stmt.Kind()))
	}
	if !ok || target == nil {
		panic(fmt.Errorf("did not find target state for branch stmt: %v", stmt))
	}

	ctx.proc.AddTransition(ctx.currentState, target)
	ctx.currentState = target
}
