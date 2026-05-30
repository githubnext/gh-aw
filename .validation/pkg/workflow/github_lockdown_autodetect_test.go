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
		name                    string
		workflow                string
		expectedGuardPolicy     string // "auto" means from step outputs, "static" means hardcoded, "none" means no guard-policy
		expectAutoDetectionStep bool   // true if automatic detection step should be present
		description             string
	}{
		{
			name: "Automatic detection when guard policy not specified",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test that automatic guard policy detection is enabled when guard policy is not specified.
`,
			expectedGuardPolicy:     "auto",
			expectAutoDetectionStep: true,
			description:             "When guard policy is not specified, automatic detection step should be present with env var refs",
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
			expectedGuardPolicy:     "auto",
			expectAutoDetectionStep: true,
			description:             "When lockdown is explicitly true but no guard policy, auto detection step should still run",
		},
		{
			name: "No auto detection when guard policy explicitly configured",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    repos: "all"
    min-integrity: approved
    toolsets: [default]
---

# Test Workflow

Test with explicit guard policy configured.
`,
			expectedGuardPolicy:     "static",
			expectAutoDetectionStep: false,
			description:             "When guard policy is explicitly configured, no auto detection step",
		},
		{
			name: "Automatic detection with remote mode when not specified",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
---

# Test Workflow

Test that remote mode uses automatic detection when guard policy not specified.
`,
			expectedGuardPolicy:     "auto",
			expectAutoDetectionStep: true,
			description:             "Remote mode without explicit guard policy should use automatic detection",
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

			// Check for automatic detection step based on expectation
			hasDetectionStep := strings.Contains(yaml, "Determine automatic lockdown") &&
				strings.Contains(yaml, "determine-automatic-lockdown")

			if tt.expectAutoDetectionStep && !hasDetectionStep {
				t.Errorf("%s: Expected automatic detection step but it was not found", tt.description)
			}
			if !tt.expectAutoDetectionStep && hasDetectionStep {
				t.Errorf("%s: Did not expect automatic detection step but it was found", tt.description)
			}

			// Check guard policy configuration based on expected value
			switch tt.expectedGuardPolicy {
			case "static":
				// Should have hardcoded guard policy values (not env var refs)
				hasStaticGuardPolicy := strings.Contains(yaml, `"guard-policies"`) &&
					!strings.Contains(yaml, `$GITHUB_MCP_GUARD_MIN_INTEGRITY`)
				if !hasStaticGuardPolicy {
					t.Errorf("%s: Expected static guard policy but not found", tt.description)
				}
			case "auto":
				// Should use step output env vars for guard policies
				hasGuardEnvVars := strings.Contains(yaml, "GITHUB_MCP_GUARD_MIN_INTEGRITY") &&
					strings.Contains(yaml, "GITHUB_MCP_GUARD_REPOS")
				if !hasGuardEnvVars {
					t.Errorf("%s: Expected guard policy env vars from step output", tt.description)
				}
				// Should reference step outputs in Start MCP Gateway env
				hasStepOutputRef := strings.Contains(yaml, "steps.determine-automatic-lockdown.outputs.min_integrity") &&
					strings.Contains(yaml, "steps.determine-automatic-lockdown.outputs.repos")
				if !hasStepOutputRef {
					t.Errorf("%s: Expected step output references in Start MCP Gateway env", tt.description)
				}
			case "none":
				// Should not have guard policy at all
				if strings.Contains(yaml, `"guard-policies"`) {
					t.Errorf("%s: Expected no guard policy but found one", tt.description)
				}
			}

			// Verify lockdown is no longer automatically set from step output
			if strings.Contains(yaml, "steps.determine-automatic-lockdown.outputs.lockdown") {
				t.Errorf("%s: lockdown output should no longer be automatically emitted from step", tt.description)
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

	// Verify automatic detection step is present (guard policy not explicitly set)
	detectStepPresent := strings.Contains(yaml, "Determine automatic lockdown mode for GitHub MCP Server") &&
		strings.Contains(yaml, "determine-automatic-lockdown")

	if !detectStepPresent {
		t.Error("Determination step should be present for Claude engine when guard policy not explicitly set")
	}

	// Check that guard policy env vars are referenced (not lockdown)
	if !strings.Contains(yaml, "GITHUB_MCP_GUARD_MIN_INTEGRITY") {
		t.Error("Expected GITHUB_MCP_GUARD_MIN_INTEGRITY env var for Claude engine")
	}
	if !strings.Contains(yaml, "GITHUB_MCP_GUARD_REPOS") {
		t.Error("Expected GITHUB_MCP_GUARD_REPOS env var for Claude engine")
	}

	// Verify lockdown is no longer automatically set from step output
	if strings.Contains(yaml, "steps.determine-automatic-lockdown.outputs.lockdown") {
		t.Error("lockdown output should no longer be automatically emitted from step")
	}
}
