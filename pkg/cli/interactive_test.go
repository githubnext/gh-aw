package cli

import (
	"strings"
	"testing"
)

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
