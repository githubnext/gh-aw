//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEngineCatalog_BuiltIns checks that all four built-in engines are registered
// and resolve to the expected runtime adapters.
func TestNewEngineCatalog_BuiltIns(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	tests := []struct {
		engineID    string
		displayName string
		provider    string
	}{
		{"claude", "Claude Code", "anthropic"},
		{"codex", "Codex", "openai"},
		{"copilot", "GitHub Copilot CLI", "github"},
		{"gemini", "Google Gemini CLI", "google"},
	}

	for _, tt := range tests {
		t.Run(tt.engineID, func(t *testing.T) {
			resolved, err := catalog.Resolve(tt.engineID, &EngineConfig{ID: tt.engineID})
			require.NoError(t, err, "expected %s to resolve without error", tt.engineID)
			require.NotNil(t, resolved, "expected non-nil ResolvedEngineTarget for %s", tt.engineID)

			assert.Equal(t, tt.engineID, resolved.Definition.ID, "Definition.ID should match")
			assert.Equal(t, tt.displayName, resolved.Definition.DisplayName, "Definition.DisplayName should match")
			assert.Equal(t, tt.provider, resolved.Definition.Provider.Name, "Definition.Provider.Name should match")
			assert.Equal(t, tt.engineID, resolved.Runtime.GetID(), "Runtime.GetID() should match engine ID")
		})
	}
}

// TestEngineCatalog_Resolve_LegacyStringFormat verifies that resolving via plain string
// ("engine: claude") and object ID format ("engine.id: claude") produce the same runtime.
func TestEngineCatalog_Resolve_LegacyStringFormat(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	// Simulate "engine: claude" — EngineConfig built from string
	stringConfig := &EngineConfig{ID: "claude"}
	resolvedString, err := catalog.Resolve("claude", stringConfig)
	require.NoError(t, err, "string-format engine should resolve without error")

	// Simulate "engine:\n  id: claude" — same logical ID
	objectConfig := &EngineConfig{ID: "claude"}
	resolvedObject, err := catalog.Resolve("claude", objectConfig)
	require.NoError(t, err, "object-format engine should resolve without error")

	assert.Equal(t, resolvedString.Runtime.GetID(), resolvedObject.Runtime.GetID(),
		"both formats should resolve to the same runtime adapter")
	assert.Equal(t, resolvedString.Definition.ID, resolvedObject.Definition.ID,
		"both formats should resolve to the same definition")
}

// TestEngineCatalog_Resolve_PrefixFallback verifies backward-compat prefix matching
// (e.g. "codex-experimental" should resolve to the codex runtime).
func TestEngineCatalog_Resolve_PrefixFallback(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	resolved, err := catalog.Resolve("codex-experimental", &EngineConfig{ID: "codex-experimental"})
	require.NoError(t, err, "prefix-matched engine should resolve without error")
	require.NotNil(t, resolved, "expected non-nil ResolvedEngineTarget for prefix match")

	assert.Equal(t, "codex", resolved.Runtime.GetID(), "prefix match should resolve to codex runtime")
}

// TestEngineCatalog_Resolve_UnknownEngine verifies that unknown engine IDs return a
// descriptive validation error containing the engine ID, a list of valid engines, and
// the documentation URL.
func TestEngineCatalog_Resolve_UnknownEngine(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	_, err := catalog.Resolve("nonexistent-engine", &EngineConfig{ID: "nonexistent-engine"})
	require.Error(t, err, "unknown engine should return an error")
	assert.Contains(t, err.Error(), "invalid engine",
		"error should mention 'invalid engine', got: %s", err.Error())
	assert.Contains(t, err.Error(), "nonexistent-engine",
		"error should mention the unknown engine ID, got: %s", err.Error())
	assert.Contains(t, err.Error(), string(constants.DocsEnginesURL),
		"error should include the engines documentation URL, got: %s", err.Error())
	assert.Contains(t, err.Error(), "engine: copilot",
		"error should include an example, got: %s", err.Error())
}

// TestEngineCatalog_Resolve_ConfigPassthrough verifies that the EngineConfig passed to
// Resolve is surfaced unchanged in the ResolvedEngineTarget.
func TestEngineCatalog_Resolve_ConfigPassthrough(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	cfg := &EngineConfig{ID: "copilot", Model: "gpt-4o", MaxTurns: "10"}
	resolved, err := catalog.Resolve("copilot", cfg)
	require.NoError(t, err, "copilot with config should resolve without error")
	assert.Equal(t, cfg, resolved.Config, "resolved Config should be the same pointer passed in")
}

// TestEngineCatalog_Register_Custom verifies that a custom engine definition can be
// registered and resolved via the catalog.
func TestEngineCatalog_Register_Custom(t *testing.T) {
	registry := NewEngineRegistry()
	// Register a test engine in the registry so the catalog can look it up
	registry.Register(NewCopilotEngine()) // reuse copilot as the backing runtime

	catalog := NewEngineCatalog(registry)
	catalog.Register(&EngineDefinition{
		ID:          "my-custom-engine",
		DisplayName: "My Custom Engine",
		Description: "A custom engine for testing",
		RuntimeID:   "copilot", // backed by copilot runtime
		Provider:    ProviderSelection{Name: "custom"},
	})

	resolved, err := catalog.Resolve("my-custom-engine", &EngineConfig{ID: "my-custom-engine"})
	require.NoError(t, err, "custom engine should resolve without error")
	assert.Equal(t, "my-custom-engine", resolved.Definition.ID, "custom engine definition ID should match")
	assert.Equal(t, "copilot", resolved.Runtime.GetID(), "custom engine should use copilot runtime")
}
