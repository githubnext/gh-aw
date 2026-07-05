//go:build !integration

package stringreplaceminusone_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/stringreplaceminusone"
)

func TestStringReplaceMinusOne(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, stringreplaceminusone.Analyzer, "stringreplaceminusone")
}
