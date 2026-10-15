//go:build !integration

package writestringerrorignored_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/writestring-error-ignored"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, writestringerrorignored.Analyzer, "writestring-error-ignored")
}
