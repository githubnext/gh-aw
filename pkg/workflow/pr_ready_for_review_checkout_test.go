package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPRCheckout verifies that PR branch checkout is added for pull_request events
func TestPRCheckout(t *testing.T) {
	tests := []struct {
		name             string
		workflowContent  string
		expectPRCheckout bool
	}{
		{
			name: "pull_request with ready_for_review should add checkout",
			workflowContent: `---
on:
  pull_request:
    types: [ready_for_review]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test Workflow
Test workflow with pull_request ready_for_review trigger.
`,
			expectPRCheckout: true,
		},
		{
			name: "pull_request with opened should add checkout",
			workflowContent: `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test Workflow
Test workflow with pull_request opened trigger.
`,
			expectPRCheckout: true,
		},
		{
			name: "push trigger should add checkout (with condition)",
			workflowContent: `---
on:
  push:
    branches: [main]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test Workflow
Test workflow with push trigger.
`,
			expectPRCheckout: true, // Step is added, but condition prevents execution
		},
		{
			name: "no contents permission should NOT add checkout",
			workflowContent: `---
on:
  pull_request:
    types: [ready_for_review]
permissions:
  issues: write
engine: claude
tools:
  github: false
---

# Test Workflow
Test workflow without contents permission.
`,
			expectPRCheckout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "pr-checkout-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create workflows directory
			workflowsDir := filepath.Join(tempDir, ".github", "workflows")
			if err := os.MkdirAll(workflowsDir, 0755); err != nil {
				t.Fatalf("Failed to create workflows directory: %v", err)
			}

			// Write test workflow file
			workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Compile workflow
			compiler := NewCompiler(false, "", "test-version")
			if err := compiler.CompileWorkflow(workflowPath); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read generated lock file
			lockPath := filepath.Join(workflowsDir, "test-workflow.lock.yml")
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockStr := string(lockContent)

			// Check for PR checkout step
			hasPRCheckout := strings.Contains(lockStr, "Checkout PR branch")
			if hasPRCheckout != tt.expectPRCheckout {
				t.Errorf("Expected PR checkout step: %v, got: %v", tt.expectPRCheckout, hasPRCheckout)
			}

			// If PR checkout is expected, verify it uses actions/github-script
			if tt.expectPRCheckout {
				// Check for actions/github-script usage
				if !strings.Contains(lockStr, "uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd") {
					t.Error("PR checkout step should use actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd")
				}
				// Check for JavaScript code patterns
				if !strings.Contains(lockStr, "pullRequest.head.ref") {
					t.Error("PR checkout step should reference PR head ref in JavaScript")
				}
				if !strings.Contains(lockStr, "exec.exec") {
					t.Error("PR checkout step should use exec.exec for git commands")
				}
				if !strings.Contains(lockStr, "checkout") {
					t.Error("PR checkout step should checkout the branch")
				}
			}
		})
	}
}

// TestPRCheckoutForAllPullRequestTypes verifies the conditional logic for PR checkout on all pull_request types
func TestPRCheckoutForAllPullRequestTypes(t *testing.T) {
	workflowContent := `---
on:
  pull_request:
    types: [ready_for_review, opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
---

# Test Workflow
Test workflow with pull_request triggers.
`

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "pr-checkout-logic-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write test workflow file
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test-version")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read generated lock file
	lockPath := filepath.Join(workflowsDir, "test-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	// Verify the checkout uses actions/github-script
	if !strings.Contains(lockStr, "uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd") {
		t.Error("Expected PR checkout to use actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd")
	}

	// Verify JavaScript code patterns
	expectedPatterns := []string{
		`pullRequest.head.ref`,
		`exec.exec`,
		`checkout`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(lockStr, pattern) {
			t.Errorf("Expected JavaScript pattern not found: %s", pattern)
		}
	}
}
