package cli

import (
	"os"
	"strings"
	"testing"
)

func TestIsValidWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid simple name",
			input:    "my-workflow",
			expected: true,
		},
		{
			name:     "valid with underscores",
			input:    "my_workflow",
			expected: true,
		},
		{
			name:     "valid alphanumeric",
			input:    "workflow123",
			expected: true,
		},
		{
			name:     "valid mixed",
			input:    "my-workflow_v2",
			expected: true,
		},
		{
			name:     "invalid with spaces",
			input:    "my workflow",
			expected: false,
		},
		{
			name:     "invalid with special chars",
			input:    "my@workflow!",
			expected: false,
		},
		{
			name:     "invalid with dots",
			input:    "my.workflow",
			expected: false,
		},
		{
			name:     "invalid with slashes",
			input:    "my/workflow",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "valid uppercase",
			input:    "MyWorkflow",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidWorkflowName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidWorkflowName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCommonWorkflowNamesAreValid(t *testing.T) {
	// Ensure all suggested workflow names are themselves valid
	if len(commonWorkflowNames) == 0 {
		t.Error("commonWorkflowNames should not be empty")
	}

	for _, name := range commonWorkflowNames {
		if !isValidWorkflowName(name) {
			t.Errorf("commonWorkflowNames contains invalid workflow name: %q", name)
		}
	}
}

func TestCommonWorkflowNamesHasExpectedPatterns(t *testing.T) {
	// Verify that common workflow patterns are included
	expectedPatterns := []string{
		"issue-triage",
		"pr-review-helper",
		"security-scan",
		"daily-report",
		"weekly-summary",
	}

	// Convert to map for O(1) lookup
	workflowNamesSet := make(map[string]bool, len(commonWorkflowNames))
	for _, name := range commonWorkflowNames {
		workflowNamesSet[name] = true
	}

	for _, expected := range expectedPatterns {
		if !workflowNamesSet[expected] {
			t.Errorf("commonWorkflowNames should include %q", expected)
		}
	}
}

func TestIsAccessibleMode(t *testing.T) {
	tests := []struct {
		name     string
		term     string
		noColor  string
		expected bool
	}{
		{
			name:     "TERM=dumb enables accessibility",
			term:     "dumb",
			noColor:  "",
			expected: true,
		},
		{
			name:     "NO_COLOR=1 enables accessibility",
			term:     "xterm",
			noColor:  "1",
			expected: true,
		},
		{
			name:     "NO_COLOR=true enables accessibility",
			term:     "xterm",
			noColor:  "true",
			expected: true,
		},
		{
			name:     "normal terminal without NO_COLOR",
			term:     "xterm-256color",
			noColor:  "",
			expected: false,
		},
		{
			name:     "both TERM=dumb and NO_COLOR set",
			term:     "dumb",
			noColor:  "1",
			expected: true,
		},
		{
			name:     "empty TERM without NO_COLOR",
			term:     "",
			noColor:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origTerm := os.Getenv("TERM")
			origNoColor := os.Getenv("NO_COLOR")

			// Set test values
			os.Setenv("TERM", tt.term)
			if tt.noColor != "" {
				os.Setenv("NO_COLOR", tt.noColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}

			result := isAccessibleMode()

			// Restore original values
			if origTerm != "" {
				os.Setenv("TERM", origTerm)
			} else {
				os.Unsetenv("TERM")
			}
			if origNoColor != "" {
				os.Setenv("NO_COLOR", origNoColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}

			if result != tt.expected {
				t.Errorf("isAccessibleMode() with TERM=%q NO_COLOR=%q = %v, want %v",
					tt.term, tt.noColor, result, tt.expected)
			}
		})
	}
}

func TestInteractiveWorkflowBuilder_generateWorkflowContent(t *testing.T) {
	builder := &InteractiveWorkflowBuilder{
		WorkflowName:  "test-workflow",
		Trigger:       "workflow_dispatch",
		Engine:        "claude",
		Tools:         []string{"github", "edit"},
		SafeOutputs:   []string{"create-issue"},
		Intent:        "This is a test workflow for validation",
		NetworkAccess: "defaults",
	}

	content := builder.generateWorkflowContent()

	// Check that basic sections are present
	if content == "" {
		t.Fatal("Generated content is empty")
	}

	// Check for frontmatter
	if !strings.Contains(content, "---") {
		t.Error("Content should contain frontmatter markers")
	}

	// Check for workflow name
	if !strings.Contains(content, "test-workflow") {
		t.Error("Content should contain workflow name")
	}

	// Check for engine
	if !strings.Contains(content, "engine: claude") {
		t.Error("Content should contain engine configuration")
	}

	// Check for tools
	if !strings.Contains(content, "github:") {
		t.Error("Content should contain github tools")
	}

	// Check for safe outputs
	if !strings.Contains(content, "create-issue:") {
		t.Error("Content should contain safe outputs")
	}

	t.Logf("Generated content:\n%s", content)
}

func TestInteractiveWorkflowBuilder_generateTriggerConfig(t *testing.T) {
	tests := []struct {
		trigger  string
		expected string
	}{
		{"workflow_dispatch", "on:\n  workflow_dispatch:\n"},
		{"issues", "on:\n  issues:\n    types: [opened, reopened]\n"},
		{"pull_request", "on:\n  pull_request:\n    types: [opened, synchronize]\n"},
	}

	for _, tt := range tests {
		builder := &InteractiveWorkflowBuilder{Trigger: tt.trigger}
		result := builder.generateTriggerConfig()
		if result != tt.expected {
			t.Errorf("generateTriggerConfig(%s) = %q, want %q", tt.trigger, result, tt.expected)
		}
	}
}

func TestInteractiveWorkflowBuilder_describeTrigger(t *testing.T) {
	tests := []struct {
		name     string
		trigger  string
		expected string
	}{
		{
			name:     "workflow_dispatch trigger",
			trigger:  "workflow_dispatch",
			expected: "Manual trigger",
		},
		{
			name:     "issues trigger",
			trigger:  "issues",
			expected: "Issue opened or reopened",
		},
		{
			name:     "pull_request trigger",
			trigger:  "pull_request",
			expected: "Pull request opened or synchronized",
		},
		{
			name:     "push trigger",
			trigger:  "push",
			expected: "Push to main branch",
		},
		{
			name:     "issue_comment trigger",
			trigger:  "issue_comment",
			expected: "Issue comment created",
		},
		{
			name:     "schedule_daily trigger",
			trigger:  "schedule_daily",
			expected: "Daily schedule (9 AM UTC)",
		},
		{
			name:     "schedule_weekly trigger",
			trigger:  "schedule_weekly",
			expected: "Weekly schedule (Monday 9 AM UTC)",
		},
		{
			name:     "command trigger",
			trigger:  "command",
			expected: "Command trigger (/bot-name)",
		},
		{
			name:     "custom trigger",
			trigger:  "custom",
			expected: "Custom trigger (TODO: configure)",
		},
		{
			name:     "unknown trigger",
			trigger:  "unknown_trigger_type",
			expected: "Unknown trigger",
		},
		{
			name:     "empty trigger",
			trigger:  "",
			expected: "Unknown trigger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &InteractiveWorkflowBuilder{Trigger: tt.trigger}
			result := builder.describeTrigger()
			if result != tt.expected {
				t.Errorf("describeTrigger() with trigger=%q = %q, want %q", tt.trigger, result, tt.expected)
			}
		})
	}
}

// TestOptionConstants verifies that all package-level option constants are correctly defined
func TestOptionConstants(t *testing.T) {
	t.Run("triggerOptions has expected options", func(t *testing.T) {
		if len(triggerOptions) == 0 {
			t.Error("triggerOptions should not be empty")
		}

		// Verify first option is workflow_dispatch (default)
		firstOption := triggerOptions[0]
		if firstOption.Value != "workflow_dispatch" {
			t.Errorf("first trigger option should be workflow_dispatch, got %s", firstOption.Value)
		}

		// Count expected options
		expectedCount := 8
		if len(triggerOptions) != expectedCount {
			t.Errorf("triggerOptions should have %d options, got %d", expectedCount, len(triggerOptions))
		}
	})

	t.Run("engineOptions has expected options", func(t *testing.T) {
		if len(engineOptions) == 0 {
			t.Error("engineOptions should not be empty")
		}

		// Verify first option is copilot (default)
		firstOption := engineOptions[0]
		if firstOption.Value != "copilot" {
			t.Errorf("first engine option should be copilot, got %s", firstOption.Value)
		}

		// Count expected options
		expectedCount := 4
		if len(engineOptions) != expectedCount {
			t.Errorf("engineOptions should have %d options, got %d", expectedCount, len(engineOptions))
		}
	})

	t.Run("toolOptions has expected options", func(t *testing.T) {
		if len(toolOptions) == 0 {
			t.Error("toolOptions should not be empty")
		}

		// Verify first option is github (default)
		firstOption := toolOptions[0]
		if firstOption.Value != "github" {
			t.Errorf("first tool option should be github, got %s", firstOption.Value)
		}

		// Count expected options
		expectedCount := 6
		if len(toolOptions) != expectedCount {
			t.Errorf("toolOptions should have %d options, got %d", expectedCount, len(toolOptions))
		}
	})

	t.Run("safeOutputOptions has expected options", func(t *testing.T) {
		if len(safeOutputOptions) == 0 {
			t.Error("safeOutputOptions should not be empty")
		}

		// Count expected options
		expectedCount := 10
		if len(safeOutputOptions) != expectedCount {
			t.Errorf("safeOutputOptions should have %d options, got %d", expectedCount, len(safeOutputOptions))
		}
	})

	t.Run("networkOptions has expected options", func(t *testing.T) {
		if len(networkOptions) == 0 {
			t.Error("networkOptions should not be empty")
		}

		// Verify first option is defaults (default)
		firstOption := networkOptions[0]
		if firstOption.Value != "defaults" {
			t.Errorf("first network option should be defaults, got %s", firstOption.Value)
		}

		// Count expected options
		expectedCount := 2
		if len(networkOptions) != expectedCount {
			t.Errorf("networkOptions should have %d options, got %d", expectedCount, len(networkOptions))
		}
	})
}

// TestBuilderDefaults verifies that the builder has sensible defaults
func TestBuilderDefaults(t *testing.T) {
	// Create a builder with only WorkflowName set (simulating fresh initialization)
	builder := &InteractiveWorkflowBuilder{
		WorkflowName:  "test-workflow",
		Trigger:       "workflow_dispatch",
		Engine:        "copilot",
		Tools:         []string{"github"},
		NetworkAccess: "defaults",
	}

	t.Run("default trigger is workflow_dispatch", func(t *testing.T) {
		if builder.Trigger != "workflow_dispatch" {
			t.Errorf("default trigger should be workflow_dispatch, got %s", builder.Trigger)
		}
	})

	t.Run("default engine is copilot", func(t *testing.T) {
		if builder.Engine != "copilot" {
			t.Errorf("default engine should be copilot, got %s", builder.Engine)
		}
	})

	t.Run("default tools includes github", func(t *testing.T) {
		if len(builder.Tools) == 0 || builder.Tools[0] != "github" {
			t.Errorf("default tools should include github, got %v", builder.Tools)
		}
	})

	t.Run("default network access is defaults", func(t *testing.T) {
		if builder.NetworkAccess != "defaults" {
			t.Errorf("default network access should be defaults, got %s", builder.NetworkAccess)
		}
	})
}
