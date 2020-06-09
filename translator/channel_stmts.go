package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t *translator) translateMakeChanStmt(stmt *ir.MakeChanStmt, ctx *context) {
	handle := t.translateVariable(stmt.Channel(), ctx)
	name := stmt.Channel().Name()
	b := stmt.BufferSize()

	made := ctx.proc.AddState("made_"+name+"_", uppaal.Renaming)
	made.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	made.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))
	make := ctx.proc.AddTrans(ctx.currentState, made)
	make.AddUpdate(fmt.Sprintf("%s = make_chan(%d)", handle, b))
	make.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))
	ctx.currentState = made
	ctx.addLocation(made.Location())
}

func (t *translator) translateChanOpStmt(stmt *ir.ChanOpStmt, ctx *context) {
	handle := t.translateVariable(stmt.Channel(), ctx)
	name := stmt.Channel().Name()
	var pendingName, confirmedName, triggerChan, confirmChan, counterOp string

	switch stmt.Op() {
	case ir.Send:
		pendingName = "sending"
		confirmedName = "sent"
		triggerChan = "sender_trigger"
		confirmChan = "sender_confirm"
		counterOp = "++"
	case ir.Receive:
		pendingName = "receiving"
		confirmedName = "received"
		triggerChan = "receiver_trigger"
		confirmChan = "receiver_confirm"
		counterOp = "--"
	default:
		t.addWarning(fmt.Errorf("unsupported ChanOp: %v", stmt.Op()))
	}

	pending := ctx.proc.AddState(pendingName+"_"+name+"_", uppaal.Renaming)
	pending.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	pending.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))

	trigger := ctx.proc.AddTrans(ctx.currentState, pending)
	trigger.SetSync(triggerChan + "[" + handle + "]!")
	trigger.AddUpdate("chan_counter[" + handle + "]" + counterOp)
	trigger.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 48}))
	trigger.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 64}))

	confirmed := ctx.proc.AddState(confirmedName+"_"+name+"_", uppaal.Renaming)
	confirmed.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	confirmed.SetLocationAndResetNameAndCommentLocation(
		pending.Location().Add(uppaal.Location{0, 136}))

	confirm := ctx.proc.AddTrans(pending, confirmed)
	confirm.SetSync(confirmChan + "[" + handle + "]?")
	confirm.SetSyncLocation(
		pending.Location().Add(uppaal.Location{4, 60}))

	ctx.proc.AddQuery(uppaal.MakeQuery(
		"A[] (not out_of_resources) imply (not (deadlock and $."+pending.Name()+"))",
		"check deadlock with pending channel operation unreachable"))

	ctx.currentState = confirmed
	ctx.addLocation(pending.Location())
	ctx.addLocation(confirmed.Location())
}

func (t *translator) translateCloseChanStmt(stmt *ir.CloseChanStmt, ctx *context) {
	handle := t.translateVariable(stmt.Channel(), ctx)
	name := stmt.Channel().Name()

	if stmt.CallKind() == ir.Call {
		closed := ctx.proc.AddState("closed_"+name+"_", uppaal.Renaming)
		closed.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		closed.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))
		close := ctx.proc.AddTrans(ctx.currentState, closed)
		close.SetSync("close[" + handle + "]!")
		close.SetSyncLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))
		ctx.currentState = closed
		ctx.addLocation(closed.Location())
	} else if stmt.CallKind() == ir.Defer {
		deferred := ctx.proc.AddState("deferred_close_"+name+"_", uppaal.Renaming)
		deferred.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		deferred.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 136}))
		xdefer := ctx.proc.AddTrans(ctx.currentState, deferred)
		xdefer.AddUpdate("deferred_is_close[deferred_count] = true")
		xdefer.AddUpdate("deferred_cid[deferred_count] = " + handle)
		xdefer.AddUpdate("deferred_count++")
		xdefer.SetUpdateLocation(ctx.currentState.Location().Add(uppaal.Location{4, 60}))
		ctx.currentState = deferred
		ctx.addLocation(deferred.Location())
	}
}

type selectCaseInfo struct {
	counterForwardUpdate string
	counterReverseUpdate string
	possibleGuard        string
	triggerChanSync      string
	confirmChanSync      string
}

func (t *translator) infoForSelectCase(selectCase *ir.SelectCase, ctx *context) selectCaseInfo {
	var info selectCaseInfo
	handle := t.translateVariable(selectCase.OpStmt().Channel(), ctx)

	var rangeGuard string
	switch selectCase.OpStmt().Op() {
	case ir.Send:
		info.counterForwardUpdate = "chan_counter[" + handle + "]++"
		info.counterReverseUpdate = "chan_counter[" + handle + "]--"
		rangeGuard = "chan_counter[" + handle + "] <= chan_buffer[" + handle + "]"
		info.triggerChanSync = "sender_trigger[" + handle + "]!"
		info.confirmChanSync = "sender_confirm[" + handle + "]?"
	case ir.Receive:
		info.counterForwardUpdate = "chan_counter[" + handle + "]--"
		info.counterReverseUpdate = "chan_counter[" + handle + "]++"
		rangeGuard = "chan_counter[" + handle + "] >= 0"
		info.triggerChanSync = "receiver_trigger[" + handle + "]!"
		info.confirmChanSync = "receiver_confirm[" + handle + "]?"
	default:
		panic("unexpected select case channel op")
	}

	closedGuard := "chan_buffer[" + handle + "] < 0"
	info.possibleGuard = closedGuard + " || " + rangeGuard

	return info
}

func (t *translator) translateSelectStmt(stmt *ir.SelectStmt, ctx *context) {
	// Generate select exit state:
	exitSelect := ctx.proc.AddState("select_end_", uppaal.Renaming)
	exitSelect.SetComment(t.program.FileSet().Position(stmt.End()).String())

	// Keep track of body position information for each case and default:
	caseXs := make([]int, len(stmt.Cases()))
	maxY := ctx.currentState.Location()[1] + 408

	// Generate pass2 state or default body:
	var exitPass1Unsuccessful *uppaal.State
	if stmt.HasDefault() {
		defaultEnter := ctx.proc.AddState("select_default_enter_", uppaal.Renaming)
		defaultEnter.SetComment(t.program.FileSet().Position(stmt.DefaultPos()).String())
		defaultEnter.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 408}))

		bodySubCtx := ctx.subContextForStmt(stmt, stmt.DefaultBody(), defaultEnter, exitSelect, nil, exitSelect)
		t.translateBody(stmt.DefaultBody(), bodySubCtx)

		if len(stmt.Cases()) > 0 {
			caseXs[0] = bodySubCtx.maxLoc[0] + 136
		}
		if maxY < bodySubCtx.maxLoc[1] {
			maxY = bodySubCtx.maxLoc[1]
		}

		ctx.addLocation(defaultEnter.Location())
		ctx.addLocationsFromSubContext(bodySubCtx)

		exitPass1Unsuccessful = defaultEnter

	} else {
		pass2 := ctx.proc.AddState("select_pass_2_", uppaal.Renaming)
		pass2.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
		pass2.SetLocationAndResetNameAndCommentLocation(
			ctx.currentState.Location().Add(uppaal.Location{0, 272}))

		ctx.proc.AddQuery(uppaal.MakeQuery(
			"A[] (not out_of_resources) imply (not (deadlock and $."+pass2.Name()+"))",
			"check deadlock with blocked select statement unreachable"))

		if len(stmt.Cases()) > 0 {
			caseXs[0] = ctx.currentState.Location()[0] + 136
		}

		ctx.addLocation(pass2.Location())

		exitPass1Unsuccessful = pass2
	}

	// Generate case bodies, keeping track of body positions:
	caseEnters := make([]*uppaal.State, len(stmt.Cases()))
	for i, c := range stmt.Cases() {
		caseEnter := ctx.proc.AddState(fmt.Sprintf("select_case_%d_enter_", i+1), uppaal.Renaming)
		caseEnter.SetComment(t.program.FileSet().Position(c.Pos()).String())
		caseEnter.SetLocationAndResetNameAndCommentLocation(uppaal.Location{caseXs[i], ctx.currentState.Location()[1] + 408})
		caseEnters[i] = caseEnter

		// Add queries for reachability:
		if c.ReachReq() == ir.Reachable {
			ctx.proc.AddQuery(uppaal.MakeQuery(
				"E<> (not out_of_resources) and $."+caseEnter.Name(),
				"check reachable: "+ctx.proc.Name()+"."+caseEnter.Name()))
		} else if c.ReachReq() == ir.Unreachable {
			ctx.proc.AddQuery(uppaal.MakeQuery(
				"A[] (not out_of_resources) imply (not $."+caseEnter.Name()+")",
				"check unreachable: "+ctx.proc.Name()+"."+caseEnter.Name()))
		}

		bodySubCtx := ctx.subContextForStmt(stmt, c.Body(), caseEnter, exitSelect, nil, exitSelect)
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

	// Position select exit state after all bodies are in place:
	exitSelect.SetLocationAndResetNameAndCommentLocation(
		uppaal.Location{ctx.currentState.Location()[0], maxY + 136})

	// Prepare channel op information for each case:
	caseInfos := make([]selectCaseInfo, len(stmt.Cases()))
	nonePossibleGuard := ""
	for i, c := range stmt.Cases() {
		caseInfos[i] = t.infoForSelectCase(c, ctx)

		impossibleGuard := "!(" + caseInfos[i].possibleGuard + ")"
		if nonePossibleGuard == "" {
			nonePossibleGuard = impossibleGuard
		} else {
			nonePossibleGuard += " && " + impossibleGuard
		}
	}

	// Generate pass1 and entry transition:
	pass1 := ctx.proc.AddState("select_pass_1_", uppaal.Renaming)
	pass1.SetComment(t.program.FileSet().Position(stmt.Pos()).String())
	pass1.SetType(uppaal.Committed)
	pass1.SetLocationAndResetNameAndCommentLocation(
		ctx.currentState.Location().Add(uppaal.Location{0, 136}))

	enteringPass1 := ctx.proc.AddTrans(ctx.currentState, pass1)
	// Update all counters when entering pass1:
	for i := range stmt.Cases() {
		enteringPass1.AddUpdate(caseInfos[i].counterForwardUpdate)
	}
	enteringPass1.SetUpdateLocation(
		ctx.currentState.Location().Add(uppaal.Location{4, 60}))

	// Poll channels (pass 1):
	for i, c := range stmt.Cases() {
		triggeredCase := ctx.proc.AddState(fmt.Sprintf("select_case_%d_trigger_", i+1), uppaal.Renaming)
		triggeredCase.SetComment(t.program.FileSet().Position(c.Pos()).String())
		triggeredCase.SetLocationAndResetNameAndCommentLocation(
			uppaal.Location{caseXs[i], ctx.currentState.Location()[1] + 272})

		triggeringCase := ctx.proc.AddTrans(pass1, triggeredCase)
		triggeringCase.SetGuard(caseInfos[i].possibleGuard)
		triggeringCase.SetGuardLocation(
			triggeredCase.Location().Sub(uppaal.Location{
				32 * (len(stmt.Cases()) - i), 32 * (len(stmt.Cases()) - i)}))
		triggeringCase.SetSync(caseInfos[i].triggerChanSync)
		triggeringCase.SetSyncLocation(
			triggeredCase.Location().Sub(uppaal.Location{
				32 * (len(stmt.Cases()) - i), 32*(len(stmt.Cases())-i) - 16}))

		enteringCase := ctx.proc.AddTrans(triggeredCase, caseEnters[i])
		enteringCase.SetSync(caseInfos[i].confirmChanSync)
		enteringCase.SetSyncLocation(
			caseEnters[i].Location().Sub(uppaal.Location{
				-4, 32 * (len(stmt.Cases()) - i)}))
		// Revert all other counters when entering case:
		for j := range stmt.Cases() {
			if i != j {
				enteringCase.AddUpdate(caseInfos[j].counterReverseUpdate)
			}
		}
		enteringCase.SetUpdateLocation(
			caseEnters[i].Location().Sub(uppaal.Location{
				-4, 32*(len(stmt.Cases())-i) - 16}))
	}

	exitingPass1Unsuccessful := ctx.proc.AddTrans(pass1, exitPass1Unsuccessful)
	exitingPass1Unsuccessful.SetGuard(nonePossibleGuard)
	exitingPass1Unsuccessful.SetGuardLocation(
		exitPass1Unsuccessful.Location().Sub(uppaal.Location{-4, len(stmt.Cases())*32 + 32}))
	if stmt.HasDefault() {
		defaultEnter := exitPass1Unsuccessful
		// Revert all counters when entering default case:
		for i := range stmt.Cases() {
			exitingPass1Unsuccessful.AddUpdate(caseInfos[i].counterReverseUpdate)
		}
		exitingPass1Unsuccessful.SetUpdateLocation(
			defaultEnter.Location().Sub(uppaal.Location{-4, len(stmt.Cases())*32 + 16}))

	} else {
		// Wait for channel (pass 2):
		pass2 := exitPass1Unsuccessful
		for i := range stmt.Cases() {
			enteringCase := ctx.proc.AddTrans(pass2, caseEnters[i])
			enteringCase.SetSync(caseInfos[i].confirmChanSync)
			enteringCase.SetSyncLocation(
				caseEnters[i].Location().Sub(uppaal.Location{
					-4, 32 * (len(stmt.Cases()) - i)}))
			// Revert all other counters when entering case:
			for j := range stmt.Cases() {
				if i != j {
					enteringCase.AddUpdate(caseInfos[j].counterReverseUpdate)
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