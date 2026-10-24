//go:build !integration

package readerstringconv_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/readerstringconv"
)

func TestReaderStringConv(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, readerstringconv.Analyzer, "readerstringconv")
}
