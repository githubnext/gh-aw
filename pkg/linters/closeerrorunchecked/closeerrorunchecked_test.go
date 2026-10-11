//go:build !integration

package closeerrorunchecked_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/closeerrorunchecked"
	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
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
	combined := analyzerutil.New("combinedclosechecks", "checks both Close() error policies together", func(pass *analysis.Pass) (any, error) {
		for _, analyzer := range []*analysis.Analyzer{closeerrorunchecked.Analyzer, unchecked_deferredclose.Analyzer} {
			analyzerPass := *pass
			analyzerPass.Analyzer = analyzer
			analyzerPass.Report = func(diagnostic analysis.Diagnostic) {
				diagnostic.Message = analyzer.Name + ": " + diagnostic.Message
				pass.Report(diagnostic)
			}
			if _, err := analyzer.Run(&analyzerPass); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	analysistest.Run(t, analysistest.TestData(), combined, "combined")
}
