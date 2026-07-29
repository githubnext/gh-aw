//go:build !integration

// Package goroutinemissingrecover_test provides tests for the goroutinemissingrecover analyzer.
package goroutinemissingrecover_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/goroutinemissingrecover"
)

func TestGoroutineMissingRecover(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), goroutinemissingrecover.Analyzer, "a")
}
