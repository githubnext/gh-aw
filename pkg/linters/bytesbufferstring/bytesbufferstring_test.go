//go:build !integration

package bytesbufferstring_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/bytesbufferstring"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, bytesbufferstring.Analyzer, "bytesbufferstring")
}
