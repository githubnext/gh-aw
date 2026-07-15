//go:build !integration

package mapdeletecheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/mapdeletecheck"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, mapdeletecheck.Analyzer, "mapdeletecheck")
}
