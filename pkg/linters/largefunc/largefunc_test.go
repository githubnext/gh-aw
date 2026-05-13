//go:build !integration

// Package largefunc_test provides tests for the largefunc analyzer.
package largefunc_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/largefunc"
)

func TestLargeFunc(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), largefunc.Analyzer, "a")
}
