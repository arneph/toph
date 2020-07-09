package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func uppaalDir() string {
	switch goos := build.Default.GOOS; goos {
	case "windows":
		return "bin-Windows"
	case "darwin":
		return "bin-Darwin"
	case "linux":
		return "bin-Linux"
	default:
		return ""
	}
}

var (
	uppaalPath     = flag.String("uppaal", uppaalDir()+"/", "path to bin-Windows, bin-Darwin, or bin-Linux Uppaal directory")
	uppaalFlags    = flag.String("uppaal-flags", "-o0 -s -q", "flags for the Uppaal verifier")
	uppaalProcesss = flag.Int("uppaal-processes", runtime.GOMAXPROCS(0), "number of parallel Uppaal verifier processes")

	shuffleSystems = flag.Bool("shuffle", false, "verify Uppaal systems in random order")

	detailedResult  = flag.Bool("details", true, "list detailed query results for each Uppaal system")
	onlyNotSatified = flag.Bool("only-not-satisfied", false, "exclude satisfied queries from results")
)

var mu sync.RWMutex
var completedSystems int
var runningSystems map[string]struct{} = make(map[string]struct{})
var ctx, cancel = context.WithCancel(context.Background())

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: uppaal-runner [flags] [Uppaal system xml files]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		return
	}

	var wg sync.WaitGroup
	sysChan := make(chan string, *uppaalProcesss)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		cancel()
	}()
	for i := 0; i < *uppaalProcesss; i++ {
		wg.Add(1)
		go func() {
			for system := range sysChan {
				runSystem(system)
			}
			wg.Done()
		}()
	}
	systems := flag.Args()
	if *shuffleSystems {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(systems), func(i, j int) {
			systems[i], systems[j] = systems[j], systems[i]
		})
	}
loop:
	for _, system := range systems {
		select {
		case sysChan <- system:
		case <-ctx.Done():
			break loop
		}
	}
	close(sysChan)
	wg.Wait()

	print("done", true)
}

func runSystem(systemPath string) {
	if !strings.HasSuffix(systemPath, ".xml") {
		print("expected xml file: "+systemPath, true)
		return
	}
	systemDir, systemFileName := filepath.Split(systemPath)
	systemName := systemFileName[:len(systemFileName)-4]

	mu.Lock()
	runningSystems[systemName] = struct{}{}
	mu.Unlock()
	printRunningSystems()

	var outBuffer bytes.Buffer
	outPath := filepath.Join(systemDir, systemName+".out.txt")
	errPath := filepath.Join(systemDir, systemName+".err.txt")
	outFile, err := os.Create(outPath)
	if err != nil {
		print(fmt.Sprint(err), true)
		return
	}
	defer outFile.Close()
	errFile, err := os.Create(errPath)
	if err != nil {
		print(fmt.Sprint(err), true)
		return
	}
	defer errFile.Close()

	uppaalArgs := strings.Split(*uppaalFlags, " ")
	uppaalArgs = append(uppaalArgs, systemPath)
	cmd := exec.CommandContext(ctx, filepath.Join(*uppaalPath, "verifyta"), uppaalArgs...)
	cmd.Stdin = strings.NewReader("")
	cmd.Stdout = io.MultiWriter(outFile, &outBuffer)
	cmd.Stderr = errFile
	t1 := time.Now()
	err = cmd.Run()
	t2 := time.Now()

	mu.Lock()
	completedSystems++
	systemIndex := completedSystems
	var result string
	if err != nil {
		result = fmt.Sprintf("%03d \x1b[31mfailed\x1b[0m    % 12.1fs %s", systemIndex, t2.Sub(t1).Seconds(), systemPath)
	} else {
		result = fmt.Sprintf("%03d \x1b[32mcompleted\x1b[0m % 12.1fs %s", systemIndex, t2.Sub(t1).Seconds(), systemPath)
	}
	print(result, true)
	if err != nil {
		print(fmt.Sprint(err), true)
	} else if *detailedResult {
		details := generateDetailedResults(systemPath, outBuffer.String())
		if details != "" {
			print(details, true)
		}
	}
	delete(runningSystems, systemName)
	mu.Unlock()
	printRunningSystems()
}

func generateDetailedResults(systemPath string, outString string) string {
	systemBytes, err := ioutil.ReadFile(systemPath)
	if err != nil {
		print(fmt.Sprint(err), true)
		return ""
	}
	systemString := string(systemBytes)
	queryStrings := regexp.MustCompile("(?s:<query>.*?</query>)").FindAllString(systemString, -1)
	type query struct {
		index          int
		formula        string
		description    string
		sourceLocation string
		category       string
		satisfied      bool
	}
	queries := make([]*query, len(queryStrings))
	categories := make(map[string][]*query)
	for i, queryString := range queryStrings {
		formula := regexp.MustCompile("(?s:<formula>.*?</formula>)").FindString(queryString)
		formula = strings.TrimPrefix(formula, "<formula>")
		formula = strings.TrimSuffix(formula, "</formula>")
		formula = strings.ReplaceAll(formula, "&gt;", ">")
		comment := regexp.MustCompile("(?s:<comment>.*?</comment>)").FindString(queryString)
		comment = strings.TrimPrefix(comment, "<comment>")
		comment = strings.TrimSuffix(comment, "</comment>")
		var description, sourceLocation, category string
		for _, line := range strings.Split(comment, "\n") {
			if strings.HasPrefix(line, "description: ") {
				description = strings.TrimPrefix(line, "description: ")
			} else if strings.HasPrefix(line, "location: ") {
				sourceLocation = strings.TrimPrefix(line, "location: ")
			} else if strings.HasPrefix(line, "category: ") {
				category = strings.TrimPrefix(line, "category: ")
			}
		}
		queries[i] = &query{
			index:          i + 1,
			formula:        formula,
			description:    description,
			sourceLocation: sourceLocation,
			category:       category,
		}
		categories[category] = append(categories[category], queries[i])
	}

	lines := strings.Split(outString, "\n")
	for i := 0; i < len(lines)-1; i++ {
		lineA := lines[i]
		lineB := lines[i+1]
		if !strings.HasPrefix(lineA, "Verifying formula") ||
			!strings.HasPrefix(lineB, " -- Formula is") {
			continue
		}
		formulaIndexString := strings.Split(lineA, " ")[2]
		formulaIndex, err := strconv.Atoi(formulaIndexString)
		if err != nil {
			continue
		}
		formulaResultString := strings.Split(lineB, " ")[4]
		formulaResult := formulaResultString == "satisfied."

		queries[formulaIndex-1].satisfied = formulaResult
	}

	var b strings.Builder
	firstCategory := true
	for category, allQueries := range categories {
		queries := allQueries
		if *onlyNotSatified {
			queries = nil
			for _, query := range allQueries {
				if !query.satisfied {
					queries = append(queries, query)
				}
			}
		}
		if len(queries) == 0 {
			continue
		}
		allSatisfied := true
		for _, query := range queries {
			if !query.satisfied {
				allSatisfied = false
				break
			}
		}
		if !firstCategory {
			fmt.Fprintln(&b)
		} else {
			firstCategory = false
		}
		if allSatisfied {
			fmt.Fprintf(&b, "    \x1b[32msatisfied:\x1b[0m %s", category)
		} else {
			fmt.Fprintf(&b, "\x1b[31mnot satisfied:\x1b[0m %s", category)
		}

		for _, query := range queries {
			fmt.Fprintln(&b)
			if query.satisfied {
				fmt.Fprintf(&b, "\t%03d     \x1b[32msatisfied:\x1b[0m %s", query.index, query.formula)
			} else {
				fmt.Fprintf(&b, "\t%03d \x1b[31mnot satisfied:\x1b[0m %s", query.index, query.formula)
			}
			if query.sourceLocation != "" {
				fmt.Fprintf(&b, "\n\t                   %s", query.sourceLocation)
			}
		}
	}

	return b.String()
}

func printRunningSystems() {
	status := "\rrunning: "
	first := true
	mu.RLock()
	for system := range runningSystems {
		if !first {
			status += ", "
		} else {
			first = false
		}
		status += system
	}
	mu.RUnlock()
	if len(status) > 100 {
		status = status[:97] + "..."
	}
	print(status, false)
}

func print(line string, newline bool) {
	n := "100"
	if strings.Contains(line, "\x1b") {
		n = "109"
	}
	if newline {
		fmt.Printf("\r%-"+n+"s\n", line)
	} else {
		fmt.Printf("\r%-"+n+"s", line)
	}
}
