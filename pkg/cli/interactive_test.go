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
