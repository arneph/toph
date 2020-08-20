package main

import (
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"strings"

	"github.com/arneph/toph/api"
	c "github.com/arneph/toph/config"
)

var (
	goos   = flag.String("goos", build.Default.GOOS, "target operating system, e.g. windows, linux")
	goarch = flag.String("goarch", build.Default.GOARCH, "target architecture, e.g. 386, amd64")

	excludeFile = flag.String("exclude", "", "set file containing a list of packages to exclude from translation")

	debug = flag.Bool("debug", false, "generate debug output files")

	maxProcessCount   = flag.Int("max-processes", 10, "set maximum number of function process instances in Uppaal")
	maxDeferCount     = flag.Int("max-defers", 10, "set maximum number of deferred function calls per function process instance in Uppaal")
	maxChannelCount   = flag.Int("max-channels", 20, "set maximum number of channels in Uppaal")
	maxMutexCount     = flag.Int("max-mutexes", 20, "set maximum number of sync.Mutexes and sync.RWMutexes in Uppaal")
	maxWaitGroupCount = flag.Int("max-wait-groups", 20, "set maximum number of sync.WaitGroups in Uppaal")
	maxOnceCount      = flag.Int("max-once", 20, "set maximum number of sync.Once in Uppaal")
	maxStructCount    = flag.Int("max-structs", 20, "set maximum number of struct instances (per defined struct) in Uppaal")
	maxContainerCount = flag.Int("max-containers", 20, "set maximum number of array, slice, or map instances (per element type) in Uppaal")

	containerCapacity = flag.Int("container-capacity", 5, "set the constant capacity of arrays, slices, and maps in Uppaal")

	optimizeIR     = flag.Bool("optimize-ir", true, "optimize intermediate representation of program")
	optimizeSystem = flag.Bool("optimize-sys", true, "optimize uppaal system")

	outName    = flag.String("out", "a", "set name out output files")
	outFormats = flag.String("out-formats", "xml", "set comma separated, generated output file formats, supports: xml, xta, ugi, q")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: toph [flags] [package directories]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		return
	}
	buildContext := build.Default
	buildContext.GOOS = *goos
	buildContext.GOARCH = *goarch
	ffmts := make(map[string]bool)
	for _, ffmt := range strings.Split(*outFormats, ",") {
		ffmts[ffmt] = true
	}
	config := c.Config{
		BuildContext:         &buildContext,
		MaxProcessCount:      *maxProcessCount,
		MaxDeferCount:        *maxDeferCount,
		MaxChannelCount:      *maxChannelCount,
		MaxMutexCount:        *maxMutexCount,
		MaxWaitGroupCount:    *maxWaitGroupCount,
		MaxOnceCount:         *maxOnceCount,
		MaxStructCount:       *maxStructCount,
		MaxContainerCount:    *maxContainerCount,
		ContainerCapacity:    *containerCapacity,
		OptimizeIR:           *optimizeIR,
		OptimizeUppaalSystem: *optimizeSystem,
		Debug:                *debug,
		OutName:              *outName,
		OutFormats:           ffmts,
	}
	if *excludeFile != "" {
		content, err := ioutil.ReadFile(*excludeFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read exclude file: %v", err)
			os.Exit(-1)
		}
		lines := strings.Split(string(content), "\n")
		for _, packagePath := range lines {
			if packagePath == "" {
				continue
			}
			config.SetExcludeEntirePackage(packagePath)
		}
	}

	result := api.Run(flag.Args(), &config)

	os.Exit(int(result))
}
