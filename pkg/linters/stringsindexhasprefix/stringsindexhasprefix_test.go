//go:build !integration

package stringsindexhasprefix_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/stringsindexhasprefix"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, stringsindexhasprefix.Analyzer, "stringsindexhasprefix")
}
