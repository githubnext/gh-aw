package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckoutStepWithPermissions(t *testing.T) {
	tests := []struct {
		name           string
		permissions    string
		shouldCheckout bool
	}{
		{
			name:           "no_permissions",
			permissions:    "",
			shouldCheckout: false, // Default is empty {} which doesn't include contents access
		},
		{
			name:           "empty_permissions",
			permissions:    "permissions: {}",
			shouldCheckout: false,
		},
		{
			name:           "issues_only",
			permissions:    "permissions:\n  issues: write",
			shouldCheckout: false,
		},
		{
			name:           "contents_read",
			permissions:    "permissions:\n  contents: read",
			shouldCheckout: true,
		},
		{
			name:           "contents_write",
			permissions:    "permissions:\n  contents: write",
			shouldCheckout: true,
		},
		{
			name:           "read_all",
			permissions:    "permissions: read-all",
			shouldCheckout: true,
		},
		{
			name:           "write_all",
			permissions:    "permissions: write-all",
			shouldCheckout: true,
		},
		{
			name:           "read",
			permissions:    "permissions: read",
			shouldCheckout: true,
		},
		{
			name:           "write",
			permissions:    "permissions: write",
			shouldCheckout: true,
		},
		{
			name:           "mixed_with_contents_read",
			permissions:    "permissions:\n  issues: write\n  contents: read",
			shouldCheckout: true,
		},
		{
			name:           "mixed_without_contents",
			permissions:    "permissions:\n  issues: write\n  pull-requests: write",
			shouldCheckout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "checkout-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test workflow content
			testContent := `---
on:
  issues:
    types: [opened]
` + tt.permissions + `
tools:
  github:
    allowed: [list_issues]
engine: claude
---

# Test Checkout Step Workflow

This workflow tests whether the checkout step is emitted based on permissions.
`

			testFile := filepath.Join(tmpDir, "test-checkout.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")

			// Compile the workflow
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Calculate the lock file path
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

			// Read the generated lock file
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)
			hasCheckout := strings.Contains(lockContentStr, "actions/checkout")

			if tt.shouldCheckout && !hasCheckout {
				t.Errorf("Expected actions/checkout step to be present but it was not found. Content:\n%s", lockContentStr)
			}
			if !tt.shouldCheckout && hasCheckout {
				t.Errorf("Expected actions/checkout step to NOT be present but it was found. Content:\n%s", lockContentStr)
			}
		})
	}
}
