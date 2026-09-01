//go:build !integration

package globwalkignorederror_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/globwalkignorederror"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, globwalkignorederror.Analyzer, "globwalkignorederror")
}
