//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

// TestPlaywrightMCPIntegration tests that compiled workflows generate correct Playwright MCP configuration
func TestPlaywrightMCPIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "gh-aw-playwright-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name                 string
		workflowContent      string
		shouldContainPackage bool
		notExpectedFlags     []string
	}{
		{
			name: "Codex engine with playwright",
			workflowContent: `---
on: push
engine: codex
tools:
  playwright:
---

# Test Workflow

Test playwright with codex engine.
`,
			shouldContainPackage: true,
			notExpectedFlags:     []string{"--allowed-hosts", "--allowed-origins"},
		},
		{
			name: "Claude engine with playwright",
			workflowContent: `---
on: push
engine: claude
tools:
  playwright:
---

# Test Workflow

Test playwright with default settings.
`,
			shouldContainPackage: true,
			notExpectedFlags:     []string{"--allowed-hosts", "--allowed-origins"},
		},
		{
			name: "Copilot engine with playwright",
			workflowContent: `---
on: push
engine: copilot
tools:
  playwright:
---

# Test Workflow

Test playwright with copilot engine.
`,
			shouldContainPackage: true,
			notExpectedFlags:     []string{"--allowed-hosts", "--allowed-origins"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test workflow file
			testFile := filepath.Join(tmpDir, "test-"+strings.ReplaceAll(tt.name, " ", "-")+".md")
			if err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read generated lock file: %v", err)
			}

			lockStr := string(lockContent)

			// Verify the official Playwright MCP Docker image is used
			if tt.shouldContainPackage {
				expectedImage := "mcr.microsoft.com/playwright/mcp"
				if !strings.Contains(lockStr, expectedImage) {
					t.Errorf("Expected lock file to contain Playwright MCP Docker image %s", expectedImage)
				}
			}

			// Verify egress flags are NOT present (controlled by firewall, not playwright flags)
			for _, flag := range tt.notExpectedFlags {
				if strings.Contains(lockStr, flag) {
					t.Errorf("Lock file should not contain flag %s (egress is controlled by firewall)", flag)
				}
			}
		})
	}
}
