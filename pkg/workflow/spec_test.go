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

// TestSpec_SafeOutputs_HasSafeOutputsEnabled validates the documented behavior of
// HasSafeOutputsEnabled as described in the workflow package README.md.
// Spec: "HasSafeOutputsEnabled — Returns whether any safe-output type is enabled".
func TestSpec_SafeOutputs_HasSafeOutputsEnabled(t *testing.T) {
	t.Run("nil config reports no safe outputs enabled", func(t *testing.T) {
		assert.False(t, workflow.HasSafeOutputsEnabled(nil),
			"a nil SafeOutputsConfig must report no safe outputs enabled")
	})

	t.Run("empty config reports no safe outputs enabled", func(t *testing.T) {
		assert.False(t, workflow.HasSafeOutputsEnabled(&workflow.SafeOutputsConfig{}),
			"an empty SafeOutputsConfig must report no safe outputs enabled")
	})

	t.Run("config built from a key reports safe outputs enabled", func(t *testing.T) {
		cfg := workflow.SafeOutputsConfigFromKeys([]string{"add-comment"})
		assert.True(t, workflow.HasSafeOutputsEnabled(cfg),
			"a config with an enabled output must report safe outputs enabled")
	})
}

// TestSpec_SecretHandling_ExtractSecretName validates the documented behavior of
// ExtractSecretName as described in the workflow package README.md.
// Spec: "ExtractSecretName — Extracts the secret name from a ${{ secrets.NAME }} expression".
func TestSpec_SecretHandling_ExtractSecretName(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "extracts name from a secrets expression",
			value:    "${{ secrets.MY_TOKEN }}",
			expected: "MY_TOKEN",
		},
		{
			name:     "extracts name from an expression with a default fallback",
			value:    "${{ secrets.DD_SITE || 'datadoghq.com' }}",
			expected: "DD_SITE",
		},
		{
			name:     "returns empty string for a plain value with no secret",
			value:    "plain value",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := workflow.ExtractSecretName(tt.value)
			assert.Equal(t, tt.expected, result,
				"ExtractSecretName(%q) should match documented behavior", tt.value)
		})
	}
}

// TestSpec_Engine_RegistryLookupAndIdentity validates the documented engine-registry
// lookup and the Engine identity contract from the workflow package README.md.
//
// Spec ("Look up an engine" usage example): a global registry is obtained via
// GetGlobalEngineRegistry() and an engine is looked up by name. Spec (Engine
// interface): "Core identity: GetID(), GetDisplayName(), GetDescription(), IsExperimental()".
//
// SPEC_MISMATCH: The README usage example shows `engine, ok := registry.Get("copilot")`
// returning a (engine, bool) pair, but the implemented method is
// `GetEngine(id string) (CodingAgentEngine, error)`. There is no `Get` method.
// This test exercises the actual API and documents the divergence.
func TestSpec_Engine_RegistryLookupAndIdentity(t *testing.T) {
	registry := workflow.GetGlobalEngineRegistry()
	require.NotNil(t, registry, "GetGlobalEngineRegistry() must return a non-nil registry")

	// SPEC_MISMATCH: spec example uses registry.Get(name) (engine, ok);
	// the real API is registry.GetEngine(name) (engine, error).
	engine, err := registry.GetEngine("copilot")
	require.NoError(t, err, "the documented copilot engine must be resolvable")
	require.NotNil(t, engine, "resolved engine must be non-nil")

	assert.Equal(t, "copilot", engine.GetID(),
		"Engine.GetID() must return the documented identity")
	assert.NotEmpty(t, engine.GetDisplayName(),
		"Engine.GetDisplayName() must return a non-empty display name")
	assert.NotEmpty(t, engine.GetDescription(),
		"Engine.GetDescription() must return a non-empty description")
}

// TestSpec_Engine_DocumentedEnginesRegistered validates that every AI engine documented
// in the workflow package README.md is registered and reports the documented identity.
// Spec: the engine architecture lists copilot, claude, codex, gemini, crush, opencode,
// pi, and antigravity engines, each created by a New<Name>Engine constructor.
func TestSpec_Engine_DocumentedEnginesRegistered(t *testing.T) {
	registry := workflow.GetGlobalEngineRegistry()
	require.NotNil(t, registry, "GetGlobalEngineRegistry() must return a non-nil registry")

	documentedEngines := []string{
		"copilot", "claude", "codex", "gemini",
		"crush", "opencode", "pi", "antigravity",
	}

	for _, id := range documentedEngines {
		t.Run(id, func(t *testing.T) {
			engine, err := registry.GetEngine(id)
			require.NoError(t, err, "documented engine %q must be registered", id)
			require.NotNil(t, engine, "documented engine %q must be non-nil", id)
			assert.Equal(t, id, engine.GetID(),
				"engine %q must report its documented ID via GetID()", id)
		})
	}
}

// TestSpec_Engine_GlobalRegistrySingleton validates the documented thread-safety contract that
// GetGlobalEngineRegistry returns a singleton initialized once at startup.
// Spec ("Thread Safety"): "The GetGlobalEngineRegistry() singleton is initialized once at startup
// and is safe for concurrent reads thereafter."
func TestSpec_Engine_GlobalRegistrySingleton(t *testing.T) {
	first := workflow.GetGlobalEngineRegistry()
	second := workflow.GetGlobalEngineRegistry()
	require.NotNil(t, first, "GetGlobalEngineRegistry() must return a non-nil registry")
	assert.Same(t, first, second,
		"GetGlobalEngineRegistry() must return the same singleton instance on repeated calls")
}

// TestSpec_Sandbox_Constants validates the documented SandboxType constant values.
// Spec ("Sandbox Constants"): SandboxTypeAWF = "awf", SandboxTypeDefault = "default" (alias for AWF).
func TestSpec_Sandbox_Constants(t *testing.T) {
	assert.Equal(t, "awf", string(workflow.SandboxTypeAWF),
		"SandboxTypeAWF must equal the documented value")
	assert.Equal(t, "default", string(workflow.SandboxTypeDefault),
		"SandboxTypeDefault must equal the documented value")
}

// TestSpec_MCPScripts_Constants validates the documented MCP Scripts constants.
// Spec ("MCP Scripts Constants"): MCPScriptsModeHTTP = "http" is the only supported transport mode;
// MCPScriptsDirectory is the runtime directory where MCP scripts files are generated.
func TestSpec_MCPScripts_Constants(t *testing.T) {
	assert.Equal(t, "http", workflow.MCPScriptsModeHTTP,
		"MCPScriptsModeHTTP must be the documented http transport mode")
	assert.NotEmpty(t, workflow.MCPScriptsDirectory,
		"MCPScriptsDirectory must be a non-empty runtime directory path")
}

// TestSpec_ActionPinning_ActionMode validates the documented ActionMode alias and DetectActionMode.
// Spec ("Action Pinning"): ActionMode is an "Action reference mode" with DetectActionMode detecting it.
//
// SPEC_MISMATCH: The README documents ActionMode values as `sha`, `tag`, `local`, but the
// implementation defines them as `dev`, `release`, `script`, and `action`. The README also
// describes DetectActionMode as detecting the "action reference mode" from a version string, but
// the implementation ignores the version parameter and instead detects from build/release context
// (the GH_AW_ACTION_MODE override, the release build flag, and GitHub Actions ref/event context).
// This test exercises the actual API and documents the divergence.
func TestSpec_ActionPinning_ActionMode(t *testing.T) {
	// SPEC_MISMATCH: documented values (sha/tag/local) do not exist; the real values are these.
	assert.Equal(t, "dev", string(workflow.ActionModeDev), "ActionModeDev value")
	assert.Equal(t, "release", string(workflow.ActionModeRelease), "ActionModeRelease value")

	// DetectActionMode honors the GH_AW_ACTION_MODE override deterministically, independent of the
	// (unused) version argument.
	t.Setenv("GH_AW_ACTION_MODE", string(workflow.ActionModeRelease))
	mode := workflow.DetectActionMode("ignored-version")
	assert.Equal(t, workflow.ActionModeRelease, mode,
		"DetectActionMode must honor the GH_AW_ACTION_MODE override regardless of the version arg")
}
