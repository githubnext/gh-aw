package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestTeamMemberCheckForCommandWorkflows tests that team member checks are only added to command workflows
func TestTeamMemberCheckForCommandWorkflows(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-team-member-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                  string
		frontmatter           string
		filename              string
		expectTeamMemberCheck bool
	}{
		{
			name: "command workflow should include team member check",
			frontmatter: `---
on:
  command:
    name: test-bot
tools:
  github:
    allowed: [list_issues]
---

# Test Bot
Test workflow content.`,
			filename:              "command-workflow.md",
			expectTeamMemberCheck: true,
		},
		{
			name: "non-command workflow with unsafe event should include team member check",
			frontmatter: `---
on:
  push:
    branches: [main]
tools:
  github:
    allowed: [list_issues]
---

# Non-Alias Workflow
Test workflow content.`,
			filename:              "non-alias-workflow.md",
			expectTeamMemberCheck: true,
		},
		{
			name: "schedule workflow should not include team member check",
			frontmatter: `---
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
    allowed: [list_issues]
---

# Schedule Workflow
Test workflow content.`,
			filename:              "schedule-workflow.md",
			expectTeamMemberCheck: false,
		},
		{
			name: "command with other events should include team member check",
			frontmatter: `---
on:
  command:
    name: multi-bot
  workflow_dispatch:
tools:
  github:
    allowed: [list_issues]
---

# Multi-trigger Workflow
Test workflow content.`,
			filename:              "multi-trigger-workflow.md",
			expectTeamMemberCheck: true,
		},
		{
			name: "command with push events should have conditional team member check",
			frontmatter: `---
on:
  command:
    name: docs-bot
  push:
    branches: [main]
  workflow_dispatch:
tools:
  github:
    allowed: [list_issues]
---

# Conditional Team Check Workflow
Test workflow content.`,
			filename:              "conditional-team-check-workflow.md",
			expectTeamMemberCheck: true,
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

			// Check for team member check (now in pre_activation job)
			hasTeamMemberCheck := strings.Contains(lockContentStr, "Check team membership for command workflow") ||
				strings.Contains(lockContentStr, constants.CheckMembershipJobName+":")

			if tt.expectTeamMemberCheck {
				if !hasTeamMemberCheck {
					t.Errorf("Expected team member check in command workflow but not found")
				}
				// Check for the specific failure message (updated for new implementation)
				if !strings.Contains(lockContentStr, "Access denied: User") {
					t.Errorf("Expected team member check failure message but not found")
				}
				// Note: As per comment feedback, the conditional if statement has been removed
				// since the JavaScript already tests membership and command filter is applied at job level
				// Verify that team member check no longer has unnecessary conditional logic
				if strings.Contains(lockContentStr, "if: contains(github.event.issue.body") {
					t.Errorf("Team member check should not have conditional if statement (per comment feedback)")
				}
				// Find the team member check section and ensure it doesn't have github.event_name logic
				teamMemberCheckStart := strings.Index(lockContentStr, "Check team membership for command workflow")
				if teamMemberCheckStart == -1 {
					// Look for the new pre_activation job structure
					teamMemberCheckStart = strings.Index(lockContentStr, constants.CheckMembershipJobName+":")
				}
				teamMemberCheckEnd := strings.Index(lockContentStr[teamMemberCheckStart:], "task:")
				if teamMemberCheckStart != -1 && teamMemberCheckEnd != -1 {
					teamMemberSection := lockContentStr[teamMemberCheckStart : teamMemberCheckStart+teamMemberCheckEnd]
					if strings.Contains(teamMemberSection, "github.event_name") {
						t.Errorf("Team member check section should not contain github.event_name logic")
					}
				}
			} else {
				if hasTeamMemberCheck {
					t.Errorf("Did not expect team member check in non-command workflow but found it")
				}
			}
		})
	}
}
