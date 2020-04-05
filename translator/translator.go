package translator

import (
	"fmt"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

const maxProcessCount = 10
const maxChannelCount = 100

// TranslateProg translates an ir.Prog to a uppaal.System.
func TranslateProg(program *ir.Program) (*uppaal.System, []error) {
	t := new(translator)
	t.program = program
	t.mainFunc = program.GetFunc("main")
	t.funcToProcess = make(map[*ir.Func]*uppaal.Process)
	t.system = uppaal.NewSystem()
	t.fcg = analyzer.BuildFuncCallGraph(program, t.mainFunc, ir.Call|ir.Go)

	if t.mainFunc == nil {
		t.addWarning(fmt.Errorf("program has no main function"))
	}

	t.translateProgram()

	return t.system, t.warnings
}

type translator struct {
	program       *ir.Program
	mainFunc      *ir.Func
	funcToProcess map[*ir.Func]*uppaal.Process

	system         *uppaal.System
	channelProcess *uppaal.Process

	fcg *analyzer.FuncCallGraph

	warnings []error
}

func (t *translator) addWarning(err error) {
	t.warnings = append(t.warnings, err)
}

type context struct {
	f    *ir.Func
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

func (c *context) subContextForBody(current, exit *uppaal.State) *context {
	ctx := new(context)

	ctx.f = c.f
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

func (c *context) subContextForInlinedCallBody(current, exit *uppaal.State) *context {
	ctx := new(context)

	ctx.f = c.f
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

func (t *translator) translateProgram() {
	t.translateScope(t.program.Scope(), t.system.Declarations())
	t.system.Declarations().SetInitFuncName("global_initialize")

	for _, f := range t.program.Funcs() {
		t.prepareProcess(f)
	}
	for _, f := range t.program.Funcs() {
		t.translateFunc(f)
	}

	t.addChannels()
}

func (t translator) callCount(f *ir.Func) int {
	callCount := t.fcg.CallCount(f)
	if callCount < 1 {
		callCount = 1
	} else if callCount > maxProcessCount {
		callCount = maxProcessCount
	}
	return callCount
}

func (t *translator) prepareProcess(f *ir.Func) {
	name := f.Name()
	proc := t.system.AddProcess(name, uppaal.NoRenaming)
	t.funcToProcess[f] = proc
	if f == t.mainFunc {
		t.system.AddProcessInstance(proc.Name(), name, uppaal.NoRenaming)
	} else {
		for i := 0; i < t.callCount(f); i++ {
			instName := fmt.Sprintf("%s%d", name, i)
			inst := t.system.AddProcessInstance(name, instName, uppaal.Renaming)
			inst.AddParameter(fmt.Sprintf("%d", i))
		}
	}
	if f != t.mainFunc {
		t.addProcessDeclarations(f, proc)
	}
}

func (t translator) addProcessDeclarations(f *ir.Func, p *uppaal.Process) {
	t.system.Declarations().AddVariable(p.Name()+"_count", "int", "0")
	t.system.Declarations().AddArray("async_"+p.Name(), t.callCount(f), "chan")
	t.system.Declarations().AddArray("sync_"+p.Name(), t.callCount(f), "chan")

	for _, arg := range f.Args() {
		t.system.Declarations().AddArray("arg_"+arg.Handle(), t.callCount(f), "int")
	}
	for _, cap := range f.Captures() {
		t.system.Declarations().AddArray("cap_"+cap.Handle(), t.callCount(f), "int")
	}
	for i, res := range f.ResultTypes() {
		name := fmt.Sprintf("res_%s_%d_%v", p.Name(), i, res)
		t.system.Declarations().AddArray(name, t.callCount(f), "int")
	}

	t.system.Declarations().AddSpace()

	t.system.Declarations().AddFunc(
		fmt.Sprintf(`int make_%[1]s() {
	int pid = %[1]s_count;
	%[1]s_count++;
	return pid;
}`, p.Name()))
}

func (t *translator) translateFunc(f *ir.Func) {
	proc := t.funcToProcess[f]

	if f != t.mainFunc {
		proc.AddParameter(fmt.Sprintf("int[0, %d] pid", t.callCount(f)-1))
	}

	// Internal helper variables:
	proc.Declarations().AddVariable("is_sync", "bool", "false")
	proc.Declarations().AddVariable("p", "int", "-1")
	proc.Declarations().AddVariable("was_pending", "bool", "false")
	proc.Declarations().AddSpace()

	starting := proc.AddState("starting", uppaal.NoRenaming)
	starting.SetLocationAndResetNameLocation(uppaal.Location{0, 0})
	started := proc.AddState("started", uppaal.NoRenaming)
	started.SetLocationAndResetNameLocation(uppaal.Location{0, 136})

	ending := proc.AddState("ending", uppaal.NoRenaming)
	ended := proc.AddState("ended", uppaal.NoRenaming)

	proc.SetInitialState(starting)

	bodyCtx := newContext(f, proc, started, ending)
	t.translateBody(f.Body(), bodyCtx)

	endingY := bodyCtx.maxLoc[1] + 136

	ending.SetLocationAndResetNameLocation(uppaal.Location{0, endingY})
	ended.SetLocationAndResetNameLocation(uppaal.Location{0, endingY + 136})

	for _, arg := range f.Args() {
		proc.Declarations().AddInitFuncStmt(
			fmt.Sprintf("%[1]s = arg_%[1]s[pid];", arg.Handle()))
	}
	for _, cap := range f.Captures() {
		proc.Declarations().AddInitFuncStmt(
			fmt.Sprintf("%[1]s = cap_%[1]s[pid];", cap.Handle()))
	}

	if f == t.mainFunc {
		start := proc.AddTrans(starting, started)
		if t.system.Declarations().RequiresInitFunc() {
			start.AddUpdate("global_initialize()")
		}
		if proc.Declarations().RequiresInitFunc() {
			start.AddUpdate("initialize()")
		}
		start.SetUpdateLocation(uppaal.Location{0, 60})

	} else {
		startAsync := proc.AddTrans(starting, started)
		startAsync.SetSync("async_" + proc.Name() + "[pid]?")
		startAsync.AddUpdate("is_sync = false")
		if proc.Declarations().RequiresInitFunc() {
			startAsync.AddUpdate("initialize()")
		}
		startAsync.AddNail(uppaal.Location{-34, 34})
		startAsync.AddNail(uppaal.Location{-34, 102})
		startAsync.SetSyncLocation(uppaal.Location{-160, 48})
		startAsync.SetUpdateLocation(uppaal.Location{-160, 64})

		startSync := proc.AddTrans(starting, started)
		startSync.SetSync("sync_" + proc.Name() + "[pid]?")
		startSync.AddUpdate("is_sync = true")
		if proc.Declarations().RequiresInitFunc() {
			startSync.AddUpdate("initialize()")
		}
		startSync.AddNail(uppaal.Location{34, 34})
		startSync.AddNail(uppaal.Location{34, 104})
		startSync.SetSyncLocation(uppaal.Location{38, 48})
		startSync.SetUpdateLocation(uppaal.Location{38, 64})
	}

	if f == t.mainFunc {
		proc.AddTrans(ending, ended)

	} else {
		endAsync := proc.AddTrans(ending, ended)
		endAsync.SetGuard("is_sync == false")
		endAsync.AddNail(uppaal.Location{-34, endingY + 34})
		endAsync.AddNail(uppaal.Location{-34, endingY + 102})
		endAsync.SetGuardLocation(uppaal.Location{-160, endingY + 48})

		endSync := proc.AddTrans(ending, ended)
		endSync.SetGuard("is_sync == true")
		endSync.SetSync("sync_" + proc.Name() + "[pid]!")
		endSync.AddNail(uppaal.Location{34, endingY + 34})
		endSync.AddNail(uppaal.Location{34, endingY + 102})
		endSync.SetGuardLocation(uppaal.Location{38, endingY + 48})
		endSync.SetSyncLocation(uppaal.Location{38, endingY + 64})
	}
}

func (t *translator) translateBody(b *ir.Body, ctx *context) {
	t.translateScope(b.Scope(), ctx.proc.Declarations())

	for _, stmt := range b.Stmts() {
		t.translateStmt(stmt, ctx)

		if ctx.isInSpecialControlFlowState() {
			break
		}
	}

	if !ctx.isInSpecialControlFlowState() {
		ctx.proc.AddTrans(ctx.currentState, ctx.exitState)
	}
}

func (t *translator) translateScope(scope *ir.Scope, decls *uppaal.Declarations) {
	for _, v := range scope.Variables() {
		initialValue := fmt.Sprintf("%d", v.InitialValue())
		decls.AddVariable(v.Handle(), "int", initialValue)
	}
	if len(scope.Variables()) > 0 {
		decls.AddSpace()
	}
}
