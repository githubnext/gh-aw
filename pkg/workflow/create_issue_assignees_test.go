package workflow

import (
	"strings"
	"testing"
)

func TestCreateIssueJobWithAssignees(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with assignees configured (but without copilot)
	// After the change, assignee steps are only generated when "copilot" is in the list
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

	// Since assignees don't include "copilot", no assignee steps should be generated
	if strings.Contains(stepsContent, "Checkout repository for gh CLI") {
		t.Error("Did not expect checkout step when copilot is not in assignees")
	}

	if strings.Contains(stepsContent, "Assign issue to") {
		t.Error("Did not expect assignee steps when copilot is not in assignees")
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

	// Test with a single assignee (not copilot)
	// After the change, no assignee steps should be generated unless copilot is in the list
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

	// Since the assignee is not "copilot", no assignee steps should be generated
	if strings.Contains(stepsContent, "Assign issue to") {
		t.Error("Did not expect assignee steps when copilot is not in the assignees list")
	}

	if strings.Contains(stepsContent, "Checkout repository for gh CLI") {
		t.Error("Did not expect checkout step when copilot is not in the assignees list")
	}
}

func TestParseIssuesConfigWithAssignees(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test parsing assignees from config (array format)
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

func TestParseIssuesConfigWithSingleStringAssignee(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test parsing assignees from config (string format)
	outputMap := map[string]any{
		"create-issue": map[string]any{
			"title-prefix": "[test] ",
			"labels":       []any{"bug"},
			"assignees":    "single-user",
		},
	}

	config := c.parseIssuesConfig(outputMap)
	if config == nil {
		t.Fatal("Expected parseIssuesConfig to return non-nil config")
	}

	if len(config.Assignees) != 1 {
		t.Errorf("Expected 1 assignee, got %d", len(config.Assignees))
	}

	if config.Assignees[0] != "single-user" {
		t.Errorf("Expected assignee 'single-user', got %s", config.Assignees[0])
	}
}

func TestCreateIssueJobWithCopilotAssignee(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with "copilot" as assignee (should be mapped to "@copilot")
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				Assignees: []string{"copilot"},
			},
		},
	}

	job, err := c.buildCreateOutputIssueJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create issue job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that the step name shows "copilot"
	if !strings.Contains(stepsContent, "Assign issue to copilot") {
		t.Error("Expected assignee step name to show 'copilot'")
	}

	// Check that the actual assignee is "@copilot" (gh CLI special value)
	if !strings.Contains(stepsContent, `ASSIGNEE: "@copilot"`) {
		t.Error("Expected ASSIGNEE environment variable to be set to '@copilot'")
	}

	// Verify that the original "copilot" without @ is NOT used as assignee
	if strings.Contains(stepsContent, `ASSIGNEE: "copilot"`) && !strings.Contains(stepsContent, `ASSIGNEE: "@copilot"`) {
		t.Error("Expected 'copilot' to be mapped to '@copilot', not used directly")
	}

	// Find the assignee step section (after "Assign issue to copilot")
	assigneeStepIndex := strings.Index(stepsContent, "Assign issue to copilot")
	if assigneeStepIndex == -1 {
		t.Fatal("Could not find assignee step")
	}
	assigneeStepContent := stepsContent[assigneeStepIndex:]

	// Find the next step or end of content (limit to this step only)
	nextStepIndex := strings.Index(assigneeStepContent[len("Assign issue to copilot"):], "- name:")
	if nextStepIndex != -1 {
		assigneeStepContent = assigneeStepContent[:len("Assign issue to copilot")+nextStepIndex]
	}

	// Verify that GH_TOKEN uses Copilot token precedence with legacy fallback in assignee step
	if !strings.Contains(assigneeStepContent, "GH_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN || secrets.COPILOT_CLI_TOKEN || secrets.GH_AW_COPILOT_TOKEN || secrets.GH_AW_GITHUB_TOKEN }}") {
		t.Error("Expected GH_TOKEN in assignee step to use Copilot token precedence with legacy fallback")
	}

	// Verify GITHUB_TOKEN is NOT in the fallback chain for copilot assignees in assignee step
	if strings.Contains(assigneeStepContent, "|| secrets.GITHUB_TOKEN }}") {
		t.Errorf("Did not expect GITHUB_TOKEN in fallback chain for copilot assignees in assignee step. Content: %s", assigneeStepContent)
	}
}

func TestCreateIssueJobWithCustomGitHubToken(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with custom GitHub token configuration and copilot as assignee
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		GitHubToken: "${{ secrets.CUSTOM_PAT }}",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.ISSUE_SPECIFIC_PAT }}",
				},
				Assignees: []string{"copilot"}, // Changed to copilot so assignee steps are generated
			},
		},
	}

	job, err := c.buildCreateOutputIssueJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create issue job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that the issue-specific token is used (highest precedence)
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.ISSUE_SPECIFIC_PAT }}") {
		t.Error("Expected issue-specific GitHub token to be used in assignee steps")
	}

	// Verify default token is NOT used
	if strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN") {
		t.Error("Did not expect default token when custom token is configured")
	}
}
