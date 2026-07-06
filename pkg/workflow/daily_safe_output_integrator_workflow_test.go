//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDailySafeOutputIntegratorHasToolBudgetAwareness(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/daily-safe-output-integrator.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	sourceContentStr := string(sourceContent)
	for _, expected := range []string{
		"If you are approaching the tool call limit, emit a partial result immediately rather than continuing to gather more data.",
		"Process at most **20 missing types** per run.",
	} {
		if !strings.Contains(sourceContentStr, expected) {
			t.Fatalf("expected daily-safe-output-integrator to contain tool budget awareness guidance %q", expected)
		}
	}
}

func TestDailySafeOutputIntegratorIncludesTempCoverageScriptAllowlist(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/daily-safe-output-integrator.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	sourceContentStr := string(sourceContent)
	bashSectionStart := strings.Index(sourceContentStr, "  bash:\n")
	if bashSectionStart == -1 {
		t.Fatal("expected workflow source to contain a bash tool allowlist")
	}
	bashSectionEnd := strings.Index(sourceContentStr[bashSectionStart:], "  cli-proxy: true")
	if bashSectionEnd == -1 {
		t.Fatal("expected workflow source bash tool allowlist to end before cli-proxy")
	}
	bashSection := sourceContentStr[bashSectionStart : bashSectionStart+bashSectionEnd]
	if !strings.Contains(bashSection, "  - \"*\"") {
		t.Fatal("expected bash allowlist to include wildcard access")
	}
	if !strings.Contains(bashSection, "  - cat > /tmp/gh-aw/agent/*.py") {
		t.Fatal("expected bash allowlist to include temporary coverage script creation")
	}

	for _, expected := range []string{
		"- Create that temporary script with the edit tool or the allowed `cat > /tmp/gh-aw/agent/*.py` bash command, then run it with `python3`.",
		"- Do not retry alternate shell-redirection paths; use only the allowed `/tmp/gh-aw/agent/` path.",
	} {
		if !strings.Contains(sourceContentStr, expected) {
			t.Fatalf("expected workflow source guidance to contain %q", expected)
		}
	}

	lockContent, err := os.ReadFile("../../.github/workflows/daily-safe-output-integrator.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}

	lockContentStr := string(lockContent)
	for _, expected := range []string{
		"--allow-all-tools",
		"{{#runtime-import .github/workflows/daily-safe-output-integrator.md}}",
	} {
		if !strings.Contains(lockContentStr, expected) {
			t.Fatalf("expected compiled workflow to contain %q", expected)
		}
	}
}
