//go:build !integration

package cli

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultsCommand(t *testing.T) {
	cmd := NewDefaultsCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "defaults", cmd.Use)

	var hasGet, hasUpdate bool
	for _, sub := range cmd.Commands() {
		if sub.Name() == "get" {
			hasGet = true
		}
		if sub.Name() == "update" {
			hasUpdate = true
		}
	}
	assert.True(t, hasGet, "defaults command should include get subcommand")
	assert.True(t, hasUpdate, "defaults command should include update subcommand")
}

func TestResolveDefaultsTarget(t *testing.T) {
	orig := defaultsGetCurrentRepoSlug
	defaultsGetCurrentRepoSlug = func() (string, error) { return "octo-org/example", nil }
	t.Cleanup(func() {
		defaultsGetCurrentRepoSlug = orig
	})

	t.Run("repo default scope uses current repo", func(t *testing.T) {
		target, err := resolveDefaultsTarget("", "", "", "", false)
		require.NoError(t, err)
		assert.Equal(t, defaultsScopeRepo, target.scope)
		assert.Equal(t, "octo-org", target.repoOwner)
		assert.Equal(t, "example", target.repoName)
	})

	t.Run("update requires scope", func(t *testing.T) {
		_, err := resolveDefaultsTarget("", "", "", "", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scope is required")
	})

	t.Run("org scope infers owner from repo", func(t *testing.T) {
		target, err := resolveDefaultsTarget(defaultsScopeOrg, "github/gh-aw", "", "", false)
		require.NoError(t, err)
		assert.Equal(t, defaultsScopeOrg, target.scope)
		assert.Equal(t, "github", target.org)
	})

	t.Run("ent scope requires enterprise", func(t *testing.T) {
		_, err := resolveDefaultsTarget(defaultsScopeEnt, "", "", "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--enterprise")
	})
}

func TestDefaultsFileYAMLKeys(t *testing.T) {
	file := defaultsFile{
		DefaultMaxEffectiveTokens: "10000",
		DefaultMaxTurns:           "42",
		DefaultTimeoutMinutes:     "90",
		DefaultDetectionModel:     "claude-sonnet-4.6",
		DefaultModelCopilot:       "claude-sonnet-4.7",
		DefaultModelClaude:        "claude-opus-4.7",
		DefaultModelCodex:         "gpt-5.5",
	}

	data, err := yaml.Marshal(&file)
	require.NoError(t, err)

	yml := string(data)
	assert.Contains(t, yml, "default_max_effective_tokens:")
	assert.Contains(t, yml, "default_max_turns:")
	assert.Contains(t, yml, "default_timeout_minutes:")
	assert.Contains(t, yml, "default_detection_model:")
	assert.Contains(t, yml, "default_model_copilot:")
	assert.Contains(t, yml, "default_model_claude:")
	assert.Contains(t, yml, "default_model_codex:")
}

func TestDefaultsTargetEndpoints(t *testing.T) {
	repoTarget := defaultsTarget{scope: defaultsScopeRepo, repoOwner: "github", repoName: "gh-aw"}
	orgTarget := defaultsTarget{scope: defaultsScopeOrg, org: "github"}
	entTarget := defaultsTarget{scope: defaultsScopeEnt, enterprise: "octo-ent"}

	assert.Equal(t, "repos/github/gh-aw/actions/variables", repoTarget.variablesEndpoint())
	assert.Equal(t, "orgs/github/actions/variables", orgTarget.variablesEndpoint())
	assert.Equal(t, "enterprises/octo-ent/actions/variables", entTarget.variablesEndpoint())
	assert.Equal(t, "repos/github/gh-aw/actions/variables/GH_AW_DEFAULT_MAX_TURNS", repoTarget.variableEndpoint("GH_AW_DEFAULT_MAX_TURNS"))
}
