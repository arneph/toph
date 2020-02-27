package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/arneph/toph/builder"
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
			runTest(testPath)
		}
	}

	fmt.Println("done")
}

func runTest(path string) {
	fmt.Printf("running: %s\n", path)
	defer fmt.Fprintln(os.Stderr)

	program, errs := builder.BuildProgram(path)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if program == nil {
		return
	}

	programFile, err := os.Create(path + "program.ir.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not write tprog.txt file: %v\n", err)
	}
	defer programFile.Close()

	fmt.Fprintln(programFile, program)

	sys, errs := translator.TranslateProg(program)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	if sys == nil {
		return
	}

	sysTxtFile, err := os.Create(path + "sys.xta.txt")
	if err != nil {
		fmt.Printf("could not write tprog.txt file: %v", err)
	}
	defer sysTxtFile.Close()

	fmt.Fprintln(sysTxtFile, sys)

	sysXtaFile, err := os.Create(path + "sys.xta")
	if err != nil {
		fmt.Printf("could not write tprog.xta file: %v", err)
	}
	defer sysXtaFile.Close()

	fmt.Fprintln(sysXtaFile, sys)
}
