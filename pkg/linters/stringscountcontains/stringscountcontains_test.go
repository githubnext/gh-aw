//go:build !integration

package stringscountcontains_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/stringscountcontains"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, stringscountcontains.Analyzer, "stringscountcontains")
}
