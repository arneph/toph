package translator

import (
	"fmt"
	"math"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

const maxProcessCount = 100
const maxChannelCount = 100

// TranslateProg translates an ir.Prog to a uppaal.System.
func TranslateProg(program *ir.Program) (*uppaal.System, []error) {
	t := new(translator)
	t.program = program
	t.funcToProcess = make(map[*ir.Func]*uppaal.Process)
	t.system = uppaal.NewSystem()
	t.fcg = analyzer.BuildFuncCallGraph(program, ir.Call|ir.Go)

	if t.program.EntryFunc() == nil {
		t.addWarning(fmt.Errorf("program has no entry function"))
	} else if len(t.fcg.AllCallers(t.program.EntryFunc())) > 0 {
		t.addWarning(fmt.Errorf("entry function gets called within program"))
	}

	t.translateProgram()

	return t.system, t.warnings
}

type translator struct {
	program       *ir.Program
	funcToProcess map[*ir.Func]*uppaal.Process

	system         *uppaal.System
	channelProcess *uppaal.Process

	fcg *analyzer.FuncCallGraph

	warnings []error
}

func (t *translator) addWarning(err error) {
	t.warnings = append(t.warnings, err)
}

func (t *translator) translateProgram() {
	t.system.Declarations().AddType(`typedef struct {
	int id;
	int par_pid;
} fid;

fid make_fid(int id, int par_pid) {
	fid t = {id, par_pid};
	return t;
}`)
	t.system.Declarations().AddVariable("out_of_resources", "bool", "false")
	t.system.AddProgressMeasure("out_of_resources")
	t.system.AddQuery(uppaal.MakeQuery(
		"A[] not out_of_resources",
		"check system never runs out of resources"))
	t.system.Declarations().AddVariable("active_go_routines", "int", "1")
	t.system.Declarations().AddSpace()

	t.translateGlobalScope()
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
	proc := t.system.AddProcess(name)
	t.funcToProcess[f] = proc
	if f == t.program.EntryFunc() {
		t.system.AddProcessInstance(proc.Name(), name)
	} else {
		c := t.callCount(f)
		if c > 1 {
			c--
		}
		d := fmt.Sprintf("%d", int(math.Log10(float64(c))+1))
		for i := 0; i < t.callCount(f); i++ {
			instName := fmt.Sprintf("%s_%0"+d+"d", name, i)
			inst := t.system.AddProcessInstance(name, instName)
			inst.AddParameter(fmt.Sprintf("%d", i))
		}
	}
	if f != t.program.EntryFunc() {
		t.addProcessDeclarations(f, proc)
	}
}

func (t translator) addProcessDeclarations(f *ir.Func, p *uppaal.Process) {
	t.system.Declarations().AddVariable(p.Name()+"_count", "int", "0")
	t.system.Declarations().AddArray("async_"+p.Name(), t.callCount(f), "chan")
	t.system.Declarations().AddArray("sync_"+p.Name(), t.callCount(f), "chan")

	if f.EnclosingFunc() != nil {
		t.system.Declarations().AddArray("par_pid_"+p.Name(), t.callCount(f), "int")
	}
	for _, arg := range f.Args() {
		name := t.translateArgName(arg)
		var typStr string
		switch arg.Type() {
		case ir.FuncType:
			typStr = "fid"
		default:
			typStr = "int"
		}
		t.system.Declarations().AddArray(name, t.callCount(f), typStr)
	}
	for i, typ := range f.ResultTypes() {
		name := t.translateResultName(f, i)
		var typStr string
		switch typ {
		case ir.FuncType:
			typStr = "fid"
		default:
			typStr = "int"
		}
		t.system.Declarations().AddArray(name, t.callCount(f), typStr)
	}

	t.system.Declarations().AddSpace()

	if f.EnclosingFunc() == nil {
		t.system.Declarations().AddFunc(
			fmt.Sprintf(`int make_%[1]s() {
	int pid;
	if (%[1]s_count == %[2]d) {
		out_of_resources = true;
		return 0;
	}
	pid = %[1]s_count;
	%[1]s_count++;
	return pid;
}`, p.Name(), maxProcessCount))
	} else {
		t.system.Declarations().AddFunc(
			fmt.Sprintf(`int make_%[1]s(int par_pid) {
	int pid;
	if (%[1]s_count == %[2]d) {
		out_of_resources = true;
		return 0;
	}
	pid = %[1]s_count;
	%[1]s_count++;
	par_pid_%[1]s[pid] = par_pid;
	return pid;
}`, p.Name(), maxProcessCount))
	}
}

func (t *translator) translateFunc(f *ir.Func) {
	proc := t.funcToProcess[f]

	if f != t.program.EntryFunc() {
		proc.AddParameter(fmt.Sprintf("int[0, %d] pid", t.callCount(f)-1))
	} else {
		proc.Declarations().AddVariable("pid", "int", "0")
	}

	// Internal helper variables:
	proc.Declarations().AddVariable("is_sync", "bool", "false")
	proc.Declarations().AddVariable("p", "int", "-1")
	proc.Declarations().AddVariable("ok", "bool", "false")
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
		argStr := t.translateArg(arg, "pid")
		varStr := t.translateVariable(arg, bodyCtx)
		proc.Declarations().AddInitFuncStmt(
			fmt.Sprintf("%s = %s;", varStr, argStr))
	}

	if f == t.program.EntryFunc() {
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
		startAsync.AddUpdate("\nactive_go_routines++")
		if proc.Declarations().RequiresInitFunc() {
			startAsync.AddUpdate("\ninitialize()")
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

	if f == t.program.EntryFunc() {
		end := proc.AddTrans(ending, ended)
		end.SetGuard("active_go_routines == 1")
		end.SetGuardLocation(uppaal.Location{4, endingY + 64})
		proc.AddQuery(uppaal.MakeQuery("$.ending --> $.ended", ""))

	} else {
		endAsync := proc.AddTrans(ending, ended)
		endAsync.SetGuard("is_sync == false")
		endAsync.AddUpdate("active_go_routines--")
		endAsync.AddNail(uppaal.Location{-34, endingY + 34})
		endAsync.AddNail(uppaal.Location{-34, endingY + 102})
		endAsync.SetGuardLocation(uppaal.Location{-160, endingY + 48})
		endAsync.SetUpdateLocation(uppaal.Location{-194, endingY + 64})

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
	t.translateScope(ctx)

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
