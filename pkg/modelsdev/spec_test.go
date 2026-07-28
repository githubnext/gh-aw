//go:build !integration

package modelsdev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_NormalizeProvider validates the documented alias and
// case-normalization behavior of NormalizeProvider as described in the
// modelsdev README.md specification.
func TestSpec_PublicAPI_NormalizeProvider(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "github alias", input: "github", expected: "github-copilot"},
		{name: "copilot alias", input: " copilot ", expected: "github-copilot"},
		{name: "github_models alias", input: "GITHUB_MODELS", expected: "github-copilot"},
		{name: "other provider lower-cased", input: "OpenAI", expected: "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeProvider(tt.input))
		})
	}
}

// TestSpec_PublicAPI_NormalizeComparableModelID validates the documented
// comparison normalization of NormalizeComparableModelID as described in the
// modelsdev README.md specification.
func TestSpec_PublicAPI_NormalizeComparableModelID(t *testing.T) {
	assert.Equal(t, "gpt-4-1-mini", NormalizeComparableModelID(" GPT_4.1_mini "))
	assert.Equal(t, "claude-3-5-sonnet", NormalizeComparableModelID("claude-3_5.sonnet"))
}
