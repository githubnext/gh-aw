//go:build !integration

package timesleepnocontext_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/timesleepnocontext"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, timesleepnocontext.Analyzer, "timesleepnocontext")
}
