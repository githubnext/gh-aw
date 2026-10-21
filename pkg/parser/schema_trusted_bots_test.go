//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSandboxMCPTrustedBotsSchema(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"one bot", []any{"github-actions[bot]"}, false},
		{"multiple bots", []any{"github-actions[bot]", "copilot-swe-agent[bot]"}, false},
		{"login without suffix", []any{"My-Bot_123"}, false},
		{"empty list", []any{}, true},
		{"empty identity", []any{""}, true},
		{"command substitution", []any{"$(id)"}, true},
		{"backtick substitution", []any{"`id`"}, true},
		{"variable expansion", []any{"${HOME}"}, true},
		{"workflow expression", []any{"${{ github.actor }}"}, true},
		{"whitespace", []any{"my bot"}, true},
		{"trailing newline", []any{"github-actions[bot]\n"}, true},
		{"quote", []any{`my"bot`}, true},
		{"backslash", []any{`my\bot`}, true},
		{"non-ASCII identity", []any{"böt"}, true},
		{"invalid suffix", []any{"my-bot[other]"}, true},
		{"mixed types", []any{"github-actions[bot]", 42}, true},
		{"scalar", "github-actions[bot]", true},
		{"null", nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": "workflow_dispatch",
				"sandbox": map[string]any{
					"mcp": map[string]any{"trusted-bots": test.value},
				},
			}, "trusted-bots.md")
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("gateway JSON key is not a frontmatter alias", func(t *testing.T) {
		err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
			"on": "workflow_dispatch",
			"sandbox": map[string]any{
				"mcp": map[string]any{"trustedBots": []any{"github-actions[bot]"}},
			},
		}, "trusted-bots.md")
		require.Error(t, err)
		require.Contains(t, err.Error(), "trustedBots")
	})
}
