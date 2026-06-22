//go:build !integration

package wgdonenotdeferred_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/wgdonenotdeferred"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wgdonenotdeferred.Analyzer, "wgdonenotdeferred")
}
