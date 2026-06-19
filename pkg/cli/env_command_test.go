//go:build !integration

package cli

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnvCommand(t *testing.T) {
	cmd := NewEnvCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "env", cmd.Use)

	var getCmd, updateCmd *cobra.Command
	var hasGet, hasUpdate bool
	for _, sub := range cmd.Commands() {
		if sub.Name() == "get" {
			hasGet = true
			getCmd = sub
		}
		if sub.Name() == "update" {
			hasUpdate = true
			updateCmd = sub
		}
	}
	assert.True(t, hasGet, "env command should include get subcommand")
	assert.True(t, hasUpdate, "env command should include update subcommand")
	require.NotNil(t, getCmd)
	require.NotNil(t, updateCmd)
	assert.NotEmpty(t, getCmd.Long)
	assert.Contains(t, getCmd.Long, "file")
	assert.Contains(t, getCmd.Long, "scope")
	assert.Contains(t, getCmd.Long, "--enterprise")
	assert.NotEmpty(t, updateCmd.Long)
	assert.Contains(t, updateCmd.Long, "file")
	assert.Contains(t, updateCmd.Long, "scope")
	assert.Contains(t, updateCmd.Long, "--dry-run")
	assert.Contains(t, updateCmd.Long, "--yes")
	assert.NotNil(t, updateCmd.Flags().Lookup("yes"))
	assert.NotNil(t, updateCmd.Flags().Lookup("dry-run"))
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
		DefaultMaxAICredits:          new("1000"),
		DefaultMaxTurnCacheMisses:    new("5"),
		DefaultDetectionMaxAICredits: new("400"),
		DefaultMaxDailyAICredits:     new("500000"),
		DefaultMaxTurns:              new("42"),
		DefaultTimeoutMinutes:        new("90"),
		DefaultDetectionModel:        new("claude-sonnet-4.6"),
		DefaultUTC:                   new("-08:00"),
		DefaultModelCopilot:          new("claude-sonnet-4.7"),
		DefaultModelClaude:           new("claude-opus-4.7"),
		DefaultModelCodex:            new("gpt-5.5"),
	}

	data, err := yaml.Marshal(&file)
	require.NoError(t, err)

	yml := string(data)
	assert.Contains(t, yml, "default_max_ai_credits:")
	assert.Contains(t, yml, "default_max_turn_cache_misses:")
	assert.Contains(t, yml, "default_detection_max_ai_credits:")
	assert.Contains(t, yml, "default_max_daily_ai_credits:")
	assert.Contains(t, yml, "default_max_turns:")
	assert.Contains(t, yml, "default_timeout_minutes:")
	assert.Contains(t, yml, "default_detection_model:")
	assert.Contains(t, yml, "default_utc:")
	assert.Contains(t, yml, "default_model_copilot:")
	assert.Contains(t, yml, "default_model_claude:")
	assert.Contains(t, yml, "default_model_codex:")
}

func TestDefaultsFileYAMLNullDelete(t *testing.T) {
	t.Run("null value unmarshals to nil pointer", func(t *testing.T) {
		var file defaultsFile
		err := yaml.Unmarshal([]byte("default_max_turns: null\n"), &file)
		require.NoError(t, err)
		assert.Nil(t, file.DefaultMaxTurns)
	})

	t.Run("string value unmarshals to non-nil pointer", func(t *testing.T) {
		var file defaultsFile
		err := yaml.Unmarshal([]byte("default_max_turns: \"42\"\ndefault_model_copilot: gpt-5-mini\n"), &file)
		require.NoError(t, err)
		require.NotNil(t, file.DefaultMaxTurns)
		assert.Equal(t, "42", *file.DefaultMaxTurns)
		require.NotNil(t, file.DefaultModelCopilot)
		assert.Equal(t, "gpt-5-mini", *file.DefaultModelCopilot)
	})

	t.Run("absent key unmarshals to nil pointer", func(t *testing.T) {
		var file defaultsFile
		err := yaml.Unmarshal([]byte("default_model_copilot: gpt-5-mini\n"), &file)
		require.NoError(t, err)
		assert.Nil(t, file.DefaultMaxTurns)
	})
}

func TestDefaultsParseFileDisallowsUnknownFields(t *testing.T) {
	_, err := defaultsParseFile("defaults.yml", []byte("default_max_turns: \"42\"\ndefault_model_copliot: gpt-5-mini\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default_model_copliot")
}

func TestDefaultsValidateFile(t *testing.T) {
	t.Run("accepts valid values", func(t *testing.T) {
		err := defaultsValidateFile(&defaultsFile{
			DefaultMaxAICredits:          new("1000"),
			DefaultMaxTurnCacheMisses:    new("5"),
			DefaultDetectionMaxAICredits: new("400"),
			DefaultMaxDailyAICredits:     new("500000"),
			DefaultMaxTurns:              new("12"),
			DefaultTimeoutMinutes:        new("30"),
			DefaultDetectionModel:        new("claude-sonnet-4.6"),
			DefaultUTC:                   new("-08:00"),
			DefaultModelCopilot:          new("gpt-5-mini"),
			DefaultModelClaude:           new("claude-haiku-4.5"),
			DefaultModelCodex:            new("gpt-5.4-mini"),
		})
		require.NoError(t, err)
	})

	t.Run("accepts -1 to disable detection budget steering", func(t *testing.T) {
		err := defaultsValidateFile(&defaultsFile{
			DefaultDetectionMaxAICredits: new("-1"),
		})
		require.NoError(t, err)
	})

	t.Run("rejects invalid numeric and empty model values", func(t *testing.T) {
		err := defaultsValidateFile(&defaultsFile{
			DefaultMaxAICredits:          new("0"),
			DefaultMaxTurnCacheMisses:    new("0"),
			DefaultDetectionMaxAICredits: new("0"),
			DefaultMaxDailyAICredits:     new("0"),
			DefaultMaxTurns:              new("abc"),
			DefaultTimeoutMinutes:        new("0"),
			DefaultUTC:                   new("west"),
			DefaultModelCopilot:          new("   "),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "default_max_ai_credits must be a non-zero integer when set")
		assert.Contains(t, err.Error(), "default_max_turn_cache_misses must be a positive integer when set")
		assert.Contains(t, err.Error(), "default_detection_max_ai_credits must be a non-zero integer when set")
		assert.Contains(t, err.Error(), "default_max_daily_ai_credits must be a non-zero integer when set")
		assert.Contains(t, err.Error(), "default_max_turns must be a positive integer when set")
		assert.Contains(t, err.Error(), "default_timeout_minutes must be a positive integer when set")
		assert.Contains(t, err.Error(), "default_utc must be a numeric UTC offset")
		assert.Contains(t, err.Error(), "default_model_copilot cannot be empty when set")
	})
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
		DefaultModelCodex: new("gpt-5.5"),
	})

	require.Len(t, changes, len(defaultsBindings))

	byField := make(map[string]defaultsUpdateChange, len(changes))
	for _, change := range changes {
		byField[change.field] = change
	}

	for _, field := range []string{
		"default_max_ai_credits",
		"default_max_turn_cache_misses",
		"default_detection_max_ai_credits",
		"default_max_daily_ai_credits",
		"default_max_turns",
		"default_timeout_minutes",
		"default_detection_model",
		"default_utc",
		"default_model_copilot",
		"default_model_claude",
	} {
		change, ok := byField[field]
		require.True(t, ok, "missing change for %s", field)
		assert.True(t, change.delete, "expected %s to be deleted", field)
	}

	change, ok := byField["default_model_codex"]
	require.True(t, ok, "missing change for default_model_codex")
	assert.False(t, change.delete)
	assert.Equal(t, "gpt-5.5", change.value)
}

func TestConfirmDefaultsUpdate(t *testing.T) {
	target := defaultsTarget{scope: defaultsScopeOrg, org: "github"}
	changes := []defaultsUpdateChange{{field: "default_max_turns", value: "42"}}

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
