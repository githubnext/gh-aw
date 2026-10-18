package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func graderEnvironmentPayload(yaml, prefix string) []byte {
	var payload strings.Builder
	pattern := regexp.MustCompile(prefix + `_\d+: '((?:[^']|'')*)'`)
	for _, match := range pattern.FindAllStringSubmatch(yaml, -1) {
		payload.WriteString(strings.ReplaceAll(match[1], "''", "'"))
	}
	return []byte(payload.String())
}

func TestParseGradersDefaults(t *testing.T) {
	for _, frontmatter := range []map[string]any{{}, {"graders": nil}, {"graders": map[string]any{}}} {
		cfg, err := ParseGradersFromFrontmatter(frontmatter)
		require.NoError(t, err)
		require.True(t, cfg.HasGraders())
		assert.Equal(t, DefaultGraderIDs, cfg.EnabledGraderIDs())
		assert.NotContains(t, cfg.EnabledGraderIDs(), "recurrence-determinism")
		assert.NotContains(t, cfg.EnabledGraderIDs(), "trajectory-ndtw")
	}
}

func TestParseGradersOverrides(t *testing.T) {
	cfg, err := ParseGradersFromFrontmatter(map[string]any{
		"graders": map[string]any{
			"loops":                  map[string]any{"enabled": false},
			"tool-success-rate":      map[string]any{"threshold": 0.95},
			"recurrence-determinism": nil,
		},
	})
	require.NoError(t, err)
	assert.NotContains(t, cfg.EnabledGraderIDs(), "loops")
	assert.Contains(t, cfg.EnabledGraderIDs(), "recurrence-determinism")
	assert.InDelta(t, 0.95, *cfg.Graders["tool-success-rate"].Threshold, 1e-9)
	assert.Len(t, cfg.EnabledGraderIDs(), len(DefaultGraderIDs))
}

func TestParseGradersRejectsExecutableConfiguration(t *testing.T) {
	for _, graders := range []map[string]any{
		{"custom": nil},
		{"custom": map[string]any{"script": "return 1"}},
		{"custom": map[string]any{"enabled": false}},
		{"operational-value": map[string]any{"run": "evaluator.sh"}},
		{"operational-value": map[string]any{"script": "#!/bin/bash\necho 1"}},
		{"loops": map[string]any{"script": "return 1"}},
		{"loops": map[string]any{"run": "evaluator.sh"}},
		{"loops": map[string]any{"unsupported": true}},
	} {
		_, err := ParseGradersFromFrontmatter(map[string]any{"graders": graders})
		require.Error(t, err)
	}
}

func TestParseGradersInvalidFields(t *testing.T) {
	for _, value := range []any{
		"loops",
		map[string]any{"bad id": nil},
		map[string]any{"loops": true},
		map[string]any{"loops": map[string]any{"enabled": "yes"}},
		map[string]any{"loops": map[string]any{"threshold": "high"}},
		map[string]any{"loops": map[string]any{"config": true}},
		map[string]any{"loops": map[string]any{"direction": "up"}},
		map[string]any{"loops": nil, " loops ": nil},
	} {
		_, err := ParseGradersFromFrontmatter(map[string]any{"graders": value})
		require.Error(t, err)
	}
}

func TestGradersCanAllBeDisabled(t *testing.T) {
	entries := map[string]any{}
	for _, id := range DefaultGraderIDs {
		entries[id] = map[string]any{"enabled": false}
	}
	cfg, err := ParseGradersFromFrontmatter(map[string]any{"graders": entries})
	require.NoError(t, err)
	assert.False(t, cfg.HasGraders())
	assert.Empty(t, cfg.EnabledGraderIDs())
}

func TestBuiltinGraderRegistryMatchesRuntime(t *testing.T) {
	content, err := os.ReadFile("../../actions/setup/js/builtin_graders.json")
	require.NoError(t, err)
	var metadata map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(content, &metadata))
	require.Len(t, metadata, len(BuiltinGraderRegistry))
	for _, meta := range BuiltinGraderRegistry {
		var actual struct {
			Name        string
			Description string
			Unit        string
			Direction   string
			Threshold   *float64
			Min         *float64
			Max         *float64
		}
		require.NoError(t, json.Unmarshal(metadata[meta.ID], &actual))
		assert.Equal(t, meta.Name, actual.Name)
		assert.Equal(t, meta.Description, actual.Description)
		assert.Equal(t, meta.Unit, actual.Unit)
		assert.Equal(t, meta.Direction, actual.Direction)
		assert.Equal(t, meta.Threshold, actual.Threshold)
		assert.Equal(t, meta.Min, actual.Min)
		assert.Equal(t, meta.Max, actual.Max)
	}
}

func TestGraderEnvironmentRoundTrip(t *testing.T) {
	cfg, err := ParseGradersFromFrontmatter(map[string]any{"graders": map[string]any{
		"loops": map[string]any{"name": "'${{ secrets.DO_NOT_EVALUATE }}", "config": map[string]any{"data": strings.Repeat("x", 6000)}},
	}})
	require.NoError(t, err)
	env, err := graderEnvironment(cfg)
	require.NoError(t, err)
	assert.NotContains(t, env, "${{")
	var actual graderManifest
	require.NoError(t, json.Unmarshal(graderEnvironmentPayload(env, "GH_AW_GRADER_MANIFEST"), &actual))
	assert.Equal(t, buildGraderManifest(cfg), &actual)
	assert.NotContains(t, env, "GH_AW_GRADER_EXEC")
}

func TestGraderDefaultEnvironmentIsCompact(t *testing.T) {
	cfg, err := ParseGradersFromFrontmatter(map[string]any{})
	require.NoError(t, err)
	env, err := graderEnvironment(cfg)
	require.NoError(t, err)
	assert.JSONEq(t, `{"version":2,"graders":[]}`, string(graderEnvironmentPayload(env, "GH_AW_GRADER_MANIFEST")))
}

func TestGradersRunOnlyInConclusion(t *testing.T) {
	for _, config := range []string{"", "graders: {}\n", "graders:\n  loops:\n    threshold: 8\n"} {
		t.Run(config, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "test.md")
			require.NoError(t, os.WriteFile(file, []byte("---\non: workflow_dispatch\nengine: copilot\nstrict: false\n"+config+"---\nTest.\n"), 0o600))
			compiler := NewCompiler(WithVersion("dev"))
			require.NoError(t, compiler.CompileWorkflow(file))
			content, err := os.ReadFile(stringutil.MarkdownToLockFile(file))
			require.NoError(t, err)
			agent := extractJobSection(string(content), "agent")
			conclusion := extractJobSection(string(content), "conclusion")
			assert.NotContains(t, agent, "GRADER")
			assert.NotContains(t, agent, "/graders/")
			assert.Contains(t, conclusion, "GH_AW_GRADER_MANIFEST")
			assert.Contains(t, conclusion, "generate_usage_artifacts.cjs")
			assert.Less(t, strings.Index(conclusion, "Generate usage activity summary and unified session"), strings.Index(conclusion, "Upload usage artifact"))
			assert.Contains(t, conclusion, "/usage/graders/grader_results.json")
			assert.NotContains(t, string(content), "GH_AW_GRADER_EXEC")
		})
	}
}

func TestMergeImportedGradersFrontmatter(t *testing.T) {
	merged, err := mergeImportedGradersFrontmatter(map[string]any{"graders": map[string]any{
		"loops": map[string]any{"threshold": 8},
	}}, `{"loops":{"threshold":2,"enabled":true},"recurrence-determinism":{}}`)
	require.NoError(t, err)
	cfg, err := ParseGradersFromFrontmatter(merged)
	require.NoError(t, err)
	assert.InDelta(t, 8.0, *cfg.Graders["loops"].Threshold, 1e-9)
	assert.Contains(t, cfg.EnabledGraderIDs(), "recurrence-determinism")
}
