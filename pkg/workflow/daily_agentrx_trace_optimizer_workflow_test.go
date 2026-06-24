//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDailyAgentRxTraceOptimizerWorkflowAllowsPythonNativeDependencies(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/daily-agentrx-trace-optimizer.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}

	sourceContentStr := string(sourceContent)
	if !strings.Contains(sourceContentStr, "allowed: [defaults, python-native, github]") {
		t.Fatalf("expected workflow source to allow the python-native ecosystem for AgentRx installation")
	}

	lockContent, err := os.ReadFile("../../.github/workflows/daily-agentrx-trace-optimizer.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}

	lockContentStr := string(lockContent)
	for _, expected := range []string{"index.crates.io", "static.crates.io", "pypi.org"} {
		if !strings.Contains(lockContentStr, expected) {
			t.Fatalf("expected compiled workflow to allow %q for AgentRx installation", expected)
		}
	}
}
