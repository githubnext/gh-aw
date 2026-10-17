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
		{"projects-item", "projects_v2_item", "edited", "project_id", "PVT_example", []string{"projects_v2_item", "project_node_id"}},
	}
	require.Len(t, webhookTriggerAliases, len(tests))
	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			require.Equal(t, tt.event, webhookTriggerAliases[tt.alias])
			config := map[string]any{
				"types":   []any{tt.action},
				tt.filter: tt.value,
			}
			on := map[string]any{
				tt.alias: config,
			}
			got, err := ParseWebhookSubscriptions(on, "first-class")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, tt.event, got[0].Event)
			assert.Equal(t, []string{tt.action}, got[0].Types)
			require.Len(t, got[0].Conditions, 1)
			assert.Equal(t, tt.path, got[0].Conditions[0].Path)
			assert.Equal(t, []any{tt.value}, got[0].Conditions[0].Values)
			on[tt.alias] = map[string]any{"project_id": tt.value}
			all, err := ParseWebhookSubscriptions(on, "first-class")
			require.NoError(t, err)
			catalog, err := WebhookCatalog()
			require.NoError(t, err)
			require.Len(t, all, 1)
			assert.ElementsMatch(t, catalog[tt.event].Actions, all[0].Types, "omitting types selects all cataloged activity types")
			assert.Equal(t, got[0].Conditions, all[0].Conditions)
			require.NoError(t, ValidateEventTypes(map[string]any{"on": on}))
			assert.Equal(t, []string{tt.event}, NormalizeGitHubAppWebhookEvents(on))

			for _, shorthand := range []any{tt.alias, []any{"push", tt.alias}} {
				_, err = ParseWebhookSubscriptions(shorthand, "first-class")
				require.ErrorContains(t, err, "requires mapping syntax")
			}
		})
	}
}

func TestFirstClassWebhookValidation(t *testing.T) {
	for alias := range webhookTriggerAliases {
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
					if config, ok := tt.config.(map[string]any); ok {
						config["project_id"] = "PVT_example"
					}
					_, err := ParseWebhookSubscriptions(map[string]any{
						alias: tt.config,
					}, "first-class")
					require.ErrorContains(t, err, tt.want)
					assert.Contains(t, err.Error(), "on."+alias)
				})
			}
		})
	}
}

func TestFirstClassWebhookPayloadRestrictions(t *testing.T) {
	for _, tt := range []struct {
		alias  string
		config map[string]any
		want   string
	}{
		{"projects-item", map[string]any{"types": []any{"moved"}}, "unsupported activity type"},
		{"projects-item", map[string]any{"filters": map[string]any{"labels": "bug"}}, "unsupported filter"},
		{"projects-item", map[string]any{
			"types": []any{"created"},
			"filters": map[string]any{
				"field": map[string]any{"to": map[string]any{"name": "Ready"}},
			},
		}, "requires one of"},
	} {
		t.Run(tt.alias+"/"+tt.want, func(t *testing.T) {
			tt.config["project_id"] = "PVT_example"
			_, err := ParseWebhookSubscriptions(map[string]any{
				tt.alias: tt.config,
			}, "first-class")
			require.ErrorContains(t, err, tt.want)
			assert.Contains(t, err.Error(), "on."+tt.alias)
		})
	}

}

func TestFirstClassWebhookRequiresProjectID(t *testing.T) {
	for name, config := range map[string]any{
		"null trigger":      nil,
		"empty trigger":     map[string]any{},
		"types only":        map[string]any{"types": []any{"edited"}},
		"nested ID only":    map[string]any{"filters": map[string]any{"project_id": "PVT_example"}},
		"null ID":           map[string]any{"project_id": nil},
		"empty ID":          map[string]any{"project_id": ""},
		"blank ID":          map[string]any{"project_id": " \t"},
		"numeric ID":        map[string]any{"project_id": 7},
		"empty IDs":         map[string]any{"project_id": []any{}},
		"blank list member": map[string]any{"project_id": []any{"PVT_example", " "}},
		"mixed list":        map[string]any{"project_id": []any{"PVT_example", true}},
		"nested ID alongside root": map[string]any{
			"project_id": "PVT_example", "filters": map[string]any{"project_id": "PVT_other"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseWebhookSubscriptions(map[string]any{"projects-item": config}, "project-agent")
			require.ErrorContains(t, err, "project_id")
		})
	}
}
