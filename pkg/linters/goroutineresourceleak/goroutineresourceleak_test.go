//go:build !integration

// Package goroutineresourceleak_test provides tests for the goroutineresourceleak analyzer.
package goroutineresourceleak_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/goroutineresourceleak"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	analysistest.Run(t, analysistest.TestData(), goroutineresourceleak.Analyzer, "a")
}
