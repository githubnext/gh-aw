//go:build !integration

package errortypeassertion_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/errortypeassertion"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errortypeassertion.Analyzer, "errortypeassertion")
}
