package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreatePullRequestTemporaryIDMapPassedToStep verifies that the temporary ID map
// from the handler manager is passed to the create_pull_request step
func TestCreatePullRequestTemporaryIDMapPassedToStep(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				TitlePrefix: "PR: ",
				Draft:       ptrBool(true),
			},
		},
	}

	// Build the create_pull_request step config
	stepConfig := compiler.buildCreatePullRequestStepConfig(workflowData, "agent", false)

	// Verify the step config includes GH_AW_TEMPORARY_ID_MAP environment variable
	found := false
	for _, envVar := range stepConfig.CustomEnvVars {
		if strings.Contains(envVar, "GH_AW_TEMPORARY_ID_MAP") {
			found = true
			// Verify it references the handler manager's output
			assert.Contains(t, envVar, "steps.process_safe_outputs.outputs.temporary_id_map",
				"GH_AW_TEMPORARY_ID_MAP should reference handler manager output")
			break
		}
	}

	assert.True(t, found, "GH_AW_TEMPORARY_ID_MAP environment variable should be present in step config")
}

// TestCreatePullRequestTemporaryIDMapInConsolidatedJob verifies that when the
// consolidated safe outputs job is built, the create_pull_request step has access
// to the handler manager's temporary ID map output
func TestCreatePullRequestTemporaryIDMapInConsolidatedJob(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 5,
				},
			},
			CreatePullRequests: &CreatePullRequestsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Draft: ptrBool(false),
			},
		},
	}

	// Build the consolidated safe outputs job
	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", "test.md")
	require.NoError(t, err, "Building consolidated safe outputs job should not error")
	assert.NotNil(t, job, "Consolidated safe outputs job should be created")

	// Convert steps to string for verification
	stepsStr := strings.Join(job.Steps, "\n")

	// Verify handler manager step exists (process_safe_outputs)
	assert.Contains(t, stepsStr, "id: process_safe_outputs",
		"Handler manager step should be present")

	// Verify handler manager outputs temporary_id_map
	assert.Contains(t, stepsStr, "steps.process_safe_outputs.outputs.temporary_id_map",
		"Handler manager temporary ID map output should be referenced")

	// Verify create_pull_request step exists
	assert.Contains(t, stepsStr, "id: create_pull_request",
		"Create pull request step should be present")

	// Verify create_pull_request step has access to temporary ID map
	assert.Contains(t, stepsStr, "GH_AW_TEMPORARY_ID_MAP: ${{ steps.process_safe_outputs.outputs.temporary_id_map }}",
		"Create pull request step should have GH_AW_TEMPORARY_ID_MAP environment variable")
}

// Helper function to create a bool pointer
func ptrBool(b bool) *bool {
	return &b
}
