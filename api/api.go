package api

import (
	"fmt"
	"go/token"
	"os"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/builder"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/optimizer"
	"github.com/arneph/toph/translator"
)

// BuilderConfig is a wrapper type for builder.Config to avoid naming collisions
type BuilderConfig = builder.Config

// TranslatorConfig is a wrapper type for translator.Config to avoid naming collisions
type TranslatorConfig = translator.Config

// Config holds paramters for the Run function.
type Config struct {
	BuilderConfig
	TranslatorConfig

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
		outputProgram(program, config.OutName, "init")
	}

	if config.TranslatorConfig.Optimize {
		// Dead Code Eliminator
		optimizer.EliminateDeadCode(program)

		if config.Debug {
			outputProgram(program, config.OutName, "opt")
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

		// Output files
		for _, ffmt := range []string{"xml", "xta", "ugi", "q"} {
			if !config.OutFormats[ffmt] {
				continue
			}

			name := config.OutName
			if len(entryFuncs) > 1 {
				name += "_" + entryFunc.Handle()
			}
			sysFile, err := os.Create(name + "." + ffmt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not write %s file: %v\n", ffmt, err)
				return RunFailedWritingOutputFiles
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
	}

	if warnings {
		return RunSuccessfulButWithWarnings
	}
	return RunSuccessful
}

// outputProgram generates debug output files showing the IR at a certain
// point during the process.
func outputProgram(program *ir.Program, outName string, stepName string) {
	fcg := analyzer.BuildFuncCallGraph(program, ir.Call|ir.Defer|ir.Go)

	// IR file
	programPath := fmt.Sprintf("%s.%s.ir.txt", outName, stepName)
	programFile, err := os.Create(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write ir.txt file: %v\n", err)
		return
	}
	defer programFile.Close()

	programFile.WriteString(program.Tree())

	// FCG files
	fcgPath := fmt.Sprintf("./%s.%s.fcg.txt", outName, stepName)
	fcgFile, err := os.Create(fcgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write call_fcg.txt file: %v\n", err)
		return
	}
	defer fcgFile.Close()

	fcgFile.WriteString(fcg.String())
}
