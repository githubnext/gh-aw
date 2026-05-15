//go:build !integration

// Package excessivefuncparams_test provides tests for the excessivefuncparams analyzer.
package excessivefuncparams_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/excessivefuncparams"
)

func TestExcessiveFuncParams(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), excessivefuncparams.Analyzer, "a")
}
