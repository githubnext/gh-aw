//go:build !integration

package writebytestring_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/writebytestring"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, writebytestring.Analyzer, "writebytestring")
}
