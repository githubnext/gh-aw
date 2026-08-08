package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParsePermissionsConfig_Shorthand verifies shorthand permission strings
// (e.g. "read-all", "write-all") are recognized regardless of whether they
// appear as the key or the value of the single-entry map.
func TestParsePermissionsConfig_Shorthand(t *testing.T) {
	tests := []struct {
		name        string
		permissions map[string]any
		wantShort   string
	}{
		{
			name:        "shorthand as value with contents key",
			permissions: map[string]any{"contents": "read-all"},
			wantShort:   "read-all",
		},
		{
			name:        "shorthand as key with any value",
			permissions: map[string]any{"write-all": "ignored"},
			wantShort:   "write-all",
		},
		{
			name:        "shorthand read",
			permissions: map[string]any{"contents": "read"},
			wantShort:   "read",
		},
		{
			name:        "shorthand write",
			permissions: map[string]any{"contents": "write"},
			wantShort:   "write",
		},
		{
			name:        "shorthand none",
			permissions: map[string]any{"contents": "none"},
			wantShort:   "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parsePermissionsConfig(tt.permissions)
			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Equal(t, tt.wantShort, config.Shorthand)
		})
	}
}

// TestParsePermissionsConfig_ShorthandNotRecognized verifies that a single-entry
// map whose key and value are both non-shorthand strings falls through to
// detailed parsing instead of being treated as shorthand.
func TestParsePermissionsConfig_ShorthandNotRecognized(t *testing.T) {
	config, err := parsePermissionsConfig(map[string]any{"contents": "custom-level"})
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Empty(t, config.Shorthand)
	assert.Equal(t, "custom-level", config.Contents)
}

// TestParsePermissionsConfig_ShorthandNonStringValue verifies a single-entry map
// whose value isn't a string is not treated as shorthand.
func TestParsePermissionsConfig_ShorthandNonStringValue(t *testing.T) {
	config, err := parsePermissionsConfig(map[string]any{"contents": 42})
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Empty(t, config.Shorthand)
	assert.Empty(t, config.Contents)
}

// TestParsePermissionsConfig_DetailedActionsScopes verifies all GitHub Actions
// GITHUB_TOKEN permission scopes are parsed into the corresponding fields.
func TestParsePermissionsConfig_DetailedActionsScopes(t *testing.T) {
	permissions := map[string]any{
		"actions":               "read",
		"checks":                "write",
		"contents":              "read",
		"deployments":           "write",
		"id-token":              "write",
		"issues":                "write",
		"discussions":           "read",
		"packages":              "read",
		"pages":                 "write",
		"pull-requests":         "write",
		"repository-projects":   "read",
		"security-events":       "write",
		"statuses":              "read",
		"vulnerability-alerts":  "read",
		"organization-projects": "read",
	}

	config, err := parsePermissionsConfig(permissions)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Empty(t, config.Shorthand)
	assert.Equal(t, "read", config.Actions)
	assert.Equal(t, "write", config.Checks)
	assert.Equal(t, "read", config.Contents)
	assert.Equal(t, "write", config.Deployments)
	assert.Equal(t, "write", config.IDToken)
	assert.Equal(t, "write", config.Issues)
	assert.Equal(t, "read", config.Discussions)
	assert.Equal(t, "read", config.Packages)
	assert.Equal(t, "write", config.Pages)
	assert.Equal(t, "write", config.PullRequests)
	assert.Equal(t, "read", config.RepositoryProjects)
	assert.Equal(t, "write", config.SecurityEvents)
	assert.Equal(t, "read", config.Statuses)
	assert.Equal(t, "read", config.VulnerabilityAlerts)
	assert.Equal(t, "read", config.OrganizationProjects)
}

// TestParsePermissionsConfig_DetailedAppScopes verifies GitHub App-only scopes
// are parsed into the corresponding fields.
func TestParsePermissionsConfig_DetailedAppScopes(t *testing.T) {
	permissions := map[string]any{
		"administration":                              "write",
		"environments":                                "read",
		"git-signing":                                 "write",
		"workflows":                                   "write",
		"repository-hooks":                            "read",
		"single-file":                                 "read",
		"codespaces":                                  "write",
		"repository-custom-properties":                "read",
		"members":                                     "write",
		"organization-administration":                 "write",
		"team-discussions":                            "read",
		"organization-hooks":                          "read",
		"organization-members":                        "read",
		"organization-packages":                       "read",
		"organization-self-hosted-runners":            "write",
		"organization-custom-org-roles":               "write",
		"organization-custom-properties":              "read",
		"organization-custom-repository-roles":        "write",
		"organization-announcement-banners":           "read",
		"organization-events":                         "read",
		"organization-plan":                           "read",
		"organization-user-blocking":                  "write",
		"organization-personal-access-token-requests": "write",
		"organization-personal-access-tokens":         "write",
		"organization-copilot":                        "read",
		"organization-codespaces":                     "write",
		"email-addresses":                             "read",
		"codespaces-lifecycle-admin":                  "write",
		"codespaces-metadata":                         "read",
	}

	config, err := parsePermissionsConfig(permissions)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "write", config.Administration)
	assert.Equal(t, "read", config.Environments)
	assert.Equal(t, "write", config.GitSigning)
	assert.Equal(t, "write", config.Workflows)
	assert.Equal(t, "read", config.RepositoryHooks)
	assert.Equal(t, "read", config.SingleFile)
	assert.Equal(t, "write", config.Codespaces)
	assert.Equal(t, "read", config.RepositoryCustomProperties)
	assert.Equal(t, "write", config.Members)
	assert.Equal(t, "write", config.OrganizationAdministration)
	assert.Equal(t, "read", config.TeamDiscussions)
	assert.Equal(t, "read", config.OrganizationHooks)
	assert.Equal(t, "read", config.OrganizationMembers)
	assert.Equal(t, "read", config.OrganizationPackages)
	assert.Equal(t, "write", config.OrganizationSelfHostedRunners)
	assert.Equal(t, "write", config.OrganizationCustomOrgRoles)
	assert.Equal(t, "read", config.OrganizationCustomProperties)
	assert.Equal(t, "write", config.OrganizationCustomRepositoryRoles)
	assert.Equal(t, "read", config.OrganizationAnnouncementBanners)
	assert.Equal(t, "read", config.OrganizationEvents)
	assert.Equal(t, "read", config.OrganizationPlan)
	assert.Equal(t, "write", config.OrganizationUserBlocking)
	assert.Equal(t, "write", config.OrganizationPersonalAccessTokenReqs)
	assert.Equal(t, "write", config.OrganizationPersonalAccessTokens)
	assert.Equal(t, "read", config.OrganizationCopilot)
	assert.Equal(t, "write", config.OrganizationCodespaces)
	assert.Equal(t, "read", config.EmailAddresses)
	assert.Equal(t, "write", config.CodespacesLifecycleAdmin)
	assert.Equal(t, "read", config.CodespacesMetadata)
}

// TestParsePermissionsConfig_UnknownScopeIgnored verifies unknown scope keys and
// non-string level values are silently ignored rather than causing an error.
func TestParsePermissionsConfig_UnknownScopeIgnored(t *testing.T) {
	permissions := map[string]any{
		"contents":      "read",
		"unknown-scope": "write",
		"issues":        42, // non-string value ignored
	}

	config, err := parsePermissionsConfig(permissions)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "read", config.Contents)
	assert.Empty(t, config.Issues)
}

// TestParsePermissionsConfig_EmptyMap verifies an empty permissions map produces
// a zero-value config without error.
func TestParsePermissionsConfig_EmptyMap(t *testing.T) {
	config, err := parsePermissionsConfig(map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, &PermissionsConfig{}, config)
}

// TestParsePermissionsConfig_MultipleEntriesNeverShorthand verifies that maps with
// more than one entry always go through detailed parsing, even if a shorthand
// value is present among them.
func TestParsePermissionsConfig_MultipleEntriesNeverShorthand(t *testing.T) {
	permissions := map[string]any{
		"contents": "read-all",
		"issues":   "write",
	}
	config, err := parsePermissionsConfig(permissions)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Empty(t, config.Shorthand)
	assert.Equal(t, "read-all", config.Contents)
	assert.Equal(t, "write", config.Issues)
}
