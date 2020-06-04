package translator

import (
	"fmt"
	"math"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

const maxProcessCount = 100
const maxDeferCount = 100
const maxChannelCount = 100

// TranslateProg translates an ir.Prog to a uppaal.System.
func TranslateProg(program *ir.Program) (*uppaal.System, []error) {
	t := new(translator)
	t.program = program
	t.funcToProcess = make(map[*ir.Func]*uppaal.Process)
	t.system = uppaal.NewSystem()
	t.completeFCG = analyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go)
	t.deferFCG = analyzer.BuildFuncCallGraph(program, ir.Defer)

	if t.program.EntryFunc() == nil {
		t.addWarning(fmt.Errorf("program has no entry function"))
	} else if len(t.completeFCG.AllCallers(t.program.EntryFunc())) > 0 {
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

	completeFCG *analyzer.FuncCallGraph
	deferFCG    *analyzer.FuncCallGraph

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
	callCount := t.completeFCG.CalleeCount(f)
	if callCount < 1 {
		callCount = 1
	} else if callCount > maxProcessCount {
		callCount = maxProcessCount
	}
	return callCount
}

func (t translator) deferCount(f *ir.Func) int {
	deferCount := t.deferFCG.CallerCount(f)
	deferCount += t.deferFCG.CloseChanCount(f)
	if deferCount > maxDeferCount {
		deferCount = maxDeferCount
	}
	return deferCount
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

	callCount := t.callCount(f)
	deferCount := t.deferCount(f)

	if f != t.program.EntryFunc() {
		proc.AddParameter(fmt.Sprintf("int[0, %d] pid", callCount-1))
	} else {
		proc.Declarations().AddVariable("pid", "int", "0")
	}

	// Internal helper variables:
	proc.Declarations().AddVariable("is_sync", "bool", "false")
	if deferCount > 0 {
		proc.Declarations().AddVariable("deferred_count", "int", "0")
		proc.Declarations().AddArray("deferred_is_close", deferCount, "bool")
		proc.Declarations().AddArray("deferred_cid", deferCount, "int")
		proc.Declarations().AddArray("deferred_fid", deferCount, "int")
		proc.Declarations().AddArray("deferred_pid", deferCount, "int")
	}
	proc.Declarations().AddVariable("p", "int", "-1")
	proc.Declarations().AddVariable("ok", "bool", "false")
	proc.Declarations().AddSpace()

	starting := proc.AddState("starting", uppaal.NoRenaming)
	starting.SetComment(t.program.FileSet().Position(f.Pos()).String())
	starting.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, 0})
	started := proc.AddState("started", uppaal.NoRenaming)
	started.SetComment(t.program.FileSet().Position(f.Pos()).String())
	started.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, 136})

	proc.SetInitialState(starting)

	ending := proc.AddState("ending", uppaal.NoRenaming)
	ending.SetComment(t.program.FileSet().Position(f.End()).String())
	ended := proc.AddState("ended", uppaal.NoRenaming)
	ended.SetComment(t.program.FileSet().Position(f.End()).String())

	var deferred *uppaal.State
	var bodyCtx *context
	if deferCount > 0 {
		deferred = proc.AddState("deferred", uppaal.NoRenaming)
		deferred.SetComment(t.program.FileSet().Position(f.End()).String())

		bodyCtx = newContext(f, proc, started, deferred)
	} else {
		bodyCtx = newContext(f, proc, started, ending)
	}

	t.translateBody(f.Body(), bodyCtx)

	bodyEndY := bodyCtx.maxLoc[1]
	endingY := bodyEndY + 136
	if deferCount > 0 {
		endingY += 272

		deferred.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 136})
		ending.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 408})
		ended.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 544})
	} else {
		ending.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 136})
		ended.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 272})
	}

	for _, arg := range f.Args() {
		argStr := t.translateArg(arg, "pid")
		varStr := t.translateVariable(arg, bodyCtx)
		proc.Declarations().AddInitFuncStmt(
			fmt.Sprintf("%s = %s;", varStr, argStr))
	}

	if deferCount > 0 {
		reachEnd := proc.AddTrans(deferred, ending)
		reachEnd.SetGuard("deferred_count == 0")
		reachEnd.SetGuardLocation(deferred.Location().Add(uppaal.Location{4, 226}))

		closed := proc.AddState("closed_", uppaal.Renaming)
		closed.SetLocationAndResetNameAndCommentLocation(
			deferred.Location().Add(uppaal.Location{136, 136}))
		close := proc.AddTrans(deferred, closed)
		close.SetGuard("deferred_count > 0 && deferred_is_close[deferred_count-1]")
		close.SetGuardLocation(deferred.Location().Add(uppaal.Location{44, 48}))
		close.SetSync("close[deferred_cid[deferred_count-1]]!")
		close.SetSyncLocation(deferred.Location().Add(uppaal.Location{44, 64}))
		next := proc.AddTrans(closed, deferred)
		next.AddUpdate("deferred_count--")
		next.SetUpdateLocation(deferred.Location().Add(uppaal.Location{44, 182}))
		next.AddNail(closed.Location().Add(uppaal.Location{0, 68}))
		next.AddNail(deferred.Location().Add(uppaal.Location{68, 204}))

		for i, calleeFunc := range t.deferFCG.AllCallees(f) {
			calleeProc := t.funcToProcess[calleeFunc]

			started := proc.AddState("started_"+calleeProc.Name()+"_", uppaal.Renaming)
			started.SetLocationAndResetNameAndCommentLocation(
				deferred.Location().Add(uppaal.Location{136 * (i + 2), 136}))
			start := proc.AddTrans(deferred, started)
			start.SetGuard("deferred_count > 0 && !deferred_is_close[deferred_count-1] && deferred_fid[deferred_count-1] == " + calleeFunc.FuncValue().String())
			start.SetGuardLocation(
				deferred.Location().Add(uppaal.Location{180 * (i + 1), 48 + 32*(i+1)}))
			start.SetSync(fmt.Sprintf("sync_%s[deferred_pid[deferred_count-1]]!", calleeProc.Name()))
			start.SetSyncLocation(
				deferred.Location().Add(uppaal.Location{180 * (i + 1), 64 + 32*(i+1)}))
			wait := proc.AddTrans(started, deferred)
			wait.SetSync(fmt.Sprintf("sync_%s[deferred_pid[deferred_count-1]]?", calleeProc.Name()))
			wait.SetSyncLocation(started.Location().Add(uppaal.Location{4, 32 + 16*i}))
			wait.AddUpdate("deferred_count--")
			wait.SetUpdateLocation(deferred.Location().Add(uppaal.Location{44, 182}))
			wait.AddNail(started.Location().Add(uppaal.Location{0, 68}))
			wait.AddNail(deferred.Location().Add(uppaal.Location{68, 204}))
		}
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

		end := proc.AddTrans(ending, ended)
		end.SetGuard("active_go_routines == 1")
		end.SetGuardLocation(uppaal.Location{4, endingY + 64})
		proc.AddQuery(uppaal.MakeQuery("$.ending --> $.ended", ""))

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
