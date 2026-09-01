//go:build !integration

package regexpdynamicpattern_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/regexpdynamicpattern"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, regexpdynamicpattern.Analyzer, "regexpdynamicpattern")
}
