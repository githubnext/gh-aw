//go:build !integration

// Package sprintferrorsnew_test provides tests for the sprintferrorsnew analyzer.
package sprintferrorsnew_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/sprintferrorsnew"
)

func TestSprintfErrorsNew(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, sprintferrorsnew.Analyzer, "sprintferrorsnew")
}
