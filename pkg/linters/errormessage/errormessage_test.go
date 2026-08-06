//go:build !integration

package errormessage_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/errormessage"
)

func TestErrorMessage(t *testing.T) {
	t.Helper()

	if err := errormessage.Analyzer.Flags.Set("changed-files", "errormessage/validation_target_validation.go,errormessage/general.go"); err != nil {
		t.Fatalf("failed to set changed-files flag: %v", err)
	}
	t.Cleanup(func() {
		_ = errormessage.Analyzer.Flags.Set("changed-files", "")
	})

	analysistest.Run(t, analysistest.TestData(), errormessage.Analyzer, "errormessage")
}

func TestErrorMessageFullRepoFlag(t *testing.T) {
	if err := errormessage.Analyzer.Flags.Set("full-repo", "true"); err != nil {
		t.Fatalf("failed to set full-repo flag: %v", err)
	}
	t.Cleanup(func() {
		_ = errormessage.Analyzer.Flags.Set("full-repo", "false")
	})

	analysistest.Run(t, analysistest.TestData(), errormessage.Analyzer, "errormessage")
}

func TestErrorMessageFullRepoSentinel(t *testing.T) {
	if err := errormessage.Analyzer.Flags.Set("changed-files", "all"); err != nil {
		t.Fatalf("failed to set changed-files flag: %v", err)
	}
	t.Cleanup(func() {
		_ = errormessage.Analyzer.Flags.Set("changed-files", "")
	})

	analysistest.Run(t, analysistest.TestData(), errormessage.Analyzer, "errormessage")
}
