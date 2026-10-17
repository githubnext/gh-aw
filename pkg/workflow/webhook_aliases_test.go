package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstClassWebhookTriggers(t *testing.T) {
	tests := []struct {
		alias  string
		event  string
		action string
		filter string
		value  any
		path   []string
	}{
		{"projects", "projects_v2", "edited", "project_id", "PVT_example", []string{"projects_v2", "node_id"}},
		{"projects-item", "projects_v2_item", "edited", "project_id", "PVT_example", []string{"projects_v2_item", "project_node_id"}},
		{"projects-status-update", "projects_v2_status_update", "created", "status", "ON_TRACK", []string{"projects_v2_status_update", "status"}},
		{"code-scanning-alert", "code_scanning_alert", "created", "severity", "error", []string{"alert", "rule", "severity"}},
		{"dependabot-alert", "dependabot_alert", "created", "severity", "critical", []string{"alert", "security_advisory", "severity"}},
		{"secret-scanning-alert", "secret_scanning_alert", "created", "secret_type", "github_personal_access_token", []string{"alert", "secret_type"}},
		{"secret-scanning-alert-location", "secret_scanning_alert_location", "created", "location_type", "commit", []string{"location", "type"}},
		{"secret-scanning-scan", "secret_scanning_scan", "completed", "repository", "octo/repo", []string{"repository", "full_name"}},
	}
	require.Len(t, webhookTriggerAliases, len(tests))
	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			require.Equal(t, tt.event, webhookTriggerAliases[tt.alias])
			config := map[string]any{
				"types":   []any{tt.action},
				"filters": map[string]any{tt.filter: tt.value},
			}
			on := map[string]any{
				tt.alias:       config,
				"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}},
			}
			got, err := ParseWebhookSubscriptions(on, "first-class")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, tt.event, got[0].Event)
			assert.Equal(t, []string{tt.action}, got[0].Types)
			require.Len(t, got[0].Conditions, 1)
			assert.Equal(t, tt.path, got[0].Conditions[0].Path)
			assert.Equal(t, []any{tt.value}, got[0].Conditions[0].Values)
			assert.Equal(t, []string{"cao[bot]"}, got[0].Dispatch.TrustedActors)

			generic := webhookTestOn(tt.event, config)
			generic["webhook-auth"] = on["webhook-auth"]
			want, err := ParseWebhookSubscriptions(generic, "first-class")
			require.NoError(t, err)
			assert.Equal(t, want, got)

			on[tt.alias] = nil
			all, err := ParseWebhookSubscriptions(on, "first-class")
			require.NoError(t, err)
			generic["webhook"].(map[string]any)[tt.event] = map[string]any{}
			want, err = ParseWebhookSubscriptions(generic, "first-class")
			require.NoError(t, err)
			assert.Equal(t, want, all, "null selects all cataloged activity types")
			require.NoError(t, ValidateEventTypes(map[string]any{"on": on}))
			assert.Equal(t, []string{tt.event}, NormalizeGitHubAppWebhookEvents(on))

			delete(on, "webhook-auth")
			_, err = ParseWebhookSubscriptions(on, "first-class")
			require.ErrorContains(t, err, "on.webhook-auth")
			for _, shorthand := range []any{tt.alias, []any{"push", tt.alias}} {
				_, err = ParseWebhookSubscriptions(shorthand, "first-class")
				require.ErrorContains(t, err, "requires mapping syntax and on.webhook-auth")
			}
		})
	}
}

func TestFirstClassWebhookValidation(t *testing.T) {
	for alias, event := range webhookTriggerAliases {
		t.Run(alias, func(t *testing.T) {
			for name, tt := range map[string]struct {
				config any
				want   string
			}{
				"scalar":           {true, "must be an object"},
				"unknown property": {map[string]any{"typo": true}, "property"},
				"invalid type":     {map[string]any{"types": []any{"not-an-action"}}, "unsupported activity type"},
				"unknown filter":   {map[string]any{"filters": map[string]any{"not-a-filter": true}}, "unsupported filter"},
				"wrong value type": {map[string]any{"filters": map[string]any{"sender": false}}, "expected string"},
			} {
				t.Run(name, func(t *testing.T) {
					_, err := ParseWebhookSubscriptions(map[string]any{
						alias:          tt.config,
						"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}},
					}, "first-class")
					require.ErrorContains(t, err, tt.want)
					assert.Contains(t, err.Error(), "on."+alias)
				})
			}
			on := webhookTestOn(event, map[string]any{})
			on[alias] = nil
			_, err := ParseWebhookSubscriptions(on, "first-class")
			require.ErrorContains(t, err, "subscribe to the same event")
			assert.Contains(t, err.Error(), "on."+alias)
			assert.Contains(t, err.Error(), "on.webhook."+event)
		})
	}
}

func TestFirstClassWebhookPayloadRestrictions(t *testing.T) {
	for _, tt := range []struct {
		alias  string
		config map[string]any
		want   string
	}{
		{"projects", map[string]any{"types": []any{"moved"}}, "unsupported activity type"},
		{"projects-item", map[string]any{"types": []any{"moved"}}, "unsupported activity type"},
		{"projects-item", map[string]any{"filters": map[string]any{"labels": "bug"}}, "unsupported filter"},
		{"projects-item", map[string]any{
			"types": []any{"created"},
			"filters": map[string]any{
				"field": map[string]any{"to": map[string]any{"name": "Ready"}},
			},
		}, "requires one of"},
		{"secret-scanning-alert", map[string]any{"filters": map[string]any{"secret": "not-a-secret"}}, "unsupported filter"},
		{"secret-scanning-alert-location", map[string]any{"filters": map[string]any{"secret": "not-a-secret"}}, "unsupported filter"},
	} {
		t.Run(tt.alias+"/"+tt.want, func(t *testing.T) {
			_, err := ParseWebhookSubscriptions(map[string]any{
				tt.alias:       tt.config,
				"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}},
			}, "first-class")
			require.ErrorContains(t, err, tt.want)
			assert.Contains(t, err.Error(), "on."+tt.alias)
		})
	}
}
