package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
---

# Test firewall disabled

This workflow tests that firewall type is empty when disabled.
`,
			expectFirewall: "",
			description:    "Should have empty firewall type when firewall is disabled",
		},
		{
			name: "no firewall configuration",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test no firewall

This workflow tests that firewall type is empty when not configured.
`,
			expectFirewall: "",
			description:    "Should have empty firewall type when firewall is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "aw-info-steps-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile workflow
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

			// Verify steps object exists
			if !strings.Contains(lockStr, "steps: {") {
				t.Error("Expected 'steps: {' to be present in awInfo")
			}

			// Verify firewall field
			expectedFirewallLine := `firewall: "` + tt.expectFirewall + `"`
			if !strings.Contains(lockStr, expectedFirewallLine) {
				t.Errorf("%s\nExpected firewall line: %s\nNot found in generated workflow", tt.description, expectedFirewallLine)
				// Print relevant section for debugging
				if strings.Contains(lockStr, "steps: {") {
					startIdx := strings.Index(lockStr, "steps: {")
					endIdx := strings.Index(lockStr[startIdx:], "},")
					if endIdx != -1 {
						t.Logf("Found steps section:\n%s", lockStr[startIdx:startIdx+endIdx+2])
					}
				}
			}

			t.Logf("✓ Firewall type correctly set to '%s' for test: %s", tt.expectFirewall, tt.description)
		})
	}
}
