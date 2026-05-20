//go:build !integration

package regexpcompileinfunction_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/regexpcompileinfunction"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, regexpcompileinfunction.Analyzer, "regexpcompileinfunction")
}
