package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func steeringTestData() *WorkflowData {
	return &WorkflowData{
		Name:            "Steering test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			Steer: true,
		},
	}
}

func TestBuildActivationJobCreatesSteeringIssue(t *testing.T) {
	compiler := NewCompiler()
	job, err := compiler.buildActivationJob(steeringTestData(), false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "id: create-steering-issue")
	assert.Contains(t, steps, "create_steering_issue.cjs")
	assert.Contains(t, job.Permissions, "issues: write")
	assert.NotContains(t, job.Permissions, "contents: write")
	assert.NotContains(t, job.Permissions, "pull-requests: write")
	assert.NotContains(t, job.Permissions, "checks: write")
	assert.Equal(t, "${{ steps.create-steering-issue.outputs.issue_number }}", job.Outputs["steering_issue_number"])
	assert.Equal(t, "${{ steps.create-steering-issue.outputs.issue_url }}", job.Outputs["steering_issue_url"])
}

func TestBuildConclusionJobReusesAndCompletesSteeringIssue(t *testing.T) {
	compiler := NewCompiler()
	job, err := compiler.buildConclusionJob(steeringTestData(), "agent", []string{"safe_outputs"})
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "GH_AW_STEERING_ISSUE_NUMBER: ${{ needs.activation.outputs.steering_issue_number }}")
	assert.Contains(t, steps, "complete_steering_issue.cjs")
	assert.Contains(t, steps, "GH_AW_CREATED_PR_NUMBER: ${{ needs.safe_outputs.outputs.created_pr_number }}")
	assert.Contains(t, job.If, "needs.activation.outputs.steering_issue_number")
	assert.Contains(t, job.Permissions, "issues: write")
}

func TestValidateSteeringIssue(t *testing.T) {
	expression := TemplatableBool("${{ inputs.staged }}")
	tests := []struct {
		name    string
		data    *WorkflowData
		wantErr string
	}{
		{name: "valid", data: steeringTestData()},
		{
			name: "supports normal pull request options",
			data: &WorkflowData{CheckoutDisabled: true, SafeOutputs: &SafeOutputsConfig{
				Steer: true,
				CreatePullRequests: &CreatePullRequestsConfig{
					TargetRepoSlug:      "owner/repo",
					AllowedBranches:     []string{"feature/*"},
					AllowedBaseBranches: []string{"release/*"},
				},
			}},
		},
		{
			name: "failure issue repo",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				Steer:            true,
				FailureIssueRepo: "owner/repo",
			}},
			wantErr: "failure-issue-repo",
		},
		{
			name: "expression staged",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				Steer:  true,
				Staged: &expression,
			}},
			wantErr: "expression-valued staged option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSteeringIssue(tt.data)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSteeringIssuePermissions(t *testing.T) {
	data := steeringTestData()
	require.ErrorContains(t, validateSteeringIssuePermissions(data, NewPermissionsContentsRead()), "requires issues: read")
	require.NoError(t, validateSteeringIssuePermissions(data, NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionIssues: PermissionRead,
	})))
	require.NoError(t, validateSteeringIssuePermissions(data, NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionIssues: PermissionWrite,
	})))
}

func TestSteeringIssueDisabledWhenStaged(t *testing.T) {
	staged := TemplatableBool("true")
	data := steeringTestData()
	data.SafeOutputs.Staged = &staged

	assert.False(t, isSteeringIssueEnabled(data))
}
