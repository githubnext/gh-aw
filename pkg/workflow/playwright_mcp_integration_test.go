package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestPlaywrightMCPIntegration tests that compiled workflows generate correct Docker Playwright commands
// This test verifies that the official Playwright MCP Docker image is used with --allowed-hosts flag
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
		expectedFlag         string
		unexpectedFlag       string
		expectedDomains      []string
		shouldContainPackage bool
	}{
		{
			name: "Codex engine with playwright and custom domains",
			workflowContent: `---
on: push
engine: codex
tools:
  playwright:
    allowed_domains:
      - "example.com"
      - "test.com"
---

# Test Workflow

Test playwright with custom domains.
`,
			expectedFlag:         "--allowed-hosts",
			unexpectedFlag:       "--allowed-origins",
			expectedDomains:      []string{"example.com", "test.com", "localhost", "127.0.0.1"},
			shouldContainPackage: true,
		},
		{
			name: "Claude engine with playwright default domains",
			workflowContent: `---
on: push
engine: claude
tools:
  playwright:
---

# Test Workflow

Test playwright with default domains only.
`,
			expectedFlag:         "--allowed-hosts",
			unexpectedFlag:       "--allowed-origins",
			expectedDomains:      []string{"localhost", "127.0.0.1"},
			shouldContainPackage: true,
		},
		{
			name: "Copilot engine with playwright",
			workflowContent: `---
on: push
engine: copilot
tools:
  playwright:
    allowed_domains:
      - "github.com"
---

# Test Workflow

Test playwright with copilot engine.
`,
			expectedFlag:         "--allowed-hosts",
			unexpectedFlag:       "--allowed-origins",
			expectedDomains:      []string{"github.com", "localhost", "127.0.0.1"},
			shouldContainPackage: true,
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
			compiler := NewCompiler(false, "", "test")
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
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

			// Verify the correct flag is used
			if !strings.Contains(lockStr, tt.expectedFlag) {
				t.Errorf("Expected lock file to contain flag %s\nActual content:\n%s", tt.expectedFlag, lockStr)
			}

			// Verify the old flag is NOT used
			if strings.Contains(lockStr, tt.unexpectedFlag) {
				t.Errorf("Did not expect lock file to contain deprecated flag %s\nActual content:\n%s", tt.unexpectedFlag, lockStr)
			}

			// Verify expected domains are present
			for _, domain := range tt.expectedDomains {
				if !strings.Contains(lockStr, domain) {
					t.Errorf("Expected lock file to contain domain %s", domain)
				}
			}
		})
	}
}

// TestPlaywrightNPXCommandWorks verifies that the generated npx command actually works
// This test requires npx to be available and will be skipped if it's not
func TestPlaywrightNPXCommandWorks(t *testing.T) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found, skipping live integration test")
	}

	// Test that the npx command with --allowed-hosts flag works
	cmd := exec.Command("npx", "@playwright/mcp@"+string(constants.DefaultPlaywrightMCPVersion), "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run npx playwright help: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Verify that --allowed-hosts is in the help output (this is the flag we use for server host restrictions)
	if !strings.Contains(outputStr, "--allowed-hosts") {
		t.Errorf("Expected npx playwright help to mention --allowed-hosts flag\nActual output:\n%s", outputStr)
	}

	// Note: --allowed-origins was added in v0.0.48 as a separate feature for browser request filtering
	// It's different from --allowed-hosts which controls which hosts the MCP server serves from
	// Both flags can now coexist, so we no longer check for its absence

	// Verify that the help output contains the expected option description
	if !strings.Contains(outputStr, "allowed-hosts") {
		t.Errorf("Expected help output to contain 'allowed-hosts' option description\nActual output:\n%s", outputStr)
	}
}
