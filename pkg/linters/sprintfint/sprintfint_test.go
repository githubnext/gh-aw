//go:build !integration

// Package sprintfint_test provides tests for the sprintfint analyzer.
package sprintfint_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/sprintfint"
)

func TestSprintfInt(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, sprintfint.Analyzer, "sprintfint")
}
