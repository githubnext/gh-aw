//go:build !integration

package hardcodedfilepath_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/hardcodedfilepath"
)

func TestHardcodedFilePath(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, hardcodedfilepath.Analyzer, "constants", "hardcodedfilepath")
}
