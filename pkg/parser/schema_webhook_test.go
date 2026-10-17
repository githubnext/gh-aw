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
					"project_id": "PVT_example",
					"types":      []any{"edited"},
					"filters":    map[string]any{"project_number": 7},
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
				"projects-item": map[string]any{"project_id": "PVT_example", "typo": true},
			},
		},
		"removed auth with valid trigger": {
			on: map[string]any{
				"projects-item": map[string]any{"project_id": "PVT_example"},
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
				"null":               {nil, false},
				"empty":              {map[string]any{}, false},
				"types without ID":   {map[string]any{"types": []any{"created"}}, false},
				"filters without ID": {map[string]any{"filters": map[string]any{"sender": "octocat"}}, false},
				"project ID":         {map[string]any{"project_id": "PVT_example"}, true},
				"project IDs":        {map[string]any{"project_id": []any{"PVT_a", "PVT_b"}}, true},
				"with types":         {map[string]any{"project_id": "PVT_example", "types": []any{"edited"}}, true},
				"with filters":       {map[string]any{"project_id": "PVT_example", "filters": map[string]any{"sender": "octocat"}}, true},
				"nested ID":          {map[string]any{"filters": map[string]any{"project_id": "PVT_example"}}, false},
				"nested ID alongside root": {map[string]any{
					"project_id": "PVT_example", "filters": map[string]any{"project_id": "PVT_other"},
				}, false},
				"null ID":           {map[string]any{"project_id": nil}, false},
				"empty ID":          {map[string]any{"project_id": ""}, false},
				"blank ID":          {map[string]any{"project_id": " \t"}, false},
				"numeric ID":        {map[string]any{"project_id": 7}, false},
				"empty IDs":         {map[string]any{"project_id": []any{}}, false},
				"mixed IDs":         {map[string]any{"project_id": []any{"PVT_example", 7}}, false},
				"blank list member": {map[string]any{"project_id": []any{"PVT_example", " "}}, false},
				"scalar":            {true, false},
				"unknown":           {map[string]any{"project_id": "PVT_example", "typo": true}, false},
				"empty types":       {map[string]any{"project_id": "PVT_example", "types": []any{}}, false},
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
		"code-scanning":    {map[string]any{"sources": "code-scanning"}, true},
		"dependabot":       {map[string]any{"sources": "dependabot"}, true},
		"secret-scanning":  {map[string]any{"sources": "secret-scanning"}, true},
		"multiple sources": {map[string]any{"sources": []any{"code-scanning", "dependabot", "secret-scanning"}}, true},
		"with severity": {map[string]any{
			"sources": []any{"dependabot"}, "types": []any{"created"}, "severity": []any{"high", "critical"},
		}, true},
		"with filters": {map[string]any{
			"sources": "dependabot", "filters": map[string]any{"sender": "octocat"},
		}, true},
		"null":             {nil, false},
		"missing sources":  {map[string]any{}, false},
		"unknown source":   {map[string]any{"sources": "security-advisory"}, false},
		"empty sources":    {map[string]any{"sources": []any{}}, false},
		"mixed sources":    {map[string]any{"sources": []any{"dependabot", true}}, false},
		"null sources":     {map[string]any{"sources": nil}, false},
		"removed type":     {map[string]any{"sources": "dependabot", "type": "dependabot"}, false},
		"scan":             {map[string]any{"sources": "secret-scanning-scan"}, false},
		"location":         {map[string]any{"sources": "secret-scanning-alert-location"}, false},
		"empty types":      {map[string]any{"sources": "dependabot", "types": []any{}}, false},
		"unknown key":      {map[string]any{"sources": "dependabot", "typo": true}, false},
		"empty severity":   {map[string]any{"sources": "dependabot", "severity": ""}, false},
		"blank severity":   {map[string]any{"sources": "dependabot", "severity": " "}, false},
		"null severity":    {map[string]any{"sources": "dependabot", "severity": nil}, false},
		"empty severities": {map[string]any{"sources": "dependabot", "severity": []any{}}, false},
		"nested severity": {map[string]any{
			"sources": "dependabot", "filters": map[string]any{"severity": "high"},
		}, false},
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

func TestWebhookStatusTransitionsSchema(t *testing.T) {
	for name, test := range map[string]struct {
		transitions any
		valid       bool
	}{
		"one pair": {[]any{map[string]any{"from": "Todo", "to": "Ready"}}, true},
		"two pairs": {[]any{
			map[string]any{"from": "Todo", "to": "Ready"},
			map[string]any{"from": "In progress", "to": "Done"},
		}, true},
		"null":         {nil, false},
		"empty":        {[]any{}, false},
		"object":       {map[string]any{"from": "Todo", "to": "Ready"}, false},
		"scalar entry": {[]any{"Todo"}, false},
		"missing from": {[]any{map[string]any{"to": "Ready"}}, false},
		"missing to":   {[]any{map[string]any{"from": "Todo"}}, false},
		"empty from":   {[]any{map[string]any{"from": "", "to": "Ready"}}, false},
		"blank to":     {[]any{map[string]any{"from": "Todo", "to": " "}}, false},
		"non-string":   {[]any{map[string]any{"from": "Todo", "to": 7}}, false},
		"extra key":    {[]any{map[string]any{"from": "Todo", "to": "Ready", "typo": true}}, false},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{
				"on": map[string]any{"projects-item": map[string]any{
					"project_id": "PVT_example", "status-transitions": test.transitions,
				}},
			}, "status-transitions.md")
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
