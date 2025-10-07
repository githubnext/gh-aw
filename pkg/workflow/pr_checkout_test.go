package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			name: "push trigger should add PR checkout (with runtime condition)",
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
			expectPRCheckout: true, // Step is added but runtime condition prevents execution
			expectPRPrompt:   false,
		},
		{
			name: "pull_request trigger should add PR checkout",
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
			expectPRCheckout: true, // Step is added and will execute for PR events
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

			// Check for PR checkout step (now uses JavaScript)
			hasPRCheckout := strings.Contains(lockStr, "Checkout PR branch")
			if hasPRCheckout != tt.expectPRCheckout {
				t.Errorf("Expected PR checkout step: %v, got: %v", tt.expectPRCheckout, hasPRCheckout)
			}

			// Check for PR context prompt
			hasPRPrompt := strings.Contains(lockStr, "Current Branch Context")
			if hasPRPrompt != tt.expectPRPrompt {
				t.Errorf("Expected PR context prompt: %v, got: %v", tt.expectPRPrompt, hasPRPrompt)
			}

			// If PR checkout is expected, verify it uses JavaScript
			if tt.expectPRCheckout {
				if !strings.Contains(lockStr, "uses: actions/github-script@v8") {
					t.Error("PR checkout step should use actions/github-script@v8")
				}
				if !strings.Contains(lockStr, "pullRequest") {
					t.Error("PR checkout step should reference pullRequest in JavaScript")
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

	// Verify the checkout step uses actions/github-script
	if !strings.Contains(lockStr, "uses: actions/github-script@v8") {
		t.Error("Expected PR checkout to use actions/github-script@v8")
	}

	// Verify JavaScript code handles PR checkout
	expectedPatterns := []string{
		"pullRequest.head.ref",
		"execSync",
		"git checkout",
		"gh pr checkout",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(lockStr, pattern) {
			t.Errorf("Expected JavaScript pattern not found: %s", pattern)
		}
	}
}
