//go:build !integration

// Package sprintfbool_test provides tests for the sprintfbool analyzer.
package sprintfbool_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/sprintfbool"
)

func TestSprintfBool(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, sprintfbool.Analyzer, "sprintfbool")
}
