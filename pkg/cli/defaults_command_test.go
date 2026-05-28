//go:build !integration

package cli

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultsCommand(t *testing.T) {
	cmd := NewDefaultsCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "defaults", cmd.Use)

	var updateCmd *cobra.Command
	var hasGet, hasUpdate bool
	for _, sub := range cmd.Commands() {
		if sub.Name() == "get" {
			hasGet = true
		}
		if sub.Name() == "update" {
			hasUpdate = true
			updateCmd = sub
		}
	}
	assert.True(t, hasGet, "defaults command should include get subcommand")
	assert.True(t, hasUpdate, "defaults command should include update subcommand")
	require.NotNil(t, updateCmd)
	assert.NotNil(t, updateCmd.Flags().Lookup("yes"))
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
		MaxEffectiveTokens: "10000",
		MaxTurns:           "42",
		TimeoutMinutes:     "90",
		DetectionModel:     "claude-sonnet-4.6",
		ModelCopilot:       "claude-sonnet-4.7",
		ModelClaude:        "claude-opus-4.7",
		ModelCodex:         "gpt-5.5",
	}

	data, err := yaml.Marshal(&file)
	require.NoError(t, err)

	yml := string(data)
	assert.Contains(t, yml, "max_effective_tokens:")
	assert.Contains(t, yml, "max_turns:")
	assert.Contains(t, yml, "timeout_minutes:")
	assert.Contains(t, yml, "detection_model:")
	assert.Contains(t, yml, "model_copilot:")
	assert.Contains(t, yml, "model_claude:")
	assert.Contains(t, yml, "model_codex:")
	assert.NotContains(t, yml, "default_")
}

func TestDefaultsFileYAMLDoesNotReadLegacyKeys(t *testing.T) {
	var file defaultsFile

	err := yaml.Unmarshal([]byte("default_max_turns: \"42\"\n"), &file)
	require.NoError(t, err)

	assert.Empty(t, file.MaxTurns)
}

func TestDefaultsFileYAMLReadsTrimmedKeys(t *testing.T) {
	var file defaultsFile

	err := yaml.Unmarshal([]byte("max_turns: \"42\"\nmodel_copilot: gpt-5-mini\n"), &file)
	require.NoError(t, err)

	assert.Equal(t, "42", file.MaxTurns)
	assert.Equal(t, "gpt-5-mini", file.ModelCopilot)
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

func TestDefaultsBuildUpdateChanges(t *testing.T) {
	changes := defaultsBuildUpdateChanges(&defaultsFile{
		MaxEffectiveTokens: " 10000 ",
		ModelCodex:         "gpt-5.5",
	})

	require.Len(t, changes, len(defaultsBindings))
	assert.Equal(t, "max_effective_tokens", changes[0].field)
	assert.Equal(t, "10000", changes[0].value)
	assert.False(t, changes[0].delete)
	assert.Equal(t, "max_turns", changes[1].field)
	assert.True(t, changes[1].delete)
	assert.Equal(t, "model_codex", changes[len(changes)-1].field)
	assert.Equal(t, "gpt-5.5", changes[len(changes)-1].value)
}

func TestConfirmDefaultsUpdate(t *testing.T) {
	target := defaultsTarget{scope: defaultsScopeOrg, org: "github"}
	changes := []defaultsUpdateChange{{field: "max_turns", value: "42"}}

	t.Run("requests confirmation by default", func(t *testing.T) {
		called := false
		confirmAction := func(title, affirmative, negative string) (bool, error) {
			called = true
			assert.Equal(t, "Do you want to update these defaults?", title)
			assert.Equal(t, "Yes, update", affirmative)
			assert.Equal(t, "No, cancel", negative)
			return true, nil
		}

		err := confirmDefaultsUpdate(target, "defaults.yml", changes, false, confirmAction)
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("skips confirmation with yes", func(t *testing.T) {
		confirmAction := func(title, affirmative, negative string) (bool, error) {
			t.Fatal("confirmation should be skipped")
			return false, nil
		}

		err := confirmDefaultsUpdate(target, "defaults.yml", changes, true, confirmAction)
		require.NoError(t, err)
	})

	t.Run("returns cancellation error", func(t *testing.T) {
		confirmAction := func(title, affirmative, negative string) (bool, error) {
			return false, nil
		}

		err := confirmDefaultsUpdate(target, "defaults.yml", changes, false, confirmAction)
		require.ErrorContains(t, err, "defaults update cancelled")
	})
}
