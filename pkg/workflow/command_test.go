package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEventAwareCommandConditions tests that command conditions are properly applied only to comment-related events
func TestEventAwareCommandConditions(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-event-aware-command-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                    string
		frontmatter             string
		filename                string
		expectedSimpleCondition bool // true if should use simple condition (command only)
		expectedEventAware      bool // true if should use event-aware condition (command + other events)
	}{
		{
			name: "command only should use simple condition",
			frontmatter: `---
on:
  command:
    name: simple-bot
tools:
  github:
    allowed: [list_issues]
---`,
			filename:                "simple-command.md",
			expectedSimpleCondition: true,
			expectedEventAware:      false,
		},
		{
			name: "command with push should use event-aware condition",
			frontmatter: `---
on:
  command:
    name: push-bot
  push:
    branches: [main]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:                "command-with-push.md",
			expectedSimpleCondition: false,
			expectedEventAware:      true,
		},
		{
			name: "command with schedule should use event-aware condition",
			frontmatter: `---
on:
  command:
    name: schedule-bot
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
    allowed: [list_issues]
---`,
			filename:                "command-with-schedule.md",
			expectedSimpleCondition: false,
			expectedEventAware:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Event-Aware Command Conditions

This test validates that command conditions are applied correctly based on event types.
`

			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			// Read the compiled workflow to check the if condition
			lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			if tt.expectedSimpleCondition {
				// Should contain simple command condition (no complex event_name logic in main job)
				expectedPattern := "contains(github.event.issue.body, '/"
				if !strings.Contains(lockContentStr, expectedPattern) {
					t.Errorf("Expected simple command condition containing '%s' but not found", expectedPattern)
				}

				// For simple command workflows, the main job condition should not contain github.event_name logic
				// We can check this by looking for any line with "if:" that contains both "contains(" and NOT "github.event_name"
				lines := strings.Split(lockContentStr, "\n")
				foundSimpleCommandCondition := false
				for _, line := range lines {
					if strings.Contains(line, "if:") && strings.Contains(line, "contains(") && !strings.Contains(line, "github.event_name") {
						foundSimpleCommandCondition = true
						break
					}
				}
				if !foundSimpleCommandCondition {
					t.Errorf("Expected to find simple command condition (contains without github.event_name) but not found")
				}
			}

			if tt.expectedEventAware {
				// Should contain event-aware condition with event_name checks (but not just in add_reaction job)
				expectedPattern := "github.event_name == 'issues'"
				if !strings.Contains(lockContentStr, expectedPattern) {
					t.Errorf("Expected event-aware condition containing '%s' but not found", expectedPattern)
				}

				// Should contain the complex condition with AND/OR logic
				expectedComplexPattern := "((github.event_name == 'issues'"
				if !strings.Contains(lockContentStr, expectedComplexPattern) {
					t.Errorf("Expected complex event-aware condition containing '%s' but not found", expectedComplexPattern)
				}

				// Should contain the OR for non-comment events
				expectedOrPattern := "!(github.event_name == 'issues'"
				if !strings.Contains(lockContentStr, expectedOrPattern) {
					t.Errorf("Expected event-aware condition with non-comment event clause containing '%s' but not found", expectedOrPattern)
				}
			}
		})
	}
}
