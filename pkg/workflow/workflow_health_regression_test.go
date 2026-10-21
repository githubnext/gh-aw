//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestGeminiShellLiteralEngineEnvironment(t *testing.T) {
	engine := NewGeminiEngine()
	data := &WorkflowData{
		Name: "conformance",
		EngineConfig: &EngineConfig{Env: map[string]string{
			"ENGINE_CONFORMANCE_SENTINEL": "conformance-gemini",
			"ANOTHER_SETTING":             "literal",
			"GEMINI_API_KEY":              "${{ secrets.GEMINI_API_KEY }}",
			"ACCESS_TOKEN":                "not-for-shell",
			"INDIRECT":                    "${{ env.SOMETHING }}",
			"APP_VALUE":                   "${{ needs.activation.outputs.token }}",
			"EXCLUDED":                    "literal",
		}},
		ExcludedEnv: []string{"EXCLUDED"},
	}
	step := engine.generateGeminiSettingsStep(data)
	var parsed []struct {
		Env map[string]string `yaml:"env"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(strings.Join(step, "\n")), &parsed))
	require.Len(t, parsed, 1)
	var config struct {
		Security struct {
			Redaction struct {
				Enabled bool     `json:"enabled"`
				Allowed []string `json:"allowed"`
			} `json:"environmentVariableRedaction"`
		} `json:"security"`
	}
	require.NoError(t, json.Unmarshal([]byte(parsed[0].Env["GH_AW_GEMINI_BASE_CONFIG"]), &config))
	require.True(t, config.Security.Redaction.Enabled)
	require.Equal(t, []string{"ANOTHER_SETTING", "ENGINE_CONFORMANCE_SENTINEL"}, config.Security.Redaction.Allowed)
	require.NotContains(t, parsed[0].Env["GH_AW_GEMINI_BASE_CONFIG"], "not-for-shell")
	require.NotContains(t, parsed[0].Env["GH_AW_GEMINI_BASE_CONFIG"], "conformance-gemini")
}

func TestConcurrencyAuditLedgerPromptExample(t *testing.T) {
	content, err := os.ReadFile("../../.github/workflows/daily-mcp-concurrency-analysis.md")
	require.NoError(t, err)
	frontmatter, err := extractMarkdownFrontmatterYAML(content)
	require.NoError(t, err)
	var workflow struct {
		Tools struct {
			Ledger map[string]struct {
				Schema map[string]any `yaml:"schema"`
			} `yaml:"ledger"`
		} `yaml:"tools"`
	}
	require.NoError(t, yaml.Unmarshal(frontmatter, &workflow))
	_, example, ok := strings.Cut(string(content), `{"ledger":"concurrency-audits","record":`)
	require.True(t, ok)
	example, _, ok = strings.Cut(example, "\n```")
	require.True(t, ok)
	var request struct {
		Record map[string]any `json:"record"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"record":`+example), &request))
	schemaJSON, err := json.Marshal(workflow.Tools.Ledger["concurrency-audits"].Schema)
	require.NoError(t, err)
	schema, err := compileSchema(string(schemaJSON), "https://github.com/github/gh-aw/concurrency-audit.schema.json")
	require.NoError(t, err)
	require.NoError(t, schema.Validate(request.Record))
	request.Record["tool"] = request.Record["analyzed_tool"]
	delete(request.Record, "analyzed_tool")
	require.Error(t, schema.Validate(request.Record))
}
