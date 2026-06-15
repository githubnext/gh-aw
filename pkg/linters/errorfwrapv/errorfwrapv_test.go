//go:build !integration

package errorfwrapv_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/errorfwrapv"
)

func TestErrorfWrapV(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, errorfwrapv.Analyzer, "errorfwrapv")
}
