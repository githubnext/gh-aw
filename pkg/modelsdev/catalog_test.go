//go:build !integration

package modelsdev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeProvider(t *testing.T) {
	cases := []struct{ input, want string }{
		{"github", "github-copilot"},
		{"copilot", "github-copilot"},
		{"github_models", "github-copilot"},
		{"GITHUB_MODELS", "github-copilot"},
		{"anthropic", "anthropic"},
		{"OpenAI", "openai"},
		{"  Anthropic  ", "anthropic"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.input+"->"+tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeProvider(tc.input))
		})
	}
}

func TestNormalizeComparableModelID(t *testing.T) {
	cases := []struct{ input, want string }{
		{"claude-sonnet-4.6", "claude-sonnet-4-6"},
		{"gpt_4o", "gpt-4o"},
		{"GPT-4O", "gpt-4o"},
		{"  claude.3  ", "claude-3"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.input+"->"+tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeComparableModelID(tc.input))
		})
	}
}
