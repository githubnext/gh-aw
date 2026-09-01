//go:build !integration

package generatedyamlheredoc_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/generatedyamlheredoc"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, generatedyamlheredoc.Analyzer, "generatedyamlheredoc")
}
