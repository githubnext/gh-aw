//go:build !integration

package ioutildeprecated_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/ioutildeprecated"
)

func TestIoutilDeprecated(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, ioutildeprecated.Analyzer, "ioutildeprecated")
}
