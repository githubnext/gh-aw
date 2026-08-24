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
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	// The pre-create step resolves the base branch through the API, so the activation job keeps
	// its sparse .github checkout (which stays cross-repo aware for workflow_call triggers).
	assert.Contains(t, steps, "name: Checkout .github and .agents folders")
	assert.NotContains(t, steps, "name: Checkout repository")
	assert.Contains(t, steps, "id: pre-create-pull-request")
	assert.Contains(t, steps, "pre_create_pull_request.cjs")
	assert.Contains(t, steps, "id: validate-pre-created-pull-request")
	assert.Contains(t, steps, "validate_pre_created_pull_request.cjs")
	assert.Contains(t, job.Permissions, "contents: write")
	assert.Contains(t, job.Permissions, "pull-requests: write")
	assert.Contains(t, job.Permissions, "checks: write")
	assert.Equal(t, "${{ steps.validate-pre-created-pull-request.outputs.branch }}", job.Outputs["pre_created_pull_request_branch"])
}

func TestBuildConclusionJobCompletesPreCreatedCheck(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name: "Pre-create test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
		},
	}

	job, err := compiler.buildConclusionJob(data, "agent", []string{"safe_outputs"})
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "Complete pre-created pull request check")
	assert.Contains(t, steps, "complete_pre_created_check_run.cjs")
	assert.Contains(t, steps, "needs.activation.outputs.pre_created_pull_request_check_run_id")
	assert.Contains(t, steps, "needs.safe_outputs.outputs.created_pr_number")
	assert.NotContains(t, steps, "GH_AW_NOOP_COMMENT_BODY")
	assert.Contains(t, job.Permissions, "checks: write")
}

func TestBuildConclusionJobPassesNoOpCommentToPreCreatedCheck(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name: "Pre-create test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
			NoOp:               &NoOpConfig{},
		},
	}

	job, err := compiler.buildConclusionJob(data, "agent", []string{"safe_outputs"})
	require.NoError(t, err)
	require.NotNil(t, job)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "GH_AW_NOOP_COMMENT_BODY: ${{ steps.noop.outputs.noop_comment_body }}")
}

func TestPreCreatePullRequestCheckoutOverride(t *testing.T) {
	manager := NewCheckoutManager(nil)
	manager.SetDefaultRefOverride(preCreatedPullRequestBranchRef(nil))

	steps := strings.Join(manager.GenerateDefaultCheckoutStep(false, "", func(action string) string {
		return action + "@sha"
	}), "")

	assert.Contains(t, steps, "ref: gh-aw/pre-created/${{ github.run_id }}-${{ github.run_attempt }}")
}

func TestPreCreatePullRequestCheckoutOverrideUsesConfiguredBranchPrefix(t *testing.T) {
	data := &WorkflowData{SafeOutputs: &SafeOutputsConfig{
		CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "signed/"},
	}}
	manager := NewCheckoutManager(nil)
	manager.SetDefaultRefOverride(preCreatedPullRequestBranchRef(data))

	steps := strings.Join(manager.GenerateDefaultCheckoutStep(false, "", func(action string) string {
		return action + "@sha"
	}), "")

	assert.Contains(t, steps, "ref: signed/${{ github.run_id }}-${{ github.run_attempt }}")
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
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
			}},
		},
		{
			name: "checkout disabled",
			data: &WorkflowData{
				CheckoutDisabled: true,
				SafeOutputs: &SafeOutputsConfig{
					CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
				},
			},
			wantErr: "requires the default checkout",
		},
		{
			name: "multiple pull requests",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BaseSafeOutputConfig: BaseSafeOutputConfig{Max: &invalidMax}},
			}},
			wantErr: "requires max: 1",
		},
		{
			name: "staged disables pre-creation",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				Staged:             stagedTrue(),
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, TargetRepoSlug: "owner/repo"},
			}},
		},
		{
			name: "expression staged",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				Staged:             stagedExpression(),
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
			}},
			wantErr: "expression-valued staged option",
		},
		{
			name: "allowed base branches",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, AllowedBaseBranches: []string{"release/*"}},
			}},
			wantErr: "allowed-base-branches",
		},
		{
			name: "branch prefix",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "signed/"},
			}},
		},
		{
			name: "expression branch prefix",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "${{ inputs.branch_prefix }}"},
			}},
			wantErr: "requires branch-prefix to be a static string",
		},
		{
			name: "invalid branch prefix",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "bad prefix/"},
			}},
			wantErr: "branch-prefix must be a valid git branch prefix",
		},
		{
			name: "whitespace branch prefix",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "  "},
			}},
			wantErr: "branch-prefix must contain valid git branch prefix characters",
		},
		{
			name: "reserved ref branch prefix",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "refs/heads/"},
			}},
			wantErr: "branch-prefix must form a valid git branch ref",
		},
		{
			name: "ambiguous branch prefix",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "foo//"},
			}},
			wantErr: "branch-prefix must form a valid git branch ref",
		},
		{
			name: "cross repository",
			data: &WorkflowData{SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{Steer: true, TargetRepoSlug: "owner/repo"},
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

func TestValidatePreCreatedPullRequestBranchPrefix(t *testing.T) {
	tests := map[string]struct {
		prefix  string
		wantErr bool
	}{
		"valid":          {prefix: "signed/"},
		"leading slash":  {prefix: "/", wantErr: true},
		"double slash":   {prefix: "foo//", wantErr: true},
		"double dot":     {prefix: "..", wantErr: true},
		"dot component":  {prefix: ".foo/", wantErr: true},
		"reserved ref":   {prefix: "refs/heads/", wantErr: true},
		"lock component": {prefix: "foo/.lock/", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.wantErr {
				require.Error(t, validatePreCreatedPullRequestBranchPrefix(tt.prefix))
			} else {
				require.NoError(t, validatePreCreatedPullRequestBranchPrefix(tt.prefix))
			}
		})
	}
}

func TestValidatePreCreatePullRequestSteerPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions *Permissions
		wantErr     string
	}{
		{
			name:        "missing pull requests permission",
			permissions: NewPermissionsContentsRead(),
			wantErr:     "steering requires pull-requests: read",
		},
		{
			name: "pull requests read",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionPullRequests: PermissionRead,
			}),
		},
		{
			name: "pull requests write implies read",
			permissions: NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
				PermissionPullRequests: PermissionWrite,
			}),
		},
	}

	data := &WorkflowData{SafeOutputs: &SafeOutputsConfig{
		CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePreCreatePullRequestSteerPermissions(data, tt.permissions)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func stagedTrue() *TemplatableBool {
	value := TemplatableBool("true")
	return &value
}

func stagedExpression() *TemplatableBool {
	value := TemplatableBool("${{ github.event.inputs.staged }}")
	return &value
}

func TestPreCreatePullRequestDisabledWhenStaged(t *testing.T) {
	data := &WorkflowData{SafeOutputs: &SafeOutputsConfig{
		Staged:             stagedTrue(),
		CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
	}}

	assert.False(t, isPreCreatePullRequestEnabled(data))
}

func TestConclusionJobRunsWhenPreCreatedCheckExists(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name: "Pre-create test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
		},
	}

	job, err := compiler.buildConclusionJob(data, "agent", []string{"safe_outputs"})
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Contains(t, job.If, "needs.activation.outputs.pre_created_pull_request_check_run_id")
}

func TestActivationPreCreateStepIgnoresDraftPolicy(t *testing.T) {
	compiler := NewCompiler()
	draft := "false"
	data := &WorkflowData{
		Name:            "Pre-create test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true, Draft: &draft},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	// The allocated pull request is always a draft; the configured draft policy is applied
	// later, in the safe outputs phase.
	assert.Contains(t, steps, "id: pre-create-pull-request")
	assert.NotContains(t, steps, "draft: false")
}

func TestActivationPreCreateStepUsesConfiguredBaseBranch(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name:            "Pre-create test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BaseBranch: "release/v1"},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, `GH_AW_CUSTOM_BASE_BRANCH: "release/v1"`)
}

func TestActivationPreCreateStepPassesConfiguredTitlePrefix(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name:            "Pre-create test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true, TitlePrefix: "[bot] "},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, `GH_AW_PR_TITLE_PREFIX: "[bot] "`)
}

func TestActivationPreCreateStepPassesConfiguredBranchPrefix(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name:            "Pre-create test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true, BranchPrefix: "signed/"},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, `GH_AW_PRE_CREATED_PULL_REQUEST_BRANCH_PREFIX: "signed/"`)
	assert.Contains(t, steps, "GH_AW_EXPECTED_PRE_CREATED_PULL_REQUEST_BRANCH: signed/${{ github.run_id }}-${{ github.run_attempt }}")
}

func TestActivationPreCreateStepOmitsEmptyTitlePrefix(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name:            "Pre-create test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	assert.NotContains(t, steps, "GH_AW_PR_TITLE_PREFIX")
}

func TestActivationPreCreateStepPassesSteerFlag(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name:            "Pre-create steer test",
		MarkdownContent: "# Test",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Steer: true},
		},
	}

	job, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)

	steps := strings.Join(job.Steps, "")
	assert.NotContains(t, steps, "GH_AW_PRE_CREATE_STEER")
}
