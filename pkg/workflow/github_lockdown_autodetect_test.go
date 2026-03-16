//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

func TestGitHubLockdownAutodetection(t *testing.T) {
	tests := []struct {
		name             string
		workflow         string
		expectedLockdown string // "true" means hardcoded true, "none" means no lockdown setting at all
		description      string
	}{
		{
			name: "No lockdown when lockdown not specified",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test that no automatic lockdown detection step is generated when lockdown is not specified.
`,
			expectedLockdown: "none",
			description:      "When lockdown is not specified, no lockdown step or hardcoded lockdown should be present",
		},
		{
			name: "Lockdown enabled when explicitly set to true",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    lockdown: true
    toolsets: [default]
---

# Test Workflow

Test with explicit lockdown enabled.
`,
			expectedLockdown: "true",
			description:      "When lockdown is explicitly true, lockdown should be hardcoded",
		},
		{
			name: "No lockdown when explicitly set to false",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    lockdown: false
    toolsets: [default]
---

# Test Workflow

Test with explicit lockdown disabled.
`,
			expectedLockdown: "none",
			description:      "When lockdown is explicitly false, no lockdown setting should be present",
		},
		{
			name: "No lockdown with remote mode when not specified",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
---

# Test Workflow

Test that remote mode has no lockdown when lockdown not specified.
`,
			expectedLockdown: "none",
			description:      "Remote mode without explicit lockdown should have no lockdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tmpDir, err := os.MkdirTemp("", "lockdown-autodetect-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write workflow file
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(tt.workflow), 0644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Compile workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(workflowPath); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			yaml := string(lockContent)

			// Verify that the automatic lockdown detection step is never present
			hasDetectionStep := strings.Contains(yaml, "Determine automatic lockdown") &&
				strings.Contains(yaml, "determine-automatic-lockdown")
			if hasDetectionStep {
				t.Errorf("%s: Automatic lockdown detection step should not be generated", tt.description)
			}

			// Check lockdown configuration based on expected value
			switch tt.expectedLockdown {
			case "true":
				// Should have hardcoded GITHUB_LOCKDOWN_MODE=1 or X-MCP-Lockdown: true
				hasDockerLockdown := strings.Contains(yaml, `"GITHUB_LOCKDOWN_MODE": "1"`)
				hasRemoteLockdown := strings.Contains(yaml, "X-MCP-Lockdown") && strings.Contains(yaml, "\"true\"")
				if !hasDockerLockdown && !hasRemoteLockdown {
					t.Errorf("%s: Expected hardcoded lockdown setting", tt.description)
				}
			case "none":
				// Should not have GITHUB_LOCKDOWN_MODE=1 or X-MCP-Lockdown
				if strings.Contains(yaml, `"GITHUB_LOCKDOWN_MODE": "1"`) {
					t.Errorf("%s: Expected no hardcoded lockdown setting", tt.description)
				}
				if strings.Contains(yaml, "X-MCP-Lockdown") {
					t.Errorf("%s: Expected no X-MCP-Lockdown header", tt.description)
				}
			}
		})
	}
}

func TestGitHubLockdownExplicitOnlyClaudeEngine(t *testing.T) {
	workflow := `---
on: issues
engine: claude
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test that Claude engine has no automatic lockdown determination.
`

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "lockdown-explicit-claude-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write workflow file
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflow), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	yaml := string(lockContent)

	// Verify no automatic detection step is present
	detectStepPresent := strings.Contains(yaml, "Determine automatic lockdown mode for GitHub MCP Server") ||
		strings.Contains(yaml, "determine-automatic-lockdown")
	if detectStepPresent {
		t.Error("Automatic lockdown detection step should not be present for Claude engine")
	}

	// Verify no hardcoded lockdown when lockdown not explicitly set
	if strings.Contains(yaml, `"GITHUB_LOCKDOWN_MODE": "1"`) {
		t.Error("Expected no hardcoded lockdown setting when lockdown not explicitly set")
	}
}
