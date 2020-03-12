package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/arneph/toph/analyzer"
	"github.com/arneph/toph/builder"
	"github.com/arneph/toph/ir"
	"github.com/arneph/toph/optimizer"
	"github.com/arneph/toph/translator"
)

func ignore(info os.FileInfo) bool {
	return !info.IsDir() ||
		strings.HasPrefix(info.Name(), ".") ||
		strings.HasPrefix(info.Name(), "_")
}

func main() {
	dirs, err := ioutil.ReadDir("tests/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read 'tests/' dir: %v", err)
		return
	}

	for _, dir := range dirs {
		if ignore(dir) {
			continue
		}

		dirPath := "tests/" + dir.Name() + "/"
		tests, err := ioutil.ReadDir(dirPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not read %q dir: %v", dirPath, err)
			continue
		}

		for _, test := range tests {
			if ignore(test) {
				continue
			}

			testPath := dirPath + test.Name() + "/"
			runTest(testPath, test.Name())
		}
	}

	fmt.Println("done")
}

func runTest(path, name string) {
	fmt.Printf("running: %s\n", path)

	// Builder
	program, errs := builder.BuildProgram(path)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if program == nil {
		return
	}

	outputProgram(program, path, name, 1)

	// Inliner
	optimizer.InlineFuncCalls(program)

	outputProgram(program, path, name, 2)

	// Translator
	sys, errs := translator.TranslateProg(program)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if sys == nil {
		return
	}

	// XTA file
	sysXTAFile, err := os.Create(path + name + ".xta")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write xta file: %v\n", err)
		return
	}
	defer sysXTAFile.Close()

	fmt.Fprintln(sysXTAFile, sys.AsXTA())

	// UGI file
	sysUGIFile, err := os.Create(path + name + ".ugi")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write ugi file: %v\n", err)
		return
	}
	defer sysUGIFile.Close()

	fmt.Fprintln(sysUGIFile, sys.AsUGI())

	// Q file
	sysQFile, err := os.Create(path + name + ".q")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write q file: %v\n", err)
		return
	}
	defer sysQFile.Close()

	fmt.Fprintln(sysQFile, sys.AsQ())
}

func outputProgram(program *ir.Program, path, name string, index int) {
	mainFunc := program.GetFunc("main")
	callFCG := analyzer.CalculateFuncCallGraph(program, mainFunc, ir.Call)
	goFCG := analyzer.CalculateFuncCallGraph(program, mainFunc, ir.Go)

	// IR file
	programPath := fmt.Sprintf("%s%s.%d.ir.txt", path, name, index)
	programFile, err := os.Create(programPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write ir.txt file: %v\n", err)
		return
	}
	defer programFile.Close()

	fmt.Fprintln(programFile, program)

	// FCG files
	callFCGPath := fmt.Sprintf("%s%s.%d.call_fcg.txt", path, name, index)
	callFCGFile, err := os.Create(callFCGPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write call_fcg.txt file: %v\n", err)
		return
	}
	defer callFCGFile.Close()

	fmt.Fprintln(callFCGFile, callFCG)

	goFCGPath := fmt.Sprintf("%s%s.%d.go_fcg.txt", path, name, index)
	goFCGFile, err := os.Create(goFCGPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write go_fcg.txt file: %v\n", err)
		return
	}
	defer goFCGFile.Close()

	fmt.Fprintln(goFCGFile, goFCG)
}
