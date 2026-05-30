//go:build !integration

package contextcancelnotdeferred_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/contextcancelnotdeferred"
)

func TestContextCancelNotDeferred(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, contextcancelnotdeferred.Analyzer, "contextcancelnotdeferred")
}
