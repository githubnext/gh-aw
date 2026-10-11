//go:build !integration

package walkdirignoredeerror_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	walkdirignoredeerror "github.com/github/gh-aw/pkg/linters/walk-dir-ignored-error"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, walkdirignoredeerror.Analyzer, "walkdirignoredeerror")
}
