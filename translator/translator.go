package translator

import (
	"fmt"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/xta"
)

// TranslateProg translates an ir.Prog to a xta.System.
func TranslateProg(program *ir.Program) (*xta.System, []error) {
	t := new(translator)
	t.program = program
	t.mainFunc = program.GetNamedFunc("main")
	t.funcToProcess = make(map[*ir.Func]*xta.Process)
	t.system = xta.NewSystem()

	if t.mainFunc == nil {
		t.addWarning(fmt.Errorf("program has no main function"))
	}

	t.translateProgram()

	return t.system, t.warnings
}

type translator struct {
	program       *ir.Program
	mainFunc      *ir.Func
	funcToProcess map[*ir.Func]*xta.Process

	system         *xta.System
	channelProcess *xta.Process

	warnings []error
}

func (t *translator) addWarning(err error) {
	t.warnings = append(t.warnings, err)
}

type context struct {
	f    *ir.Func
	proc *xta.Process

	currentState *xta.State
	exitState    *xta.State

	exitFuncState      *xta.State
	continueLoopStates map[ir.Loop]*xta.State
	breakLoopStates    map[ir.Loop]*xta.State
}

func newContext(f *ir.Func, p *xta.Process, current, exit *xta.State) *context {
	ctx := new(context)
	ctx.f = f
	ctx.proc = p

	ctx.currentState = current
	ctx.exitState = exit

	ctx.exitFuncState = exit

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

func (c *context) subContextForBody(current, exit *xta.State) *context {
	ctx := new(context)

	ctx.f = c.f
	ctx.proc = c.proc

	ctx.currentState = current
	ctx.exitState = exit

	ctx.exitFuncState = ctx.exitFuncState
	ctx.continueLoopStates = ctx.continueLoopStates
	ctx.breakLoopStates = ctx.breakLoopStates

	return ctx
}

func (c *context) subContextForLoopBody(loop ir.Loop, current, continueLoop, breakLoop *xta.State) *context {
	ctx := new(context)
	ctx.f = c.f
	ctx.proc = c.proc

	ctx.currentState = current
	ctx.exitState = continueLoop

	ctx.exitFuncState = ctx.exitFuncState
	ctx.continueLoopStates = make(map[ir.Loop]*xta.State)
	for l, s := range c.continueLoopStates {
		ctx.continueLoopStates[l] = s
	}
	ctx.continueLoopStates[loop] = continueLoop
	ctx.breakLoopStates = make(map[ir.Loop]*xta.State)
	for l, s := range c.breakLoopStates {
		ctx.breakLoopStates[l] = s
	}
	ctx.breakLoopStates[loop] = breakLoop

	return ctx
}

func (t *translator) translateProgram() {
	t.translateScope(t.program.Scope(), t.system.Declarations())
	t.system.Declarations().AddVariableDeclaration("")
	t.system.Declarations().SetInitFuncName("global_initialize")

	for _, f := range t.program.Funcs() {
		t.prepareProcess(f)
	}
	for _, f := range t.program.Funcs() {
		t.translateFunc(f)
	}

	t.addChannels()
}

func (t *translator) translateScope(scope *ir.Scope, decls *xta.Declarations) {
	for _, v := range scope.Variables() {
		varDecl := fmt.Sprintf("int %s = %d;", v.Handle(), v.InitialValue())
		decls.AddVariableDeclaration(varDecl)
	}
}

func (t *translator) prepareProcess(f *ir.Func) {
	name := f.Name()
	proc := t.system.AddProcess(name, xta.NoRenaming)
	t.funcToProcess[f] = proc
	if f == t.mainFunc {
		t.system.AddProcessInstance(name, name, xta.NoRenaming)
	} else {
		for i := 0; i < maxProcessCount; i++ {
			instName := fmt.Sprintf("%s%d", name, i)
			inst := t.system.AddProcessInstance(name, instName, xta.NoRenaming)
			inst.AddParameter(fmt.Sprintf("%d", i))
		}
	}
	if f != t.mainFunc {
		t.addProcessDeclarations(f, proc)
	}
}

func (t *translator) translateFunc(f *ir.Func) {
	proc := t.funcToProcess[f]

	if f != t.mainFunc {
		proc.AddParameter(fmt.Sprintf("int[0, %d] pid", maxProcessCount-1))
	}

	proc.Declarations().AddVariableDeclaration("bool is_sync;")
	proc.Declarations().AddVariableDeclaration("int p = -1;")

	starting := proc.AddState("starting", xta.NoRenaming)
	started := proc.AddState("started", xta.NoRenaming)
	ending := proc.AddState("ending", xta.NoRenaming)
	ended := proc.AddState("ended", xta.NoRenaming)

	proc.SetInitialState(starting)

	t.translateBody(f.Body(), newContext(f, proc, started, ending))

	for _, arg := range f.Args() {
		proc.Declarations().AddInitFuncInstr(
			fmt.Sprintf("%[1]s = arg_%[1]s[pid];", arg.Handle()))
	}
	for _, cap := range f.Captures() {
		proc.Declarations().AddInitFuncInstr(
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

	} else {
		startAsync := proc.AddTrans(starting, started)
		startAsync.SetSync("async_" + proc.Name() + "[pid]?")
		startAsync.AddUpdate("is_sync = false")
		if proc.Declarations().RequiresInitFunc() {
			startAsync.AddUpdate("initialize()")
		}

		startSync := proc.AddTrans(starting, started)
		startSync.SetSync("sync_" + proc.Name() + "[pid]?")
		startSync.AddUpdate("is_sync = true")
		if proc.Declarations().RequiresInitFunc() {
			startSync.AddUpdate("initialize()")
		}
	}

	if f == t.mainFunc {
		proc.AddTrans(ending, ended)

	} else {
		endAsync := proc.AddTrans(ending, ended)
		endAsync.SetGuard("is_sync == false")

		endSync := proc.AddTrans(ending, ended)
		endSync.SetGuard("is_sync == true")
		endSync.SetSync("sync_" + proc.Name() + "[pid]!")
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

const maxProcessCount = 1

func (t translator) addProcessDeclarations(f *ir.Func, p *xta.Process) {
	t.system.Declarations().AddVariableDeclaration(
		fmt.Sprintf("int %s_count = 0;", p.Name()))

	for _, arg := range f.Args() {
		t.system.Declarations().AddVariableDeclaration(
			fmt.Sprintf("int arg_%s[%d];", arg.Handle(), maxProcessCount))
	}
	for _, cap := range f.Captures() {
		t.system.Declarations().AddVariableDeclaration(
			fmt.Sprintf("int cap_%s[%d];", cap.Handle(), maxProcessCount))
	}
	for i, res := range f.ResultTypes() {
		t.system.Declarations().AddVariableDeclaration(
			fmt.Sprintf("int res_%s_%d_%v[%d];", p.Name(), i, res, maxProcessCount))
	}

	t.system.Declarations().AddVariableDeclaration(
		fmt.Sprintf("chan async_%s[%d];", p.Name(), maxProcessCount))
	t.system.Declarations().AddVariableDeclaration(
		fmt.Sprintf("chan sync_%s[%d];", p.Name(), maxProcessCount))

	t.system.Declarations().AddVariableDeclaration(fmt.Sprintf(
		`int make_%[1]s() {
    int pid = %[1]s_count;
    %[1]s_count++;
    return pid;
}
`, p.Name()))
}
