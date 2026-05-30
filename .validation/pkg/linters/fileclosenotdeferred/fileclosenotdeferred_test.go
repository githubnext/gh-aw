//go:build !integration

package fileclosenotdeferred_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/fileclosenotdeferred"
)

func TestFileCloseNotDeferred(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, fileclosenotdeferred.Analyzer, "fileclosenotdeferred")
}
