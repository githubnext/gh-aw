//go:build !integration

package lenstringzero_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/lenstringzero"
)

func TestLenStringZero(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, lenstringzero.Analyzer, "lenstringzero")
}
