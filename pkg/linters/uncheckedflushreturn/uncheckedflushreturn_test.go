//go:build !integration

package uncheckedflushreturn_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/uncheckedflushreturn"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, uncheckedflushreturn.Analyzer, "uncheckedflushreturn")
}
