package api

import (
	"fmt"
	"go/token"
	"os"

	"github.com/arneph/toph/builder"
	c "github.com/arneph/toph/config"
	"github.com/arneph/toph/ir"
	irAnalyzer "github.com/arneph/toph/ir/analyzer"
	irOptimizer "github.com/arneph/toph/ir/optimizer"
	"github.com/arneph/toph/translator"
	"github.com/arneph/toph/uppaal"
	uppaalOptimizer "github.com/arneph/toph/uppaal/optimizer"
)

// Result indicates if the Run function was successful or how it failed.
type Result int

const (
	// RunSuccessful indicates that the Run function completed successfully
	// without wanring.
	RunSuccessful Result = iota
	// RunSuccessfulButWithWarnings indicates that the Run function completed
	// successfully but generated errors.
	RunSuccessfulButWithWarnings
	// RunFailedWithBuilder indicates that the Run function failed while the
	// builder was working.
	RunFailedWithBuilder
	// RunFailedWithTranslator indicates that the Run function failed while the
	// translator was working.
	RunFailedWithTranslator
	// RunFailedWritingOutputFiles indicates that the Run function failed
	// writing the generated Uppaal files to disk.
	RunFailedWritingOutputFiles
)

// Run translates the package at the given path and returns whether it was
// successful or failed.
func Run(path string, config *c.Config) Result {
	warnings := false

	// Builder
	program, entryFuncs, errs := builder.BuildProgram(path, config)
	warnings = warnings || len(errs) > 0
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if program == nil {
		return RunFailedWithBuilder
	} else if len(entryFuncs) == 0 {
		fmt.Fprintf(os.Stderr, "found no entry functions (main or tests)\n")
	}

	initStmts := program.InitFunc().Body().Stmts()
	outNames := make([]string, len(entryFuncs))
	for i, entryFunc := range entryFuncs {
		outName := config.OutName
		if len(entryFuncs) > 1 {
			outName += "_" + entryFunc.Handle()
		}
		outNames[i] = outName
	}

	if config.Debug {
		if len(entryFuncs) == 0 {
			outputIRProgram(program, config.OutName+"_no_entry_func", "init", config)
		}

		for i, entryFunc := range entryFuncs {
			callStmt := ir.NewCallStmt(entryFunc, entryFunc.Signature(), ir.Call, token.NoPos, token.NoPos)
			program.InitFunc().Body().SetStmts(initStmts)
			program.InitFunc().Body().AddStmt(callStmt)

			outputIRProgram(program, outNames[i], "init", config)
		}
		program.InitFunc().Body().SetStmts(initStmts)
	}

	if config.OptimizeIR {
		// Dead Code Eliminator
		irOptimizer.EliminateDeadCode(program, config)

		if config.Debug {
			if len(entryFuncs) == 0 {
				outputIRProgram(program, config.OutName+"_no_entry_func", "opt", config)
			}

			for i, entryFunc := range entryFuncs {
				callStmt := ir.NewCallStmt(entryFunc, entryFunc.Signature(), ir.Call, token.NoPos, token.NoPos)
				program.InitFunc().Body().SetStmts(initStmts)
				program.InitFunc().Body().AddStmt(callStmt)

				outputIRProgram(program, outNames[i], "opt", config)
			}
			program.InitFunc().Body().SetStmts(initStmts)
		}
	}

	for i, entryFunc := range entryFuncs {
		callStmt := ir.NewCallStmt(entryFunc, entryFunc.Signature(), ir.Call, token.NoPos, token.NoPos)
		program.InitFunc().Body().SetStmts(initStmts)
		program.InitFunc().Body().AddStmt(callStmt)

		// Translator
		sys, errs := translator.TranslateProg(program, config)
		warnings = warnings || len(errs) > 0
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		if sys == nil {
			return RunFailedWithTranslator
		}

		if config.OptimizeUppaalSystem {
			if config.Debug {
				ok := outputUppaalSystem(sys, outNames[i]+".init", config.OutFormats)
				if !ok {
					return RunFailedWritingOutputFiles
				}
			}

			uppaalOptimizer.ReduceStates(sys)
			uppaalOptimizer.ReduceTransitions(sys)

			if config.Debug {
				ok := outputUppaalSystem(sys, outNames[i]+".opt", config.OutFormats)
				if !ok {
					return RunFailedWritingOutputFiles
				}
			}
		}
		if !config.Debug {
			ok := outputUppaalSystem(sys, outNames[i], config.OutFormats)
			if !ok {
				return RunFailedWritingOutputFiles
			}
		}
	}

	if warnings {
		return RunSuccessfulButWithWarnings
	}
	return RunSuccessful
}

func outputIRProgram(program *ir.Program, outName string, stepName string, config *c.Config) {
	fcg := irAnalyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go, config)
	tg := irAnalyzer.BuildTypeGraph(program)
	vi := irAnalyzer.FindVarInfo(program)

	// IR file
	programPath := fmt.Sprintf("%s.%s.ir.txt", outName, stepName)
	programFile, err := os.Create(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write ir.txt file: %v\n", err)
		return
	}
	defer programFile.Close()

	programFile.WriteString(program.Tree())

	// FCG file
	fcgPath := fmt.Sprintf("./%s.%s.fcg.txt", outName, stepName)
	fcgFile, err := os.Create(fcgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write fcg.txt file: %v\n", err)
		return
	}
	defer fcgFile.Close()

	fcgFile.WriteString(fcg.String())

	// TG file
	tgPath := fmt.Sprintf("./%s.%s.tg.txt", outName, stepName)
	tgFile, err := os.Create(tgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write tg.txt file: %v\n", err)
		return
	}
	defer tgFile.Close()

	tgFile.WriteString(tg.String())

	// VI file
	viPath := fmt.Sprintf("./%s.%s.vi.txt", outName, stepName)
	viFile, err := os.Create(viPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write vi.txt file: %v\n", err)
		return
	}
	defer viFile.Close()

	viFile.WriteString(vi.String())
}

func outputUppaalSystem(sys *uppaal.System, outName string, outFormats map[string]bool) bool {
	for _, ffmt := range []string{"xml", "xta", "ugi", "q"} {
		if !outFormats[ffmt] {
			continue
		}

		sysFile, err := os.Create(outName + "." + ffmt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not write %s file: %v\n", ffmt, err)
			return false
		}
		defer sysFile.Close()

		switch ffmt {
		case "xml":
			fmt.Fprintln(sysFile, sys.AsXML())
		case "xta":
			fmt.Fprintln(sysFile, sys.AsXTA())
		case "ugi":
			fmt.Fprintln(sysFile, sys.AsUGI())
		case "q":
			fmt.Fprintln(sysFile, sys.AsQ())
		}
	}

	return true
}
