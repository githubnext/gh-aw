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
		{"empty list", []any{}, true},
		{"empty identity", []any{""}, true},
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
}
