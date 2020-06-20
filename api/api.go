package api

import (
	"fmt"
	"go/build"
	"os"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/builder"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/optimizer"
	"github.com/arneph/toph/translator"
)

// Config holds paramters for the Run function.
type Config struct {
	// EntryFuncName is the name of the entry function to the program.
	EntryFuncName string
	// BuildContext is the context used by the builder package to find packages
	// and files.
	BuildContext build.Context
	// Debug indicates if debug output files should be generated.
	Debug bool
	// Optimize indicates if the optimizer should be run on the IR.
	Optimize bool
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
	program, errs := builder.BuildProgram(path, config.EntryFuncName, config.BuildContext)
	warnings = warnings || len(errs) > 0
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if program == nil {
		return RunFailedWithBuilder
	}

	if config.Debug {
		outputProgram(program, config.OutName, "init")
	}

	if config.Optimize {
		// Dead Code Eliminator
		optimizer.EliminateDeadCode(program)
		optimizer.EliminateUnusedFunctions(program)

		if config.Debug {
			outputProgram(program, config.OutName, "opt")
		}
	}

	// Translator
	sys, errs := translator.TranslateProg(program)
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

		sysFile, err := os.Create(config.OutName + "." + ffmt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\tcould not write %s file: %v\n", ffmt, err)
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
