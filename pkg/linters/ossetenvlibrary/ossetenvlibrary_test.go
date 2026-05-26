//go:build !integration

package ossetenvlibrary_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/ossetenvlibrary"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ossetenvlibrary.Analyzer, "ossetenvlibrary", "mainpkg", "fixtures/cmd/tool")
}
