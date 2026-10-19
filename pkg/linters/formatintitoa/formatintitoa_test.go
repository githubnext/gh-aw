//go:build !integration

// Package formatintitoa_test provides tests for the formatintitoa analyzer.
package formatintitoa_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/formatintitoa"
)

func TestFormatIntItoa(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, formatintitoa.Analyzer, "formatintitoa")
}
