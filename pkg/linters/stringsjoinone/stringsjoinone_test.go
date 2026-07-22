//go:build !integration

package stringsjoinone_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/stringsjoinone"
)

func TestStringsJoinOne(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, stringsjoinone.Analyzer, "stringsjoinone")
}
