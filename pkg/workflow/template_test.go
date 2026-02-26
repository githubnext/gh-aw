//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateInterpolationAndTemplateStep_DeduplicatesEnvVars tests that duplicate
// expression mappings (same EnvVar) are only emitted once in the env section,
// preventing runtime errors when import and main workflow reference the same variable.
func TestGenerateInterpolationAndTemplateStep_DeduplicatesEnvVars(t *testing.T) {
	compiler := &Compiler{}
	data := &WorkflowData{
		MarkdownContent: "hello",
		ParsedTools:     NewTools(map[string]any{}),
	}

	// Simulate the same variable referenced in both imported content and main workflow
	expressionMappings := []*ExpressionMapping{
		{EnvVar: "GH_AW_VARS_MY_VAR", Content: "vars.MY_VAR"},
		{EnvVar: "GH_AW_VARS_MY_VAR", Content: "vars.MY_VAR"}, // duplicate
		{EnvVar: "GH_AW_VARS_OTHER_VAR", Content: "vars.OTHER_VAR"},
	}

	var yaml strings.Builder
	compiler.generateInterpolationAndTemplateStep(&yaml, expressionMappings, data)

	result := yaml.String()

	// MY_VAR should appear exactly once
	count := strings.Count(result, "GH_AW_VARS_MY_VAR: ${{ vars.MY_VAR }}")
	assert.Equal(t, 1, count, "duplicate env var GH_AW_VARS_MY_VAR should appear exactly once")

	// OTHER_VAR should be present
	assert.Contains(t, result, "GH_AW_VARS_OTHER_VAR: ${{ vars.OTHER_VAR }}", "OTHER_VAR should be present")
}
