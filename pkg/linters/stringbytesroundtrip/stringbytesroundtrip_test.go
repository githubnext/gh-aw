//go:build !integration

package stringbytesroundtrip_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/stringbytesroundtrip"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, stringbytesroundtrip.Analyzer, "stringbytesroundtrip")
}
