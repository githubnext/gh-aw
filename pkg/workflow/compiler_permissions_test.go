package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestRunsOnSection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "workflow-runs-on-test")

	compiler := NewCompiler(false, "", "test")

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
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
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
	compiler := NewCompiler(false, "", "test")

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

		// When no network is specified, firewall is NOT enabled (defaults to full access)
		// AWF is only enabled when network restrictions are configured
		if strings.Contains(string(lockContent), "sudo -E awf") {
			t.Error("Should NOT contain AWF wrapper when no network field specified (defaults to full access)")
		}
	})

	t.Run("network: defaults should not enable AWF for Claude without firewall config", func(t *testing.T) {
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

		// network: defaults without explicit firewall config does NOT enable AWF
		// (firewall must be explicitly enabled or network.allowed must be specified)
		if strings.Contains(string(lockContent), "sudo -E awf") {
			t.Error("Should NOT contain AWF wrapper for network: defaults without firewall config")
		}
	})

	t.Run("network: {} should not enable AWF without firewall config", func(t *testing.T) {
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

		// Empty network config without explicit firewall config does NOT enable AWF
		if strings.Contains(string(lockContent), "sudo -E awf") {
			t.Error("Should NOT contain AWF wrapper for network: {} without firewall config")
		}
	})

	t.Run("network with allowed domains and firewall enabled should use AWF", func(t *testing.T) {
		testContent := `---
on: push
strict: false
engine:
  id: claude
network:
  allowed: ["example.com", "api.github.com"]
  firewall: true
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
			t.Error("Should contain AWF wrapper with explicit network permissions and firewall: true")
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
