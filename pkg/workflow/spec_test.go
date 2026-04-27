//go:build !integration

package workflow_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_Permissions_ContentsWritePRWrite validates the documented permission combination
// for NewPermissionsContentsWritePRWrite as described in the workflow package README.md.
// Spec: "NewPermissionsContentsWritePRWrite — contents:write + pull-requests:write"
func TestSpec_Permissions_ContentsWritePRWrite(t *testing.T) {
	p := workflow.NewPermissionsContentsWritePRWrite()
	require.NotNil(t, p, "NewPermissionsContentsWritePRWrite must return a non-nil Permissions")

	level, ok := p.Get(workflow.PermissionContents)
	assert.True(t, ok, "contents scope must be present")
	assert.Equal(t, workflow.PermissionWrite, level, "contents must be write")

	level, ok = p.Get(workflow.PermissionPullRequests)
	assert.True(t, ok, "pull-requests scope must be present")
	assert.Equal(t, workflow.PermissionWrite, level, "pull-requests must be write")
}

// TestSpec_Permissions_ContentsWriteIssuesWritePRWrite validates the documented permission combination.
// Spec: "NewPermissionsContentsWriteIssuesWritePRWrite — contents:write + issues:write + pull-requests:write"
func TestSpec_Permissions_ContentsWriteIssuesWritePRWrite(t *testing.T) {
	p := workflow.NewPermissionsContentsWriteIssuesWritePRWrite()
	require.NotNil(t, p, "NewPermissionsContentsWriteIssuesWritePRWrite must return a non-nil Permissions")

	contentsLevel, ok := p.Get(workflow.PermissionContents)
	assert.True(t, ok, "contents scope must be present")
	assert.Equal(t, workflow.PermissionWrite, contentsLevel, "contents must be write")

	issuesLevel, ok := p.Get(workflow.PermissionIssues)
	assert.True(t, ok, "issues scope must be present")
	assert.Equal(t, workflow.PermissionWrite, issuesLevel, "issues must be write")

	prLevel, ok := p.Get(workflow.PermissionPullRequests)
	assert.True(t, ok, "pull-requests scope must be present")
	assert.Equal(t, workflow.PermissionWrite, prLevel, "pull-requests must be write")
}

// TestSpec_Permissions_ContentsReadDiscussionsWrite validates the documented permission combination.
// Spec: "NewPermissionsContentsReadDiscussionsWrite — contents:read + discussions:write"
func TestSpec_Permissions_ContentsReadDiscussionsWrite(t *testing.T) {
	p := workflow.NewPermissionsContentsReadDiscussionsWrite()
	require.NotNil(t, p, "NewPermissionsContentsReadDiscussionsWrite must return a non-nil Permissions")

	contentsLevel, ok := p.Get(workflow.PermissionContents)
	assert.True(t, ok, "contents scope must be present")
	assert.Equal(t, workflow.PermissionRead, contentsLevel, "contents must be read")

	discussionsLevel, ok := p.Get(workflow.PermissionDiscussions)
	assert.True(t, ok, "discussions scope must be present")
	assert.Equal(t, workflow.PermissionWrite, discussionsLevel, "discussions must be write")
}

// TestSpec_Permissions_ContentsReadIssuesWriteDiscussionsWrite validates the documented combination.
// Spec: "NewPermissionsContentsReadIssuesWriteDiscussionsWrite — contents:read + issues:write + discussions:write"
func TestSpec_Permissions_ContentsReadIssuesWriteDiscussionsWrite(t *testing.T) {
	p := workflow.NewPermissionsContentsReadIssuesWriteDiscussionsWrite()
	require.NotNil(t, p, "NewPermissionsContentsReadIssuesWriteDiscussionsWrite must return a non-nil Permissions")

	contentsLevel, ok := p.Get(workflow.PermissionContents)
	assert.True(t, ok, "contents scope must be present")
	assert.Equal(t, workflow.PermissionRead, contentsLevel, "contents must be read")

	issuesLevel, ok := p.Get(workflow.PermissionIssues)
	assert.True(t, ok, "issues scope must be present")
	assert.Equal(t, workflow.PermissionWrite, issuesLevel, "issues must be write")

	discussionsLevel, ok := p.Get(workflow.PermissionDiscussions)
	assert.True(t, ok, "discussions scope must be present")
	assert.Equal(t, workflow.PermissionWrite, discussionsLevel, "discussions must be write")
}

// TestSpec_Permissions_ContentsReadPRWrite validates the documented permission combination.
// Spec: "NewPermissionsContentsReadPRWrite — contents:read + pull-requests:write"
func TestSpec_Permissions_ContentsReadPRWrite(t *testing.T) {
	p := workflow.NewPermissionsContentsReadPRWrite()
	require.NotNil(t, p, "NewPermissionsContentsReadPRWrite must return a non-nil Permissions")

	contentsLevel, ok := p.Get(workflow.PermissionContents)
	assert.True(t, ok, "contents scope must be present")
	assert.Equal(t, workflow.PermissionRead, contentsLevel, "contents must be read")

	prLevel, ok := p.Get(workflow.PermissionPullRequests)
	assert.True(t, ok, "pull-requests scope must be present")
	assert.Equal(t, workflow.PermissionWrite, prLevel, "pull-requests must be write")
}

// TestSpec_Permissions_ContentsReadSecurityEventsWrite validates the documented permission combination.
// Spec: "NewPermissionsContentsReadSecurityEventsWrite — contents:read + security-events:write"
func TestSpec_Permissions_ContentsReadSecurityEventsWrite(t *testing.T) {
	p := workflow.NewPermissionsContentsReadSecurityEventsWrite()
	require.NotNil(t, p, "NewPermissionsContentsReadSecurityEventsWrite must return a non-nil Permissions")

	contentsLevel, ok := p.Get(workflow.PermissionContents)
	assert.True(t, ok, "contents scope must be present")
	assert.Equal(t, workflow.PermissionRead, contentsLevel, "contents must be read")

	secLevel, ok := p.Get(workflow.PermissionSecurityEvents)
	assert.True(t, ok, "security-events scope must be present")
	assert.Equal(t, workflow.PermissionWrite, secLevel, "security-events must be write")
}

// TestSpec_Permissions_ContentsReadProjectsWrite validates the documented permission combination.
// Spec: "NewPermissionsContentsReadProjectsWrite — contents:read + projects:write"
// Note: organization-projects is a GitHub App-only scope per the spec.
func TestSpec_Permissions_ContentsReadProjectsWrite(t *testing.T) {
	p := workflow.NewPermissionsContentsReadProjectsWrite()
	require.NotNil(t, p, "NewPermissionsContentsReadProjectsWrite must return a non-nil Permissions")

	contentsLevel, ok := p.Get(workflow.PermissionContents)
	assert.True(t, ok, "contents scope must be present")
	assert.Equal(t, workflow.PermissionRead, contentsLevel, "contents must be read")

	// Spec: the permissions factory uses "organization-projects" for GitHub App tokens
	projLevel, ok := p.GetExplicit(workflow.PermissionOrganizationProj)
	assert.True(t, ok, "organization-projects scope must be explicitly set")
	assert.Equal(t, workflow.PermissionWrite, projLevel, "organization-projects must be write")
}

// TestSpec_SafeOutputs_SafeOutputsConfigFromKeys validates that SafeOutputsConfigFromKeys
// builds a minimal SafeOutputsConfig from a list of safe-output key names, as documented
// in the workflow package README.md.
// Spec: "SafeOutputsConfigFromKeys — Creates a config from a list of type keys"
func TestSpec_SafeOutputs_SafeOutputsConfigFromKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		validate func(t *testing.T, cfg *workflow.SafeOutputsConfig)
	}{
		{
			name: "empty keys returns empty config",
			keys: []string{},
			validate: func(t *testing.T, cfg *workflow.SafeOutputsConfig) {
				require.NotNil(t, cfg, "SafeOutputsConfigFromKeys must never return nil")
				assert.False(t, workflow.HasSafeOutputsEnabled(cfg), "empty key list must produce no enabled outputs")
			},
		},
		{
			name: "add-comment key enables the add-comment output",
			keys: []string{"add-comment"},
			validate: func(t *testing.T, cfg *workflow.SafeOutputsConfig) {
				require.NotNil(t, cfg, "config must not be nil")
				assert.True(t, workflow.HasSafeOutputsEnabled(cfg), "add-comment key must enable safe outputs")
			},
		},
		{
			name: "create-issue key enables the create-issue output",
			keys: []string{"create-issue"},
			validate: func(t *testing.T, cfg *workflow.SafeOutputsConfig) {
				require.NotNil(t, cfg, "config must not be nil")
				assert.True(t, workflow.HasSafeOutputsEnabled(cfg), "create-issue key must enable safe outputs")
			},
		},
		{
			name: "create-pull-request key enables the create-pull-request output",
			keys: []string{"create-pull-request"},
			validate: func(t *testing.T, cfg *workflow.SafeOutputsConfig) {
				require.NotNil(t, cfg, "config must not be nil")
				assert.True(t, workflow.HasSafeOutputsEnabled(cfg), "create-pull-request key must enable safe outputs")
			},
		},
		{
			name: "multiple keys enable multiple outputs",
			keys: []string{"add-comment", "close-issue", "create-pull-request"},
			validate: func(t *testing.T, cfg *workflow.SafeOutputsConfig) {
				require.NotNil(t, cfg, "config must not be nil")
				assert.True(t, workflow.HasSafeOutputsEnabled(cfg), "multiple keys must enable safe outputs")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := workflow.SafeOutputsConfigFromKeys(tt.keys)
			tt.validate(t, cfg)
		})
	}
}
