//go:build !integration

package manualmutexunlock_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/manualmutexunlock"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, manualmutexunlock.Analyzer, "manualmutexunlock")
}
