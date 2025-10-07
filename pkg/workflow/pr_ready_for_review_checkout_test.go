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

			// If PR checkout is expected, verify the conditional logic
			if tt.expectPRCheckout {
				// New simpler condition: just check for github.event.pull_request
				if !strings.Contains(lockStr, "github.event.pull_request") {
					t.Error("PR checkout step should check for github.event.pull_request")
				}
				if !strings.Contains(lockStr, "github.event.pull_request.head.ref") {
					t.Error("PR checkout step should reference PR head ref")
				}
				if !strings.Contains(lockStr, "git fetch origin") {
					t.Error("PR checkout step should fetch from origin")
				}
				if !strings.Contains(lockStr, "git checkout") {
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

	// Verify the conditional structure uses simple PR context check
	if !strings.Contains(lockStr, "github.event.pull_request") {
		t.Error("Expected condition 'github.event.pull_request' not found")
	}

	// Verify PR branch reference
	expectedPRLogic := []string{
		`PR_BRANCH="${{ github.event.pull_request.head.ref }}"`,
		`git fetch origin "$PR_BRANCH"`,
		`git checkout "$PR_BRANCH"`,
	}

	for _, logic := range expectedPRLogic {
		if !strings.Contains(lockStr, logic) {
			t.Errorf("Expected PR logic not found: %s", logic)
		}
	}
}
