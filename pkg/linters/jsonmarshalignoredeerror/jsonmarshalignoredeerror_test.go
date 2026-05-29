//go:build !integration

package jsonmarshalignoredeerror_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/jsonmarshalignoredeerror"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, jsonmarshalignoredeerror.Analyzer, "jsonmarshalignoredeerror")
}
