package cli

import (
	"testing"
)

func TestWorkflowInfoTableRendering(t *testing.T) {
	// Create test workflow info
	workflows := []WorkflowInfo{
		{
			ID:          "ci-workflow",
			Name:        "CI Workflow",
			Description: "Continuous integration workflow",
			Path:        "workflows/ci-workflow.md",
		},
		{
			ID:          "deploy",
			Name:        "Deploy to Production",
			Description: "Deployment workflow for production environment",
			Path:        "workflows/deploy.md",
		},
		{
			ID:          "test-only",
			Name:        "Test Suite",
			Description: "",
			Path:        "workflows/test-only.md",
		},
	}

	// Build table configuration (mimicking handleRepoOnlySpec)
	var rows [][]string
	for _, workflow := range workflows {
		rows = append(rows, []string{workflow.ID, workflow.GetDisplayName()})
	}

	// Verify the rows were built correctly
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}

	// Verify first row
	if rows[0][0] != "ci-workflow" {
		t.Errorf("Row 0 ID: got %s, want ci-workflow", rows[0][0])
	}
	if rows[0][1] != "Continuous integration workflow" {
		t.Errorf("Row 0 Name/Desc: got %s, want 'Continuous integration workflow'", rows[0][1])
	}

	// Verify second row (has description, should use description)
	if rows[1][0] != "deploy" {
		t.Errorf("Row 1 ID: got %s, want deploy", rows[1][0])
	}
	if rows[1][1] != "Deployment workflow for production environment" {
		t.Errorf("Row 1 Name/Desc: got %s, want 'Deployment workflow for production environment'", rows[1][1])
	}

	// Verify third row (no description, should use name)
	if rows[2][0] != "test-only" {
		t.Errorf("Row 2 ID: got %s, want test-only", rows[2][0])
	}
	if rows[2][1] != "Test Suite" {
		t.Errorf("Row 2 Name/Desc: got %s, want 'Test Suite'", rows[2][1])
	}
}

func TestWorkflowInfoStruct(t *testing.T) {
	// Test WorkflowInfo struct creation and fields
	wf := WorkflowInfo{
		ID:          "my-workflow",
		Name:        "My Workflow Name",
		Description: "My workflow description",
		Path:        "workflows/my-workflow.md",
	}

	if wf.ID != "my-workflow" {
		t.Errorf("ID: got %s, want my-workflow", wf.ID)
	}

	if wf.Name != "My Workflow Name" {
		t.Errorf("Name: got %s, want My Workflow Name", wf.Name)
	}

	if wf.Description != "My workflow description" {
		t.Errorf("Description: got %s, want 'My workflow description'", wf.Description)
	}

	if wf.Path != "workflows/my-workflow.md" {
		t.Errorf("Path: got %s, want workflows/my-workflow.md", wf.Path)
	}
}

func TestWorkflowInfoPreferDescription(t *testing.T) {
	// Test that description is preferred over name when both exist
	workflows := []WorkflowInfo{
		{
			ID:          "workflow1",
			Name:        "Workflow One",
			Description: "Description for workflow one",
			Path:        "workflows/workflow1.md",
		},
		{
			ID:          "workflow2",
			Name:        "Workflow Two",
			Description: "", // Empty description
			Path:        "workflows/workflow2.md",
		},
	}

	// Test first workflow - should use description
	if workflows[0].GetDisplayName() != "Description for workflow one" {
		t.Errorf("Workflow 1: got %s, want 'Description for workflow one'", workflows[0].GetDisplayName())
	}

	// Test second workflow - should use name (no description)
	if workflows[1].GetDisplayName() != "Workflow Two" {
		t.Errorf("Workflow 2: got %s, want 'Workflow Two'", workflows[1].GetDisplayName())
	}
}
