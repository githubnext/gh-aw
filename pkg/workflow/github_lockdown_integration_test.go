//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

func TestGitHubGuardPolicyIntegration(t *testing.T) {
	tests := []struct {
		name        string
		workflow    string
		expected    []string
		notExpected []string
		description string
	}{
		{
			name: "copilot engine local mode gets default guard policy",
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
			expected: []string{
				`"type": "stdio"`,
				`"guard-policies"`,
				`"allow-only"`,
				`"repos": "all"`,
				`"min-integrity": "approved"`,
			},
			notExpected: []string{
				`"GITHUB_LOCKDOWN_MODE"`,
				`"X-MCP-Lockdown"`,
			},
			description: "Copilot with local mode should render default guard policy, no lockdown",
		},
		{
			name: "copilot engine remote mode gets default guard policy",
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
			expected: []string{
				`"type": "http"`,
				`"guard-policies"`,
				`"repos": "all"`,
				`"min-integrity": "approved"`,
			},
			notExpected: []string{
				`"X-MCP-Lockdown"`,
				`"GITHUB_LOCKDOWN_MODE"`,
			},
			description: "Copilot with remote mode should render default guard policy, no lockdown",
		},
		{
			name: "claude engine gets default guard policy",
			workflow: `---
on: issues
engine: claude
tools:
  github:
    mode: local
    toolsets: [default]
---

# Test Workflow

Test guard policy with Claude engine.
`,
			expected: []string{
				`"guard-policies"`,
				`"repos": "all"`,
				`"min-integrity": "approved"`,
			},
			notExpected: []string{
				`"GITHUB_LOCKDOWN_MODE"`,
				`"type": "stdio"`, // Claude doesn't include type field
			},
			description: "Claude should render default guard policy, no lockdown",
		},
		{
			name: "explicit guard policy overrides default",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    repos: [github/gh-aw]
    min-integrity: merged
    toolsets: [default]
---

# Test Workflow

Test explicit guard policy override.
`,
			expected: []string{
				`"guard-policies"`,
				`"min-integrity": "merged"`,
			},
			notExpected: []string{
				`"GITHUB_LOCKDOWN_MODE"`,
				`"X-MCP-Lockdown"`,
				`"repos": "all"`,
			},
			description: "Explicit guard policy should override default",
		},
		{
			name: "read-only with default guard policy",
			workflow: `---
on: issues
engine: copilot
tools:
  github:
    mode: local
    toolsets: [repos]
---

# Test Workflow

Test read-only with guard policy.
`,
			expected: []string{
				`"GITHUB_READ_ONLY": "1"`,
				`"guard-policies"`,
				`"repos": "all"`,
			},
			notExpected: []string{
				`"GITHUB_LOCKDOWN_MODE"`,
			},
			description: "Read-only mode and guard policy can coexist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "guard-policy-test-*")
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

			for _, expected := range tt.expected {
				if !strings.Contains(yaml, expected) {
					t.Errorf("%s: Expected output to contain %q, but it doesn't", tt.description, expected)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(yaml, notExpected) {
					t.Errorf("%s: Expected output NOT to contain %q, but it does", tt.description, notExpected)
				}
			}
		})
	}
}
