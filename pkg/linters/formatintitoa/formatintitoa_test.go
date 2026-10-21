//go:build !integration

package formatintitoa_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/formatintitoa"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), formatintitoa.Analyzer, "formatintitoa", "alias", "dotimport")
}
