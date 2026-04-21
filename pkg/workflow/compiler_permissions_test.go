//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestRunsOnSection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "workflow-runs-on-test")

	compiler := NewCompiler()

	tests := []struct {
		name           string
		frontmatter    string
		expectedRunsOn string
	}{
		{
			name: "default runs-on",
			frontmatter: `---
on: push
tools:
  github:
    allowed: [list_issues]
---`,
			expectedRunsOn: "runs-on: ubuntu-latest",
		},
		{
			name: "custom runs-on",
			frontmatter: `---
on: push
runs-on: windows-latest
tools:
  github:
    allowed: [list_issues]
---`,
			expectedRunsOn: "runs-on: windows-latest",
		},
		{
			name: "custom runs-on with array",
			frontmatter: `---
on: push
runs-on: [self-hosted, linux, x64]
tools:
  github:
    allowed: [list_issues]
---`,
			expectedRunsOn: `runs-on:
                - self-hosted
				- linux
				- x64`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

This is a test workflow.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that the expected runs-on value is present
			if !strings.Contains(lockContent, "    "+tt.expectedRunsOn) {
				// For array format, check differently
				if strings.Contains(tt.expectedRunsOn, "\n") {
					// For multiline YAML, just check that it contains the main components
					if !strings.Contains(lockContent, "runs-on:") || !strings.Contains(lockContent, "- self-hosted") {
						t.Errorf("Expected lock file to contain runs-on with array format but it didn't.\nContent:\n%s", lockContent)
					}
				} else {
					t.Errorf("Expected lock file to contain '    %s' but it didn't.\nContent:\n%s", tt.expectedRunsOn, lockContent)
				}
			}
		})
	}
}

func TestNetworkPermissionsDefaultBehavior(t *testing.T) {
	compiler := NewCompiler()

	tmpDir := testutil.TempDir(t, "test-*")

	t.Run("no network field defaults to full access", func(t *testing.T) {
		testContent := `---
on: push
engine: claude
strict: false
---

# Test Workflow

This is a test workflow without network permissions.
`
		testFile := filepath.Join(tmpDir, "no-network-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "no-network-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// AWF is enabled by default for all engines (copilot, claude, codex) even without explicit network config
		// This ensures sandbox.agent: awf is the default behavior
		if !strings.Contains(string(lockContent), "sudo -E awf") {
			t.Error("Should contain AWF wrapper by default for Claude engine")
		}
	})

	t.Run("network: defaults enables AWF by default for Claude", func(t *testing.T) {
		testContent := `---
on: push
engine: claude
strict: false
network: defaults
---

# Test Workflow

This is a test workflow with explicit defaults network permissions.
`
		testFile := filepath.Join(tmpDir, "defaults-network-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "defaults-network-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// AWF is enabled by default for Claude engine with network: defaults
		if !strings.Contains(string(lockContent), "sudo -E awf") {
			t.Error("Should contain AWF wrapper for Claude engine with network: defaults")
		}
	})

	t.Run("network: {} enables AWF by default for Claude", func(t *testing.T) {
		testContent := `---
on: push
engine: claude
strict: false
network: {}
---

# Test Workflow

This is a test workflow with empty network permissions (deny all).
`
		testFile := filepath.Join(tmpDir, "deny-all-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "deny-all-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// AWF is enabled by default for Claude engine with network: {}
		if !strings.Contains(string(lockContent), "sudo -E awf") {
			t.Error("Should contain AWF wrapper for Claude engine with network: {}")
		}
	})

	t.Run("network with allowed domains should use AWF", func(t *testing.T) {
		testContent := `---
on: push
strict: false
engine:
  id: claude
network:
  allowed: ["example.com", "api.github.com"]
---

# Test Workflow

This is a test workflow with explicit network permissions.
`
		testFile := filepath.Join(tmpDir, "allowed-domains-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "allowed-domains-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// Should contain AWF wrapper with --allow-domains
		if !strings.Contains(string(lockContent), "sudo -E awf") {
			t.Error("Should contain AWF wrapper with explicit network permissions")
		}
		if !strings.Contains(string(lockContent), "--allow-domains") {
			t.Error("Should contain --allow-domains flag in AWF command")
		}
		if !strings.Contains(string(lockContent), "example.com") {
			t.Error("Should contain example.com in allowed domains")
		}
		if !strings.Contains(string(lockContent), "api.github.com") {
			t.Error("Should contain api.github.com in allowed domains")
		}
	})

	t.Run("network permissions with non-claude engine should be ignored", func(t *testing.T) {
		testContent := `---
on: push
engine: codex
strict: false
network:
  allowed: ["example.com"]
---

# Test Workflow

This is a test workflow with network permissions and codex engine.
`
		testFile := filepath.Join(tmpDir, "codex-network-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "codex-network-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// Should not contain claude-specific network hook setup
		if strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should not contain network hook setup for non-claude engines")
		}
	})
}

// TestCopilotRequestsFeaturePermissions verifies that when permissions: read-all is combined
// with features: copilot-requests: true, the agent job receives all read-all permissions merged
// with copilot-requests: write (not replaced by it), and the detection job receives at minimum
// copilot-requests: write.
func TestCopilotRequestsFeaturePermissions(t *testing.T) {
	tmpDir := testutil.TempDir(t, "copilot-requests-permissions-test")

	compiler := NewCompiler()

	t.Run("agent job merges read-all with copilot-requests: write", func(t *testing.T) {
		testContent := `---
on:
  issues:
    types: [opened]
engine: copilot
permissions: read-all
features:
  copilot-requests: true
---

# Test Workflow

This is a test workflow with read-all permissions and copilot-requests feature.
`
		testFile := filepath.Join(tmpDir, "copilot-requests-agent.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(testFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		content := string(lockContent)

		// The agent job must include copilot-requests: write (added by the feature).
		if !strings.Contains(content, "copilot-requests: write") {
			t.Error("Agent job should contain 'copilot-requests: write'")
		}
		// The agent job must also include the read-all-derived permissions.
		if !strings.Contains(content, "contents: read") {
			t.Error("Agent job should contain 'contents: read' (preserved from read-all)")
		}
		if !strings.Contains(content, "issues: read") {
			t.Error("Agent job should contain 'issues: read' (preserved from read-all)")
		}
		if !strings.Contains(content, "pull-requests: read") {
			t.Error("Agent job should contain 'pull-requests: read' (preserved from read-all)")
		}
	})

	t.Run("detection job gets copilot-requests: write when feature enabled", func(t *testing.T) {
		testContent := `---
on:
  issues:
    types: [opened]
engine: copilot
permissions: read-all
features:
  copilot-requests: true
safe-outputs:
  threat-detection: true
---

# Test Workflow With Detection

This is a test workflow with read-all permissions, copilot-requests, and threat detection.
`
		testFile := filepath.Join(tmpDir, "copilot-requests-detection.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		lockFile := stringutil.MarkdownToLockFile(testFile)
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		content := string(lockContent)

		// The detection job section must include copilot-requests: write.
		// We verify this by checking that the key appears in the detection job block.
		detectionIdx := strings.Index(content, "  detection:")
		if detectionIdx == -1 {
			t.Fatal("Lock file should contain a 'detection:' job")
		}
		detectionSection := content[detectionIdx:]
		if !strings.Contains(detectionSection, "copilot-requests: write") {
			t.Error("Detection job should contain 'copilot-requests: write' when copilot-requests feature is enabled")
		}
	})
}
