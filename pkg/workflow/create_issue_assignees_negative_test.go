package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAssigneeStepsOnlyWithCreateIssue verifies that assignee steps are ONLY generated
// when safe-outputs.create-issue is configured, not with other safe-outputs
func TestAssigneeStepsOnlyWithCreateIssue(t *testing.T) {
	tests := []struct {
		name               string
		frontmatter        string
		expectAssigneeStep bool
		description        string
	}{
		{
			name: "create-issue with assignees should have assignee steps",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-issue:
    assignees: [copilot]
---

# Test workflow`,
			expectAssigneeStep: true,
			description:        "Assignee steps should be generated when create-issue is configured with assignees",
		},
		{
			name: "create-issue without assignees should NOT have assignee steps",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-issue:
    title-prefix: "[test] "
---

# Test workflow`,
			expectAssigneeStep: false,
			description:        "Assignee steps should NOT be generated when create-issue has no assignees",
		},
		{
			name: "add-comment without create-issue should NOT have assignee steps",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
safe-outputs:
  add-comment:
    max: 1
---

# Test workflow`,
			expectAssigneeStep: false,
			description:        "Assignee steps should NOT be generated when only add-comment is configured",
		},
		{
			name: "create-discussion without create-issue should NOT have assignee steps",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-discussion:
    category: "general"
---

# Test workflow`,
			expectAssigneeStep: false,
			description:        "Assignee steps should NOT be generated when only create-discussion is configured",
		},
		{
			name: "create-pull-request without create-issue should NOT have issue assignee steps",
			frontmatter: `---
on: push
permissions:
  contents: read
engine: copilot
safe-outputs:
  create-pull-request:
---

# Test workflow`,
			expectAssigneeStep: false,
			description:        "Issue assignee steps should NOT be generated when only create-pull-request is configured",
		},
		{
			name: "multiple safe-outputs but no create-issue should NOT have assignee steps",
			frontmatter: `---
on: issues
permissions:
  contents: read
engine: copilot
safe-outputs:
  add-comment:
    max: 1
  create-discussion:
    category: "general"
  add-labels:
---

# Test workflow`,
			expectAssigneeStep: false,
			description:        "Assignee steps should NOT be generated when multiple safe-outputs are configured but none is create-issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "assignee-negative-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the compiled output
			outputFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
			compiledContent, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read compiled output: %v", err)
			}

			yamlStr := string(compiledContent)

			// Check for assignee steps
			hasAssigneeStep := strings.Contains(yamlStr, "Assign issue to")

			if tt.expectAssigneeStep && !hasAssigneeStep {
				t.Errorf("%s: Expected assignee step to be present, but it was not found", tt.description)
			}

			if !tt.expectAssigneeStep && hasAssigneeStep {
				t.Errorf("%s: Expected NO assignee step, but one was found", tt.description)
			}
		})
	}
}

// TestCopilotAssigneeSpecialHandling verifies that "copilot" assignee is mapped to "@copilot"
func TestCopilotAssigneeSpecialHandling(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with "copilot" and other users as assignees
	// After the change, only copilot gets an assignee step, other users are ignored
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				Assignees: []string{"copilot", "user1"},
			},
		},
	}

	job, err := c.buildCreateOutputIssueJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create issue job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Verify that "copilot" is mapped to "@copilot"
	if !strings.Contains(stepsContent, `ASSIGNEE: "@copilot"`) {
		t.Error("Expected copilot to be mapped to @copilot")
	}

	// Verify that copilot assignee step is present
	if !strings.Contains(stepsContent, "Assign issue to copilot") {
		t.Error("Expected assignee step for copilot")
	}

	// After the change, only copilot gets an assignee step
	// user1 should NOT have an assignee step
	if strings.Contains(stepsContent, "Assign issue to user1") {
		t.Error("Did not expect assignee step for user1 (only copilot should have a step)")
	}
}
