//go:build !integration

package manualpathconcat_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/manualpathconcat"
)

func TestManualPathConcat(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, manualpathconcat.Analyzer, "manualpathconcat")
}
