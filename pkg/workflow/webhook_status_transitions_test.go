package workflow

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookNativeStatusTransitions(t *testing.T) {
	config := map[string]any{
		"project_id": "PVT_example",
		"status-transitions": []any{
			map[string]any{"from": "Todo", "to": "Ready"},
			map[string]any{"from": "In progress", "to": "Done"},
		},
	}
	on := map[string]any{"gateway-dispatch": map[string]any{"projects-items": config}}
	before, err := json.Marshal(on)
	require.NoError(t, err)
	subscriptions, err := ParseWebhookSubscriptions(on, "project-agent")
	require.NoError(t, err)
	require.Len(t, subscriptions, 1)
	subscription := subscriptions[0]
	assert.Equal(t, []string{"edited"}, subscription.Types)
	require.Len(t, subscription.Conditions, 3)
	assert.Equal(t, "field.name", subscription.Conditions[0].Filter)
	assert.Equal(t, []any{"Status"}, subscription.Conditions[0].Values)
	assert.Equal(t, "field.type", subscription.Conditions[1].Filter)
	assert.Equal(t, []any{"single_select"}, subscription.Conditions[1].Values)
	assert.Equal(t, "project_id", subscription.Conditions[2].Filter)
	assert.Equal(t, []any{"PVT_example"}, subscription.Conditions[2].Values)
	require.Len(t, subscription.AnyOf, 2)
	require.Len(t, subscription.AnyOf[0], 2)
	assert.Equal(t, []string{"changes", "field_value", "from", "name"}, subscription.AnyOf[0][0].Path)
	assert.Equal(t, []any{"Todo"}, subscription.AnyOf[0][0].Values)
	assert.Equal(t, []string{"changes", "field_value", "to", "name"}, subscription.AnyOf[0][1].Path)
	assert.Equal(t, []any{"Ready"}, subscription.AnyOf[0][1].Values)
	after, err := json.Marshal(on)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
	contract, err := json.Marshal(subscription)
	require.NoError(t, err)
	assert.Contains(t, string(contract), "any_of")
	assert.Contains(t, string(contract), "field.from.name")
	assert.Contains(t, string(contract), "field.to.name")
	fixture, err := os.ReadFile("testdata/webhook-project-item-edited.json")
	require.NoError(t, err)
	var nativePayload map[string]any
	require.NoError(t, json.Unmarshal(fixture, &nativePayload))
	assert.True(t, matchWebhookContract(subscription, nativePayload))
	for _, tt := range []struct {
		from, to, field, project, action, fieldType string
		match                                       bool
	}{
		{"Todo", "Ready", "Status", "PVT_example", "edited", "single_select", true},
		{"In progress", "Done", "Status", "PVT_example", "edited", "single_select", true},
		{"Todo", "Done", "Status", "PVT_example", "edited", "single_select", false},
		{"In progress", "Ready", "Status", "PVT_example", "edited", "single_select", false},
		{"Ready", "Todo", "Status", "PVT_example", "edited", "single_select", false},
		{"Todo", "ready", "Status", "PVT_example", "edited", "single_select", false},
		{"Todo", "Ready", "Priority", "PVT_example", "edited", "single_select", false},
		{"Todo", "Ready", "Status", "PVT_other", "edited", "single_select", false},
		{"Todo", "Ready", "Status", "PVT_example", "created", "single_select", false},
		{"Todo", "Ready", "Status", "PVT_example", "edited", "text", false},
	} {
		t.Run(tt.from+"/"+tt.to+"/"+tt.field+"/"+tt.project+"/"+tt.action+"/"+tt.fieldType, func(t *testing.T) {
			var payload map[string]any
			require.NoError(t, json.Unmarshal(fixture, &payload))
			payload["action"] = tt.action
			payload["projects_v2_item"].(map[string]any)["project_node_id"] = tt.project
			field := payload["changes"].(map[string]any)["field_value"].(map[string]any)
			field["field_name"] = tt.field
			field["field_type"] = tt.fieldType
			field["from"] = map[string]any{"name": tt.from}
			field["to"] = map[string]any{"name": tt.to}
			assert.Equal(t, tt.match, matchWebhookContract(subscription, payload))
			for _, direction := range []string{"from", "to"} {
				value := field[direction]
				field[direction] = nil
				assert.False(t, matchWebhookContract(subscription, payload))
				delete(field, direction)
				assert.False(t, matchWebhookContract(subscription, payload))
				field[direction] = value
			}
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
			_, err := ParseWebhookSubscriptions(map[string]any{"gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example", "status-transitions": transitions}}}, "project-agent")
			require.ErrorContains(t, err, "on.gateway-dispatch.projects-items.status-transitions")
		})
	}
	_, err := ParseWebhookSubscriptions(map[string]any{"gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example", "types": []any{"created"}, "status-transitions": []any{map[string]any{"from": "Todo", "to": "Ready"}}}}}, "project-agent")
	require.ErrorContains(t, err, "requires types to include edited")
}
