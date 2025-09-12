package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestActionsWritePermissionForSelfCancellation tests that actions: write permission
// is added to jobs that include team member checks for self-cancellation functionality
func TestActionsWritePermissionForSelfCancellation(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "actions-write-permission-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name               string
		frontmatter        string
		filename           string
		expectActionsWrite bool
		jobName            string
		description        string
	}{
		{
			name: "command workflow task job should have actions: write",
			frontmatter: `---
on:
  command:
    name: test-bot
tools:
  github:
    allowed: [list_issues]
---

# Command Workflow
Test workflow with command trigger.`,
			filename:           "command-workflow.md",
			expectActionsWrite: true,
			jobName:            "task",
			description:        "Task job should have actions: write for self-cancellation",
		},
		{
			name: "push workflow main job should have actions: write",
			frontmatter: `---
on:
  push:
    branches: [main]
tools:
  github:
    allowed: [list_issues]
---

# Push Workflow
Test workflow with push trigger that needs permission checks.`,
			filename:           "push-workflow.md",
			expectActionsWrite: true,
			jobName:            "push-workflow",
			description:        "Main job should have actions: write for permission checks",
		},
		{
			name: "workflow_dispatch should not have actions: write",
			frontmatter: `---
on:
  workflow_dispatch:
tools:
  github:
    allowed: [list_issues]
---

# Workflow Dispatch
Test workflow with safe event.`,
			filename:           "workflow-dispatch.md",
			expectActionsWrite: false,
			jobName:            "",
			description:        "Safe events should not need actions: write permission",
		},
		{
			name: "schedule workflow should not have actions: write",
			frontmatter: `---
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
    allowed: [list_issues]
---

# Schedule Workflow
Test workflow with schedule trigger.`,
			filename:           "schedule-workflow.md",
			expectActionsWrite: false,
			jobName:            "",
			description:        "Schedule events should not need actions: write permission",
		},
		{
			name: "roles: all should not have actions: write",
			frontmatter: `---
on:
  push:
    branches: [main]
roles: all
tools:
  github:
    allowed: [list_issues]
---

# Unrestricted Workflow
Test workflow with unrestricted access.`,
			filename:           "unrestricted-workflow.md",
			expectActionsWrite: false,
			jobName:            "",
			description:        "Unrestricted workflows should not need actions: write permission",
		},
		{
			name: "main job should have actions: write when no task job",
			frontmatter: `---
on:
  issues:
    types: [opened]
tools:
  github:
    allowed: [list_issues]
---

# Issues Workflow
Test workflow with permission checks but no task job.`,
			filename:           "issues-workflow.md",
			expectActionsWrite: true,
			jobName:            "issues-workflow",
			description:        "Main job should have actions: write for permission checks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(testFile, []byte(tt.frontmatter), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockContentStr := string(lockContent)

			if tt.expectActionsWrite {
				// Check that the specified job exists
				if !strings.Contains(lockContentStr, "  "+tt.jobName+":") {
					t.Fatalf("Job '%s' not found in workflow", tt.jobName)
				}

				// Check for actions: write permission anywhere in the workflow (should be in the right job)
				if !strings.Contains(lockContentStr, "permissions:") || !strings.Contains(lockContentStr, "actions: write") {
					t.Errorf("%s: Expected 'actions: write' permission in workflow but not found", tt.description)
				}

				// Check that setCancelled function is present
				if !strings.Contains(lockContentStr, "async function setCancelled(message)") {
					t.Errorf("%s: Expected custom setCancelled function but not found", tt.description)
				}

				// Check that github.rest.actions.cancelWorkflowRun is called
				if !strings.Contains(lockContentStr, "github.rest.actions.cancelWorkflowRun") {
					t.Errorf("%s: Expected self-cancellation API call but not found", tt.description)
				}

			} else {
				// Check that actions: write permission is not present
				if strings.Contains(lockContentStr, "actions: write") {
					t.Errorf("%s: Did not expect 'actions: write' permission but found it", tt.description)
				}
			}
		})
	}
}

// TestMainJobActionsWritePermission tests that the main job gets actions: write
// permission when permission checks are needed and no task job exists
func TestMainJobActionsWritePermission(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "main-job-actions-write-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	testContent := `---
on:
  issues:
    types: [opened]
tools:
  github:
    allowed: [list_issues]
---

# Issues Workflow
This workflow needs permission checks but has no task job.`

	// Create test file
	testFile := filepath.Join(tmpDir, "issues-workflow.md")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContentStr := string(lockContent)

	// Check that the main job has actions: write permission
	// Look for the main job (should be "issues-workflow" based on filename)
	if !strings.Contains(lockContentStr, "  issues-workflow:") {
		t.Fatal("Main job 'issues-workflow' not found in workflow")
	}

	// Check for actions: write permission anywhere in the workflow
	if !strings.Contains(lockContentStr, "permissions:") || !strings.Contains(lockContentStr, "actions: write") {
		t.Error("Expected 'actions: write' permission in main job but not found")
	}

	// Check that permission check step is present
	if !strings.Contains(lockContentStr, "Check team membership for workflow") {
		t.Error("Expected team membership check step in main job but not found")
	}

	// Check that setCancelled function is present
	if !strings.Contains(lockContentStr, "async function setCancelled(message)") {
		t.Error("Expected custom setCancelled function but not found")
	}
}
