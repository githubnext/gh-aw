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
		"subscription without separate auth": {
			on: map[string]any{
				"projects-item": map[string]any{
					"types":   []any{"edited"},
					"filters": map[string]any{"project_number": 7},
				},
			},
			valid: true,
		},
		"removed general registration": {
			on: map[string]any{"webhook": map[string]any{"projects_v2_item": map[string]any{}}},
		},
		"removed auth": {
			on: map[string]any{"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}}},
		},
		"unknown event config property": {
			on: map[string]any{
				"projects-item": map[string]any{"typo": true},
			},
		},
		"removed auth with valid trigger": {
			on: map[string]any{
				"projects-item": nil,
				"webhook-auth":  map[string]any{"actors": []any{"cao[bot]"}},
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
	for _, alias := range []string{"projects-item"} {
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
						alias: test.config,
					}
					err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "first-class.md")
					if test.valid {
						require.NoError(t, err)
					} else {
						require.Error(t, err)
					}
				})
			}
		})
	}
}

func TestSecurityAlertSchema(t *testing.T) {
	for name, test := range map[string]struct {
		config any
		valid  bool
	}{
		"code-scanning":   {map[string]any{"type": "code-scanning"}, true},
		"dependabot":      {map[string]any{"type": "dependabot"}, true},
		"secret-scanning": {map[string]any{"type": "secret-scanning"}, true},
		"with filters": {map[string]any{
			"type": "dependabot", "types": []any{"created"}, "filters": map[string]any{"severity": "critical"},
		}, true},
		"null":         {nil, false},
		"missing type": {map[string]any{}, false},
		"unknown type": {map[string]any{"type": "security-advisory"}, false},
		"list type":    {map[string]any{"type": []any{"code-scanning", "dependabot"}}, false},
		"null type":    {map[string]any{"type": nil}, false},
		"scan":         {map[string]any{"type": "secret-scanning-scan"}, false},
		"location":     {map[string]any{"type": "secret-scanning-alert-location"}, false},
		"empty types":  {map[string]any{"type": "dependabot", "types": []any{}}, false},
		"unknown key":  {map[string]any{"type": "dependabot", "typo": true}, false},
	} {
		t.Run(name, func(t *testing.T) {
			on := map[string]any{
				"security-alert": test.config,
			}

			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "security-alert.md")
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestRemovedFirstClassWebhookTriggers(t *testing.T) {
	for _, trigger := range []string{
		"projects", "projects-status-update", "code-scanning-alert", "dependabot-alert",
		"secret-scanning-alert", "secret-scanning-alert-location", "secret-scanning-scan",
	} {
		t.Run(trigger, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": map[string]any{trigger: nil},
			}, "webhook.md")
			require.Error(t, err)
		})
	}
}
