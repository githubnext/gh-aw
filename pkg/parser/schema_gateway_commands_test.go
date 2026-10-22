package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayCommandSchemaParity(t *testing.T) {
	for _, command := range []struct{ gateway, native string }{{"slash-command", "slash_command"}, {"label-command", "label_command"}} {
		for _, config := range []any{
			nil, "review", true, []any{"review"}, map[string]any{},
			map[string]any{"name": []any{"review", "fix*"}, "events": []any{"issues", "pull_request"}, "strategy": "inline"},
			map[string]any{"names": []any{"review"}, "remove_label": true, "strategy": "decentralized"},
			map[string]any{"name": "review", "strategy": "centralized"},
			map[string]any{"events": "issue_comment"}, map[string]any{"unknown": true},
			map[string]any{"name": "/review"}, map[string]any{"name": []any{}},
		} {
			nativeErr := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{command.native: config}}, "native.md")
			gatewayErr := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{command.gateway: config}}}, "gateway.md")
			require.Equal(t, nativeErr == nil, gatewayErr == nil, "%s: %#v; native=%v; gateway=%v", command.gateway, config, nativeErr, gatewayErr)
		}
	}
}

func TestGatewayLabelCommandRequiresConsumption(t *testing.T) {
	config := map[string]any{"name": "review", "remove_label": false}
	require.NoError(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"label_command": config}}, "native.md"))
	require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{"label-command": config}}}, "gateway.md"))
}
