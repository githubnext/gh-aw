//go:build !integration

package walkfuncerrshadow_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/walkfuncerrshadow"
)

func TestWalkFuncErrShadow(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), walkfuncerrshadow.Analyzer, "walkfuncerrshadow")
}
