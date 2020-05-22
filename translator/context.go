package translator

import (
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

type context struct {
	f    *ir.Func
	body *ir.Body
	proc *uppaal.Process

	currentState *uppaal.State
	exitState    *uppaal.State

	exitFuncState      *uppaal.State
	continueLoopStates map[ir.Loop]*uppaal.State
	breakLoopStates    map[ir.Loop]*uppaal.State

	minLoc, maxLoc uppaal.Location
}

func newContext(f *ir.Func, p *uppaal.Process, current, exit *uppaal.State) *context {
	ctx := new(context)
	ctx.f = f
	ctx.body = f.Body()
	ctx.proc = p

	ctx.currentState = current
	ctx.exitState = exit
	ctx.exitFuncState = exit

	ctx.minLoc = current.Location()
	ctx.maxLoc = current.Location()

	return ctx
}

func (c *context) isInSpecialControlFlowState() bool {
	if c.currentState == c.exitFuncState {
		return true
	}
	for _, s := range c.continueLoopStates {
		if c.currentState == s {
			return true
		}
	}
	for _, s := range c.breakLoopStates {
		if c.currentState == s {
			return true
		}
	}
	return false
}

func (c *context) subContextForBody(body *ir.Body, current, exit *uppaal.State) *context {
	ctx := new(context)

	ctx.f = c.f
	ctx.body = body
	ctx.proc = c.proc

	ctx.currentState = current
	ctx.exitState = exit

	ctx.exitFuncState = c.exitFuncState
	ctx.continueLoopStates = c.continueLoopStates
	ctx.breakLoopStates = c.breakLoopStates

	ctx.minLoc = current.Location()
	ctx.maxLoc = current.Location()

	return ctx
}

func (c *context) subContextForInlinedCallBody(body *ir.Body, current, exit *uppaal.State) *context {
	ctx := new(context)

	ctx.f = c.f
	ctx.body = body
	ctx.proc = c.proc

	ctx.currentState = current
	ctx.exitState = exit

	ctx.exitFuncState = exit
	ctx.continueLoopStates = c.continueLoopStates
	ctx.breakLoopStates = c.breakLoopStates

	ctx.minLoc = current.Location()
	ctx.maxLoc = current.Location()

	return ctx
}

func (c *context) subContextForLoopBody(loop ir.Loop, current, continueLoop, breakLoop *uppaal.State) *context {
	ctx := new(context)
	ctx.f = c.f
	ctx.body = loop.Body()
	ctx.proc = c.proc

	ctx.currentState = current
	ctx.exitState = continueLoop

	ctx.exitFuncState = c.exitFuncState
	ctx.continueLoopStates = make(map[ir.Loop]*uppaal.State)
	for l, s := range c.continueLoopStates {
		ctx.continueLoopStates[l] = s
	}
	ctx.continueLoopStates[loop] = continueLoop
	ctx.breakLoopStates = make(map[ir.Loop]*uppaal.State)
	for l, s := range c.breakLoopStates {
		ctx.breakLoopStates[l] = s
	}
	ctx.breakLoopStates[loop] = breakLoop

	ctx.minLoc = current.Location()
	ctx.maxLoc = current.Location()

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
