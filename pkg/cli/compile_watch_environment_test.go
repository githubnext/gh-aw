//go:build !integration

package cli

import (
	"context"
	"strings"
	"testing"
)

func TestCompileWorkflowsRejectsWatchInAutomatedEnvironments(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
	}{
		{name: "CI", envVar: "CI", envValue: "true"},
		{name: "Copilot coding agent", envVar: "COPILOT_AGENT_SESSION_ID", envValue: "session-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, envVar := range []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "COPILOT_AGENT_SESSION_ID"} {
				t.Setenv(envVar, "")
			}
			t.Setenv(tt.envVar, tt.envValue)

			_, err := CompileWorkflows(context.Background(), CompileConfig{Watch: true})
			if err == nil {
				t.Fatal("expected watch mode to be rejected")
			}
			if !strings.Contains(err.Error(), "watch mode cannot be used in CI or Copilot coding agent environments") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
