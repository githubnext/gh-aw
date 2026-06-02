//go:build !integration

package tolowerequalfold_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/tolowerequalfold"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, tolowerequalfold.Analyzer, "tolowerequalfold")
}
