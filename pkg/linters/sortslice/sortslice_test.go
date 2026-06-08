//go:build !integration

package sortslice_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/sortslice"
)

func TestSortSlice(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, sortslice.Analyzer, "sortslice")
}
