//go:build !integration

package stringshasprefixslice_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/stringshasprefixslice"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, stringshasprefixslice.Analyzer, "stringshasprefixslice")
}
