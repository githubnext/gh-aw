//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

func TestGitHubDefaultGuardPolicy(t *testing.T) {
	tests := []struct {
		name        string
		workflow    string
		description string
	}{
		{
			name: "Default guard policy applied when no guard policy configured",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test that default guard policy is applied.
`,
			description: "When no guard policy is configured, default repos:all min-integrity:approved should be present",
		},
		{
			name: "Default guard policy in remote mode",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
---

# Test Workflow

Test that remote mode gets default guard policy.
`,
			description: "Remote mode should also get default guard policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "guard-policy-default-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(tt.workflow), 0644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(workflowPath); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			yaml := string(lockContent)

			// Verify no lockdown detection step
			if strings.Contains(yaml, "Determine automatic lockdown") || strings.Contains(yaml, "determine-automatic-lockdown") {
				t.Errorf("%s: Lockdown detection step should not be present", tt.description)
			}

			// Verify no lockdown mode in MCP config
			if strings.Contains(yaml, "GITHUB_LOCKDOWN_MODE") {
				t.Errorf("%s: GITHUB_LOCKDOWN_MODE should not be present", tt.description)
			}

			// Verify default guard policy is present
			if !strings.Contains(yaml, "guard-policies") {
				t.Errorf("%s: Expected guard-policies to be present", tt.description)
			}
			if !strings.Contains(yaml, `"repos": "all"`) && !strings.Contains(yaml, `repos = "all"`) {
				t.Errorf("%s: Expected repos:all in guard policy", tt.description)
			}
			if !strings.Contains(yaml, `"min-integrity": "approved"`) && !strings.Contains(yaml, `min-integrity = "approved"`) {
				t.Errorf("%s: Expected min-integrity:approved in guard policy", tt.description)
			}
		})
	}
}

func TestGitHubNoLockdownDetectionStepClaudeEngine(t *testing.T) {
	workflow := `---
on: issues
engine: claude
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test that Claude engine has no lockdown detection step.
`

	tmpDir, err := os.MkdirTemp("", "no-lockdown-claude-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	yaml := string(lockContent)

	// Verify no lockdown detection step
	if strings.Contains(yaml, "Determine automatic lockdown mode for GitHub MCP Server") ||
		strings.Contains(yaml, "determine-automatic-lockdown") {
		t.Error("Lockdown detection step should not be present for Claude engine")
	}

	// Verify default guard policy is present
	if !strings.Contains(yaml, "guard-policies") {
		t.Error("Expected guard-policies to be present for Claude engine")
	}
}
