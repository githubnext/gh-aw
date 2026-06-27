//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDailyCompilerQualityWorkflowRequiresExplicitSafeOutputCompletion(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/daily-compiler-quality.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	sourceContentStr := string(sourceContent)
	for _, expected := range []string{
		"  noop:",
		"**Before finishing, confirm you called either `create_discussion` or `noop`.**",
	} {
		if !strings.Contains(sourceContentStr, expected) {
			t.Fatalf("expected workflow source to contain %q", expected)
		}
	}

	lockContent, err := os.ReadFile("../../.github/workflows/daily-compiler-quality.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}

	lockContentStr := string(lockContent)
	for _, expected := range []string{
		"Tools: create_discussion, missing_tool, missing_data, noop",
		"{{#runtime-import .github/workflows/daily-compiler-quality.md}}",
	} {
		if !strings.Contains(lockContentStr, expected) {
			t.Fatalf("expected compiled workflow to contain %q", expected)
		}
	}
}
