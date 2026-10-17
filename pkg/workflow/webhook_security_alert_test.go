package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityAlertSubscriptions(t *testing.T) {
	tests := []struct {
		kind   string
		event  string
		filter string
		value  string
		path   []string
	}{
		{"code-scanning", "code_scanning_alert", "severity", "error", []string{"alert", "rule", "severity"}},
		{"dependabot", "dependabot_alert", "severity", "critical", []string{"alert", "security_advisory", "severity"}},
		{"secret-scanning", "secret_scanning_alert", "secret_type", "github_personal_access_token", []string{"alert", "secret_type"}},
	}
	require.Len(t, securityAlertEvents, len(tests))
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			config := map[string]any{
				"type":    tt.kind,
				"types":   []any{"created"},
				"filters": map[string]any{tt.filter: tt.value},
			}
			on := map[string]any{
				"security-alert": config,
				"webhook-auth":   map[string]any{"actors": []any{"cao[bot]"}},
			}
			before, err := json.Marshal(on)
			require.NoError(t, err)
			subscriptions, err := ParseWebhookSubscriptions(on, "security-agent")
			require.NoError(t, err)
			require.Len(t, subscriptions, 1)
			sub := subscriptions[0]
			assert.Equal(t, tt.event, sub.Event)
			assert.Equal(t, []string{"created"}, sub.Types)
			require.Len(t, sub.Conditions, 1)
			assert.Equal(t, tt.path, sub.Conditions[0].Path)
			assert.Equal(t, []any{tt.value}, sub.Conditions[0].Values)
			assert.Equal(t, []string{"cao[bot]"}, sub.Dispatch.TrustedActors)
			after, err := json.Marshal(on)
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after))
			require.NoError(t, ValidateEventTypes(map[string]any{"on": on}))
			assert.Equal(t, []string{tt.event}, NormalizeGitHubAppWebhookEvents(on))

			generic := webhookTestOn(tt.event, map[string]any{
				"types": []any{"created"}, "filters": map[string]any{tt.filter: tt.value},
			})
			generic["webhook-auth"] = on["webhook-auth"]
			want, err := ParseWebhookSubscriptions(generic, "security-agent")
			require.NoError(t, err)
			assert.Equal(t, want, subscriptions)

			on["security-alert"] = map[string]any{"type": tt.kind}
			all, err := ParseWebhookSubscriptions(on, "security-agent")
			require.NoError(t, err)
			generic["webhook"].(map[string]any)[tt.event] = map[string]any{}
			want, err = ParseWebhookSubscriptions(generic, "security-agent")
			require.NoError(t, err)
			assert.Equal(t, want, all)
			assert.True(t, matchWebhookContract(all[0], map[string]any{"action": "created"}))
			assert.False(t, matchWebhookContract(all[0], map[string]any{"action": "completed"}))

			delete(on, "webhook-auth")
			_, err = ParseWebhookSubscriptions(on, "security-agent")
			require.ErrorContains(t, err, "on.webhook-auth")
			on = webhookTestOn(tt.event, map[string]any{})
			on["security-alert"] = map[string]any{"type": tt.kind}
			_, err = ParseWebhookSubscriptions(on, "security-agent")
			require.ErrorContains(t, err, "subscribe to the same event")
		})
	}
}

func TestSecurityAlertRejectsInvalidConfiguration(t *testing.T) {
	for name, tt := range map[string]struct {
		config any
		want   string
	}{
		"null":         {nil, "must be an object"},
		"scalar":       {"dependabot", "must be an object"},
		"missing type": {map[string]any{}, "on.security-alert.type"},
		"null type":    {map[string]any{"type": nil}, "on.security-alert.type"},
		"unknown type": {map[string]any{"type": "security_advisory"}, "on.security-alert.type"},
		"list type":    {map[string]any{"type": []any{"code-scanning", "dependabot"}}, "on.security-alert.type"},
		"scan event":   {map[string]any{"type": "secret-scanning-scan"}, "on.security-alert.type"},
		"location event": {map[string]any{
			"type": "secret-scanning-alert-location",
		}, "on.security-alert.type"},
		"unsupported property": {map[string]any{"type": "dependabot", "typo": true}, "unsupported on.security-alert property"},
		"wrong action":         {map[string]any{"type": "secret-scanning", "types": []any{"completed"}}, "unsupported activity type"},
		"wrong source action":  {map[string]any{"type": "code-scanning", "types": []any{"publicly_leaked"}}, "unsupported activity type"},
		"empty actions":        {map[string]any{"type": "dependabot", "types": []any{}}, "non-empty list"},
		"unsupported filter":   {map[string]any{"type": "dependabot", "filters": map[string]any{"not-a-filter": true}}, "unsupported filter"},
		"wrong filter type":    {map[string]any{"type": "code-scanning", "filters": map[string]any{"severity": true}}, "expected string"},
		"secret material":      {map[string]any{"type": "secret-scanning", "filters": map[string]any{"secret": "not-a-secret"}}, "unsupported filter"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseWebhookSubscriptions(map[string]any{
				"security-alert": tt.config,
				"webhook-auth":   map[string]any{"actors": []any{"cao[bot]"}},
			}, "security-agent")
			require.ErrorContains(t, err, tt.want)
			assert.Contains(t, err.Error(), "on.security-alert")
		})
	}
	for _, shorthand := range []any{"security-alert", []any{"push", "security-alert"}} {
		_, err := ParseWebhookSubscriptions(shorthand, "security-agent")
		require.ErrorContains(t, err, "requires mapping syntax and on.webhook-auth")
	}
}
