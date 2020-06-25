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
var testIndex int
var runningTests map[string]struct{} = make(map[string]struct{})
var wg sync.WaitGroup
var mu sync.RWMutex

func main() {
	var requiredSubString string
	if len(os.Args) > 1 {
		requiredSubString = os.Args[1]
	}

	dirs, err := ioutil.ReadDir("tests/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read 'tests/' dir: %v", err)
		return
	}

	for i := 0; i < runtime.GOMAXPROCS(120); i++ {
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
			if !strings.Contains(testPath, requiredSubString) {
				continue
			}
			testFiles, err := ioutil.ReadDir(testPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not read %q dir: %v", dirPath, err)
				continue
			}
			for _, testFile := range testFiles {
				name := testFile.Name()
				if testFile.IsDir() || !strings.HasSuffix(name, ".xml") {
					continue
				}
				testsChan <- testPath + name[:len(name)-4]
			}
		}
	}
	close(testsChan)
	wg.Wait()

	fmt.Println("\rdone")
}

func runTests() {
	for test := range testsChan {
		mu.Lock()
		runningTests[test[strings.LastIndex(test, "/")+1:]] = struct{}{}
		mu.Unlock()
		printRunning()
		xml := test + ".xml"
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

		cmd := exec.Command(uppaalPath+"verifyta", "-o0", "-s", "-q", "-t1", xml)
		cmd.Stdin = strings.NewReader("")
		cmd.Stdout = outG
		cmd.Stderr = errG
		t1 := time.Now()
		err = cmd.Run()
		t2 := time.Now()
		testIndex++
		fmt.Printf("\r%100s", "")
		if err != nil {
			fmt.Printf("\r%03d failed    % 12.1fs %s\n", testIndex, t2.Sub(t1).Seconds(), xml[6:])
		} else {
			fmt.Printf("\r%03d completed % 12.1fs %s\n", testIndex, t2.Sub(t1).Seconds(), xml[6:])
		}
		mu.Lock()
		delete(runningTests, test[strings.LastIndex(test, "/")+1:])
		mu.Unlock()
	}
	printRunning()
	wg.Done()
}

func printRunning() {
	s := "\rrunning: "
	first := true
	mu.RLock()
	for test := range runningTests {
		if !first {
			s += ", "
		} else {
			first = false
		}
		s += test
	}
	mu.RUnlock()
	if len(s) > 100 {
		s = s[:97] + "..."
	}
	fmt.Print(s)
}
