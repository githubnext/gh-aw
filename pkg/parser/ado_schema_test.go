package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestADOToolSchema(t *testing.T) {
	tests := []struct {
		name    string
		ado     any
		wantErr bool
	}{
		{
			name: "required configuration",
			ado: map[string]any{
				"organization": "contoso",
				"token":        "${{ secrets.ADO_MCP_AUTH_TOKEN }}",
			},
		},
		{
			name: "restricted toolsets",
			ado: map[string]any{
				"organization": "contoso",
				"token":        "${{ secrets.ADO_MCP_AUTH_TOKEN }}",
				"toolsets":     []any{"repos", "wit"},
			},
		},
		{name: "disabled", ado: false},
		{
			name: "missing token",
			ado: map[string]any{
				"organization": "contoso",
			},
			wantErr: true,
		},
		{
			name: "literal token",
			ado: map[string]any{
				"organization": "contoso",
				"token":        "literal-token",
			},
			wantErr: true,
		},
		{
			name: "unsupported toolset",
			ado: map[string]any{
				"organization": "contoso",
				"token":        "${{ secrets.ADO_MCP_AUTH_TOKEN }}",
				"toolsets":     []any{"issues"},
			},
			wantErr: true,
		},
		{
			name: "unknown property",
			ado: map[string]any{
				"organization": "contoso",
				"token":        "${{ secrets.ADO_MCP_AUTH_TOKEN }}",
				"read-only":    false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(
				map[string]any{
					"on":    map[string]any{"workflow_dispatch": nil},
					"tools": map[string]any{"ado": tt.ado},
				},
				"/tmp/gh-aw/ado-schema-test.md",
			)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
