package config

import "go/build"

// Config holds paramters for the Run function.
type Config struct {
	BuildContext *build.Context

	MaxProcessCount   int
	MaxDeferCount     int
	MaxChannelCount   int
	MaxMutexCount     int
	MaxWaitGroupCount int
	MaxStructCount    int
	MaxContainerCount int
	ContainerCapacity int

	OptimizeIR           bool
	OptimizeUppaalSystem bool

	// Debug indicates if debug output files should be generated.
	Debug bool

	// OutName is the file name of all output files.
	OutName string
	// OutFormats lists the generated output file formats (supports xml, xta, ugi, q)
	OutFormats map[string]bool
}
