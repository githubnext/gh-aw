//go:build !integration

package linters

import "testing"

func TestErrorMessageAnalyzerExported(t *testing.T) {
	t.Parallel()
	t.Helper()
	if ErrorMessageAnalyzer == nil {
		t.Fatal("ErrorMessageAnalyzer should be exported and non-nil")
	}
}
