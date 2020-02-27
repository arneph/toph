package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/xta"
)

func (t *translator) translateStmt(stmt ir.Stmt, ctx *context) {
	switch stmt := stmt.(type) {
	case *ir.AssignStmt:
		t.translateAssignStmt(stmt, ctx)
	case *ir.CallStmt:
		t.translateCallStmt(stmt, ctx)
	case *ir.ReturnStmt:
		t.translateReturnStmt(stmt, ctx)
	case *ir.ForStmt:
		t.translateForStmt(stmt, ctx)
	case *ir.RangeStmt:
		t.translateRangeStmt(stmt, ctx)
	case *ir.MakeChanStmt:
		t.translateMakeChanStmt(stmt, ctx)
	case *ir.ChanOpStmt:
		t.translateChanOpStmt(stmt, ctx)
	default:
		t.addWarning(fmt.Errorf("ignoring %T statement", stmt))
	}
}

func (t *translator) translateAssignStmt(stmt *ir.AssignStmt, ctx *context) {
	s := stmt.Source().Handle()
	d := stmt.Destination().Handle()

	assigned := ctx.proc.AddState("assigned_"+d+"_", xta.Renaming)
	assign := ctx.proc.AddTrans(ctx.currentState, assigned)

	assign.AddUpdate(fmt.Sprintf("%s = %s", d, s))

	ctx.currentState = assigned
}

func (t *translator) translateCallStmt(stmt *ir.CallStmt, ctx *context) {
	calleeFunc := stmt.Callee()
	calleeProc := t.funcToProcess[calleeFunc]

	createdInst := ctx.proc.AddState("created_"+calleeProc.Name()+"_", xta.Renaming)
	create := ctx.proc.AddTrans(ctx.currentState, createdInst)
	create.AddUpdate("p = make_" + calleeProc.Name() + "()")

	for i, calleeArg := range calleeFunc.Args() {
		callerArg := stmt.Args()[i]
		create.AddUpdate(
			fmt.Sprintf("arg_%s[p] = %s",
				calleeArg.Handle(), callerArg.Handle()))
	}
	for callerArg, calleeCap := range calleeFunc.Captures() {
		create.AddUpdate(
			fmt.Sprintf("cap_%s[p] = %s",
				calleeCap.Handle(), callerArg.Handle()))
		t.addWarning(fmt.Errorf("treating captured as regular arg"))
	}

	startedInst := ctx.proc.AddState("started_"+calleeProc.Name()+"_", xta.Renaming)
	start := ctx.proc.AddTrans(createdInst, startedInst)
	switch stmt.Kind() {
	case ir.Go:
		start.SetSync(fmt.Sprintf("async_%s[p]!", calleeProc.Name()))

		ctx.currentState = startedInst

	case ir.Call:
		start.SetSync(fmt.Sprintf("sync_%s[p]!", calleeProc.Name()))

		awaitedInst := ctx.proc.AddState("awaited_"+calleeProc.Name()+"_", xta.Renaming)
		wait := ctx.proc.AddTrans(startedInst, awaitedInst)
		wait.SetSync(fmt.Sprintf("sync_%s[p]?", calleeProc.Name()))

		for i, calleeRes := range calleeFunc.ResultTypes() {
			callerRes := stmt.Results()[i]
			wait.AddUpdate(
				fmt.Sprintf("%s = res_%s_%d_%v[p]",
					callerRes.Handle(), calleeProc.Name(), i, calleeRes))
		}

		ctx.currentState = awaitedInst

	default:
		panic(fmt.Errorf("unsupported CallKind: %v", stmt.Kind()))
	}
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

	ctx.currentState = ctx.exitFuncState
}

func (t *translator) translateForStmt(stmt *ir.ForStmt, ctx *context) {
	condEnter := ctx.proc.AddState("enter_loop_cond", xta.Renaming)
	condExit := ctx.proc.AddState("exit_loop_cond", xta.Renaming)
	bodyEnter := ctx.proc.AddState("enter_loop_body", xta.Renaming)
	bodyExit := ctx.proc.AddState("exit_loop_body", xta.Renaming)
	loopExit := ctx.proc.AddState("exit_loop", xta.Renaming)

	cond := stmt.Cond()
	body := stmt.Body()

	t.translateBody(cond, ctx.subContextForBody(condEnter, condExit))
	t.translateBody(body, ctx.subContextForLoopBody(stmt, bodyEnter, bodyExit, loopExit))

	ctx.proc.AddTrans(ctx.currentState, condEnter)
	ctx.proc.AddTrans(condExit, bodyEnter)
	ctx.proc.AddTrans(condExit, loopExit)
	ctx.proc.AddTrans(bodyExit, condEnter)
	ctx.currentState = loopExit
}

func (t *translator) translateRangeStmt(stmt *ir.RangeStmt, ctx *context) {
	h := stmt.Channel().Handle()

	condEval := ctx.proc.AddState("range_cond", xta.Renaming)
	receiving := ctx.proc.AddState("range_receiving_"+h+"_", xta.Renaming)
	bodyEnter := ctx.proc.AddState("enter_loop_body", xta.Renaming)
	bodyExit := ctx.proc.AddState("exit_loop_body", xta.Renaming)
	loopExit := ctx.proc.AddState("exit_loop", xta.Renaming)

	body := stmt.Body()

	t.translateBody(body, ctx.subContextForLoopBody(stmt, bodyEnter, bodyExit, loopExit))

	ctx.proc.AddTrans(ctx.currentState, condEval)

	request := ctx.proc.AddTrans(condEval, receiving)
	request.SetGuard("chan_buffer[" + h + "] >= 0")
	request.SetSync("receiver_alpha[" + h + "]!")

	confirm := ctx.proc.AddTrans(receiving, bodyEnter)
	confirm.SetSync("receiver_omega[" + h + "]?")

	exit := ctx.proc.AddTrans(condEval, loopExit)
	exit.SetGuard("chan_buffer[" + h + "] < 0")

	ctx.proc.AddTrans(bodyExit, condEval)
	ctx.currentState = loopExit
}

func (t *translator) translateMakeChanStmt(stmt *ir.MakeChanStmt, ctx *context) {
	h := stmt.Channel().Handle()
	b := stmt.BufferSize()

	made := ctx.proc.AddState("made_"+h+"_", xta.Renaming)
	make := ctx.proc.AddTrans(ctx.currentState, made)
	make.AddUpdate(fmt.Sprintf("%s = make_chan(%d)", h, b))
	ctx.currentState = made
}

func (t *translator) translateChanOpStmt(stmt *ir.ChanOpStmt, ctx *context) {
	h := stmt.Channel().Handle()

	switch stmt.Op() {
	case ir.Send:
		sending := ctx.proc.AddState("sending_"+h+"_", xta.Renaming)
		request := ctx.proc.AddTrans(ctx.currentState, sending)
		request.SetSync("sender_alpha[" + h + "]!")
		sent := ctx.proc.AddState("sent_"+h+"_", xta.Renaming)
		confirm := ctx.proc.AddTrans(sending, sent)
		confirm.SetSync("sender_omega[" + h + "]?")
		ctx.currentState = sent

	case ir.Receive:
		receiving := ctx.proc.AddState("receiving_"+h+"_", xta.Renaming)
		request := ctx.proc.AddTrans(ctx.currentState, receiving)
		request.SetSync("receiver_alpha[" + h + "]!")
		received := ctx.proc.AddState("received_"+h+"_", xta.Renaming)
		confirm := ctx.proc.AddTrans(receiving, received)
		confirm.SetSync("receiver_omega[" + h + "]?")
		ctx.currentState = received

	case ir.Close:
		closing := ctx.proc.AddState("closed_"+h+"_", xta.Renaming)
		close := ctx.proc.AddTrans(ctx.currentState, closing)
		close.SetSync("close[" + h + "]!")
		ctx.currentState = closing

	default:
		t.addWarning(fmt.Errorf("unsupported ChanOp: %v", stmt.Op()))
	}
}
