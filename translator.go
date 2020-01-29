package main

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/xta"
)

type translation struct {
	tprog   *ir.Prog
	xsystem *xta.System

	tfuncToXProcess map[*ir.Func]*xta.Process
}

func translateProg(tprog *ir.Prog) (*xta.System, error) {
	var trans translation
	trans.tprog = tprog
	trans.xsystem = xta.NewSystem()
	trans.tfuncToXProcess = make(map[*ir.Func]*xta.Process)

	tmain := trans.tprog.Scope().FindNamedFunc("main")
	if tmain == nil {
		return nil, fmt.Errorf("program has no main function")
	}

	err := translateFunc(trans, tmain)
	if err != nil {
		return nil, err
	}

	return trans.xsystem, nil
}

func translateFunc(trans translation, tfunc *ir.Func) error {
	name := trans.tprog.Scope().FindNameOfFunc(tfunc)
	xproc := trans.xsystem.AddProcess(name)
	trans.tfuncToXProcess[tfunc] = xproc

	ctx := tStmtsCtx{
		xproc:  xproc,
		xstart: xproc.GetState(xta.Start),
		xend:   xproc.GetState(xta.End),
	}

	err := translateFuncBody(trans, &ctx, tfunc)
	if err != nil {
		return nil
	}

	xproc.AddTrans(ctx.xstart, ctx.xend)

	return nil
}

type tStmtsCtx struct {
	xproc  *xta.Process
	xstart *xta.State
	xend   *xta.State
}

func translateFuncBody(trans translation, ctx *tStmtsCtx, tfunc *ir.Func) error {
	xenter := ctx.xproc.AddState("enter")
	xreturn := ctx.xproc.AddState("return")
	xcomplete := ctx.xproc.AddState("exit")
	xend := ctx.xend

	ctx.xproc.AddTrans(ctx.xstart, xenter)

	ctx.xstart = xenter
	ctx.xend = xreturn

	err := translateStmts(trans, ctx, tfunc.Body().Stmts())
	if err != nil {
		return err
	}

	if tfunc.DeferredCount() == 0 {
		ctx.xproc.AddTrans(xreturn, xcomplete)
	}
	for i := 0; i < tfunc.DeferredCount(); i++ {
		deferredBody := tfunc.DeferredAt(i)
		ctx.xstart = ctx.xend
		if i > 0 {
			ctx.xend = ctx.xproc.AddState("deferred")
		} else {
			ctx.xend = xcomplete
		}
		err := translateStmts(trans, ctx, deferredBody.Stmts())
		if err != nil {
			return err
		}
	}
	ctx.xstart = xcomplete
	ctx.xend = xend
	return nil
}

func translateStmts(trans translation, ctx *tStmtsCtx, tstmts []ir.Stmt) error {
	for _, tstmt := range tstmts {
		err := translateStmt(trans, ctx, tstmt)
		if err != nil {
			return err
		}
	}
	ctx.xproc.AddTrans(ctx.xstart, ctx.xend)
	return nil
}

func translateStmt(trans translation, ctx *tStmtsCtx, tstmt ir.Stmt) error {
	switch tstmt := tstmt.(type) {
	case *ir.GoStmt:
		callee := tstmt.Callee()
		if _, ok := trans.tfuncToXProcess[callee]; !ok {
			translateFunc(trans, callee)
		}
		// TODO: lots more
		return nil

	case *ir.CallStmt:
		callee := tstmt.Callee()
		err := translateFuncBody(trans, ctx, callee)
		if err != nil {
			return err
		}
		// TODO: lots more
		return nil

	case *ir.ForStmt:
		xCondEnter := ctx.xproc.AddState("enterLoopCond")
		xCondExit := ctx.xproc.AddState("exitLoopCond")
		condCtx := tStmtsCtx{
			xproc:  ctx.xproc,
			xstart: xCondEnter,
			xend:   xCondExit,
		}
		err := translateStmts(trans, &condCtx, tstmt.Cond().Stmts())
		if err != nil {
			return err
		}

		xBodyEnter := ctx.xproc.AddState("enterLoopBody")
		xBodyExit := ctx.xproc.AddState("exitLoopBody")
		bodyCtx := tStmtsCtx{
			xproc:  ctx.xproc,
			xstart: xBodyEnter,
			xend:   xBodyExit,
		}
		translateStmts(trans, &bodyCtx, tstmt.Body().Stmts())
		if err != nil {
			return err
		}

		xLoopExit := ctx.xproc.AddState("exitLoop")

		ctx.xproc.AddTrans(ctx.xstart, xCondEnter)
		ctx.xproc.AddTrans(xCondExit, xBodyEnter)
		ctx.xproc.AddTrans(xCondExit, xLoopExit)
		ctx.xproc.AddTrans(xBodyExit, xCondEnter)
		ctx.xstart = xLoopExit

		return nil

	case *ir.SendStmt:
		xsend := ctx.xproc.AddState(fmt.Sprintf("send_%p_", tstmt.Channel()))
		ctx.xproc.AddTrans(ctx.xstart, xsend)
		ctx.xstart = xsend

		return nil

	case *ir.ReceiveStmt:
		xreceive := ctx.xproc.AddState(fmt.Sprintf("receive_%p_", tstmt.Channel()))
		ctx.xproc.AddTrans(ctx.xstart, xreceive)
		ctx.xstart = xreceive

		return nil

	case *ir.CloseStmt:
		xclose := ctx.xproc.AddState(fmt.Sprintf("close_%p_", tstmt.Channel()))
		ctx.xproc.AddTrans(ctx.xstart, xclose)
		ctx.xstart = xclose

		return nil

	default:
		fmt.Printf("ignoring %T statement\n", tstmt)
		return nil
	}
}
