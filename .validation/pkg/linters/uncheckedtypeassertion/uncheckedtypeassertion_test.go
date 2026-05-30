//go:build !integration

package uncheckedtypeassertion_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/uncheckedtypeassertion"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, uncheckedtypeassertion.Analyzer, "uncheckedtypeassertion")
}
