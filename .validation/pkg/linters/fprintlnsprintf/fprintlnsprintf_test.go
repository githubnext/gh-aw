//go:build !integration

package fprintlnsprintf_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/fprintlnsprintf"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fprintlnsprintf.Analyzer, "fprintlnsprintf")
}
