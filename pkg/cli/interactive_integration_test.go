package cli

import (
	"strings"
	"testing"
)

func TestCreateWorkflowInteractively_Integration(t *testing.T) {
	// Test that the function exists and has correct signature
	// Since we can't actually test the interactive flow in automated tests,
	// we just verify the function signature and basic setup

	// This should be callable (but will fail due to no interactive terminal)
	err := CreateWorkflowInteractively("test-workflow", true, true)

	// We expect this to fail because there's no terminal for survey to use
	if err == nil {
		t.Error("Expected error when no terminal is available for interactive prompts")
	}

	// Error should be from survey, not from our code structure
	if err != nil && err.Error() == "" {
		t.Error("Expected descriptive error message")
	}
}

func TestInteractiveWorkflowBuilder_Validation(t *testing.T) {
	// Test that the builder struct can be created and used
	builder := &InteractiveWorkflowBuilder{
		WorkflowName:  "test",
		Trigger:       "workflow_dispatch",
		Engine:        "claude",
		Tools:         []string{"github"},
		SafeOutputs:   []string{"create-issue"},
		Intent:        "Test workflow",
		NetworkAccess: "defaults",
	}

	if builder.WorkflowName != "test" {
		t.Error("Workflow name not set correctly")
	}

	// Test content generation
	content := builder.generateWorkflowContent()
	if content == "" {
		t.Error("Generated content should not be empty")
	}

	// Verify basic structure
	if !strings.Contains(content, "test") {
		t.Error("Content should contain workflow name")
	}
}
