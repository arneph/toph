package translator

import (
	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/uppaal"
)

// Config contains configuration settings for translation.
type Config struct {
	MaxProcessCount   int
	MaxDeferCount     int
	MaxChannelCount   int
	MaxMutexCount     int
	MaxWaitGroupCount int
	MaxStructCount    int

	Optimize bool
}

// TranslateProg translates an ir.Prog to a uppaal.System.
func TranslateProg(program *ir.Program, config *Config) (*uppaal.System, []error) {
	t := new(translator)
	t.program = program
	t.funcToProcess = make(map[*ir.Func]*uppaal.Process)
	t.system = uppaal.NewSystem()
	t.tg = analyzer.BuildTypeGraph(program)
	t.completeFCG = analyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go)
	t.deferFCG = analyzer.BuildFuncCallGraph(program, ir.Defer)
	t.config = config

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

	tg *analyzer.TypeGraph

	completeFCG *analyzer.FuncCallGraph
	deferFCG    *analyzer.FuncCallGraph

	config *Config

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
		"check system never runs out of resources", "",
		uppaal.ResourceBoundUnreached))
	t.system.Declarations().AddVariable("active_go_routines", "int", "1")
	t.system.Declarations().AddSpace()

	t.translateGlobalScope()
	t.system.Declarations().SetInitFuncName("global_initialize")

	for _, u := range t.tg.TopologicalOrder() {
		t.addType(u)
	}
	for _, f := range t.program.Funcs() {
		if t.config.Optimize && t.completeFCG.CalleeCount(f) == 0 {
			continue
		}
		t.addFuncProcess(f)
		if f == t.program.InitFunc() {
			continue
		}
		t.addFuncDeclarations(f)
	}
	for _, f := range t.program.Funcs() {
		if t.config.Optimize && t.completeFCG.CalleeCount(f) == 0 {
			continue
		}
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
