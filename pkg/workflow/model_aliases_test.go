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
		"sonnet", "sonnet-6x", "haiku", "opus",
		"gpt-4.1", "gpt-5", "gpt-5.4", "gpt-5-mini", "gpt-5-nano", "gpt-5-codex", "gpt-5-pro", "reasoning",
		"gemini-flash", "gemini-flash-lite", "gemini-pro", "gemini-3-pro", "gemini-3-flash", "gemini-3.1-pro", "gemini-3.1-flash", "antigravity", "computer-use", "robotics", "deep-research",
		"mini", "large", "any", "agent", "copilot", "claude", "codex", "gemini", "summarization",
	}
	for _, family := range expectedFamilies {
		patterns, ok := aliases[family]
		assert.True(t, ok, "expected builtin alias for family %q", family)
		assert.NotEmpty(t, patterns, "builtin alias %q should have at least one pattern", family)
	}

	// Vendor aliases should include at least one copilot/* pattern.
	// Meta-aliases (mini, large, auto) reference other alias names and are excluded here.
	vendorFamilies := []string{"sonnet", "sonnet-6x", "haiku", "opus", "gpt-4.1", "gpt-5", "gpt-5.4", "gpt-5-mini", "gpt-5-nano", "gpt-5-codex", "gpt-5-pro", "reasoning", "gemini-flash", "gemini-flash-lite", "gemini-pro", "gemini-3-pro", "gemini-3-flash", "gemini-3.1-pro", "gemini-3.1-flash", "antigravity", "computer-use", "robotics", "deep-research"}
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

	assert.Contains(t, aliases["gemini-flash"], "gemini/gemini-*flash*", "gemini-flash should support direct gemini/ provider models")
	assert.Contains(t, aliases["gemini-flash-lite"], "gemini/gemini-*flash*lite*", "gemini-flash-lite should support direct gemini/ provider models")
	assert.Contains(t, aliases["gemini-pro"], "gemini/gemini-*pro*", "gemini-pro should support direct gemini/ provider models")
	assert.Equal(t, []string{"copilot/gpt-5.4*", "openai/gpt-5.4*"}, aliases["gpt-5.4"], "gpt-5.4 should map to copilot/openai gpt-5.4 family")
	assert.Contains(t, aliases["gemini-3-pro"], "gemini/gemini-3*pro*", "gemini-3-pro should support direct gemini/ provider models")
	assert.Contains(t, aliases["gemini-3-flash"], "gemini/gemini-3*flash*", "gemini-3-flash should support direct gemini/ provider models")
	assert.Contains(t, aliases["gemini-3.1-pro"], "gemini/gemini-3.1*pro*", "gemini-3.1-pro should support direct gemini/ provider models")
	assert.Contains(t, aliases["gemini-3.1-flash"], "gemini/gemini-3.1*flash*", "gemini-3.1-flash should support direct gemini/ provider models")
	assert.Contains(t, aliases["antigravity"], "copilot/antigravity*", "antigravity should include copilot/ provider pattern")
	assert.Equal(t, []string{"copilot/*sonnet-4-5-*", "anthropic/*sonnet-4-5-*", "copilot/*sonnet-4-6*", "anthropic/*sonnet-4-6*"}, aliases["sonnet-6x"], "sonnet-6x should target Sonnet 4.5/4.6 dated model families")
	assert.Contains(t, aliases["deep-research"], "gemini/deep-research*", "deep-research should support direct gemini/ provider models")

	// Meta-aliases reference other alias names (resolved recursively by AWF).
	assert.Equal(t, []string{"haiku", "gpt-5-mini", "gpt-5-nano", "gemini-flash-lite"}, aliases["mini"], "mini should reference haiku, gpt-5-mini, gpt-5-nano, and gemini-flash-lite")
	assert.Equal(t, []string{"haiku", "gpt-5-mini", "gemini-flash-lite", "mini"}, aliases["summarization"], "summarization should reference fast/lightweight models")
	assert.Equal(t, []string{"sonnet", "gpt-5-pro", "gpt-5", "gemini-pro"}, aliases["large"], "large should reference sonnet, gpt-5-pro, gpt-5, and gemini-pro")
	assert.Equal(t, []string{"large"}, aliases["agent"], "agent should fall back to large")
	assert.Equal(t, []string{"copilot/*", "anthropic/*", "openai/*", "google/*", "gemini/*"}, aliases["any"], "any should provide a provider-wide catch-all fallback chain")
	assert.Equal(t, []string{"sonnet-6x", "gpt-5.4", "gpt-5", "gemini-pro", "haiku", "any"}, aliases["agent"], "agent should default to <=6x-capable model families before any-model fallback")
	assert.Equal(t, []string{"agent", "gpt-5.4", "sonnet", "gpt-5", "any"}, aliases["copilot"], "copilot should define per-engine default fallback chain")
	assert.Equal(t, []string{"agent", "sonnet-6x", "haiku", "any"}, aliases["claude"], "claude should define per-engine default fallback chain")
	assert.Equal(t, []string{"agent", "gpt-5-codex", "gpt-5", "any"}, aliases["codex"], "codex should define per-engine default fallback chain")
	assert.Equal(t, []string{"agent", "gemini-pro", "gemini-flash", "any"}, aliases["gemini"], "gemini should define per-engine default fallback chain")
	assert.NotContains(t, aliases["agent"], "opus", "agent default chain must not include opus")

	// Returns a fresh copy — mutating one call's map must not affect another call.
	aliases["sonnet"] = []string{"custom/model"}
	aliases2 := BuiltinModelAliases()
	assert.NotEqual(t, aliases["sonnet"], aliases2["sonnet"], "BuiltinModelAliases should return a fresh copy each time")
}

// awfConfigModelsResult is a helper type for parsing the apiProxy.models section
// from generated AWF config JSON in tests.
type awfConfigModelsResult struct {
	APIProxy struct {
		Models map[string][]string `json:"models"`
	} `json:"apiProxy"`
}

// TestBuildAWFConfigJSON_ModelsSection verifies model alias behaviour in BuildAWFConfigJSON.
//
// Models are serialised under apiProxy.models per the AWF config schema (apiProxy.models
// is supported in AWF v0.25.38+). The builtin aliases are included when ModelMappings is set.
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
				ModelMappings: MergeImportedModelAliases(nil, nil),
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		var parsed awfConfigModelsResult
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed), "result must be valid JSON")

		// models must appear nested under apiProxy
		assert.NotEmpty(t, parsed.APIProxy.Models, "models section must be present and non-empty under apiProxy in AWF config JSON")
		assert.Contains(t, jsonStr, `"models"`, "models key must appear in AWF config JSON")

		// the alias map is populated in WorkflowData
		assert.NotEmpty(t, config.WorkflowData.ModelMappings, "ModelMappings should be populated on WorkflowData")
		assert.Contains(t, config.WorkflowData.ModelMappings, "sonnet", "ModelMappings should include sonnet alias")
		assert.Contains(t, config.WorkflowData.ModelMappings, "haiku", "ModelMappings should include haiku alias")
	})

	t.Run("frontmatter override is reflected in WorkflowData and in AWF config JSON", func(t *testing.T) {
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
				ModelMappings: MergeImportedModelAliases(nil, custom),
			},
		}

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should not return an error")

		var parsed awfConfigModelsResult
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed), "result must be valid JSON")

		// models must appear nested under apiProxy
		assert.Contains(t, jsonStr, `"models"`, "models section must be present in AWF config JSON")

		// frontmatter overrides are visible in both WorkflowData and the JSON
		assert.Equal(t, []string{"myvendor/sonnet-v3"}, config.WorkflowData.ModelMappings["sonnet"],
			"frontmatter override for sonnet should be stored in ModelMappings")
		assert.Equal(t, []string{"myvendor/sonnet-v3"}, parsed.APIProxy.Models["sonnet"],
			"frontmatter override for sonnet should be emitted under apiProxy.models in AWF config JSON")
		assert.Equal(t, []string{"sonnet"}, config.WorkflowData.ModelMappings[""],
			"default policy should be stored in ModelMappings")
		assert.Equal(t, []string{"sonnet"}, parsed.APIProxy.Models[""],
			"default policy should be emitted under apiProxy.models in AWF config JSON")
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

// TestMergeImportedModelAliases verifies the three-layer merge: builtins → imports → main.
func TestMergeImportedModelAliases(t *testing.T) {
	t.Run("no imports and no frontmatter returns builtins", func(t *testing.T) {
		merged := MergeImportedModelAliases(nil, nil)
		builtins := BuiltinModelAliases()
		assert.Len(t, merged, len(builtins), "should return exactly the builtins")
		for k, v := range builtins {
			assert.Equal(t, v, merged[k], "builtin alias %q should be present unchanged", k)
		}
	})

	t.Run("imported alias is added when not in builtins", func(t *testing.T) {
		imported := []map[string][]string{
			{"my-imported": {"vendor/imported-model"}},
		}
		merged := MergeImportedModelAliases(imported, nil)
		assert.Equal(t, []string{"vendor/imported-model"}, merged["my-imported"],
			"imported alias should be present in merged map")
		assert.NotEmpty(t, merged["sonnet"], "builtin sonnet should still be present")
	})

	t.Run("import cannot override a builtin alias", func(t *testing.T) {
		imported := []map[string][]string{
			{"sonnet": {"imported/sonnet-override"}},
		}
		merged := MergeImportedModelAliases(imported, nil)
		builtins := BuiltinModelAliases()
		assert.Equal(t, builtins["sonnet"], merged["sonnet"],
			"import should NOT override a builtin alias; builtin takes precedence over import")
	})

	t.Run("first import wins among multiple imports for the same key", func(t *testing.T) {
		imported := []map[string][]string{
			{"shared-alias": {"first-import/model"}},
			{"shared-alias": {"second-import/model"}},
		}
		merged := MergeImportedModelAliases(imported, nil)
		assert.Equal(t, []string{"first-import/model"}, merged["shared-alias"],
			"first import should win among competing imports for the same alias key")
	})

	t.Run("main workflow frontmatter overrides imported alias", func(t *testing.T) {
		imported := []map[string][]string{
			{"my-alias": {"import/model"}},
		}
		frontmatter := map[string][]string{
			"my-alias": {"main/model"},
		}
		merged := MergeImportedModelAliases(imported, frontmatter)
		assert.Equal(t, []string{"main/model"}, merged["my-alias"],
			"main workflow frontmatter should win over imported alias")
	})

	t.Run("main workflow frontmatter overrides builtin alias", func(t *testing.T) {
		frontmatter := map[string][]string{
			"sonnet": {"mygateway/sonnet-v3"},
		}
		merged := MergeImportedModelAliases(nil, frontmatter)
		assert.Equal(t, []string{"mygateway/sonnet-v3"}, merged["sonnet"],
			"main workflow frontmatter should override builtin sonnet alias")
		assert.NotEmpty(t, merged["haiku"], "other builtins should still be present")
	})

	t.Run("all three layers are combined correctly", func(t *testing.T) {
		imported := []map[string][]string{
			{
				"import-only": {"import/model"},
				"both":        {"import/both"},
				"sonnet":      {"import/sonnet"}, // shadowed by builtin
			},
		}
		frontmatter := map[string][]string{
			"main-only": {"main/model"},
			"both":      {"main/both"},
		}
		merged := MergeImportedModelAliases(imported, frontmatter)

		// import-only key comes from import (no conflict)
		assert.Equal(t, []string{"import/model"}, merged["import-only"],
			"import-only alias should come from the import layer")

		// main-only key comes from main workflow
		assert.Equal(t, []string{"main/model"}, merged["main-only"],
			"main-only alias should come from the main workflow layer")

		// 'both' key: main workflow wins over import
		assert.Equal(t, []string{"main/both"}, merged["both"],
			"main workflow should win over import for the 'both' key")

		// 'sonnet' key: builtin wins over import
		builtins := BuiltinModelAliases()
		assert.Equal(t, builtins["sonnet"], merged["sonnet"],
			"builtin should win over import for the 'sonnet' key")
	})
}

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
