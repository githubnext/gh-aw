//go:build !integration

package httpstatuscode_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/httpstatuscode"
)

func TestHTTPStatusCode(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, httpstatuscode.Analyzer, "httpstatuscode")
}
