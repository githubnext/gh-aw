package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildActivationJobPreCreatesPullRequest(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name:            "Pre-create test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{PreCreate: true},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "name: Checkout repository")
	assert.Contains(t, steps, "id: pre-create-pull-request")
	assert.Contains(t, steps, "pre_create_pull_request.cjs")
	assert.Contains(t, job.Permissions, "contents: write")
	assert.Contains(t, job.Permissions, "pull-requests: write")
	assert.Contains(t, job.Permissions, "checks: write")
	assert.Equal(t, "${{ steps.pre-create-pull-request.outputs.branch }}", job.Outputs["pre_created_pull_request_branch"])
}

func TestBuildConclusionJobCompletesPreCreatedCheck(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name: "Pre-create test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{PreCreate: true},
		},
	}

	job, err := compiler.buildConclusionJob(data, "agent", []string{"safe_outputs"})
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "Complete pre-created pull request check")
	assert.Contains(t, steps, "complete_pre_created_check_run.cjs")
	assert.Contains(t, steps, "needs.activation.outputs.pre_created_pull_request_check_run_id")
	assert.Contains(t, job.Permissions, "checks: write")
}

func TestPreCreatePullRequestCheckoutOverride(t *testing.T) {
	manager := NewCheckoutManager(nil)
	manager.SetDefaultRefOverride("${{ needs.activation.outputs.pre_created_pull_request_branch }}")

	steps := strings.Join(manager.GenerateDefaultCheckoutStep(false, "", func(action string) string {
		return action + "@sha"
	}), "")

	assert.Contains(t, steps, "ref: ${{ needs.activation.outputs.pre_created_pull_request_branch }}")
}

func TestValidatePreCreatePullRequest(t *testing.T) {
	invalidMax := "2"
	tests := []struct {
		name    string
		data    *WorkflowData
		wantErr string
	}{
		{
			name: "valid",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{PreCreate: true},
			}},
		},
		{
			name: "checkout disabled",
			data: &WorkflowData{
				CheckoutDisabled: true,
				SafeOutputs: &SafeOutputsConfig{
					CreatePullRequests: &CreatePullRequestsConfig{PreCreate: true},
				},
			},
			wantErr: "requires the default checkout",
		},
		{
			name: "multiple pull requests",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{PreCreate: true, BaseSafeOutputConfig: BaseSafeOutputConfig{Max: &invalidMax}},
			}},
			wantErr: "requires max: 1",
		},
		{
			name: "cross repository",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{PreCreate: true, TargetRepoSlug: "owner/repo"},
			}},
			wantErr: "only supports pull requests in the workflow repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePreCreatePullRequest(tt.data)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
