//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuiltinModelAliases verifies that the builtin model alias map covers the main
// model families and returns a fresh map on each call.
func TestBuiltinModelAliases(t *testing.T) {
	aliases := BuiltinModelAliases()

	expectedFamilies := []string{
		"sonnet", "haiku", "opus",
		"gpt-5", "gpt-5-mini", "gpt-5-codex",
		"gemini-flash", "gemini-pro",
		"mini", "large", "auto",
	}
	for _, family := range expectedFamilies {
		patterns, ok := aliases[family]
		assert.True(t, ok, "expected builtin alias for family %q", family)
		assert.NotEmpty(t, patterns, "builtin alias %q should have at least one pattern", family)
	}

	// Vendor aliases should include at least one copilot/* pattern.
	// Meta-aliases (mini, large, auto) reference other alias names and are excluded here.
	vendorFamilies := []string{"sonnet", "haiku", "opus", "gpt-5", "gpt-5-mini", "gpt-5-codex", "gemini-flash", "gemini-pro"}
	for _, family := range vendorFamilies {
		patterns := aliases[family]
		hasCopilot := false
		for _, p := range patterns {
			if len(p) > 7 && p[:7] == "copilot" {
				hasCopilot = true
				break
			}
		}
		assert.True(t, hasCopilot, "builtin alias %q should include a copilot/* pattern", family)
	}

	// Meta-aliases reference other alias names (resolved recursively by AWF).
	assert.Equal(t, []string{"haiku", "gpt-5-mini", "gemini-flash"}, aliases["mini"], "mini should reference haiku, gpt-5-mini, and gemini-flash")
	assert.Equal(t, []string{"sonnet", "gpt-5", "gemini-pro"}, aliases["large"], "large should reference sonnet, gpt-5, and gemini-pro")
	assert.Equal(t, []string{"large"}, aliases["auto"], "auto should fall back to large")

	// Returns a fresh copy — mutating one call's map must not affect another call.
	aliases["sonnet"] = []string{"custom/model"}
	aliases2 := BuiltinModelAliases()
	assert.NotEqual(t, aliases["sonnet"], aliases2["sonnet"], "BuiltinModelAliases should return a fresh copy each time")
}

// TestMergeModelAliases verifies that frontmatter-defined aliases are merged on top
// of the builtins.
func TestMergeModelAliases(t *testing.T) {
	t.Run("nil frontmatter returns all builtins", func(t *testing.T) {
		merged := MergeModelAliases(nil)
		builtins := BuiltinModelAliases()
		assert.Len(t, merged, len(builtins), "nil frontmatter should return exactly the builtins")
		for k, v := range builtins {
			assert.Equal(t, v, merged[k], "builtin alias %q should be present unchanged", k)
		}
	})

	t.Run("empty frontmatter returns all builtins", func(t *testing.T) {
		merged := MergeModelAliases(map[string][]string{})
		builtins := BuiltinModelAliases()
		assert.Len(t, merged, len(builtins), "empty frontmatter should return exactly the builtins")
	})

	t.Run("frontmatter override replaces builtin entry", func(t *testing.T) {
		custom := map[string][]string{
			"sonnet": {"myvendor/sonnet-custom"},
		}
		merged := MergeModelAliases(custom)
		assert.Equal(t, []string{"myvendor/sonnet-custom"}, merged["sonnet"],
			"frontmatter override should replace the builtin sonnet alias")
		// Other builtins should be unaffected.
		assert.NotEmpty(t, merged["haiku"], "haiku builtin should still be present")
	})

	t.Run("frontmatter adds new alias", func(t *testing.T) {
		custom := map[string][]string{
			"my-alias": {"copilot/my-model"},
		}
		merged := MergeModelAliases(custom)
		assert.Equal(t, []string{"copilot/my-model"}, merged["my-alias"],
			"new frontmatter alias should be present in merged map")
		// Builtins should still be present.
		assert.NotEmpty(t, merged["sonnet"], "sonnet builtin should still be present")
	})

	t.Run("default policy key is supported", func(t *testing.T) {
		custom := map[string][]string{
			"": {"sonnet", "gpt-5-codex"},
		}
		merged := MergeModelAliases(custom)
		assert.Equal(t, []string{"sonnet", "gpt-5-codex"}, merged[""],
			"default policy (empty key) should be stored and returned")
	})
}

// TestBuildAWFConfigJSON_ModelsSection verifies that the models map is included in
// the generated AWF config JSON when WorkflowData.ModelMappings is set.
func TestBuildAWFConfigJSON_ModelsSection(t *testing.T) {
	t.Run("builtin model aliases are included when WorkflowData has ModelMappings", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				ModelMappings: MergeModelAliases(nil),
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed), "result must be valid JSON")

		models, ok := parsed["models"]
		assert.True(t, ok, "models section should be present in AWF config JSON")
		modelsMap, ok := models.(map[string]any)
		require.True(t, ok, "models should be a JSON object")
		assert.Contains(t, modelsMap, "sonnet", "models should include sonnet alias")
		assert.Contains(t, modelsMap, "haiku", "models should include haiku alias")
		assert.Contains(t, modelsMap, "opus", "models should include opus alias")
		assert.Contains(t, modelsMap, "gpt-5", "models should include gpt-5 alias")
		assert.Contains(t, modelsMap, "gpt-5-mini", "models should include gpt-5-mini alias")
		assert.Contains(t, modelsMap, "gpt-5-codex", "models should include gpt-5-codex alias")
		assert.Contains(t, modelsMap, "gemini-flash", "models should include gemini-flash alias")
		assert.Contains(t, modelsMap, "gemini-pro", "models should include gemini-pro alias")
		assert.Contains(t, modelsMap, "mini", "models should include mini alias")
		assert.Contains(t, modelsMap, "large", "models should include large alias")
		assert.Contains(t, modelsMap, "auto", "models should include auto alias")
	})

	t.Run("frontmatter override is reflected in AWF config JSON", func(t *testing.T) {
		custom := map[string][]string{
			"sonnet": {"myvendor/sonnet-v3"},
			"":       {"sonnet"},
		}
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				ModelMappings: MergeModelAliases(custom),
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		var parsed struct {
			Models map[string][]string `json:"models"`
		}
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

		assert.Equal(t, []string{"myvendor/sonnet-v3"}, parsed.Models["sonnet"],
			"frontmatter override for sonnet should appear in AWF config")
		assert.Equal(t, []string{"sonnet"}, parsed.Models[""],
			"default policy should appear in AWF config JSON")
	})

	t.Run("no models section when ModelMappings is nil", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				ModelMappings: nil,
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		assert.NotContains(t, jsonStr, `"models"`, "models section should be absent when ModelMappings is nil")
	})
}

// TestFrontmatterModelsField verifies that the models field in frontmatter is parsed
// correctly by ParseFrontmatterConfig.
func TestFrontmatterModelsField(t *testing.T) {
	t.Run("models field is parsed from frontmatter", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "test-workflow",
			"models": map[string]any{
				"my-model": []any{"copilot/my-model-v1", "openai/my-model-v1"},
				"":         []any{"my-model"},
			},
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		require.NoError(t, err, "ParseFrontmatterConfig should succeed with models field")
		require.NotNil(t, config, "parsed config should not be nil")

		assert.Equal(t, []string{"copilot/my-model-v1", "openai/my-model-v1"}, config.Models["my-model"],
			"models[my-model] should be parsed correctly")
		assert.Equal(t, []string{"my-model"}, config.Models[""],
			"models default policy (empty key) should be parsed correctly")
	})

	t.Run("models field is optional", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "test-workflow",
		}

		config, err := ParseFrontmatterConfig(frontmatter)
		require.NoError(t, err, "ParseFrontmatterConfig should succeed without models field")
		require.NotNil(t, config, "parsed config should not be nil")
		assert.Nil(t, config.Models, "models should be nil when not specified in frontmatter")
	})
}
