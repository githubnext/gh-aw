package workflow

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookNativeStatusTransitions(t *testing.T) {
	on := map[string]any{"projects-item": map[string]any{
		"project_id": "PVT_example",
		"status-transitions": []any{
			map[string]any{"from": "Todo", "to": "Ready"},
			map[string]any{"from": "In progress", "to": "Done"},
		},
	}}
	before, err := json.Marshal(on)
	require.NoError(t, err)
	subscriptions, err := ParseWebhookSubscriptions(on, "project-agent")
	require.NoError(t, err)
	require.Len(t, subscriptions, 1)
	subscription := subscriptions[0]
	assert.Equal(t, []string{"edited"}, subscription.Types)
	require.Len(t, subscription.Conditions, 2)
	assert.Equal(t, "field.name", subscription.Conditions[0].Filter)
	assert.Equal(t, []any{"Status"}, subscription.Conditions[0].Values)
	assert.Equal(t, "project_id", subscription.Conditions[1].Filter)
	assert.Equal(t, []any{"PVT_example"}, subscription.Conditions[1].Values)
	require.Len(t, subscription.AnyOf, 2)
	require.Len(t, subscription.AnyOf[0], 2)
	assert.Equal(t, []string{"changes", "field_value", "from", "name"}, subscription.AnyOf[0][0].Path)
	assert.Equal(t, []any{"Todo"}, subscription.AnyOf[0][0].Values)
	assert.Equal(t, []string{"changes", "field_value", "to", "name"}, subscription.AnyOf[0][1].Path)
	assert.Equal(t, []any{"Ready"}, subscription.AnyOf[0][1].Values)
	after, err := json.Marshal(on)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
	fixture, err := os.ReadFile("testdata/webhook-project-item-edited.json")
	require.NoError(t, err)
	var nativePayload map[string]any
	require.NoError(t, json.Unmarshal(fixture, &nativePayload))
	assert.False(t, matchWebhookContract(subscription, nativePayload), "native deliveries lack status names and transition operands")
	for _, tt := range []struct {
		from, to, field, project, action string
		match                            bool
	}{
		{"Todo", "Ready", "Status", "PVT_example", "edited", true},
		{"In progress", "Done", "Status", "PVT_example", "edited", true},
		{"Todo", "Done", "Status", "PVT_example", "edited", false},
		{"In progress", "Ready", "Status", "PVT_example", "edited", false},
		{"Ready", "Todo", "Status", "PVT_example", "edited", false},
		{"Todo", "ready", "Status", "PVT_example", "edited", false},
		{"Todo", "Ready", "Priority", "PVT_example", "edited", false},
		{"Todo", "Ready", "Status", "PVT_other", "edited", false},
		{"Todo", "Ready", "Status", "PVT_example", "created", false},
	} {
		t.Run(tt.from+"/"+tt.to+"/"+tt.field+"/"+tt.project+"/"+tt.action, func(t *testing.T) {
			var payload map[string]any
			require.NoError(t, json.Unmarshal(fixture, &payload))
			payload["action"] = tt.action
			item, ok := payload["projects_v2_item"].(map[string]any)
			require.True(t, ok)
			item["project_node_id"] = tt.project
			// Synthetic values verify paired predicates without claiming native payload support.
			payload["changes"] = map[string]any{"field_value": map[string]any{
				"field_name": tt.field,
				"from":       map[string]any{"name": tt.from},
				"to":         map[string]any{"name": tt.to},
			}}
			assert.Equal(t, tt.match, matchWebhookContract(subscription, payload))
			delete(payload, "changes")
			assert.False(t, matchWebhookContract(subscription, payload))
		})
	}
}

func TestWebhookNativeStatusTransitionsValidation(t *testing.T) {
	for name, transitions := range map[string]any{
		"null":            nil,
		"empty":           []any{},
		"object":          map[string]any{"from": "Todo", "to": "Ready"},
		"scalar entry":    []any{"Todo"},
		"missing from":    []any{map[string]any{"to": "Ready"}},
		"missing to":      []any{map[string]any{"from": "Todo"}},
		"empty from":      []any{map[string]any{"from": "", "to": "Ready"}},
		"blank to":        []any{map[string]any{"from": "Todo", "to": " "}},
		"non-string from": []any{map[string]any{"from": 7, "to": "Ready"}},
		"null to":         []any{map[string]any{"from": "Todo", "to": nil}},
		"expression":      []any{map[string]any{"from": "Todo", "to": "${{ github.actor }}"}},
		"extra key":       []any{map[string]any{"from": "Todo", "to": "Ready", "typo": true}},
		"wrong keys":      []any{map[string]any{"old": "Todo", "new": "Ready"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseWebhookSubscriptions(map[string]any{"projects-item": map[string]any{
				"project_id": "PVT_example", "status-transitions": transitions,
			}}, "project-agent")
			require.ErrorContains(t, err, "on.projects-item.status-transitions")
		})
	}
	_, err := ParseWebhookSubscriptions(map[string]any{"projects-item": map[string]any{
		"project_id": "PVT_example", "types": []any{"created"},
		"status-transitions": []any{map[string]any{"from": "Todo", "to": "Ready"}},
	}}, "project-agent")
	require.ErrorContains(t, err, "requires types to include edited")
}
