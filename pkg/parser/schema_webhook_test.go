package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebhookSchema(t *testing.T) {
	for name, test := range map[string]struct {
		on    map[string]any
		valid bool
	}{
		"subscription": {
			on: map[string]any{
				"webhook": map[string]any{
					"projects_v2_item": map[string]any{
						"types":   []any{"edited"},
						"filters": map[string]any{"project-number": 7},
					},
				},
				"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}},
			},
			valid: true,
		},
		"missing auth": {
			on: map[string]any{"webhook": map[string]any{"projects_v2_item": map[string]any{}}},
		},
		"auth without subscription": {
			on: map[string]any{"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}}},
		},
		"unknown event config property": {
			on: map[string]any{
				"webhook":      map[string]any{"projects_v2_item": map[string]any{"typo": true}},
				"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}},
			},
		},
		"empty actors": {
			on: map[string]any{
				"webhook":      map[string]any{"projects_v2_item": map[string]any{}},
				"webhook-auth": map[string]any{"actors": []any{}},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": test.on}, "webhook.md")
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestFirstClassWebhookSchema(t *testing.T) {
	for _, alias := range []string{
		"projects", "projects-item", "projects-status-update",
		"code-scanning-alert", "dependabot-alert", "secret-scanning-alert",
		"secret-scanning-alert-location", "secret-scanning-scan",
	} {
		t.Run(alias, func(t *testing.T) {
			for name, test := range map[string]struct {
				config any
				valid  bool
			}{
				"null":        {nil, true},
				"empty":       {map[string]any{}, true},
				"types":       {map[string]any{"types": []any{"created"}}, true},
				"filters":     {map[string]any{"filters": map[string]any{"sender": "octocat"}}, true},
				"scalar":      {true, false},
				"unknown":     {map[string]any{"typo": true}, false},
				"empty types": {map[string]any{"types": []any{}}, false},
			} {
				t.Run(name, func(t *testing.T) {
					on := map[string]any{
						alias:          test.config,
						"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}},
					}
					err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "first-class.md")
					if test.valid {
						require.NoError(t, err)
					} else {
						require.Error(t, err)
					}
				})
			}
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": map[string]any{alias: map[string]any{}},
			}, "first-class.md")
			require.Error(t, err, "first-class triggers require relay authentication")
		})
	}
}
