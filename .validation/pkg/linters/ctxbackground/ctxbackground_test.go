//go:build !integration

package ctxbackground_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/ctxbackground"
)

func TestCtxBackground(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, ctxbackground.Analyzer, "ctxbackground")
}
