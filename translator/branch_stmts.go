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

	ctx.proc.AddTrans(ctx.currentState, ifEnter)
	ctx.proc.AddTrans(ctx.currentState, elseEnter)

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
			trans := ctx.proc.AddTrans(caseBodyEnds[i], exitSwitch)
			trans.AddNail(caseBodyEnds[i].Location().Add(uppaal.Location{-136, 0}))
		} else {
			trans := ctx.proc.AddTrans(caseBodyEnds[i], caseBodyStarts[i+1])
			trans.AddNail(caseBodyEnds[i].Location().Add(uppaal.Location{-34, 0}))
			trans.AddNail(caseBodyStarts[i+1].Location().Add(uppaal.Location{-34, 0}))
		}
		if switchCase.IsDefault() {
			defaultCaseIndex = i
			continue
		}

		for j := range switchCase.Conds() {
			trans1 := ctx.proc.AddTrans(lastCondState, caseCondStarts[i][j])
			if i > 0 && j == 0 {
				trans1.AddNail(lastCondState.Location().Add(uppaal.Location{-102, 0}))
				trans1.AddNail(caseCondStarts[i][j].Location().Add(uppaal.Location{-102, 0}))
			}
			trans2 := ctx.proc.AddTrans(caseCondEnds[i][j], caseBodyStarts[i])
			trans2.AddNail(caseCondEnds[i][j].Location().Add(uppaal.Location{-68, 0}))
			trans2.AddNail(caseBodyStarts[i].Location().Add(uppaal.Location{-68, 0}))
			lastCondState = caseCondEnds[i][j]
		}
	}
	if defaultCaseIndex != -1 {
		trans := ctx.proc.AddTrans(lastCondState, caseBodyStarts[defaultCaseIndex])
		if lastCondState != ctx.currentState {
			x := 68
			if defaultCaseIndex < len(stmt.Cases())-1 {
				x += 17
			}
			trans.AddNail(lastCondState.Location().Add(uppaal.Location{-x, 0}))
			trans.AddNail(caseBodyStarts[defaultCaseIndex].Location().Add(uppaal.Location{-x, 0}))
		}
	} else {
		trans := ctx.proc.AddTrans(lastCondState, exitSwitch)
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

	bodyEnter := ctx.proc.AddState("loop_body_enter_", uppaal.Renaming)
	bodyEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	bodyEnter.SetLocationAndResetNameAndCommentLocation(
		condExit.Location().Add(uppaal.Location{0, 136}))
	bodyExit := ctx.proc.AddState("loop_body_exit_", uppaal.Renaming)
	bodyExit.SetComment(t.program.FileSet().Position(stmt.End()).String())
	loopExit := ctx.proc.AddState("loop_exit_", uppaal.Renaming)
	loopExit.SetComment(t.program.FileSet().Position(stmt.End()).String())

	bodySubCtx := ctx.subContextForStmt(stmt, body, bodyEnter, loopExit, bodyExit, bodyExit)
	t.translateBody(body, bodySubCtx)

	bodyExitY := bodySubCtx.maxLoc[1] + 136
	bodyExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{bodyEnter.Location()[0], bodyExitY})
	loopExit.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], bodyExitY})

	var counterVar string
	if stmt.HasMinIterations() || stmt.HasMaxIterations() {
		loopCount := len(ctx.continueStates)
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
	handle := t.translateLValue(stmt.Channel(), ctx)
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
		"check deadlock with pending channel operation unreachable",
		t.program.FileSet().Position(stmt.Pos()).String(),
		uppaal.NoChannelRelatedDeadlocks))

	bodyEnter := ctx.proc.AddState("loop_body_enter_", uppaal.Renaming)
	bodyEnter.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	bodyEnter.SetLocationAndResetNameAndCommentLocation(
		received.Location().Add(uppaal.Location{0, 136}))
	bodyExit := ctx.proc.AddState("loop_body_exit_", uppaal.Renaming)
	bodyExit.SetComment(t.program.FileSet().Position(stmt.End()).String())
	loopExit := ctx.proc.AddState("loop_exit_", uppaal.Renaming)
	loopExit.SetComment(t.program.FileSet().Position(stmt.End()).String())

	bodySubCtx := ctx.subContextForStmt(stmt, body, bodyEnter, loopExit, bodyExit, bodyExit)
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

	ctx.proc.AddTrans(ctx.currentState, target)
	ctx.currentState = target
}
