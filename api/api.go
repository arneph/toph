package api

import (
	"fmt"
	"go/token"
	"os"

	"github.com/arneph/toph/builder"
	"github.com/arneph/toph/ir"
	irAnalyzer "github.com/arneph/toph/ir/analyzer"
	irOptimizer "github.com/arneph/toph/ir/optimizer"
	"github.com/arneph/toph/translator"
	"github.com/arneph/toph/uppaal"
	uppaalOptimizer "github.com/arneph/toph/uppaal/optimizer"
)

// BuilderConfig is a wrapper type for builder.Config to avoid naming collisions
type BuilderConfig = builder.Config

// TranslatorConfig is a wrapper type for translator.Config to avoid naming collisions
type TranslatorConfig = translator.Config

// Config holds paramters for the Run function.
type Config struct {
	BuilderConfig
	TranslatorConfig

	// OptimizeUppaalSystem indicates if the Uppaal optimizer should be run
	// before the final output gets generated.
	OptimizeUppaalSystem bool

	// Debug indicates if debug output files should be generated.
	Debug bool

	// OutName is the file name of all output files.
	OutName string
	// OutFormats lists the generated output file formats (supports xml, xta, ugi, q)
	OutFormats map[string]bool
}

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
func Run(path string, config Config) Result {
	warnings := false

	// Builder
	program, entryFuncs, errs := builder.BuildProgram(path, &config.BuilderConfig)
	warnings = warnings || len(errs) > 0
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if program == nil {
		return RunFailedWithBuilder
	} else if len(entryFuncs) == 0 {
		fmt.Fprintf(os.Stderr, "found no entry functions (main or tests)\n")
	}

	if config.Debug {
		outputIRProgram(program, config.OutName, "init")
	}

	if config.TranslatorConfig.OptimizeIR {
		// Dead Code Eliminator
		irOptimizer.EliminateDeadCode(program)

		if config.Debug {
			outputIRProgram(program, config.OutName, "opt")
		}
	}

	initStmts := program.InitFunc().Body().Stmts()
	for _, entryFunc := range entryFuncs {
		callStmt := ir.NewCallStmt(entryFunc, entryFunc.Signature(), ir.Call, token.NoPos, token.NoPos)
		program.InitFunc().Body().SetStmts(initStmts)
		program.InitFunc().Body().AddStmt(callStmt)

		// Translator
		sys, errs := translator.TranslateProg(program, &config.TranslatorConfig)
		warnings = warnings || len(errs) > 0
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		if sys == nil {
			return RunFailedWithTranslator
		}

		outName := config.OutName
		if len(entryFuncs) > 0 {
			outName += "_" + entryFunc.Handle()
		}

		if config.OptimizeUppaalSystem {
			ok := outputUppaalSystem(sys, outName+".init", config.OutFormats)
			if !ok {
				return RunFailedWritingOutputFiles
			}

			uppaalOptimizer.ReduceStates(sys)

			ok = outputUppaalSystem(sys, outName+".opt", config.OutFormats)
			if !ok {
				return RunFailedWritingOutputFiles
			}
		} else {
			ok := outputUppaalSystem(sys, outName, config.OutFormats)
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

func outputIRProgram(program *ir.Program, outName string, stepName string) {
	fcg := irAnalyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go)
	tg := irAnalyzer.BuildTypeGraph(program)

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
