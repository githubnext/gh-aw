//go:build !integration

package execcommandwithoutcontext_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/execcommandwithoutcontext"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, execcommandwithoutcontext.Analyzer, "execcommandwithoutcontext")
}
