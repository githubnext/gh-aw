package workflow

import (
	"strings"
	"testing"
)

func TestCreatePullRequestJobWithAllowEmpty(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with allow-empty configured
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				AllowEmpty: true,
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that GH_AW_PR_ALLOW_EMPTY environment variable is set
	if !strings.Contains(stepsContent, `GH_AW_PR_ALLOW_EMPTY: "true"`) {
		t.Error("Expected GH_AW_PR_ALLOW_EMPTY environment variable to be set to true")
	}

	// Check that the JavaScript code includes allow-empty logic
	if !strings.Contains(stepsContent, "allowEmpty") {
		t.Error("Expected JavaScript code to include allowEmpty variable")
	}
}

func TestCreatePullRequestJobWithoutAllowEmpty(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test without allow-empty (default should be false)
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{},
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that GH_AW_PR_ALLOW_EMPTY environment variable is set to false
	if !strings.Contains(stepsContent, `GH_AW_PR_ALLOW_EMPTY: "false"`) {
		t.Error("Expected GH_AW_PR_ALLOW_EMPTY environment variable to be set to false by default")
	}
}
