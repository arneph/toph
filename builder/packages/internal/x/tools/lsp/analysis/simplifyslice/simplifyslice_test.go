// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package simplifyslice_test

import (
	"testing"

	"github.com/arneph/toph/builder/packages/internal/x/tools/lsp/analysis/simplifyslice"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, simplifyslice.Analyzer, "a")
}
