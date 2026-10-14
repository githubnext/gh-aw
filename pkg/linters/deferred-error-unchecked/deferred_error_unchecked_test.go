//go:build !integration

// Package deferrerrorunchecked_test provides tests for the deferrerrorunchecked analyzer.
package deferrerrorunchecked_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	deferrerrorunchecked "github.com/github/gh-aw/pkg/linters/deferred-error-unchecked"
)

func TestDeferredErrorUnchecked(t *testing.T) {
	t.Parallel()
	analysistest.Run(t, analysistest.TestData(), deferrerrorunchecked.Analyzer, "deferred_error_unchecked")
}
