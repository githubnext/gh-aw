//go:build !integration

package nilctxpassed_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/nilctxpassed"
)

func TestNilCtxPassed(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, nilctxpassed.Analyzer, "nilctxpassed", "nilctxpassed_noctx")
}
