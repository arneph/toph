package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"strings"

	"github.com/arneph/toph/api"
	"github.com/arneph/toph/builder"
	"github.com/arneph/toph/translator"
)

var (
	goos   = flag.String("goos", build.Default.GOOS, "target operating system, e.g. windows, linux")
	goarch = flag.String("goarch", build.Default.GOARCH, "target architecture, e.g. 386, amd64")

	debug = flag.Bool("debug", false, "generate debug output files")

	maxProcessCount   = flag.Int("max-processes", 100, "set maximum number of function process instances in Uppaal")
	maxDeferCount     = flag.Int("max-defers", 100, "set maximum number of deferred function calls per function process instance in Uppaal")
	maxChannelCount   = flag.Int("max-channels", 100, "set maximum number of channels in Uppaal")
	maxMutexCount     = flag.Int("max-mutexes", 100, "set maximum number of sync.Mutexes and sync.RWMutexes in Uppaal")
	maxWaitGroupCount = flag.Int("max-wait-groups", 100, "set maximum number of sync.WaitGroups in Uppaal")
	maxStructCount    = flag.Int("max-structs", 100, "set maximum number of struct instances (per defined struct) in Uppaal")

	optimize = flag.Bool("optimize", true, "optimize program")

	outName    = flag.String("out", "a", "set name out output files")
	outFormats = flag.String("out-formats", "xml", "set comma separated, generated output file formats, supports: xml, xta, ugi, q")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: toph [flags] [package directory]\n\n")
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
	path := flag.Arg(0)
	ffmts := make(map[string]bool)
	for _, ffmt := range strings.Split(*outFormats, ",") {
		ffmts[ffmt] = true
	}
	config := api.Config{
		BuilderConfig: builder.BuilderConfig{
			BuildContext: &buildContext,
		},
		TranslatorConfig: translator.TranslatorConfig{
			MaxProcessCount:   *maxProcessCount,
			MaxDeferCount:     *maxDeferCount,
			MaxChannelCount:   *maxChannelCount,
			MaxMutexCount:     *maxMutexCount,
			MaxWaitGroupCount: *maxWaitGroupCount,
			MaxStructCount:    *maxStructCount,
			Optimize:          *optimize,
		},
		Debug:      *debug,
		OutName:    *outName,
		OutFormats: ffmts,
	}
	result := api.Run(path, config)

	os.Exit(int(result))
}
