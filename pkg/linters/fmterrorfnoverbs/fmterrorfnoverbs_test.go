//go:build !integration

package fmterrorfnoverbs_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/fmterrorfnoverbs"
)

func TestFmtErrorfNoVerbs(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, fmterrorfnoverbs.Analyzer, "fmterrorfnoverbs")
}
