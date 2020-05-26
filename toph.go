package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"

	"github.com/arneph/toph/api"
)

var entryFuncName = flag.String("entry", "main", "name of program entry function")
var goos = flag.String("goos", build.Default.GOOS, "target operating system, e.g. windows, linux")
var goarch = flag.String("goarch", build.Default.GOARCH, "target architecture, e.g. 386, amd64")
var debug = flag.Bool("debug", false, "generate debug output files")
var optimize = flag.Bool("optimize", true, "optimize program")
var outName = flag.String("out", "a", "set name out output files")

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
	build.Default.GOOS = *goos
	build.Default.GOARCH = *goarch
	path := flag.Arg(0)
	config := api.Config{
		EntryFuncName: *entryFuncName,
		Debug:         *debug,
		Optimize:      *optimize,
		OutName:       *outName,
	}
	result := api.Run(path, config)

	os.Exit(int(result))
}
