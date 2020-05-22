package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const uppaalPath string = "test-runners/bin-Darwin/"

func ignore(info os.FileInfo) bool {
	return !info.IsDir() ||
		strings.HasPrefix(info.Name(), ".") ||
		strings.HasPrefix(info.Name(), "_")
}

var testsChan chan string = make(chan string, 10)
var wg sync.WaitGroup

func main() {
	dirs, err := ioutil.ReadDir("tests/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read 'tests/' dir: %v", err)
		return
	}

	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go runTests()
	}

	//rand.Seed(time.Now().UnixNano())
	//rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
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

		//rand.Seed(time.Now().UnixNano())
		//rand.Shuffle(len(tests), func(i, j int) { tests[i], tests[j] = tests[j], tests[i] })
		for _, test := range tests {
			if ignore(test) {
				continue
			}

			testPath := dirPath + test.Name() + "/"
			testsChan <- testPath + test.Name()
		}
	}
	close(testsChan)
	wg.Wait()

	fmt.Println("done")
}

var z int

func runTests() {
	for test := range testsChan {
		xta := test + ".xta"
		q := test + ".q"
		outF := test + ".out.txt"
		errF := test + ".err.txt"
		outG, err := os.Create(outF)
		if err != nil {
			fmt.Println(err)
			continue
		}
		defer outG.Close()
		errG, err := os.Create(errF)
		if err != nil {
			fmt.Println(err)
			continue
		}
		defer errG.Close()

		cmd := exec.Command(uppaalPath+"verifyta", "-o0", "-s", "-q", "-t1", xta, q)
		cmd.Stdin = strings.NewReader("")
		cmd.Stdout = outG
		cmd.Stderr = errG
		t1 := time.Now()
		err = cmd.Run()
		t2 := time.Now()
		z++
		if err != nil {
			fmt.Printf("%03d failed    % 12.1fs %s\n", z, t2.Sub(t1).Seconds(), xta[6:])
		} else {
			fmt.Printf("%03d completed % 12.1fs %s\n", z, t2.Sub(t1).Seconds(), xta[6:])
		}
	}
	wg.Done()
}
