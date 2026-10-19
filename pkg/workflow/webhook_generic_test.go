package workflow

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genericWebhookOn(config any) map[string]any {
	return map[string]any{"gateway-dispatch": map[string]any{"webhooks": []any{config}}}
}

func TestGenericWebhookSubscriptions(t *testing.T) {
	on := genericWebhookOn(map[string]any{
		"type": "com.example.release",
		"filters": map[string]any{
			"data":   map[string]any{"state": []any{"ready", "published", "ready"}, "approved": false, "attempt": 2},
			"labels": map[string]any{"*": map[string]any{"name": "deploy"}},
		},
	})
	on["gateway-dispatch"].(map[string]any)["repos"] = []any{"owner/*"}
	before, err := json.Marshal(on)
	require.NoError(t, err)
	subscriptions, err := ParseWebhookSubscriptions(on, "generic")
	require.NoError(t, err)
	require.Len(t, subscriptions, 1)
	subscription := subscriptions[0]
	assert.Equal(t, "webhook", subscription.Event)
	assert.Equal(t, []string{"com.example.release"}, subscription.Types)
	assert.Empty(t, subscription.Repos)
	assert.Empty(t, NormalizeGitHubAppWebhookEvents(on))
	assert.Empty(t, ComputeGitHubAppManifestWebhookPermissions(on))
	assert.Equal(t, WebhookPayloadProjectionPolicy, subscription.Dispatch.PayloadProjection.Policy)
	assert.Equal(t, []WebhookCondition{
		{Filter: "data.approved", Path: []string{"data", "approved"}, Operator: "equals", ValueType: "boolean", Values: []any{false}},
		{Filter: "data.attempt", Path: []string{"data", "attempt"}, Operator: "equals", ValueType: "integer", Values: []any{2}},
		{Filter: "data.state", Path: []string{"data", "state"}, Operator: "equals", ValueType: "string", Values: []any{"published", "ready"}},
		{Filter: "labels.*.name", Path: []string{"labels", "*", "name"}, Operator: "equals", ValueType: "string", Values: []any{"deploy"}},
	}, subscription.Conditions)
	after, err := json.Marshal(on)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
	var notice bytes.Buffer
	NewCompiler().emitExperimentalFeatureWarningsTo(&WorkflowData{WebhookSubscriptions: subscriptions}, &notice)
	assert.Contains(t, notice.String(), "on.gateway-dispatch.webhooks (approved webhook bridge required)")
}

func TestGenericWebhookValidation(t *testing.T) {
	for _, config := range []any{
		nil, true, "example.event", map[string]any{}, map[string]any{"type": nil},
		map[string]any{"type": []any{"example.event"}}, map[string]any{"type": ""},
		map[string]any{"type": "example event"}, map[string]any{"type": "*"},
		map[string]any{"type": "${{ github.event_name }}"},
		map[string]any{"type": "example.event", "types": []any{"created"}},
		map[string]any{"type": "example.event", "repos": []any{"owner/repo"}},
		map[string]any{"type": "example.event", "script": "true"},
	} {
		_, err := ParseWebhookSubscriptions(genericWebhookOn(config), "invalid")
		require.Error(t, err, "%#v", config)
	}
	for _, filters := range []any{
		nil, true, "jq .data", map[string]any{}, map[string]any{"data.state": "ready"},
		map[string]any{"data": map[string]any{}}, map[string]any{"state": nil},
		map[string]any{"state": []any{}}, map[string]any{"state": []any{"ready", true}},
		map[string]any{"state": []any{map[string]any{"value": "ready"}}},
		map[string]any{"state": "${{ secrets.VALUE }}"}, map[string]any{"attempt": 1.5},
		map[string]any{"attempt": int64(9007199254740992)},
		map[string]any{"__proto__": map[string]any{"state": "ready"}},
		map[string]any{"constructor": "ready"}, map[string]any{"$(command)": "ready"},
	} {
		_, err := ParseWebhookSubscriptions(genericWebhookOn(map[string]any{"type": "example.event", "filters": filters}), "invalid")
		require.Error(t, err, "%#v", filters)
	}
	subscriptions, err := ParseWebhookSubscriptions(genericWebhookOn(map[string]any{"type": "example.event"}), "unfiltered")
	require.NoError(t, err)
	assert.Empty(t, subscriptions[0].Conditions)
}

func TestGenericWebhookArray(t *testing.T) {
	on := map[string]any{"gateway-dispatch": map[string]any{"webhooks": []any{
		map[string]any{"type": "example.z", "filters": map[string]any{"state": "ready"}},
		map[string]any{"type": "example.a"},
	}}}
	subscriptions, err := ParseWebhookSubscriptions(on, "multiple")
	require.NoError(t, err)
	require.Len(t, subscriptions, 2)
	assert.Equal(t, []string{"example.a"}, subscriptions[0].Types)
	assert.Equal(t, []string{"example.z"}, subscriptions[1].Types)
	for _, invalid := range []any{nil, []any{}, map[string]any{"type": "example.event"}, []any{map[string]any{"type": "example.event"}, map[string]any{"type": "example.event", "filters": map[string]any{"state": "ready"}}}} {
		_, err := ParseWebhookSubscriptions(map[string]any{"gateway-dispatch": map[string]any{"webhooks": invalid}}, "invalid")
		require.Error(t, err)
	}
	_, err = ParseWebhookSubscriptions(map[string]any{"gateway-dispatch": map[string]any{"webhook": map[string]any{"type": "example.event"}}}, "singular")
	require.ErrorContains(t, err, "unsupported on.gateway-dispatch property")
}
