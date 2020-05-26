package api

import (
	"fmt"
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
	program, errs := builder.BuildProgram(path, config.EntryFuncName)
	warnings = warnings || len(errs) > 0
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if program == nil {
		return RunFailedWithBuilder
	}

	if config.Debug {
		outputProgram(program, path, config.OutName, 1)
	}

	if config.Optimize {
		// Dead Code Eliminator
		optimizer.EliminateDeadCode(program)

		if config.Debug {
			outputProgram(program, path, config.OutName, 2)
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

		sysFile, err := os.Create(path + "/" + config.OutName + "." + ffmt)
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
func outputProgram(program *ir.Program, path, name string, index int) {
	callFCG := analyzer.BuildFuncCallGraph(program, ir.Call)
	goFCG := analyzer.BuildFuncCallGraph(program, ir.Go)

	// IR file
	programPath := fmt.Sprintf("%s/%s.%d.ir.txt", path, name, index)
	programFile, err := os.Create(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write ir.txt file: %v\n", err)
		return
	}
	defer programFile.Close()

	fmt.Fprintln(programFile, program.String())

	// FCG files
	callFCGPath := fmt.Sprintf("%s/%s.%d.call_fcg.txt", path, name, index)
	callFCGFile, err := os.Create(callFCGPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write call_fcg.txt file: %v\n", err)
		return
	}
	defer callFCGFile.Close()

	fmt.Fprintln(callFCGFile, callFCG.String())

	goFCGPath := fmt.Sprintf("%s/%s.%d.go_fcg.txt", path, name, index)
	goFCGFile, err := os.Create(goFCGPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write go_fcg.txt file: %v\n", err)
		return
	}
	defer goFCGFile.Close()

	fmt.Fprintln(goFCGFile, goFCG.String())
}
