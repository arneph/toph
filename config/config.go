package config

import (
	"go/build"
)

// Config holds paramters for the Run function.
type Config struct {
	BuildContext *build.Context
	PackageExcludeInfo

	MaxProcessCount   int
	MaxDeferCount     int
	MaxChannelCount   int
	MaxMutexCount     int
	MaxWaitGroupCount int
	MaxOnceCount      int
	MaxStructCount    int
	MaxContainerCount int
	ContainerCapacity int

	GenerateResourceBoundQueries            bool
	GenerateIndividualResourceBoundQueries  bool
	GenerateChannelSafetyQueries            bool
	GenerateMutexSafetyQueries              bool
	GenerateWaitGroupSafetyQueries          bool
	GenerateChannelRelatedDeadlockQueries   bool
	GenerateMutexRelatedDeadlockQueries     bool
	GenerateWaitGroupRelatedDeadlockQueries bool
	GenerateOnceRelatedDeadlockQueries      bool
	GenerateFunctionCallsWithNilQueries     bool
	GenerateGoroutineExitWithPanicQueries   bool
	GenerateReachabilityQueries             bool

	OptimizeIR           bool
	OptimizeUppaalSystem bool

	// Debug indicates if debug output files should be generated.
	Debug bool

	// OutName is the file name of all output files.
	OutName string
	// OutFormats lists the generated output file formats (supports xml, xta, ugi, q)
	OutFormats map[string]bool
}

// PackageExcludeInfo stores which packages and members of packages should be excluded from translation.
type PackageExcludeInfo map[string]struct{}

func (pei *PackageExcludeInfo) ShouldExcludeEntirePackage(packagePath string) bool {
	if (*pei) == nil {
		return false
	}
	_, ok := (*pei)[packagePath]
	return ok
}

func (pei *PackageExcludeInfo) SetExcludeEntirePackage(packagePath string) {
	if (*pei) == nil {
		*pei = make(PackageExcludeInfo)
	}
	(*pei)[packagePath] = struct{}{}
}
