//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractEngineConfig_InlineDefinition verifies that an inline engine definition
// (engine.runtime + optional engine.provider) is correctly parsed from frontmatter
// into an EngineConfig with IsInlineDefinition=true.
func TestExtractEngineConfig_InlineDefinition(t *testing.T) {
	tests := []struct {
		name                  string
		frontmatter           map[string]any
		expectedID            string
		expectedVersion       string
		expectedModel         string
		expectedProviderID    string
		expectedSecret        string
		expectInlineFlag      bool
		expectedEngineSetting string
	}{
		{
			name: "runtime only",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"runtime": map[string]any{
						"id": "codex",
					},
				},
			},
			expectedID:            "codex",
			expectedEngineSetting: "codex",
			expectInlineFlag:      true,
		},
		{
			name: "runtime with version",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"runtime": map[string]any{
						"id":      "codex",
						"version": "0.105.0",
					},
				},
			},
			expectedID:            "codex",
			expectedVersion:       "0.105.0",
			expectedEngineSetting: "codex",
			expectInlineFlag:      true,
		},
		{
			name: "runtime and provider full",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"runtime": map[string]any{
						"id": "codex",
					},
					"provider": map[string]any{
						"id":    "openai",
						"model": "gpt-5",
						"auth": map[string]any{
							"secret": "OPENAI_API_KEY",
						},
					},
				},
			},
			expectedID:            "codex",
			expectedModel:         "gpt-5",
			expectedProviderID:    "openai",
			expectedSecret:        "OPENAI_API_KEY",
			expectedEngineSetting: "codex",
			expectInlineFlag:      true,
		},
		{
			name: "runtime with provider model only",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"runtime": map[string]any{
						"id": "claude",
					},
					"provider": map[string]any{
						"model": "claude-3-7-sonnet-20250219",
					},
				},
			},
			expectedID:            "claude",
			expectedModel:         "claude-3-7-sonnet-20250219",
			expectedEngineSetting: "claude",
			expectInlineFlag:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()
			engineSetting, config := c.ExtractEngineConfig(tt.frontmatter)

			require.NotNil(t, config, "should return non-nil EngineConfig for inline definition")
			assert.Equal(t, tt.expectedEngineSetting, engineSetting, "engineSetting should equal runtime.id")
			assert.Equal(t, tt.expectedID, config.ID, "config.ID should equal runtime.id")
			assert.Equal(t, tt.expectInlineFlag, config.IsInlineDefinition, "IsInlineDefinition flag should be set")
			assert.Equal(t, tt.expectedVersion, config.Version, "Version should match runtime.version")
			assert.Equal(t, tt.expectedModel, config.Model, "Model should match provider.model")
			assert.Equal(t, tt.expectedProviderID, config.InlineProviderID, "InlineProviderID should match provider.id")
			assert.Equal(t, tt.expectedSecret, config.InlineProviderSecret, "InlineProviderSecret should match provider.auth.secret")
		})
	}
}

// TestExtractEngineConfig_InlineDefinition_NotTriggeredByIDField verifies that the
// object variant with an 'id' field (not 'runtime') does NOT set IsInlineDefinition.
func TestExtractEngineConfig_InlineDefinition_NotTriggeredByIDField(t *testing.T) {
	c := NewCompiler()
	frontmatter := map[string]any{
		"engine": map[string]any{
			"id": "copilot",
		},
	}
	_, config := c.ExtractEngineConfig(frontmatter)

	require.NotNil(t, config, "should return non-nil EngineConfig")
	assert.Equal(t, "copilot", config.ID, "ID should be set from 'id' field")
	assert.False(t, config.IsInlineDefinition, "IsInlineDefinition should be false for object-with-id format")
}

// TestExtractEngineConfig_LegacyStringFormat_Regression verifies that the legacy
// string format "engine: copilot" still parses correctly and does not set
// IsInlineDefinition.
func TestExtractEngineConfig_LegacyStringFormat_Regression(t *testing.T) {
	c := NewCompiler()

	for _, engineID := range []string{"copilot", "claude", "codex", "gemini"} {
		t.Run(engineID, func(t *testing.T) {
			frontmatter := map[string]any{"engine": engineID}
			engineSetting, config := c.ExtractEngineConfig(frontmatter)

			require.NotNil(t, config, "should return non-nil EngineConfig for string format")
			assert.Equal(t, engineID, engineSetting, "engineSetting should equal the engine string")
			assert.Equal(t, engineID, config.ID, "config.ID should equal the engine string")
			assert.False(t, config.IsInlineDefinition, "IsInlineDefinition should be false for string format")
		})
	}
}

// TestValidateEngineInlineDefinition_MissingRuntimeID verifies that an inline
// definition with an empty runtime.id produces a clear validation error.
func TestValidateEngineInlineDefinition_MissingRuntimeID(t *testing.T) {
	c := NewCompiler()

	config := &EngineConfig{
		IsInlineDefinition: true,
		ID:                 "", // deliberately empty
	}

	err := c.validateEngineInlineDefinition(config)
	require.Error(t, err, "missing runtime.id should return an error")
	assert.Contains(t, err.Error(), "runtime.id", "error should mention the missing field")
	assert.Contains(t, err.Error(), string(constants.DocsEnginesURL), "error should include docs URL")
}

// TestValidateEngineInlineDefinition_ValidRuntimeID verifies that a valid inline
// definition (with runtime.id present) passes validation.
func TestValidateEngineInlineDefinition_ValidRuntimeID(t *testing.T) {
	c := NewCompiler()

	config := &EngineConfig{
		IsInlineDefinition: true,
		ID:                 "codex",
	}

	err := c.validateEngineInlineDefinition(config)
	assert.NoError(t, err, "valid runtime.id should pass validation")
}

// TestValidateEngineInlineDefinition_NonInlineIsNoop verifies that the validator is
// a no-op for configs that are not inline definitions.
func TestValidateEngineInlineDefinition_NonInlineIsNoop(t *testing.T) {
	c := NewCompiler()

	config := &EngineConfig{
		IsInlineDefinition: false,
		ID:                 "",
	}

	err := c.validateEngineInlineDefinition(config)
	assert.NoError(t, err, "non-inline config should not produce an error")
}

// TestRegisterInlineEngineDefinition_PreservesBuiltInDisplayName verifies that
// registering an inline definition for a built-in engine ID preserves the existing
// display name and description from the catalog.
func TestRegisterInlineEngineDefinition_PreservesBuiltInDisplayName(t *testing.T) {
	c := NewCompiler()

	// Retrieve the built-in display name before registration.
	builtIn := c.engineCatalog.Get("codex")
	require.NotNil(t, builtIn, "codex built-in should exist")
	originalDisplayName := builtIn.DisplayName

	config := &EngineConfig{
		IsInlineDefinition:   true,
		ID:                   "codex",
		InlineProviderID:     "openai",
		InlineProviderSecret: "MY_KEY",
	}
	c.registerInlineEngineDefinition(config)

	updated := c.engineCatalog.Get("codex")
	require.NotNil(t, updated, "codex should still be in catalog after registration")
	assert.Equal(t, originalDisplayName, updated.DisplayName,
		"display name should be preserved from built-in definition")
	assert.Equal(t, "openai", updated.Provider.Name,
		"provider should be overridden by inline definition")
	require.Len(t, updated.Auth, 1,
		"auth binding should be set from inline definition")
	assert.Equal(t, "MY_KEY", updated.Auth[0].Secret,
		"auth secret should be set from inline provider auth")
}

// TestInlineEngineDefinition_ResolvesViaCatalog verifies the full flow: inline
// definition parsed → registered in catalog → resolved to a runtime adapter.
func TestInlineEngineDefinition_ResolvesViaCatalog(t *testing.T) {
	c := NewCompiler()

	frontmatter := map[string]any{
		"engine": map[string]any{
			"runtime": map[string]any{
				"id":      "copilot",
				"version": "beta",
			},
			"provider": map[string]any{
				"id": "github",
			},
		},
	}

	_, config := c.ExtractEngineConfig(frontmatter)
	require.NotNil(t, config, "should extract EngineConfig from inline definition")
	require.True(t, config.IsInlineDefinition, "should be flagged as inline definition")

	// Validate and register as setupEngineAndImports would do.
	require.NoError(t, c.validateEngineInlineDefinition(config), "valid inline definition should pass")
	c.registerInlineEngineDefinition(config)

	// Resolve via catalog.
	resolved, err := c.engineCatalog.Resolve(config.ID, config)
	require.NoError(t, err, "inline definition should resolve through catalog without error")
	require.NotNil(t, resolved, "resolved target should not be nil")
	assert.Equal(t, "copilot", resolved.Runtime.GetID(),
		"inline definition should resolve to the copilot runtime adapter")
	assert.Equal(t, "github", resolved.Definition.Provider.Name,
		"provider override from inline definition should be applied to the resolved definition")
}

// TestInlineEngineDefinition_UnknownRuntimeID verifies that an inline definition with
// an unknown runtime.id produces a helpful error listing known runtimes.
func TestInlineEngineDefinition_UnknownRuntimeID(t *testing.T) {
	c := NewCompiler()

	config := &EngineConfig{
		IsInlineDefinition: true,
		ID:                 "nonexistent-runtime",
	}

	// validateEngineInlineDefinition should catch the unknown runtime ID with a clear error.
	err := c.validateEngineInlineDefinition(config)
	require.Error(t, err, "unknown runtime.id should produce a validation error")
	assert.Contains(t, err.Error(), "nonexistent-runtime",
		"error should mention the unknown runtime ID")
	assert.Contains(t, err.Error(), "runtime.id",
		"error should mention the 'runtime.id' field")
	assert.Contains(t, err.Error(), string(constants.DocsEnginesURL),
		"error should include the docs URL")
}

// TestExtractEngineConfig_InlineDefinition_Bare verifies that engine.bare is
// parsed correctly in the inline engine definition path.
func TestExtractEngineConfig_InlineDefinition_Bare(t *testing.T) {
	compiler := NewCompiler()

	t.Run("bare true with inline definition", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"runtime": map[string]any{
					"id": "claude",
				},
				"bare": true,
			},
		}

		_, config := compiler.ExtractEngineConfig(frontmatter)
		require.NotNil(t, config, "inline definition with bare:true should produce a config")
		assert.True(t, config.IsInlineDefinition, "Expected inline definition")
		assert.True(t, config.Bare, "Expected Bare=true for inline definition with bare:true")
	})

	t.Run("bare false with inline definition", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"runtime": map[string]any{
					"id": "codex",
				},
				"bare": false,
			},
		}

		_, config := compiler.ExtractEngineConfig(frontmatter)
		require.NotNil(t, config, "inline definition with bare:false should produce a config")
		assert.True(t, config.IsInlineDefinition, "Expected inline definition")
		assert.False(t, config.Bare, "Expected Bare=false")
	})

	t.Run("bare not set defaults to false with inline definition", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"runtime": map[string]any{
					"id": "claude",
				},
			},
		}

		_, config := compiler.ExtractEngineConfig(frontmatter)
		require.NotNil(t, config, "inline definition without bare field should produce a config")
		assert.True(t, config.IsInlineDefinition, "Expected inline definition")
		assert.False(t, config.Bare, "Expected Bare=false by default for inline definition")
	})
}
