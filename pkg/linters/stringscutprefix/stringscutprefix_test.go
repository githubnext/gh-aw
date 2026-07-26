package stringscutprefix_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/stringscutprefix"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), stringscutprefix.Analyzer, "stringscutprefix")
}
