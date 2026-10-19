package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericWebhookSchema(t *testing.T) {
	validate := func(config any) error {
		return ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{"webhooks": []any{config}}}}, "generic.md")
	}
	for _, config := range []any{
		map[string]any{"type": "example.event"},
		map[string]any{"type": "jira:issue_updated", "filters": map[string]any{"data": map[string]any{"enabled": false, "attempts": []any{1, 2}, "state": []any{"open", "closed"}}}},
		map[string]any{"type": "example.event", "filters": map[string]any{"labels": map[string]any{"*": map[string]any{"name": "deploy"}}}},
	} {
		require.NoError(t, validate(config))
	}
	for _, config := range []any{nil, true, "example.event", map[string]any{}, map[string]any{"type": "example.event", "types": []any{"created"}}} {
		require.Error(t, validate(config))
	}
	for _, eventType := range []any{nil, "", "*", "event type", " event", "${{ github.actor }}", strings.Repeat("a", 101), []any{"event"}} {
		require.Error(t, validate(map[string]any{"type": eventType}))
	}
	for _, filters := range []any{
		nil, "", map[string]any{}, map[string]any{"state": nil}, map[string]any{"state": []any{}},
		map[string]any{"state": []any{"open", true}}, map[string]any{"number": 1.5},
		map[string]any{"number": int64(9007199254740992)}, map[string]any{"state": "${{ secrets.VALUE }}"},
		map[string]any{"data.state": "open"}, map[string]any{"__proto__": "open"}, map[string]any{"constructor": "open"},
		map[string]any{"key[]": "open"}, map[string]any{"state": map[string]any{}},
	} {
		require.Error(t, validate(map[string]any{"type": "example.event", "filters": filters}), "%#v", filters)
	}
	for _, value := range []any{nil, []any{}, map[string]any{"type": "example.event"}, []any{"example.event"}} {
		require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{"webhooks": value}}}, "array.md"))
	}
	require.NoError(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{"webhooks": []any{
		map[string]any{"type": "example.first"}, map[string]any{"type": "example.second", "filters": map[string]any{"state": "ready"}},
	}}}}, "multiple.md"))
}
