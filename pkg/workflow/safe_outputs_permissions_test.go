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
			name: "push-to-pull-request-branch - no issues permission",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:     PermissionWrite,
				PermissionPullRequests: PermissionWrite,
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
			name: "upload-asset requires contents write",
			safeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents: PermissionWrite,
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
			name: "create-project requires organization-projects write",
			safeOutputs: &SafeOutputsConfig{
				CreateProjects: &CreateProjectsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
			expected: map[PermissionScope]PermissionLevel{
				PermissionContents:         PermissionRead,
				PermissionOrganizationProj: PermissionWrite,
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
