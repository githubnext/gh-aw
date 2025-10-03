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
				// We can check this by looking for conditions that use "contains(" without "github.event_name"
				// Handle both single-line and multi-line YAML conditions
				lines := strings.Split(lockContentStr, "\n")
				foundSimpleCommandCondition := false

				for i, line := range lines {
					// Check for single-line if condition
					if strings.Contains(line, "if:") && strings.Contains(line, "contains(") && !strings.Contains(line, "github.event_name") {
						foundSimpleCommandCondition = true
						break
					}
					// Check for multi-line if condition (if: > or if: | format)
					if strings.Contains(line, "if:") && (strings.Contains(line, ">") || strings.Contains(line, "|")) {
						// Check the following lines for contains() without github.event_name
						for j := i + 1; j < len(lines) && strings.TrimSpace(lines[j]) != ""; j++ {
							nextLine := lines[j]
							if strings.Contains(nextLine, "contains(") && !strings.Contains(nextLine, "github.event_name") {
								foundSimpleCommandCondition = true
								break
							}
							// Stop if we hit the next YAML key (starts without indentation)
							if len(nextLine) > 0 && nextLine[0] != ' ' && nextLine[0] != '\t' {
								break
							}
						}
						if foundSimpleCommandCondition {
							break
						}
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

// TestCommandEventsFiltering tests that the events field filters which events the command is active on
func TestCommandEventsFiltering(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-command-events-filtering-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                 string
		frontmatter          string
		filename             string
		expectedEvents       []string // Events that should be in the generated workflow
		unexpectedEvents     []string // Events that should NOT be in the generated workflow
		expectedBodyChecks   []string // Body properties that should be checked
		unexpectedBodyChecks []string // Body properties that should NOT be checked
	}{
		{
			name: "command with events: [issue]",
			frontmatter: `---
on:
  command:
    name: issue-bot
    events: [issue]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:             "command-issue-only.md",
			expectedEvents:       []string{"issues:"},
			unexpectedEvents:     []string{"issue_comment:", "pull_request:", "pull_request_review_comment:"},
			expectedBodyChecks:   []string{"github.event.issue.body"},
			unexpectedBodyChecks: []string{"github.event.comment.body", "github.event.pull_request.body"},
		},
		{
			name: "command with events: [issue, comment]",
			frontmatter: `---
on:
  command:
    name: dual-bot
    events: [issue, comment]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:             "command-issue-comment.md",
			expectedEvents:       []string{"issues:", "issue_comment:"},
			unexpectedEvents:     []string{"pull_request:", "pull_request_review_comment:"},
			expectedBodyChecks:   []string{"github.event.issue.body", "github.event.comment.body"},
			unexpectedBodyChecks: []string{"github.event.pull_request.body"},
		},
		{
			name: "command with events: '*' (all events)",
			frontmatter: `---
on:
  command:
    name: all-bot
    events: "*"
tools:
  github:
    allowed: [list_issues]
---`,
			filename:       "command-all-events.md",
			expectedEvents: []string{"issues:", "issue_comment:", "pull_request:", "pull_request_review_comment:"},
			expectedBodyChecks: []string{"github.event.issue.body", "github.event.comment.body", 
				"github.event.pull_request.body"},
		},
		{
			name: "command with events: [pr]",
			frontmatter: `---
on:
  command:
    name: pr-bot
    events: [pr]
tools:
  github:
    allowed: [list_pull_requests]
---`,
			filename:             "command-pr-only.md",
			expectedEvents:       []string{"pull_request:"},
			unexpectedEvents:     []string{"issues:", "issue_comment:", "pull_request_review_comment:"},
			expectedBodyChecks:   []string{"github.event.pull_request.body"},
			unexpectedBodyChecks: []string{"github.event.issue.body", "github.event.comment.body"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Command Events Filtering

This test validates that command events filtering works correctly.
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

			// Read the compiled workflow
			lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			// Extract the "on:" section to check for events (not permissions)
			onSectionStart := strings.Index(lockContentStr, "on:")
			onSectionEnd := strings.Index(lockContentStr[onSectionStart:], "\npermissions:")
			if onSectionEnd == -1 {
				onSectionEnd = strings.Index(lockContentStr[onSectionStart:], "\nconcurrency:")
			}
			if onSectionEnd == -1 {
				onSectionEnd = strings.Index(lockContentStr[onSectionStart:], "\njobs:")
			}
			onSection := ""
			if onSectionEnd > 0 {
				onSection = lockContentStr[onSectionStart : onSectionStart+onSectionEnd]
			} else {
				onSection = lockContentStr[onSectionStart:]
			}

			// Check for expected events in the "on:" section only
			for _, expectedEvent := range tt.expectedEvents {
				if !strings.Contains(onSection, expectedEvent) {
					t.Errorf("Expected to find event '%s' in 'on:' section, but not found.\nOn section:\n%s", expectedEvent, onSection)
				}
			}

			// Check for unexpected events in the "on:" section only
			for _, unexpectedEvent := range tt.unexpectedEvents {
				if strings.Contains(onSection, unexpectedEvent) {
					t.Errorf("Did not expect to find event '%s' in 'on:' section, but found it.\nOn section:\n%s", unexpectedEvent, onSection)
				}
			}

			// Check for expected body checks in the if condition
			for _, expectedCheck := range tt.expectedBodyChecks {
				if !strings.Contains(lockContentStr, expectedCheck) {
					t.Errorf("Expected to find body check '%s' in generated workflow", expectedCheck)
				}
			}

			// Check for unexpected body checks
			for _, unexpectedCheck := range tt.unexpectedBodyChecks {
				if strings.Contains(lockContentStr, unexpectedCheck) {
					t.Errorf("Did not expect to find body check '%s' in generated workflow", unexpectedCheck)
				}
			}
		})
	}
}
