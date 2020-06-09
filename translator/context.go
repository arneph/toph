package translator

import (
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

type context struct {
	f    *ir.Func
	body *ir.Body
	proc *uppaal.Process

	currentState   *uppaal.State
	exitBodyState  *uppaal.State
	exitFuncState  *uppaal.State
	breakStates    map[ir.Stmt]*uppaal.State
	continueStates map[ir.Stmt]*uppaal.State

	minLoc, maxLoc uppaal.Location
}

func newContext(f *ir.Func, p *uppaal.Process, currentState, exitFuncState *uppaal.State) *context {
	ctx := new(context)
	ctx.f = f
	ctx.body = f.Body()
	ctx.proc = p

	ctx.currentState = currentState
	ctx.exitBodyState = exitFuncState
	ctx.exitFuncState = exitFuncState
	ctx.breakStates = make(map[ir.Stmt]*uppaal.State)
	ctx.continueStates = make(map[ir.Stmt]*uppaal.State)

	ctx.minLoc = currentState.Location()
	ctx.maxLoc = currentState.Location()

	return ctx
}

func (c *context) isInSpecialControlFlowState() bool {
	if c.currentState == c.exitFuncState {
		return true
	}
	for _, s := range c.breakStates {
		if c.currentState == s {
			return true
		}
	}
	for _, s := range c.continueStates {
		if c.currentState == s {
			return true
		}
	}
	return false
}

func (c *context) subContextForStmt(stmt ir.Stmt, body *ir.Body, currentState, breakState, continueState, exitBodyState *uppaal.State) *context {
	ctx := new(context)
	ctx.f = c.f
	ctx.body = body
	ctx.proc = c.proc

	ctx.currentState = currentState
	ctx.exitBodyState = exitBodyState
	ctx.exitFuncState = c.exitFuncState
	ctx.continueStates = make(map[ir.Stmt]*uppaal.State)
	for l, s := range c.continueStates {
		ctx.continueStates[l] = s
	}
	ctx.continueStates[stmt] = continueState
	ctx.breakStates = make(map[ir.Stmt]*uppaal.State)
	for l, s := range c.breakStates {
		ctx.breakStates[l] = s
	}
	ctx.breakStates[stmt] = breakState

	ctx.minLoc = currentState.Location()
	ctx.maxLoc = currentState.Location()

	return ctx
}

func (c *context) subContextForInlinedCallBody(body *ir.Body, currentState, exitState *uppaal.State) *context {
	ctx := new(context)

	ctx.f = c.f
	ctx.body = body
	ctx.proc = c.proc

	ctx.currentState = currentState
	ctx.exitBodyState = exitState
	ctx.exitFuncState = exitState
	ctx.continueStates = c.continueStates
	ctx.breakStates = c.breakStates

	ctx.minLoc = currentState.Location()
	ctx.maxLoc = currentState.Location()

	return ctx
}

func (c *context) addLocation(l uppaal.Location) {
	c.minLoc = uppaal.Min(c.minLoc, l)
	c.maxLoc = uppaal.Max(c.maxLoc, l)
}

func (c *context) addLocationsFromSubContext(s *context) {
	c.minLoc = uppaal.Min(c.minLoc, s.minLoc)
	c.minLoc = uppaal.Min(c.minLoc, s.maxLoc)
	c.maxLoc = uppaal.Max(c.maxLoc, s.minLoc)
	c.maxLoc = uppaal.Max(c.maxLoc, s.maxLoc)
}
