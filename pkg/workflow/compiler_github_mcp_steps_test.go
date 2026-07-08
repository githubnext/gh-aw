//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeEnvStringValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "plain string", value: "myvalue", want: "myvalue"},
		{name: "string slice", value: []string{"a", "b"}, want: `["a","b"]`},
		{name: "empty slice", value: []string{}, want: `[]`},
		{name: "nil", value: nil, want: `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, serializeEnvStringValue(tt.value))
		})
	}
}

func TestQuoteYAMLEnvValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "'hello'", quoteYAMLEnvValue("hello"))
	assert.Equal(t, "'it''s'", quoteYAMLEnvValue("it's"))
	assert.Equal(t, `'["a","b"]'`, quoteYAMLEnvValue(`["a","b"]`))
}

func TestGenerateGitHubMCPLockdownDetectionStepSkipsWhenGuardPolicyExplicit(t *testing.T) {
	t.Parallel()

	var yaml strings.Builder
	data := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"min-integrity": "approved",
				"allowed-repos": []string{"github/gh-aw", "github/*"},
			},
		},
	}

	NewCompiler().generateGitHubMCPLockdownDetectionStep(&yaml, data)
	output := yaml.String()

	// When guard policies are explicitly configured, the detection step must not be generated.
	assert.Empty(t, output, "detection step should be skipped when guard policy is explicitly configured")
}

func TestGenerateGitHubMCPLockdownDetectionStepGeneratedWhenNoGuardPolicy(t *testing.T) {
	t.Parallel()

	var yaml strings.Builder
	data := &WorkflowData{
		Tools: map[string]any{
			"github": map[string]any{
				"mode": "local",
			},
		},
	}

	NewCompiler().generateGitHubMCPLockdownDetectionStep(&yaml, data)
	output := yaml.String()

	assert.Contains(t, output, "determine-automatic-lockdown", "detection step should be generated when no explicit guard policy")
	assert.NotContains(t, output, "GH_AW_GITHUB_MIN_INTEGRITY", "env var should not be present when min-integrity is not set")
	assert.NotContains(t, output, "GH_AW_GITHUB_REPOS", "env var should not be present when repos is not set")
}
