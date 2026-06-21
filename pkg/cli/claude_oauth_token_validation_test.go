package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUnsupportedClaudeOAuthTokenForEngine(t *testing.T) {
	t.Setenv(claudeCodeOAuthTokenEnvVar, "")
	if err := validateUnsupportedClaudeOAuthTokenForEngine("claude"); err != nil {
		t.Fatalf("expected no error when token is unset, got %v", err)
	}

	t.Setenv(claudeCodeOAuthTokenEnvVar, "gho_test")
	if err := validateUnsupportedClaudeOAuthTokenForEngine("copilot"); err != nil {
		t.Fatalf("expected no error for non-claude engine, got %v", err)
	}
	if err := validateUnsupportedClaudeOAuthTokenForEngine("claude"); err == nil || !strings.Contains(err.Error(), "set ANTHROPIC_API_KEY instead") {
		t.Fatalf("expected guidance error for claude engine, got %v", err)
	}
}

func TestValidateUnsupportedClaudeOAuthTokenForWorkflowFiles(t *testing.T) {
	t.Setenv(claudeCodeOAuthTokenEnvVar, "gho_test")

	tempDir := t.TempDir()
	copilotWorkflow := filepath.Join(tempDir, "copilot.md")
	claudeWorkflow := filepath.Join(tempDir, "claude.md")

	if err := os.WriteFile(copilotWorkflow, []byte("---\nengine: copilot\n---\n"), 0o644); err != nil {
		t.Fatalf("failed to write copilot workflow: %v", err)
	}
	if err := os.WriteFile(claudeWorkflow, []byte("---\nengine: claude\n---\n"), 0o644); err != nil {
		t.Fatalf("failed to write claude workflow: %v", err)
	}

	if err := validateUnsupportedClaudeOAuthTokenForWorkflowFiles([]string{copilotWorkflow}, ""); err != nil {
		t.Fatalf("expected no error for non-claude workflow, got %v", err)
	}
	if err := validateUnsupportedClaudeOAuthTokenForWorkflowFiles([]string{claudeWorkflow}, ""); err == nil || !strings.Contains(err.Error(), claudeCodeOAuthTokenEnvVar) {
		t.Fatalf("expected unsupported token error for claude workflow, got %v", err)
	}
	if err := validateUnsupportedClaudeOAuthTokenForWorkflowFiles([]string{filepath.Join(tempDir, "missing.md")}, ""); err == nil || !strings.Contains(err.Error(), "failed to inspect workflow") {
		t.Fatalf("expected inspection error for missing workflow file, got %v", err)
	}
}
