//go:build !integration

package osgetenvlibrary_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/osgetenvlibrary"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), osgetenvlibrary.Analyzer, "osgetenvlibrary", "mainpkg")
}
