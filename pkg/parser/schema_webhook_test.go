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
