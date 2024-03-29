// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/cmd"
	cmdtest "github.com/arneph/toph/builder/packages/internal/x/tools/lsp/cmd/test"
	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/tests"
	"github.com/arneph/toph/builder/packages/internal/x/tools/testenv"
	"golang.org/x/tools/go/packages/packagestest"
)

func TestMain(m *testing.M) {
	testenv.ExitIfSmallMachine()
	os.Exit(m.Run())
}

func TestCommandLine(t *testing.T) {
	packagestest.TestAll(t,
		cmdtest.TestCommandLine(
			"../testdata",
			nil,
		),
	)
}

func TestDefinitionHelpExample(t *testing.T) {
	// TODO: https://golang.org/issue/32794.
	t.Skip()
	if runtime.GOOS == "android" {
		t.Skip("not all source files are available on android")
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("could not get wd: %v", err)
		return
	}
	ctx := tests.Context(t)
	ts := cmdtest.NewTestServer(ctx, nil)
	thisFile := filepath.Join(dir, "definition.go")
	baseArgs := []string{"query", "definition"}
	expect := regexp.MustCompile(`(?s)^[\w/\\:_-]+flag[/\\]flag.go:\d+:\d+-\d+: defined here as FlagSet struct {.*}$`)
	for _, query := range []string{
		fmt.Sprintf("%v:%v:%v", thisFile, cmd.ExampleLine, cmd.ExampleColumn),
		fmt.Sprintf("%v:#%v", thisFile, cmd.ExampleOffset)} {
		args := append(baseArgs, query)
		r := cmdtest.NewRunner(nil, nil, ctx, ts.Addr, nil)
		got, _ := r.NormalizeGoplsCmd(t, args...)
		if !expect.MatchString(got) {
			t.Errorf("test with %v\nexpected:\n%s\ngot:\n%s", args, expect, got)
		}
	}
}
