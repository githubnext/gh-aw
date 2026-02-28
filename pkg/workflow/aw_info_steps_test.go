//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestAwInfoStepsFirewall(t *testing.T) {
	tests := []struct {
		name            string
		workflowContent string
		expectFirewall  string
		description     string
	}{
		{
			name: "firewall enabled with copilot",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
network:
  firewall: true
---

# Test firewall enabled

This workflow tests that firewall type is set to squid when enabled.
`,
			expectFirewall: "squid",
			description:    "Should have firewall type squid when firewall is enabled",
		},
		{
			name: "firewall disabled",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
network:
  firewall: false
strict: false
---

# Test firewall disabled

This workflow tests that firewall type is empty when disabled.
`,
			expectFirewall: "",
			description:    "Should have empty firewall type when firewall is disabled",
		},
		{
			name: "no firewall configuration - claude defaults",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test no firewall

This workflow tests that Claude has firewall enabled by default when network is not configured.
`,
			expectFirewall: "squid",
			description:    "Should have firewall type squid when no network is configured (firewall enabled by default)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir := testutil.TempDir(t, "aw-info-steps-test")

			// Create test file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile workflow
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

			// Verify the generate_aw_info step is present
			if !strings.Contains(lockStr, "id: generate_aw_info") {
				t.Error("Expected 'id: generate_aw_info' to be present in generated workflow")
			}

			// Verify firewall type env var
			expectedFirewallLine := `GH_AW_INFO_FIREWALL_TYPE: "` + tt.expectFirewall + `"`
			if !strings.Contains(lockStr, expectedFirewallLine) {
				t.Errorf("%s\nExpected firewall line: %s\nNot found in generated workflow", tt.description, expectedFirewallLine)
				// Print relevant section for debugging
				if idx := strings.Index(lockStr, "id: generate_aw_info"); idx != -1 {
					endIdx := strings.Index(lockStr[idx:], "uses:")
					if endIdx != -1 {
						t.Logf("Found generate_aw_info step env:\n%s", lockStr[idx:idx+endIdx])
					}
				}
			}

			t.Logf("✓ Firewall type correctly set to '%s' for test: %s", tt.expectFirewall, tt.description)
		})
	}
}
