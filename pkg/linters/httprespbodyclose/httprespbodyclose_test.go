//go:build !integration

package httprespbodyclose_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/httprespbodyclose"
)

func TestHTTPRespBodyClose(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, httprespbodyclose.Analyzer, "httprespbodyclose")
}
