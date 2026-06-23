//go:build !integration

package lenstringsplit_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/lenstringsplit"
)

func TestLenStringSplit(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, lenstringsplit.Analyzer, "lenstringsplit")
}
