//go:build !integration

// Package sprintferrdot_test provides tests for the sprintferrdot analyzer.
package sprintferrdot_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/sprintferrdot"
)

func TestSprintfErrDot(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, sprintferrdot.Analyzer, "sprintferrdot")
}
