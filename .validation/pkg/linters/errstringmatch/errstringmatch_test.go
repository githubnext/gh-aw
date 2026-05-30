//go:build !integration

package errstringmatch_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/errstringmatch"
)

func TestErrStringMatch(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errstringmatch.Analyzer, "errstringmatch")
}
