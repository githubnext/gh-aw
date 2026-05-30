//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestGetGitIdentityEnvVars verifies the helper returns all four required git identity variables
func TestGetGitIdentityEnvVars(t *testing.T) {
	vars := getGitIdentityEnvVars()

	expected := map[string]string{
		"GIT_AUTHOR_NAME":     "github-actions[bot]",
		"GIT_AUTHOR_EMAIL":    "github-actions[bot]@users.noreply.github.com",
		"GIT_COMMITTER_NAME":  "github-actions[bot]",
		"GIT_COMMITTER_EMAIL": "github-actions[bot]@users.noreply.github.com",
	}

	for key, want := range expected {
		got, ok := vars[key]
		if !ok {
			t.Errorf("missing expected key %s", key)
			continue
		}
		if got != want {
			t.Errorf("key %s: got %q, want %q", key, got, want)
		}
	}

	if len(vars) != len(expected) {
		t.Errorf("expected %d entries, got %d", len(expected), len(vars))
	}
}

// sandboxWorkflowData builds a WorkflowData with AWF firewall enabled.
func sandboxWorkflowData(name string) *WorkflowData {
	return &WorkflowData{
		Name: name,
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}
}

// gitIdentityKeys are the four environment variable names that must be present in sandbox runs.
var gitIdentityKeys = []string{
	"GIT_AUTHOR_NAME",
	"GIT_AUTHOR_EMAIL",
	"GIT_COMMITTER_NAME",
	"GIT_COMMITTER_EMAIL",
}

// TestCopilotEngineGitIdentityEnvSandbox verifies git identity env vars are present in sandbox mode
// and absent in non-sandbox mode.
func TestCopilotEngineGitIdentityEnvSandbox(t *testing.T) {
	engine := NewCopilotEngine()

	t.Run("sandbox mode includes git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(sandboxWorkflowData("test-copilot"), "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if !strings.Contains(stepContent, key+": github-actions") {
				t.Errorf("expected %s in sandbox execution step, but not found:\n%s", key, stepContent)
			}
		}
	})

	t.Run("non-sandbox mode does not include git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-copilot-no-fw"}, "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if strings.Contains(stepContent, key+":") {
				t.Errorf("did not expect %s in non-sandbox execution step, but found:\n%s", key, stepContent)
			}
		}
	})
}

// TestClaudeEngineGitIdentityEnvSandbox verifies git identity env vars are present in sandbox mode
// and absent in non-sandbox mode.
func TestClaudeEngineGitIdentityEnvSandbox(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("sandbox mode includes git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(sandboxWorkflowData("test-claude"), "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if !strings.Contains(stepContent, key+": github-actions") {
				t.Errorf("expected %s in sandbox execution step, but not found:\n%s", key, stepContent)
			}
		}
	})

	t.Run("non-sandbox mode does not include git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-claude-no-fw"}, "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if strings.Contains(stepContent, key+":") {
				t.Errorf("did not expect %s in non-sandbox execution step, but found:\n%s", key, stepContent)
			}
		}
	})
}

// TestCodexEngineGitIdentityEnvSandbox verifies git identity env vars are present in sandbox mode
// and absent in non-sandbox mode.
func TestCodexEngineGitIdentityEnvSandbox(t *testing.T) {
	engine := NewCodexEngine()

	t.Run("sandbox mode includes git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(sandboxWorkflowData("test-codex"), "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if !strings.Contains(stepContent, key+": github-actions") {
				t.Errorf("expected %s in sandbox execution step, but not found:\n%s", key, stepContent)
			}
		}
	})

	t.Run("non-sandbox mode does not include git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-codex-no-fw"}, "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if strings.Contains(stepContent, key+":") {
				t.Errorf("did not expect %s in non-sandbox execution step, but found:\n%s", key, stepContent)
			}
		}
	})
}

// TestGeminiEngineGitIdentityEnvSandbox verifies git identity env vars are present in sandbox mode
// and absent in non-sandbox mode.
func TestGeminiEngineGitIdentityEnvSandbox(t *testing.T) {
	engine := NewGeminiEngine()

	t.Run("sandbox mode includes git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(sandboxWorkflowData("test-gemini"), "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if !strings.Contains(stepContent, key+": github-actions") {
				t.Errorf("expected %s in sandbox execution step, but not found:\n%s", key, stepContent)
			}
		}
	})

	t.Run("non-sandbox mode does not include git identity env vars", func(t *testing.T) {
		steps := engine.GetExecutionSteps(&WorkflowData{Name: "test-gemini-no-fw"}, "test.log")
		stepContent := joinSteps(steps)

		for _, key := range gitIdentityKeys {
			if strings.Contains(stepContent, key+":") {
				t.Errorf("did not expect %s in non-sandbox execution step, but found:\n%s", key, stepContent)
			}
		}
	})
}

// joinSteps joins all lines from all GitHubActionStep entries into a single string for assertion.
func joinSteps(steps []GitHubActionStep) string {
	var sb strings.Builder
	for _, step := range steps {
		for _, line := range step {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
