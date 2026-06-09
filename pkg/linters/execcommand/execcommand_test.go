//go:build !integration

package execcommand_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/execcommand"
)

func TestExecCommand(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, execcommand.Analyzer, "execcommand")
}
