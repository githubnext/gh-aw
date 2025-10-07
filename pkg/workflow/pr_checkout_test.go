package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestPRBranchCheckout verifies that PR branch checkout is added for comment triggers
func TestPRBranchCheckout(t *testing.T) {
	tests := []struct {
		name             string
		workflowContent  string
		expectPRCheckout bool
		expectPRPrompt   bool
	}{
		{
			name: "issue_comment trigger should add PR checkout",
			workflowContent: `---
on:
  issue_comment:
    types: [created]
permissions:
  contents: read
engine: claude
---

# Test Workflow
Test workflow with issue_comment trigger.
`,
			expectPRCheckout: true,
			expectPRPrompt:   true,
		},
		{
			name: "pull_request_review_comment trigger should add PR checkout",
			workflowContent: `---
on:
  pull_request_review_comment:
    types: [created]
permissions:
  contents: read
engine: claude
---

# Test Workflow
Test workflow with pull_request_review_comment trigger.
`,
			expectPRCheckout: true,
			expectPRPrompt:   true,
		},
		{
			name: "multiple comment triggers should add PR checkout",
			workflowContent: `---
on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
permissions:
  contents: read
engine: claude
---

# Test Workflow
Test workflow with multiple comment triggers.
`,
			expectPRCheckout: true,
			expectPRPrompt:   true,
		},
		{
			name: "command trigger should add PR checkout (expands to comments)",
			workflowContent: `---
on:
  command:
    name: test-bot
permissions:
  contents: read
engine: claude
---

# Test Workflow
Test workflow with command trigger.
`,
			expectPRCheckout: true,
			expectPRPrompt:   true,
		},
		{
			name: "push trigger should NOT add PR checkout",
			workflowContent: `---
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: claude
---

# Test Workflow
Test workflow with push trigger only.
`,
			expectPRCheckout: false,
			expectPRPrompt:   false,
		},
		{
			name: "pull_request trigger should NOT add PR checkout",
			workflowContent: `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
engine: claude
---

# Test Workflow
Test workflow with pull_request trigger only.
`,
			expectPRCheckout: false,
			expectPRPrompt:   false,
		},
		{
			name: "no contents permission should NOT add PR checkout",
			workflowContent: `---
on:
  issue_comment:
    types: [created]
permissions:
  issues: write
engine: claude
---

# Test Workflow
Test workflow without contents read permission.
`,
			expectPRCheckout: false,
			expectPRPrompt:   false,
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
			workflowsDir := filepath.Join(tempDir, constants.GetWorkflowDir())
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
			hasPRCheckout := strings.Contains(lockStr, "Checkout PR branch if applicable")
			if hasPRCheckout != tt.expectPRCheckout {
				t.Errorf("Expected PR checkout step: %v, got: %v", tt.expectPRCheckout, hasPRCheckout)
			}

			// Check for PR context prompt
			hasPRPrompt := strings.Contains(lockStr, "Current Branch Context")
			if hasPRPrompt != tt.expectPRPrompt {
				t.Errorf("Expected PR context prompt: %v, got: %v", tt.expectPRPrompt, hasPRPrompt)
			}

			// If PR checkout is expected, verify the conditional logic
			if tt.expectPRCheckout {
				if !strings.Contains(lockStr, "github.event_name == 'issue_comment'") {
					t.Error("PR checkout step should check for issue_comment event")
				}
				if !strings.Contains(lockStr, "github.event.issue.pull_request != null") {
					t.Error("PR checkout step should check for pull_request field in issue")
				}
				if !strings.Contains(lockStr, "gh pr checkout") {
					t.Error("PR checkout step should use gh pr checkout command")
				}
			}

			// If PR prompt is expected, verify key content
			if tt.expectPRPrompt {
				if !strings.Contains(lockStr, "automatically checked out to the PR's branch") {
					t.Error("PR context prompt should explain the branch context")
				}
				if !strings.Contains(lockStr, "Current Branch Context") {
					t.Error("PR context prompt should include branch context heading")
				}
			}
		})
	}
}

// TestPRCheckoutConditionalLogic verifies the conditional logic for PR checkout
func TestPRCheckoutConditionalLogic(t *testing.T) {
	workflowContent := `---
on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
permissions:
  contents: read
engine: claude
---

# Test Workflow
Test workflow with multiple comment triggers.
`

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "pr-checkout-logic-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create workflows directory
	workflowsDir := filepath.Join(tempDir, constants.GetWorkflowDir())
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

	// Verify the conditional structure includes all event types
	expectedConditions := []string{
		"github.event_name == 'issue_comment'",
		"github.event.issue.pull_request != null",
		"github.event_name == 'pull_request_review_comment'",
		"github.event_name == 'pull_request_review'",
	}

	for _, condition := range expectedConditions {
		if !strings.Contains(lockStr, condition) {
			t.Errorf("Expected condition not found: %s", condition)
		}
	}

	// Verify PR number determination logic
	expectedPRLogic := []string{
		`if [ "${{ github.event_name }}" = "issue_comment" ]`,
		`PR_NUMBER="${{ github.event.issue.number }}"`,
		`elif [ "${{ github.event_name }}" = "pull_request_review_comment" ]`,
		`PR_NUMBER="${{ github.event.pull_request.number }}"`,
	}

	for _, logic := range expectedPRLogic {
		if !strings.Contains(lockStr, logic) {
			t.Errorf("Expected PR logic not found: %s", logic)
		}
	}

	// Verify environment variable for gh token
	if !strings.Contains(lockStr, "GH_TOKEN: ${{ github.token }}") {
		t.Error("Expected GH_TOKEN environment variable for gh pr checkout")
	}
}
