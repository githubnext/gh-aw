package workflow

import (
	"strings"
	"testing"
)

func TestCreatePullRequestJobWithStagedFlag(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with staged: true
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{},
			Staged:             &[]bool{true}[0], // pointer to true
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that GITHUB_AW_SAFE_OUTPUTS_STAGED is included in the env section
	if !strings.Contains(stepsContent, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED environment variable to be set to true in create-pull-request job")
	}

	// Test with staged: false
	workflowData.SafeOutputs.Staged = &[]bool{false}[0] // pointer to false

	job, err = c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	stepsContent = strings.Join(job.Steps, "")

	// Check that GITHUB_AW_SAFE_OUTPUTS_STAGED is not included in the env section when false
	// We need to be specific to avoid matching the JavaScript code that references the variable
	if strings.Contains(stepsContent, "          GITHUB_AW_SAFE_OUTPUTS_STAGED:") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED environment variable not to be set when staged is false")
	}

	// Test with staged: nil (not specified)
	workflowData.SafeOutputs.Staged = nil

	job, err = c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	stepsContent = strings.Join(job.Steps, "")

	// Check that GITHUB_AW_SAFE_OUTPUTS_STAGED is not included in the env section when nil
	// We need to be specific to avoid matching the JavaScript code that references the variable
	if strings.Contains(stepsContent, "          GITHUB_AW_SAFE_OUTPUTS_STAGED:") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED environment variable not to be set when staged is nil")
	}
}

func TestCreatePullRequestJobWithoutSafeOutputs(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with no SafeOutputs config - this should fail
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: nil,
	}

	_, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err == nil {
		t.Error("Expected error when SafeOutputs is nil")
	}

	expectedError := "safe-outputs.create-pull-request configuration is required"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedError, err)
	}

	// Test with SafeOutputs but no CreatePullRequests config - this should also fail
	workflowData.SafeOutputs = &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{},
		Staged:       &[]bool{true}[0],
	}

	_, err = c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err == nil {
		t.Error("Expected error when CreatePullRequests is nil")
	}

	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedError, err)
	}
}
