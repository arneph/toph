package translator

import (
	"fmt"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

const maxProcessCount = 100
const maxDeferCount = 100
const maxChannelCount = 100
const maxMutexCount = 100
const maxWaitGroupCount = 100

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

	system           *uppaal.System
	channelProcess   *uppaal.Process
	mutexProcess     *uppaal.Process
	waitGroupProcess *uppaal.Process

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
	t.system.Declarations().AddSpace()
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
		t.addFuncProcess(f)
		if f == t.program.EntryFunc() {
			continue
		}
		t.addFuncDeclarations(f)
	}
	for _, f := range t.program.Funcs() {
		t.translateFunc(f)
	}

	t.addChannels()
	t.addMutexes()
	t.addWaitGroups()
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
		ctx.proc.AddTrans(ctx.currentState, ctx.exitBodyState)
	}
}
