package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPRReadyForReviewCheckout verifies that PR branch checkout is added for pull_request ready_for_review events
func TestPRReadyForReviewCheckout(t *testing.T) {
	tests := []struct {
		name                         string
		workflowContent              string
		expectReadyForReviewCheckout bool
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
			expectReadyForReviewCheckout: true,
		},
		{
			name: "pull_request with opened should NOT add ready_for_review checkout",
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
			expectReadyForReviewCheckout: false,
		},
		{
			name: "push trigger should NOT add ready_for_review checkout",
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
			expectReadyForReviewCheckout: false,
		},
		{
			name: "no contents permission should NOT add ready_for_review checkout",
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
			expectReadyForReviewCheckout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "pr-ready-for-review-test")
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

			// Check for ready_for_review checkout step
			hasReadyForReviewCheckout := strings.Contains(lockStr, "Checkout PR branch for ready_for_review")
			if hasReadyForReviewCheckout != tt.expectReadyForReviewCheckout {
				t.Errorf("Expected ready_for_review checkout step: %v, got: %v", tt.expectReadyForReviewCheckout, hasReadyForReviewCheckout)
			}

			// If ready_for_review checkout is expected, verify the conditional logic
			if tt.expectReadyForReviewCheckout {
				if !strings.Contains(lockStr, "github.event_name == 'pull_request'") {
					t.Error("Ready for review checkout step should check for pull_request event")
				}
				if !strings.Contains(lockStr, "github.event.action == 'ready_for_review'") {
					t.Error("Ready for review checkout step should check for ready_for_review action")
				}
				if !strings.Contains(lockStr, "github.event.pull_request.head.ref") {
					t.Error("Ready for review checkout step should reference PR head ref")
				}
				if !strings.Contains(lockStr, "git fetch origin") {
					t.Error("Ready for review checkout step should fetch from origin")
				}
				if !strings.Contains(lockStr, "git checkout") {
					t.Error("Ready for review checkout step should checkout the branch")
				}
			}
		})
	}
}

// TestPRReadyForReviewCheckoutConditionalLogic verifies the conditional logic for ready_for_review checkout
func TestPRReadyForReviewCheckoutConditionalLogic(t *testing.T) {
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
	tempDir, err := os.MkdirTemp("", "pr-ready-for-review-logic-test")
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

	// Verify the conditional structure includes the event type and action
	expectedConditions := []string{
		"github.event_name == 'pull_request'",
		"github.event.action == 'ready_for_review'",
	}

	for _, condition := range expectedConditions {
		if !strings.Contains(lockStr, condition) {
			t.Errorf("Expected condition not found: %s", condition)
		}
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
