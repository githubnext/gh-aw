//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormal_CTR016_NilManifestSkipsEnforcement(t *testing.T) {
	err := EnforceSafeUpdate(nil, []string{"MY_SECRET"}, []string{"evil-org/action@deadbeef # v1"}, "")
	require.NoError(t, err)
}

func TestFormal_CTR016_EmptyManifestRejectsNewSecret(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"MY_SECRET"}, nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MY_SECRET")
}

func TestFormal_CTR016_GitHubTokenExempt_BareForm(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"GITHUB_TOKEN"}, nil, "")
	require.NoError(t, err)
}

func TestFormal_CTR016_GitHubTokenExempt_PrefixedForm(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"secrets.GITHUB_TOKEN"}, nil, "")
	require.NoError(t, err)
}

func TestFormal_CTR016_GhAwInternalSecretExempt(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"GH_AW_GITHUB_TOKEN"}, nil, "")
	require.NoError(t, err)
}

func TestFormal_CTR016_SecretPrefixNormalization(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Secrets: []string{"MY_SECRET"}}
	err := EnforceSafeUpdate(manifest, []string{"secrets.MY_SECRET"}, nil, "")
	require.NoError(t, err)
}

func TestFormal_CTR016_NewActionDriftRejected(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc1234", Version: "v4"}}}
	err := EnforceSafeUpdate(manifest, nil, []string{"actions/checkout@abc1234 # v4", "evil-org/steal@deadbeef # v1"}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evil-org/steal")
}

func TestFormal_CTR016_RemovedActionDriftRejected(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Actions: []GHAWManifestAction{{Repo: "my-org/approved-action", SHA: "abc1234", Version: "v1"}}}
	err := EnforceSafeUpdate(manifest, nil, []string{}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Previously-approved action")
	assert.Contains(t, err.Error(), "my-org/approved-action")
}

func TestFormal_CTR016_KnownActionPinUpdateAllowed(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Actions: []GHAWManifestAction{{Repo: "my-org/action", SHA: "abc1234", Version: "v1"}}}
	err := EnforceSafeUpdate(manifest, nil, []string{"my-org/action@def5678 # v2"}, "")
	require.NoError(t, err)
}

func TestFormal_CTR016_RedirectWhitespaceNormalization(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Redirect: "owner/repo/workflows/new.md@main"}
	err := EnforceSafeUpdate(manifest, nil, nil, "  owner/repo/workflows/new.md@main  ")
	require.NoError(t, err)
}

func TestFormal_CTR016_RedirectChangeRejected(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Redirect: "owner/repo/workflows/old.md@main"}
	err := EnforceSafeUpdate(manifest, nil, nil, "owner/repo/workflows/new.md@main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "New redirect configured")
	assert.Contains(t, err.Error(), "Previously-approved redirect removed")
}

func TestFormal_CTR001_WritePermissionsRejected(t *testing.T) {
	// PermissionIdToken: id-token:write is allowed for OIDC auth and does not grant repo write access.
	// PermissionMetadata: metadata is always implicitly read-only, so it is excluded from the write-rejection rule.
	for _, scope := range GetAllPermissionScopes() {
		if scope == PermissionIdToken || scope == PermissionMetadata {
			continue
		}
		t.Run(string(scope), func(t *testing.T) {
			perms := NewPermissions()
			perms.Set(scope, PermissionWrite)
			err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "write permissions")
		})
	}
}

func TestFormal_CTR001_ReadOnlyPermissionsAllowed(t *testing.T) {
	perms := NewPermissions()
	for _, scope := range GetAllPermissionScopes() {
		// PermissionIdToken is intentionally omitted because GitHub Actions treats it as write-or-absent, not read-or-write.
		if scope == PermissionIdToken {
			continue
		}
		perms.Set(scope, PermissionRead)
	}
	err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
	require.NoError(t, err)
}

func TestFormal_CTR001_EmptyPermissionsAllowed(t *testing.T) {
	err := validateDangerousPermissions(&WorkflowData{Permissions: ""}, NewPermissions())
	require.NoError(t, err)
}

func TestFormal_CTR011_AllowURLsRequiresSSLBump(t *testing.T) {
	err := validateNetworkFirewallConfig(&NetworkPermissions{
		Firewall: &FirewallConfig{
			AllowURLs: []string{"https://github.com/githubnext/*"},
			SSLBump:   false,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allow-urls requires ssl-bump: true")
}

func TestFormal_CTR011_AllowURLsWithSSLBumpAllowed(t *testing.T) {
	err := validateNetworkFirewallConfig(&NetworkPermissions{
		Firewall: &FirewallConfig{
			AllowURLs: []string{"https://github.com/githubnext/*"},
			SSLBump:   true,
		},
	})
	require.NoError(t, err)
}

func TestFormal_CTR011_WildcardOnlyDomainRejected(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateStrictNetwork(&NetworkPermissions{Allowed: []string{"*"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard '*' is not allowed")
}

func TestFormal_CTR015_WildcardLabelRejected(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(&SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{AllowedLabels: []string{"*"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CTR-015")
}

func TestFormal_CTR015_WildcardLabelRejected_CreateDiscussion(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(&SafeOutputsConfig{
		CreateDiscussions: &CreateDiscussionsConfig{AllowedLabels: []string{"*"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CTR-015")
}

func TestFormal_CTR015_SpecificLabelsAllowed(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(&SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{AllowedLabels: []string{"bug", "team-*"}},
	})
	require.NoError(t, err)
}

func TestFormal_CTR015_NilConfigAllowed(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(nil)
	require.NoError(t, err)
}

func TestFormal_CTR014_StrictModeEnabledRejected(t *testing.T) {
	compiler := NewCompiler()
	compiler.SetStrictMode(true)
	err := compiler.validateRunInstallScripts(&WorkflowData{RunInstallScripts: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict mode")
}

func TestFormal_CTR014_DisabledAlwaysAllowed(t *testing.T) {
	t.Run("strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.SetStrictMode(true)
		err := compiler.validateRunInstallScripts(&WorkflowData{RunInstallScripts: false})
		require.NoError(t, err)
	})

	t.Run("non-strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		err := compiler.validateRunInstallScripts(&WorkflowData{RunInstallScripts: false})
		require.NoError(t, err)
	})
}
