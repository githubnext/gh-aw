//go:build !integration

package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestRunActionlintOnFiles_EmptyList(t *testing.T) {
	err := RunActionlintOnFiles(nil, false, false)
	if err != nil {
		t.Fatalf("expected nil error for empty lock file list, got %v", err)
	}
}

func TestRunBatchDirectoryTool_NonStrictSwallowsErrors(t *testing.T) {
	runner := func(_ string, _ bool, _ bool) error {
		return errors.New("boom")
	}

	err := runBatchDirectoryTool("poutine", "/tmp/workflows", false, false, runner)
	if err != nil {
		t.Fatalf("expected nil error in non-strict mode, got %v", err)
	}
}

func TestRunBatchDirectoryTool_StrictWrapsErrors(t *testing.T) {
	runner := func(_ string, _ bool, _ bool) error {
		return errors.New("boom")
	}

	err := runBatchDirectoryTool("runner-guard", "/tmp/workflows", false, true, runner)
	if err == nil {
		t.Fatal("expected error in strict mode, got nil")
	}
	if !strings.Contains(err.Error(), "runner-guard failed: boom") {
		t.Fatalf("expected wrapped error message, got %q", err.Error())
	}
}
