package main

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"strings"

	"github.com/arneph/toph/api"
	"github.com/arneph/toph/builder"
	"github.com/arneph/toph/translator"
)

func ignore(info os.FileInfo) bool {
	return !info.IsDir() ||
		strings.HasPrefix(info.Name(), ".") ||
		strings.HasPrefix(info.Name(), "_")
}

func main() {
	var requiredSubString string
	if len(os.Args) > 1 {
		requiredSubString = os.Args[1]
	}

	buildContext := build.Default
	buildContext.GOOS = "linux"

	dirs, err := ioutil.ReadDir("tests/")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not read 'tests/' dir: %v", err)
		return
	}

	attemptedTests := 0
	perfectTests := 0
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
			if !strings.Contains(testPath, requiredSubString) {
				continue
			}
			fmt.Printf("running test: %s\n", testPath)
			config := api.Config{
				BuilderConfig: builder.BuilderConfig{
					BuildContext: &buildContext,
				},
				TranslatorConfig: translator.TranslatorConfig{
					MaxProcessCount:   10,
					MaxDeferCount:     10,
					MaxChannelCount:   100,
					MaxMutexCount:     100,
					MaxWaitGroupCount: 100,
					MaxStructCount:    100,
					Optimize:          true,
				},
				Debug:      true,
				OutName:    testPath + test.Name(),
				OutFormats: map[string]bool{"xml": true},
			}
			result := api.Run(testPath, config)
			perfect := result == api.RunSuccessful
			attemptedTests++
			if perfect {
				perfectTests++
			}
		}
	}
	fmt.Printf("%d/%d tests ran without warnings\n", perfectTests, attemptedTests)
	fmt.Println("done")
}
