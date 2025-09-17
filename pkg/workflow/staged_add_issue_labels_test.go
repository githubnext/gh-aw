package workflow

import (
	"strings"
	"testing"
)

func TestAddIssueLabelsJobWithStagedFlag(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with staged: true
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			AddIssueLabels: &AddIssueLabelsConfig{},
			Staged:         &[]bool{true}[0], // pointer to true
		},
	}

	job, err := c.buildCreateOutputLabelJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building add labels job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that GITHUB_AW_SAFE_OUTPUTS_STAGED is included in the env section
	if !strings.Contains(stepsContent, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED environment variable to be set to true in add-issue-labels job")
	}

	// Test with staged: false
	workflowData.SafeOutputs.Staged = &[]bool{false}[0] // pointer to false

	job, err = c.buildCreateOutputLabelJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building add labels job: %v", err)
	}

	stepsContent = strings.Join(job.Steps, "")

	// Check that GITHUB_AW_SAFE_OUTPUTS_STAGED is not included in the env section when false
	// We need to be specific to avoid matching the JavaScript code that references the variable
	if strings.Contains(stepsContent, "          GITHUB_AW_SAFE_OUTPUTS_STAGED:") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED environment variable not to be set when staged is false")
	}

	// Test with staged: nil (not specified)
	workflowData.SafeOutputs.Staged = nil

	job, err = c.buildCreateOutputLabelJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building add labels job: %v", err)
	}

	stepsContent = strings.Join(job.Steps, "")

	// Check that GITHUB_AW_SAFE_OUTPUTS_STAGED is not included in the env section when nil
	// We need to be specific to avoid matching the JavaScript code that references the variable
	if strings.Contains(stepsContent, "          GITHUB_AW_SAFE_OUTPUTS_STAGED:") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED environment variable not to be set when staged is nil")
	}
}

func TestAddIssueLabelsJobWithNilSafeOutputs(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with no SafeOutputs config - this should fail
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: nil,
	}

	_, err := c.buildCreateOutputLabelJob(workflowData, "main_job")
	if err == nil {
		t.Error("Expected error when SafeOutputs is nil")
	}

	expectedError := "safe-outputs configuration is required"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedError, err)
	}
}

func TestAddIssueLabelsJobWithNilAddIssueLabelsConfig(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with SafeOutputs but nil AddIssueLabels config - this should work as it's a valid case
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			AddIssueLabels: nil, // This is valid - means empty configuration
			Staged:         &[]bool{true}[0],
		},
	}

	job, err := c.buildCreateOutputLabelJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Expected no error when AddIssueLabels is nil (should use defaults): %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that GITHUB_AW_SAFE_OUTPUTS_STAGED is included in the env section
	if !strings.Contains(stepsContent, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED environment variable to be set to true even with nil AddIssueLabels config")
	}

	// Check that default max count is used
	if !strings.Contains(stepsContent, "          GITHUB_AW_LABELS_MAX_COUNT: 3\n") {
		t.Error("Expected default max count of 3 when AddIssueLabels is nil")
	}
}
