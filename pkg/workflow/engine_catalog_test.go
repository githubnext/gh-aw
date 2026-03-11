//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineCatalogBuiltIns(t *testing.T) {
	catalog := GetGlobalEngineCatalog()
	require.NotNil(t, catalog, "Global engine catalog must not be nil")

	// All four built-in engines must be present
	expectedIDs := []string{
		string(constants.CopilotEngine),
		string(constants.ClaudeEngine),
		string(constants.CodexEngine),
		string(constants.GeminiEngine),
	}

	allIDs := catalog.GetAllEngineIDs()
	assert.Len(t, allIDs, len(expectedIDs), "Catalog must contain exactly the four built-in engines")

	for _, id := range expectedIDs {
		def, err := catalog.GetDefinition(id)
		require.NoError(t, err, "Built-in engine %q must be found in catalog", id)
		assert.Equal(t, id, def.ID, "Engine definition ID must match lookup key")
		assert.NotEmpty(t, def.DisplayName, "Engine %q must have a DisplayName", id)
		assert.NotEmpty(t, def.Description, "Engine %q must have a Description", id)
		assert.NotEmpty(t, def.Secrets.Primary, "Engine %q must have a primary secret", id)
	}
}

func TestEngineCatalogGetDefinition(t *testing.T) {
	catalog := GetGlobalEngineCatalog()

	tests := []struct {
		id              string
		wantDisplayName string
		wantPrimary     string
	}{
		{string(constants.CopilotEngine), "GitHub Copilot CLI", "COPILOT_GITHUB_TOKEN"},
		{string(constants.ClaudeEngine), "Claude Code", "ANTHROPIC_API_KEY"},
		{string(constants.CodexEngine), "Codex", "OPENAI_API_KEY"},
		{string(constants.GeminiEngine), "Google Gemini CLI", "GEMINI_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			def, err := catalog.GetDefinition(tt.id)
			require.NoError(t, err, "GetDefinition must succeed for built-in engine %q", tt.id)
			assert.Equal(t, tt.wantDisplayName, def.DisplayName, "DisplayName for %q", tt.id)
			assert.Equal(t, tt.wantPrimary, def.Secrets.Primary, "Primary secret for %q", tt.id)
		})
	}

	// Unknown engine must return an error
	_, err := catalog.GetDefinition("unknown-engine-xyz")
	assert.Error(t, err, "GetDefinition must return error for unknown engine")
}

func TestEngineCatalogIsKnownEngine(t *testing.T) {
	catalog := GetGlobalEngineCatalog()

	assert.True(t, catalog.IsKnownEngine(string(constants.CopilotEngine)), "copilot must be known")
	assert.True(t, catalog.IsKnownEngine(string(constants.ClaudeEngine)), "claude must be known")
	assert.True(t, catalog.IsKnownEngine(string(constants.CodexEngine)), "codex must be known")
	assert.True(t, catalog.IsKnownEngine(string(constants.GeminiEngine)), "gemini must be known")
	assert.False(t, catalog.IsKnownEngine("unknown-engine"), "unknown engine must not be known")
}

func TestEngineCatalogGetEngineOptions(t *testing.T) {
	catalog := GetGlobalEngineCatalog()
	opts := catalog.GetEngineOptions()

	// Must include all four engines
	assert.Len(t, opts, 4, "GetEngineOptions must return options for all four built-in engines")

	// Build a map for easy lookup
	optMap := make(map[string]constants.EngineOption)
	for _, opt := range opts {
		optMap[opt.Value] = opt
	}

	// Each engine must have the required option fields
	for _, id := range []string{"copilot", "claude", "codex", "gemini"} {
		opt, ok := optMap[id]
		require.True(t, ok, "EngineOption must exist for %q", id)
		assert.NotEmpty(t, opt.Label, "EngineOption.Label must not be empty for %q", id)
		assert.NotEmpty(t, opt.Description, "EngineOption.Description must not be empty for %q", id)
		assert.NotEmpty(t, opt.SecretName, "EngineOption.SecretName must not be empty for %q", id)
	}

	// Codex must have CODEX_API_KEY as an alternative
	codexOpt := optMap["codex"]
	assert.Contains(t, codexOpt.AlternativeSecrets, "CODEX_API_KEY",
		"Codex EngineOption must include CODEX_API_KEY as an alternative secret")
}

func TestEngineCatalogGetAllSecretNames(t *testing.T) {
	catalog := GetGlobalEngineCatalog()
	secrets := catalog.GetAllSecretNames()

	require.NotEmpty(t, secrets, "GetAllSecretNames must not return empty slice")

	secretSet := make(map[string]bool)
	for _, s := range secrets {
		secretSet[s] = true
	}

	// All primary secrets must be present
	for _, primary := range []string{
		"COPILOT_GITHUB_TOKEN",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GEMINI_API_KEY",
	} {
		assert.True(t, secretSet[primary], "GetAllSecretNames must include %q", primary)
	}

	// Codex alternative secret must be present
	assert.True(t, secretSet["CODEX_API_KEY"], "GetAllSecretNames must include CODEX_API_KEY")

	// No duplicates
	seen := make(map[string]bool)
	for _, s := range secrets {
		assert.False(t, seen[s], "GetAllSecretNames must not return duplicate: %q", s)
		seen[s] = true
	}
}

func TestEngineCatalogSyncWithRegistry(t *testing.T) {
	// The engine catalog and engine registry must be in sync:
	// every engine ID in the catalog must have a corresponding runtime in the registry,
	// and vice versa. This test catches drift like the gemini incident where the
	// runtime was registered but the metadata was missing from the catalog.
	catalog := GetGlobalEngineCatalog()
	registry := NewEngineRegistry()

	catalogIDs := catalog.GetAllEngineIDs()
	registryIDs := registry.GetSupportedEngines()

	// Convert to sets for comparison
	catalogSet := make(map[string]bool)
	for _, id := range catalogIDs {
		catalogSet[id] = true
	}
	registrySet := make(map[string]bool)
	for _, id := range registryIDs {
		registrySet[id] = true
	}

	// Every catalog engine must be in the registry
	for _, id := range catalogIDs {
		assert.True(t, registrySet[id],
			"Engine %q is in catalog but missing from registry — add it to NewEngineRegistry()", id)
	}

	// Every registry engine must be in the catalog
	for _, id := range registryIDs {
		assert.True(t, catalogSet[id],
			"Engine %q is in registry but missing from catalog — add it to newBuiltInEngineCatalog()", id)
	}
}

func TestResolvedEngineTarget(t *testing.T) {
	registry := NewEngineRegistry()

	copilot, err := registry.GetEngine("copilot")
	require.NoError(t, err, "GetEngine must succeed for copilot")

	catalog := GetGlobalEngineCatalog()
	def, err := catalog.GetDefinition("copilot")
	require.NoError(t, err, "GetDefinition must succeed for copilot")

	config := &EngineConfig{ID: "copilot", Model: "gpt-5"}
	target := NewResolvedEngineTarget(def, copilot, config)

	assert.Equal(t, "copilot", target.Definition.ID, "Definition.ID must be copilot")
	assert.Equal(t, copilot, target.Runtime, "Runtime must be the copilot engine")
	assert.Equal(t, "gpt-5", target.Config.Model, "Config.Model must be gpt-5")
	assert.Equal(t, "copilot", target.EngineID(), "EngineID() must return copilot")

	// Verify the runtime adapter is functional (not just a pointer match)
	assert.Equal(t, "copilot", target.Runtime.GetID(), "Runtime.GetID() must return copilot")
	assert.Equal(t, "GitHub Copilot CLI", target.Runtime.GetDisplayName(), "Runtime.GetDisplayName() must return expected name")
}

func TestResolvedEngineTargetEngineID(t *testing.T) {
	catalog := GetGlobalEngineCatalog()
	def, err := catalog.GetDefinition("codex")
	require.NoError(t, err, "GetDefinition must succeed for codex")

	registry := NewEngineRegistry()
	codex, err := registry.GetEngine("codex")
	require.NoError(t, err, "GetEngine must succeed for codex")

	t.Run("config ID takes precedence", func(t *testing.T) {
		// When engine was resolved via prefix (e.g., "codex-experimental" → codex),
		// the config.ID holds the original user-specified string
		config := &EngineConfig{ID: "codex"}
		target := NewResolvedEngineTarget(def, codex, config)
		assert.Equal(t, "codex", target.EngineID(), "EngineID() must use Config.ID when set")
	})

	t.Run("falls back to definition ID when config ID is empty", func(t *testing.T) {
		config := &EngineConfig{ID: ""}
		target := NewResolvedEngineTarget(def, codex, config)
		assert.Equal(t, "codex", target.EngineID(), "EngineID() must fall back to Definition.ID")
	})

	t.Run("falls back to definition ID when config is nil", func(t *testing.T) {
		target := NewResolvedEngineTarget(def, codex, nil)
		assert.Equal(t, "codex", target.EngineID(), "EngineID() must fall back to Definition.ID when config is nil")
	})
}
