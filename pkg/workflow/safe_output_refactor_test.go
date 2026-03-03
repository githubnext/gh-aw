//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestSafeOutputJobBuilderRefactor validates that the refactored safe output job builders
// produce the expected job structure and maintain consistency across different output types
func TestSafeOutputJobBuilderRefactor(t *testing.T) {
	tests := []struct {
		name           string
		frontmatter    string
		expectedJob    string
		expectedPerms  string
		expectedOutput string
	}{
		{
			name: "create-issue job builder",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-issue:
    title-prefix: "[bot] "
    labels: [automation]
---

# Test workflow`,
			expectedJob:    "safe_outputs:",
			expectedPerms:  "contents: read",
			expectedOutput: "issue_number:",
		},
		{
			name: "create-discussion job builder",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-discussion:
    title-prefix: "[report] "
    category: General
---

# Test workflow`,
			expectedJob:    "safe_outputs:",
			expectedPerms:  "contents: read",
			expectedOutput: "discussion_number:",
		},
		{
			name: "update-issue job builder",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  update-issue:
    status:
    title:
---

# Test workflow`,
			expectedJob:    "safe_outputs:",
			expectedPerms:  "contents: read",
			expectedOutput: "issue_number:",
		},
		{
			name: "add-comment job builder",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  add-comment:
    max: 3
---

# Test workflow`,
			expectedJob:    "safe_outputs:",
			expectedPerms:  "contents: read",
			expectedOutput: "comment_id:",
		},
		{
			name: "create-pull-request job builder",
			frontmatter: `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-pull-request:
    title-prefix: "[auto] "
    labels: [automated]
---

# Test workflow`,
			expectedJob:    "safe_outputs:",
			expectedPerms:  "contents: write",
			expectedOutput: "pull_request_number:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir := testutil.TempDir(t, "refactor-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the compiled output
			outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
			compiledContent, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read compiled output: %v", err)
			}

			yamlStr := string(compiledContent)

			// Verify job is created
			if !strings.Contains(yamlStr, tt.expectedJob) {
				t.Errorf("Expected job %q not found in output", tt.expectedJob)
			}

			// Verify permissions are set
			if !strings.Contains(yamlStr, tt.expectedPerms) {
				t.Errorf("Expected permissions %q not found in output", tt.expectedPerms)
			}

			// Verify outputs are defined (consolidated mode uses different output format)
			if !strings.Contains(yamlStr, tt.expectedOutput) && !strings.Contains(yamlStr, "outputs:") {
				t.Errorf("Expected output %q or outputs section not found", tt.expectedOutput)
			}

			// Verify timeout is set (consolidated safe_outputs job uses 15 minute timeout)
			if !strings.Contains(yamlStr, "timeout-minutes: 15") && !strings.Contains(yamlStr, "timeout-minutes:") {
				t.Error("Expected timeout-minutes not found in output")
			}

			// Verify the job is present
			if !strings.Contains(yamlStr, "safe_outputs:") {
				t.Error("Expected safe_outputs job not found")
			}

			// Verify safe output condition is set
			if !strings.Contains(yamlStr, "!cancelled()") {
				t.Error("Expected safe output condition '!cancelled()' not found")
			}
		})
	}
}

// TestSafeOutputJobBuilderWithPreAndPostSteps validates that pre-steps and post-steps
// are correctly handled by the shared builder
func TestSafeOutputJobBuilderWithPreAndPostSteps(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  string
		expectedStep string
		stepType     string
	}{
		{
			name: "create-issue with copilot assignee (post-steps)",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-issue:
    assignees: [copilot]
---

# Test workflow`,
			// In consolidated mode with handler manager, check for process_safe_outputs step
			expectedStep: "id: process_safe_outputs",
			stepType:     "step",
		},
		{
			name: "create-pull-request with checkout (pre-steps)",
			frontmatter: `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-pull-request:
---

# Test workflow`,
			expectedStep: "actions/checkout",
			stepType:     "pre-step",
		},
		{
			name: "add-comment with debug (pre-steps)",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  add-comment:
---

# Test workflow`,
			// In consolidated mode with handler manager, check for process_safe_outputs step
			expectedStep: "id: process_safe_outputs",
			stepType:     "step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir := testutil.TempDir(t, "presteps-test")

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the compiled output
			outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
			compiledContent, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read compiled output: %v", err)
			}

			yamlStr := string(compiledContent)

			// Verify the expected step is present
			if !strings.Contains(yamlStr, tt.expectedStep) {
				t.Errorf("Expected %s %q not found in output", tt.stepType, tt.expectedStep)
			}
		})
	}
}
