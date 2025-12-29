package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubLockdownAutodetection(t *testing.T) {
	tests := []struct {
		name               string
		workflow           string
		expectedDetectStep bool
		expectedLockdown   string // "auto" means use step output expression, "true" means hardcoded true, "false" means not present
		description        string
	}{
		{
			name: "Auto-detection enabled when lockdown not specified",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test automatic lockdown detection.
`,
			expectedDetectStep: true,
			expectedLockdown:   "auto",
			description:        "When lockdown is not specified, detection step should be added and lockdown should use step output",
		},
		{
			name: "No auto-detection when lockdown explicitly set to true",
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
			expectedDetectStep: false,
			expectedLockdown:   "true",
			description:        "When lockdown is explicitly true, no detection step and lockdown should be hardcoded",
		},
		{
			name: "No auto-detection when lockdown explicitly set to false",
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
			expectedDetectStep: false,
			expectedLockdown:   "false",
			description:        "When lockdown is explicitly false, no detection step and no lockdown setting",
		},
		{
			name: "Auto-detection with remote mode",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
---

# Test Workflow

Test auto-detection with remote GitHub MCP.
`,
			expectedDetectStep: true,
			expectedLockdown:   "auto",
			description:        "Auto-detection should work with remote mode too",
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
			compiler := NewCompiler(false, "", "test")
			if err := compiler.CompileWorkflow(workflowPath); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			yaml := string(lockContent)

			// Check if detection step is present
			detectStepPresent := strings.Contains(yaml, "Detect repository visibility for GitHub MCP lockdown") &&
				strings.Contains(yaml, "detect-repo-visibility") &&
				strings.Contains(yaml, "detect_repo_visibility.cjs")

			if detectStepPresent != tt.expectedDetectStep {
				t.Errorf("%s: Detection step presence = %v, want %v", tt.description, detectStepPresent, tt.expectedDetectStep)
			}

			// Check lockdown configuration based on expected value
			switch tt.expectedLockdown {
			case "auto":
				// Should use step output expression
				if !strings.Contains(yaml, "steps.detect-repo-visibility.outputs.lockdown") {
					t.Errorf("%s: Expected lockdown to use step output expression", tt.description)
				}
			case "true":
				// Should have hardcoded GITHUB_LOCKDOWN_MODE=1 or X-MCP-Lockdown: true
				hasDockerLockdown := strings.Contains(yaml, "GITHUB_LOCKDOWN_MODE=1")
				hasRemoteLockdown := strings.Contains(yaml, "X-MCP-Lockdown") && strings.Contains(yaml, "\"true\"")
				if !hasDockerLockdown && !hasRemoteLockdown {
					t.Errorf("%s: Expected hardcoded lockdown setting", tt.description)
				}
			case "false":
				// Should not have GITHUB_LOCKDOWN_MODE or X-MCP-Lockdown
				if strings.Contains(yaml, "GITHUB_LOCKDOWN_MODE") || strings.Contains(yaml, "X-MCP-Lockdown") {
					t.Errorf("%s: Expected no lockdown setting", tt.description)
				}
			}
		})
	}
}

func TestGitHubLockdownAutodetectionClaudeEngine(t *testing.T) {
	workflow := `---
on: issues
engine: claude
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test automatic lockdown detection with Claude.
`

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "lockdown-autodetect-claude-test-*")
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
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	yaml := string(lockContent)

	// Check if detection step is present
	detectStepPresent := strings.Contains(yaml, "Detect repository visibility for GitHub MCP lockdown") &&
		strings.Contains(yaml, "detect-repo-visibility")

	if !detectStepPresent {
		t.Error("Detection step should be present for Claude engine")
	}

	// Check if lockdown uses step output expression
	if !strings.Contains(yaml, "steps.detect-repo-visibility.outputs.lockdown") {
		t.Error("Expected lockdown to use step output expression for Claude engine")
	}
}
