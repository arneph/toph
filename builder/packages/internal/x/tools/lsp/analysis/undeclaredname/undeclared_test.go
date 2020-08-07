// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package undeclaredname_test

import (
	"testing"

	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/analysis/undeclaredname"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, undeclaredname.Analyzer, "a")
}
