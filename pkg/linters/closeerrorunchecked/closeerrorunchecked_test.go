//go:build !integration

package closeerrorunchecked_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/closeerrorunchecked"
	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil/analyzertest"
	unchecked_deferredclose "github.com/github/gh-aw/pkg/linters/unchecked-deferred-close"
)

func TestCloseErrorUnchecked(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, closeerrorunchecked.Analyzer, "closeerrorunchecked")
	analysistest.Run(t, testdata, closeerrorunchecked.Analyzer, "closeerrorunchecked/customerror")
}

func TestComplementaryCloseErrorAnalyzers(t *testing.T) {
	t.Parallel()
	analyzertest.RunTogether(t, analysistest.TestData(), []*analysis.Analyzer{
		closeerrorunchecked.Analyzer,
		unchecked_deferredclose.Analyzer,
	}, "combined")
}
