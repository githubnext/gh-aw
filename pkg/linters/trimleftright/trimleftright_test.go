//go:build !integration

package trimleftright_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/trimleftright"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), trimleftright.Analyzer, "trimleftright")
}
