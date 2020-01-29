package main

import (
	"fmt"
	"os"
)

func main() {
	for i := 1; i <= 3; i++ {
		path := fmt.Sprintf("tests/test%d/", i)
		fmt.Printf("running test %d\n", i)

		prog, err := buildProg(path)
		if err != nil {
			fmt.Printf("could not parse input: %v\n", err)
		}

		progFile, err := os.Create(path + "prog.ir.txt")
		if err != nil {
			fmt.Printf("could not write tprog.txt file: %v", err)
		} else {
			fmt.Fprintln(progFile, prog)
		}

		sys, err := translateProg(prog)
		if err != nil {
			fmt.Printf("could not translate to xta: %v\n", err)
		}

		sysTxtFile, err := os.Create(path + "sys.xta.txt")
		if err != nil {
			fmt.Printf("could not write tprog.txt file: %v", err)
		} else {
			fmt.Fprintln(sysTxtFile, sys)
		}

		sysXtaFile, err := os.Create(path + "sys.xta")
		if err != nil {
			fmt.Printf("could not write tprog.xta file: %v", err)
		} else {
			fmt.Fprintln(sysXtaFile, sys)
		}
	}
	fmt.Println("done")
}
