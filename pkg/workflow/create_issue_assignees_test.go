package workflow

import (
	"strings"
	"testing"
)

func TestCreateIssueJobWithAssignees(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with assignees configured
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				Assignees: []string{"user1", "user2", "bot-user"},
			},
		},
	}

	job, err := c.buildCreateOutputIssueJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create issue job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that assignee steps are included
	if !strings.Contains(stepsContent, "Assign issue to user1") {
		t.Error("Expected assignee step for user1")
	}
	if !strings.Contains(stepsContent, "Assign issue to user2") {
		t.Error("Expected assignee step for user2")
	}
	if !strings.Contains(stepsContent, "Assign issue to bot-user") {
		t.Error("Expected assignee step for bot-user")
	}

	// Check that gh issue edit command is present
	if !strings.Contains(stepsContent, "gh issue edit") {
		t.Error("Expected gh issue edit command in steps")
	}

	// Check that --add-assignee flag is present
	if !strings.Contains(stepsContent, "--add-assignee") {
		t.Error("Expected --add-assignee flag in gh issue edit command")
	}

	// Check that ISSUE_NUMBER environment variable is set from step output
	if !strings.Contains(stepsContent, "ISSUE_NUMBER: ${{ steps.create_issue.outputs.issue_number }}") {
		t.Error("Expected ISSUE_NUMBER to be set from create_issue step output")
	}

	// Check that condition is set to only run if issue_number is not empty
	if !strings.Contains(stepsContent, "if: steps.create_issue.outputs.issue_number != ''") {
		t.Error("Expected conditional if statement for assignee steps")
	}

	// Verify that GH_TOKEN is set
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ github.token }}") {
		t.Error("Expected GH_TOKEN environment variable to be set")
	}
}

func TestCreateIssueJobWithoutAssignees(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test without assignees
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				// No assignees configured
			},
		},
	}

	job, err := c.buildCreateOutputIssueJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create issue job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that no assignee steps are included
	if strings.Contains(stepsContent, "Assign issue to") {
		t.Error("Did not expect assignee steps when no assignees configured")
	}
	if strings.Contains(stepsContent, "gh issue edit") {
		t.Error("Did not expect gh issue edit command when no assignees configured")
	}
}

func TestCreateIssueJobWithSingleAssignee(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with a single assignee
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				Assignees: []string{"single-user"},
			},
		},
	}

	job, err := c.buildCreateOutputIssueJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create issue job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that single assignee step is included
	if !strings.Contains(stepsContent, "Assign issue to single-user") {
		t.Error("Expected assignee step for single-user")
	}

	// Check that gh issue edit command is present
	if !strings.Contains(stepsContent, "gh issue edit") {
		t.Error("Expected gh issue edit command in steps")
	}

	// Verify environment variable for assignee
	if !strings.Contains(stepsContent, `ASSIGNEE: "single-user"`) {
		t.Error("Expected ASSIGNEE environment variable to be set")
	}
}

func TestParseIssuesConfigWithAssignees(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test parsing assignees from config
	outputMap := map[string]any{
		"create-issue": map[string]any{
			"title-prefix": "[test] ",
			"labels":       []any{"bug", "enhancement"},
			"assignees":    []any{"user1", "user2", "github-bot"},
		},
	}

	config := c.parseIssuesConfig(outputMap)
	if config == nil {
		t.Fatal("Expected parseIssuesConfig to return non-nil config")
	}

	if len(config.Assignees) != 3 {
		t.Errorf("Expected 3 assignees, got %d", len(config.Assignees))
	}

	expectedAssignees := []string{"user1", "user2", "github-bot"}
	for i, expected := range expectedAssignees {
		if i >= len(config.Assignees) {
			t.Errorf("Missing assignee at index %d, expected %s", i, expected)
			continue
		}
		if config.Assignees[i] != expected {
			t.Errorf("Assignee at index %d: expected %s, got %s", i, expected, config.Assignees[i])
		}
	}
}
