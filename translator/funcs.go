package translator

import (
	"fmt"
	"math"

	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

func (t translator) isFuncUsed(f *ir.Func) bool {
	if !t.config.OptimizeIR {
		return true
	}
	return t.completeFCG.CalleeCount(f) > 0
}

func (t translator) callCount(f *ir.Func) int {
	callCount := t.completeFCG.CalleeCount(f)
	if callCount < 1 {
		callCount = 1
	} else if callCount > t.config.MaxProcessCount {
		callCount = t.config.MaxProcessCount
	}
	return callCount
}

func (t translator) deferCount(f *ir.Func) int {
	deferCount := t.deferFCG.CallerCount(f)
	if deferCount > t.config.MaxDeferCount {
		deferCount = t.config.MaxDeferCount
	}
	return deferCount
}

func (t *translator) addFuncProcess(f *ir.Func) {
	procName := f.Handle()
	proc := t.system.AddProcess(procName)
	t.funcToProcess[f] = proc
	if f == t.program.InitFunc() {
		t.system.AddProcessInstance(proc, procName)
	} else {
		c := t.callCount(f)
		if c > 1 {
			c--
		}
		d := fmt.Sprintf("%d", int(math.Log10(float64(c))+1))
		for i := 0; i < t.callCount(f); i++ {
			instName := fmt.Sprintf("%s_%0"+d+"d", procName, i)
			inst := t.system.AddProcessInstance(proc, instName)
			inst.AddParameter(fmt.Sprintf("%d", i))
		}
	}
}

func (t translator) addFuncDeclarations(f *ir.Func) {
	proc := t.funcToProcess[f]

	t.system.Declarations().AddVariable(proc.Name()+"_count", "int", "0")
	t.system.Declarations().AddArray("async_"+proc.Name(), []int{t.callCount(f)}, "chan")
	t.system.Declarations().AddArray("sync_"+proc.Name(), []int{t.callCount(f)}, "chan")

	if f.EnclosingFunc() != nil {
		t.system.Declarations().AddArray("par_pid_"+proc.Name(), []int{t.callCount(f)}, "int")
	}

	externalPanicInit := ""
	if !t.config.OptimizeIR || t.completeFCG.CanPanic(f) || t.completeFCG.CanRecover(f) {
		t.system.Declarations().AddArray("external_panic_"+proc.Name(), []int{t.callCount(f)}, "bool")
		externalPanicInit = fmt.Sprintf("\n    external_panic_%s[pid] = false;", proc.Name())
	}

	for _, arg := range f.Args() {
		name := t.translateArgName(arg)
		typStr := t.uppaalReferenceTypeForIrType(arg.Type())
		t.system.Declarations().AddArray(name, []int{t.callCount(f)}, typStr)
	}
	for i, typ := range f.ResultTypes() {
		name := t.translateResultName(f, i)
		typStr := t.uppaalReferenceTypeForIrType(typ)
		t.system.Declarations().AddArray(name, []int{t.callCount(f)}, typStr)
	}

	t.system.Declarations().AddSpaceBetweenVariables()

	if f.EnclosingFunc() == nil {
		t.system.Declarations().AddFunc(
			fmt.Sprintf(`int make_%[1]s() {
	int pid;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	pid = %[1]s_count;
	%[1]s_count++;%[3]s
	return pid;
}`, proc.Name(), t.callCount(f), externalPanicInit))
	} else {
		t.system.Declarations().AddFunc(
			fmt.Sprintf(`int make_%[1]s(int par_pid) {
	int pid;
	if (%[1]s_count >= %[2]d) {
		%[1]s_count++;
		out_of_resources = true;
		return 0;
	}
	pid = %[1]s_count;
	%[1]s_count++;
	par_pid_%[1]s[pid] = par_pid;%[3]s
	return pid;
}`, proc.Name(), t.callCount(f), externalPanicInit))
	}
	if t.config.GenerateIndividualResourceBoundQueries {
		t.system.AddQuery(uppaal.NewQuery(
			fmt.Sprintf("A[] %s_count < %d", proc.Name(), t.callCount(f)+1),
			fmt.Sprintf("check resource bound never reached through %s creation", proc.Name()),
			"",
			uppaal.ResourceBoundUnreached))
	}
}

func (t *translator) translateFunc(f *ir.Func) {
	proc := t.funcToProcess[f]

	callCount := t.callCount(f)
	deferCount := t.deferCount(f)

	if f != t.program.InitFunc() {
		proc.AddParameter(fmt.Sprintf("int[0, %d] pid", callCount-1))
	} else {
		proc.Declarations().AddVariable("pid", "int", "0")
	}

	// Internal helper variables:
	proc.Declarations().AddVariable("is_sync", "bool", "false")
	if !t.config.OptimizeIR || t.completeFCG.CanPanic(f) {
		proc.Declarations().AddVariable("internal_panic", "bool", "false")
	}
	if deferCount > 0 {
		proc.Declarations().AddVariable("deferred_count", "int", "0")
		proc.Declarations().AddArray("deferred_fid", []int{deferCount}, "int")
		proc.Declarations().AddArray("deferred_pid", []int{deferCount}, "int")
	}
	proc.Declarations().AddVariable("p", "int", "-1")
	proc.Declarations().AddVariable("ok", "bool", "false")
	proc.Declarations().AddSpaceBetweenVariables()

	starting := proc.AddState("starting", uppaal.NoRenaming)
	starting.SetComment(t.program.FileSet().Position(f.Pos()).String())
	starting.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, 0})
	started := proc.AddState("started", uppaal.NoRenaming)
	started.SetComment(t.program.FileSet().Position(f.Pos()).String())
	started.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, 136})

	proc.SetInitialState(starting)

	finalizing := proc.AddState("finalizing", uppaal.NoRenaming)
	finalizing.SetComment(t.program.FileSet().Position(f.End()).String())
	ending := proc.AddState("ending", uppaal.NoRenaming)
	ending.SetComment(t.program.FileSet().Position(f.End()).String())
	ended := proc.AddState("ended", uppaal.NoRenaming)
	ended.SetComment(t.program.FileSet().Position(f.End()).String())

	sourceLocation := t.program.FileSet().Position(f.End()).String()
	if f == t.program.InitFunc() {
		sourceLocation = ""
	}
	if t.config.GenerateGoroutineExitWithPanicQueries {
		if !t.config.OptimizeIR || t.completeFCG.CanPanic(f) {
			proc.AddQuery(uppaal.NewQuery(
				"A[] (not out_of_resources) imply (not ($.ending and !$.is_sync and $.internal_panic))",
				"check goroutine does not exit with panic",
				sourceLocation,
				uppaal.NoGoroutineExitWithPanic))
		}
	}

	var deferred *uppaal.State
	var exitFuncState *uppaal.State
	if deferCount > 0 {
		deferred = proc.AddState("deferred", uppaal.NoRenaming)
		deferred.SetComment(t.program.FileSet().Position(f.End()).String())

		exitFuncState = deferred
	} else {
		exitFuncState = finalizing
	}

	bodyCtx := newContext(f, proc, started, exitFuncState)

	t.translateBody(f.Body(), bodyCtx)

	bodyEndY := bodyCtx.maxLoc[1]
	endingY := bodyEndY + 136
	if deferCount > 0 {
		endingY += 272

		deferred.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 136})
		finalizing.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 408})
		ending.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 544})
		ended.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 680})
	} else {
		finalizing.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 136})
		ending.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 272})
		ended.SetLocationAndResetNameAndCommentLocation(uppaal.Location{0, bodyEndY + 408})
	}

	for returnTrans := range bodyCtx.returnTransitions {
		returnTrans.AddNail(exitFuncState.Location().Sub(uppaal.Location{68, 0}))
	}

	for _, arg := range f.Args() {
		argStr := t.translateArg(arg, "pid")
		varStr, _ := t.translateVariable(arg, bodyCtx)
		proc.Declarations().AddInitFuncStmt(
			fmt.Sprintf("%s = %s;", varStr, argStr))
	}

	if deferCount > 0 {
		reachEnd := proc.AddTransition(deferred, finalizing)
		reachEnd.SetGuard("deferred_count == 0", false)
		reachEnd.SetGuardLocation(deferred.Location().Add(uppaal.Location{4, 226}))

		t.translateDeferredCalls(proc, deferred, f)
	}

	finalize := proc.AddTransition(finalizing, ending)
	if f != t.program.InitFunc() && (!t.config.OptimizeIR || t.completeFCG.CanPanic(f)) {
		finalize.AddUpdate(fmt.Sprintf("external_panic_%s[pid] |= internal_panic", proc.Name()), false)
		finalize.SetUpdateLocation(uppaal.Location{4, endingY + 60})
	}

	endingY += 136

	if f == t.program.InitFunc() {
		start := proc.AddTransition(starting, started)
		if t.system.Declarations().RequiresInitFunc() {
			start.AddUpdate("global_initialize()", true)
		}
		if proc.Declarations().RequiresInitFunc() {
			start.AddUpdate("initialize()", true)
		}
		start.SetUpdateLocation(uppaal.Location{0, 60})

		end := proc.AddTransition(ending, ended)
		end.SetGuard("active_go_routines == 1", true)
		end.SetGuardLocation(uppaal.Location{4, endingY + 64})

	} else {
		startAsync := proc.AddTransition(starting, started)
		startAsync.SetSync("async_" + proc.Name() + "[pid]?")
		startAsync.AddUpdate("is_sync = false", false)
		startAsync.AddUpdate("\nactive_go_routines++", true)
		if proc.Declarations().RequiresInitFunc() {
			startAsync.AddUpdate("\ninitialize()", true)
		}
		startAsync.AddNail(uppaal.Location{-34, 34})
		startAsync.AddNail(uppaal.Location{-34, 102})
		startAsync.SetSyncLocation(uppaal.Location{-160, 48})
		startAsync.SetUpdateLocation(uppaal.Location{-160, 64})

		startSync := proc.AddTransition(starting, started)
		startSync.SetSync("sync_" + proc.Name() + "[pid]?")
		startSync.AddUpdate("is_sync = true", false)
		if proc.Declarations().RequiresInitFunc() {
			startSync.AddUpdate("\ninitialize()", true)
		}
		startSync.AddNail(uppaal.Location{34, 34})
		startSync.AddNail(uppaal.Location{34, 104})
		startSync.SetSyncLocation(uppaal.Location{38, 48})
		startSync.SetUpdateLocation(uppaal.Location{38, 64})

		endAsync := proc.AddTransition(ending, ended)
		endAsync.SetGuard("is_sync == false", false)
		endAsync.AddUpdate("active_go_routines--", true)
		endAsync.AddNail(uppaal.Location{-34, endingY + 34})
		endAsync.AddNail(uppaal.Location{-34, endingY + 102})
		endAsync.SetGuardLocation(uppaal.Location{-160, endingY + 48})
		endAsync.SetUpdateLocation(uppaal.Location{-194, endingY + 64})

		endSync := proc.AddTransition(ending, ended)
		endSync.SetGuard("is_sync == true", false)
		endSync.SetSync("sync_" + proc.Name() + "[pid]!")

		endSync.AddNail(uppaal.Location{34, endingY + 34})
		endSync.AddNail(uppaal.Location{34, endingY + 102})
		endSync.SetGuardLocation(uppaal.Location{38, endingY + 48})
		endSync.SetSyncLocation(uppaal.Location{38, endingY + 64})
	}
}
