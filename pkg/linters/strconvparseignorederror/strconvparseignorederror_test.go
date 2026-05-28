//go:build !integration

package strconvparseignorederror_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/strconvparseignorederror"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, strconvparseignorederror.Analyzer, "strconvparseignorederror")
}
