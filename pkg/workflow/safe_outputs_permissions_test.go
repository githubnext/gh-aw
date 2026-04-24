//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputePermissionsForSafeOutputs(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expected    map[PermissionScope]PermissionLevel
	}{
		{
			name:        "nil safe outputs returns empty permissions",
			safeOutputs: nil,
			expected:    map[PermissionScope]PermissionLevel{},
		},
		{
			name: "create-issue only - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
				PermissionIssues:   PermissionWrite,
			},
		},
		{
			name: "create-discussion requires discussions permission",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:    PermissionRead,
				PermissionIssues:      PermissionWrite,
				PermissionDiscussions: PermissionWrite,
			},
		},
		{
			name: "close-discussion requires discussions permission",
			safeOutputs: &SafeOutputsConfig{
				CloseDiscussions: &CloseDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:    PermissionRead,
				PermissionDiscussions: PermissionWrite,
			},
		},
		{
			name: "update-discussion requires discussions permission",
			safeOutputs: &SafeOutputsConfig{
				UpdateDiscussions: &UpdateDiscussionsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:    PermissionRead,
				PermissionDiscussions: PermissionWrite,
			},
		},
		{
			name: "add-comment default - includes pull-requests and discussions",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
				PermissionDiscussions:  PermissionWrite,
			},
		},
		{
			name: "add-comment with discussions:true - includes discussions permission",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Discussions:          ptrBool(true),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
				PermissionDiscussions:  PermissionWrite,
			},
		},
		{
			name: "add-comment with discussions:false - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Discussions:          ptrBool(false),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "add-comment with pull-requests:false - no pull-requests permission",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					PullRequests:         ptrBool(false),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:    PermissionRead,
				PermissionIssues:      PermissionWrite,
				PermissionDiscussions: PermissionWrite,
			},
		},
		{
			name: "add-comment with issues:false - no issues permission",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Issues:               ptrBool(false),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionPullRequests: PermissionWrite,
				PermissionDiscussions:  PermissionWrite,
			},
		},
		{
			name: "hide-comment default - includes discussions permission",
			safeOutputs: &SafeOutputsConfig{
				HideComment: &HideCommentConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:    PermissionRead,
				PermissionIssues:      PermissionWrite,
				PermissionDiscussions: PermissionWrite,
			},
		},
		{
			name: "hide-comment with discussions:false - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				HideComment: &HideCommentConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Discussions:          ptrBool(false),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
				PermissionIssues:   PermissionWrite,
			},
		},
		{
			name: "add-labels only - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				AddLabels: &AddLabelsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "remove-labels only - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				RemoveLabels: &RemoveLabelsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("2")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "close-issue only - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				CloseIssues: &CloseIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
				PermissionIssues:   PermissionWrite,
			},
		},
		{
			name: "close-pull-request only - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				ClosePullRequests: &ClosePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "update-pull-request without update-branch requires contents read",
			safeOutputs: &SafeOutputsConfig{
				UpdatePullRequests: &UpdatePullRequestsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "update-pull-request with update-branch requires contents write",
			safeOutputs: &SafeOutputsConfig{
				UpdatePullRequests: &UpdatePullRequestsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					},
					UpdateBranch: boolPtr(true),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "create-pull-request with fallback-as-issue (default) - includes issues permission",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "create-pull-request with fallback-as-issue false - no issues permission",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					FallbackAsIssue:      boolPtr(false),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "push-to-pull-request-branch default fallback - requires pull-requests write and administration read",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:       PermissionWrite,
				PermissionPullRequests:   PermissionWrite,
				PermissionAdministration: PermissionRead,
			},
		},
		{
			name: "push-to-pull-request-branch with fallback-as-pull-request false - no pull-requests permission but administration read",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig:  BaseSafeOutputConfig{},
					FallbackAsPullRequest: boolPtr(false),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:       PermissionWrite,
				PermissionAdministration: PermissionRead,
			},
		},
		{
			name: "push-to-pull-request-branch with check-branch-protection false - no administration permission",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig:  BaseSafeOutputConfig{},
					CheckBranchProtection: boolPtr(false),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "push-to-pull-request-branch with check-branch-protection explicit true - includes administration read",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig:  BaseSafeOutputConfig{},
					CheckBranchProtection: boolPtr(true),
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:       PermissionWrite,
				PermissionPullRequests:   PermissionWrite,
				PermissionAdministration: PermissionRead,
			},
		},
		{
			name: "multiple safe outputs without discussions - no discussions permission",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
				AddLabels: &AddLabelsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
				},
				AssignToUser: &AssignToUserConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "multiple safe outputs with one discussion - includes discussions permission",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
				AddLabels: &AddLabelsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
				PermissionDiscussions:  PermissionWrite,
			},
		},
		{
			name: "upload-asset requires contents read",
			safeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
			},
		},
		{
			name: "create-code-scanning-alert requires security-events write",
			safeOutputs: &SafeOutputsConfig{
				CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:       PermissionRead,
				PermissionSecurityEvents: PermissionWrite,
			},
		},
		{
			name: "autofix-code-scanning-alert requires security-events and actions",
			safeOutputs: &SafeOutputsConfig{
				AutofixCodeScanningAlert: &AutofixCodeScanningAlertConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:       PermissionRead,
				PermissionSecurityEvents: PermissionWrite,
				PermissionActions:        PermissionRead,
			},
		},
		{
			name: "dispatch-workflow requires actions write",
			safeOutputs: &SafeOutputsConfig{
				DispatchWorkflow: &DispatchWorkflowConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionActions: PermissionWrite,
			},
		},
		{
			name: "create-project requires organization-projects write and issues read",
			safeOutputs: &SafeOutputsConfig{
				CreateProjects: &CreateProjectsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:         PermissionRead,
				PermissionOrganizationProj: PermissionWrite,
				PermissionIssues:           PermissionRead,
			},
		},
		{
			name: "update-project requires organization-projects write and issues read",
			safeOutputs: &SafeOutputsConfig{
				UpdateProjects: &UpdateProjectConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:         PermissionRead,
				PermissionOrganizationProj: PermissionWrite,
				PermissionIssues:           PermissionRead,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permissions := ComputePermissionsForSafeOutputs(tt.safeOutputs)
			require.NotNil(t, permissions, "Permissions should not be nil")

			// Check that all expected permissions are present
			for scope, expectedLevel := range tt.expected {
				actualLevel, exists := permissions.Get(scope)
				assert.True(t, exists, "Permission scope %s should exist", scope)
				assert.Equal(t, expectedLevel, actualLevel, "Permission level for %s should match", scope)
			}

			// Check that no unexpected permissions are present
			for scope := range permissions.permissions {
				_, expected := tt.expected[scope]
				assert.True(t, expected, "Unexpected permission scope: %s", scope)
			}
		})
	}
}

func TestComputePermissionsForSafeOutputs_NoOpAndMissingTool(t *testing.T) {
	// NoOp and MissingTool don't add any permissions on their own
	// They rely on add-comment permissions if comments are needed
	safeOutputs := &SafeOutputsConfig{
		NoOp: &NoOpConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
		},
		MissingTool: &MissingToolConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("3")},
		},
	}

	permissions := ComputePermissionsForSafeOutputs(safeOutputs)
	require.NotNil(t, permissions, "Permissions should not be nil")

	// NoOp and MissingTool alone don't require any permissions
	// The conclusion job will handle commenting through add-comment if configured
	assert.Empty(t, permissions.permissions, "NoOp and MissingTool alone should not add permissions")
}

func TestStepsRequireIDToken(t *testing.T) {
	tests := []struct {
		name     string
		steps    []any
		expected bool
	}{
		{
			name:     "nil steps",
			steps:    nil,
			expected: false,
		},
		{
			name:     "empty steps",
			steps:    []any{},
			expected: false,
		},
		{
			name: "no uses field",
			steps: []any{
				map[string]any{"name": "run something", "run": "echo hello"},
			},
			expected: false,
		},
		{
			name: "aws-actions/configure-aws-credentials with version",
			steps: []any{
				map[string]any{"uses": "aws-actions/configure-aws-credentials@v4"},
			},
			expected: true,
		},
		{
			name: "azure/login",
			steps: []any{
				map[string]any{"uses": "azure/login@v2"},
			},
			expected: true,
		},
		{
			name: "google-github-actions/auth",
			steps: []any{
				map[string]any{"uses": "google-github-actions/auth@v2"},
			},
			expected: true,
		},
		{
			name: "hashicorp/vault-action",
			steps: []any{
				map[string]any{"uses": "hashicorp/vault-action@v3"},
			},
			expected: true,
		},
		{
			name: "cyberark/conjur-action",
			steps: []any{
				map[string]any{"uses": "cyberark/conjur-action@v2"},
			},
			expected: true,
		},
		{
			name: "non-vault action",
			steps: []any{
				map[string]any{"uses": "actions/checkout@v4"},
			},
			expected: false,
		},
		{
			name: "mixed steps - vault action present",
			steps: []any{
				map[string]any{"uses": "actions/checkout@v4"},
				map[string]any{"uses": "aws-actions/configure-aws-credentials@v4"},
				map[string]any{"run": "echo hello"},
			},
			expected: true,
		},
		{
			name: "mixed steps - no vault action",
			steps: []any{
				map[string]any{"uses": "actions/checkout@v4"},
				map[string]any{"run": "echo hello"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stepsRequireIDToken(tt.steps)
			assert.Equal(t, tt.expected, result, "stepsRequireIDToken result")
		})
	}
}

func TestComputePermissionsForSafeOutputs_IDToken(t *testing.T) {
	writeStr := "write"
	noneStr := "none"

	tests := []struct {
		name          string
		safeOutputs   *SafeOutputsConfig
		expectIDToken bool
	}{
		{
			name: "no steps - no id-token permission",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expectIDToken: false,
		},
		{
			name: "step with vault action - auto-detects id-token: write",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				Steps: []any{
					map[string]any{"uses": "aws-actions/configure-aws-credentials@v4"},
				},
			},
			expectIDToken: true,
		},
		{
			name: "step with vault action but id-token: none overrides",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				IDToken:      &noneStr,
				Steps: []any{
					map[string]any{"uses": "aws-actions/configure-aws-credentials@v4"},
				},
			},
			expectIDToken: false,
		},
		{
			name: "no vault action but id-token: write explicitly set",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				IDToken:      &writeStr,
				Steps: []any{
					map[string]any{"uses": "actions/checkout@v4"},
				},
			},
			expectIDToken: true,
		},
		{
			name: "no steps with id-token: write explicitly set",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				IDToken:      &writeStr,
			},
			expectIDToken: true,
		},
		{
			name: "id-token: none with no steps",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				IDToken:      &noneStr,
			},
			expectIDToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permissions := ComputePermissionsForSafeOutputs(tt.safeOutputs)
			require.NotNil(t, permissions, "Permissions should not be nil")

			level, exists := permissions.Get(PermissionIdToken)
			if tt.expectIDToken {
				assert.True(t, exists, "Expected id-token permission to be set")
				assert.Equal(t, PermissionWrite, level, "Expected id-token: write")
			} else {
				assert.False(t, exists, "Expected id-token permission NOT to be set")
			}
		})
	}
}

func TestComputePermissionsForSafeOutputs_Staged(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expected    map[PermissionScope]PermissionLevel
	}{
		{
			name: "global staged=true - no permissions for any handler",
			safeOutputs: &SafeOutputsConfig{
				Staged:            true,
				CreateIssues:      &CreateIssuesConfig{},
				CreateDiscussions: &CreateDiscussionsConfig{},
				AddLabels:         &AddLabelsConfig{},
			},
			expected: map[PermissionScope]PermissionLevel{},
		},
		{
			name: "per-handler staged=true - staged handler contributes no permissions",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true},
				},
				AddLabels: &AddLabelsConfig{},
			},
			// create-issue is staged so it contributes nothing; add-labels is not staged
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionIssues:       PermissionWrite,
				PermissionPullRequests: PermissionWrite,
			},
		},
		{
			name: "all handlers per-handler staged - no permissions",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true},
				},
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true},
				},
			},
			expected: map[PermissionScope]PermissionLevel{},
		},
		{
			name: "global staged=true overrides per-handler staged=false",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: false},
				},
				DispatchWorkflow: &DispatchWorkflowConfig{},
			},
			expected: map[PermissionScope]PermissionLevel{},
		},
		{
			name: "global staged=false, one handler staged=true",
			safeOutputs: &SafeOutputsConfig{
				Staged: false,
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true},
				},
				CloseIssues: &CloseIssuesConfig{},
			},
			// create-pull-request is staged; close-issue is not
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionRead,
				PermissionIssues:   PermissionWrite,
			},
		},
		{
			name: "global staged=true - upload-asset staged, no contents:write",
			safeOutputs: &SafeOutputsConfig{
				Staged:       true,
				UploadAssets: &UploadAssetsConfig{},
			},
			expected: map[PermissionScope]PermissionLevel{},
		},
		{
			name: "pr review operations - all staged via global flag",
			safeOutputs: &SafeOutputsConfig{
				Staged:                          true,
				CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{},
				SubmitPullRequestReview:         &SubmitPullRequestReviewConfig{},
			},
			expected: map[PermissionScope]PermissionLevel{},
		},
		{
			name: "pr review operations - one staged, one not",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true},
				},
				SubmitPullRequestReview: &SubmitPullRequestReviewConfig{},
			},
			// submit-pull-request-review is not staged, so PR write permissions are added
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionRead,
				PermissionPullRequests: PermissionWrite,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permissions := ComputePermissionsForSafeOutputs(tt.safeOutputs)
			require.NotNil(t, permissions, "Permissions should not be nil")

			// Check that all expected permissions are present
			for scope, expectedLevel := range tt.expected {
				actualLevel, exists := permissions.Get(scope)
				assert.True(t, exists, "Permission scope %s should exist", scope)
				assert.Equal(t, expectedLevel, actualLevel, "Permission level for %s should match", scope)
			}

			// Check that no unexpected permissions are present
			for scope := range permissions.permissions {
				_, expected := tt.expected[scope]
				assert.True(t, expected, "Unexpected permission scope: %s", scope)
			}
		})
	}
}

// TestComputePermissionsForSafeOutputs_StagedYAMLRendering validates that fully-staged
// safe output configurations produce explicit "permissions: {}" in YAML rendering,
// rather than an empty string that would cause the job to inherit workflow-level permissions.
func TestComputePermissionsForSafeOutputs_StagedYAMLRendering(t *testing.T) {
	tests := []struct {
		name             string
		safeOutputs      *SafeOutputsConfig
		expectedRendered string
	}{
		{
			name: "globally staged - renders permissions: {}",
			safeOutputs: &SafeOutputsConfig{
				Staged:       true,
				CreateIssues: &CreateIssuesConfig{},
				AddLabels:    &AddLabelsConfig{},
			},
			expectedRendered: "permissions: {}",
		},
		{
			name: "all per-handler staged - renders permissions: {}",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true}},
				AddLabels:    &AddLabelsConfig{BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: true}},
			},
			expectedRendered: "permissions: {}",
		},
		{
			name: "staged PR handlers - renders permissions: {}",
			safeOutputs: &SafeOutputsConfig{
				Staged:             true,
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectedRendered: "permissions: {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permissions := ComputePermissionsForSafeOutputs(tt.safeOutputs)
			require.NotNil(t, permissions, "Permissions should not be nil")
			rendered := permissions.RenderToYAML()
			assert.Equal(t, tt.expectedRendered, rendered, "Fully-staged safe-outputs must render explicit empty permissions block")
		})
	}
}
