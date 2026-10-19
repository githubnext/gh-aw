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
			on:    map[string]any{"gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example", "types": []any{"edited"}, "filters": map[string]any{"project_number": 7}}}},
			valid: true,
		},
		"removed general registration": {
			on: map[string]any{"webhook": map[string]any{"projects_v2_item": map[string]any{}}},
		},
		"removed auth": {
			on: map[string]any{"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}}},
		},
		"unknown event config property": {
			on: map[string]any{"gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example", "typo": true}}},
		},
		"removed auth with valid trigger": {
			on: map[string]any{"webhook-auth": map[string]any{"actors": []any{"cao[bot]"}}, "gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example"}}},
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
	for _, alias := range []string{"projects-items"} {
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
					on := map[string]any{"gateway-dispatch": map[string]any{alias: test.config}}
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
		"secret-scanning":  {map[string]any{"sources": "secret-scanning"}, false},
		"secret list":      {map[string]any{"sources": []any{"secret-scanning"}}, false},
		"mixed secret":     {map[string]any{"sources": []any{"code-scanning", "dependabot", "secret-scanning"}}, false},
		"multiple sources": {map[string]any{"sources": []any{"code-scanning", "dependabot"}}, true},
		"with severity": {map[string]any{
			"sources": []any{"dependabot"}, "types": []any{"created"}, "severity": []any{"high", "critical"},
		}, true},
		"with filters": {map[string]any{
			"sources": "dependabot", "filters": map[string]any{"sender": "octocat"},
		}, true},
		"null":             {nil, false},
		"default sources":  {map[string]any{}, true},
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
		for _, alias := range []string{"security-alerts"} {
			t.Run(alias+"/"+name, func(t *testing.T) {
				err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{alias: test.config}}}, "security-alert.md")
				if test.valid {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			})
		}
	}
}

func TestSecurityAlertRepositorySchema(t *testing.T) {
	for _, alias := range []string{"security-alerts"} {
		for name, test := range map[string]struct {
			repos any
			valid bool
		}{
			"exact and wildcard": {[]any{"bar/baz", "foo/*"}, true},
			"case insensitive":   {[]any{"Bar/Baz", "FOO/*"}, true},
			"dot and underscore": {[]any{"bar/repo.name_x"}, true},
			"null":               {nil, false},
			"scalar":             {"bar/baz", false},
			"empty":              {[]any{}, false},
			"blank":              {[]any{""}, false},
			"mixed":              {[]any{"bar/baz", false}, false},
			"wildcard owner":     {[]any{"*/baz"}, false},
			"global wildcard":    {[]any{"*"}, false},
			"prefix wildcard":    {[]any{"foo/prefix*"}, false},
			"recursive wildcard": {[]any{"foo/**"}, false},
			"expression":         {[]any{"${{ github.repository }}"}, false},
			"url":                {[]any{"https://github.com/bar/baz"}, false},
			"dot":                {[]any{"foo/."}, false},
			"dotdot":             {[]any{"foo/.."}, false},
			"duplicate":          {[]any{"foo/*", "foo/*"}, false},
			"whitespace":         {[]any{" bar/baz"}, false},
			"newline":            {[]any{"bar/baz\n"}, false},
		} {
			t.Run(alias+"/"+name, func(t *testing.T) {
				err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{"repos": test.repos, alias: map[string]any{}}}}, "security-repos.md")
				if test.valid {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			})
		}
	}
}

func TestGatewayOrganizationSchema(t *testing.T) {
	for _, orgs := range []any{[]any{"GitHub", "my-*", "*-labs", "a*b"}, []any{"*"}} {
		require.NoError(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{"orgs": orgs, "security-alerts": map[string]any{}}}}, "orgs.md"))
	}
	for _, orgs := range []any{nil, "github", []any{}, []any{true}, []any{""}, []any{"github/repo"}, []any{"github/*"}, []any{"foo?"}, []any{"foo.bar"}, []any{"foo_bar"}, []any{"-foo"}, []any{"foo-"}, []any{"github\n"}, []any{"${{ github.actor }}"}, []any{"github", "github"}} {
		require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": map[string]any{"orgs": orgs, "security-alerts": map[string]any{}}}}, "invalid-orgs.md"))
	}
	for _, on := range []any{
		"orgs",
		map[string]any{"orgs": []any{"github"}},
		map[string]any{"gateway-dispatch": map[string]any{"orgs": []any{"github"}}},
		map[string]any{"gateway-dispatch": map[string]any{"security-alerts": map[string]any{"orgs": []any{"github"}}}},
	} {
		require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "invalid-orgs-placement.md"))
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
				"on": map[string]any{"gateway-dispatch": map[string]any{"projects-items": map[string]any{"project_id": "PVT_example", "status-transitions": test.transitions}}},
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

func TestWebhookGatewayNamespaceSchema(t *testing.T) {
	for name, on := range map[string]any{
		"null":              map[string]any{"gateway": nil},
		"scalar":            map[string]any{"gateway": "projects-items"},
		"empty":             map[string]any{"gateway": map[string]any{}},
		"list":              map[string]any{"gateway": []any{"security-alerts"}},
		"unknown":           map[string]any{"gateway": map[string]any{"unknown": map[string]any{}}},
		"nested roles":      map[string]any{"gateway": map[string]any{"roles": "write", "security-alerts": map[string]any{}}},
		"singular project":  map[string]any{"gateway": map[string]any{"projects-item": map[string]any{"project_id": "PVT_example"}}},
		"singular security": map[string]any{"gateway": map[string]any{"security-alert": map[string]any{}}},
		"string shorthand":  "gateway",
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "gateway.md"))
		})
	}
	for _, name := range []string{"projects-item", "projects-items", "security-alert", "security-alerts"} {
		for _, on := range []any{name, map[string]any{name: map[string]any{}},
			map[string]any{"gateway": map[string]any{"security-alerts": map[string]any{}}, name: map[string]any{}}} {
			require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "legacy.md"))
		}
	}

}

func TestGatewayDispatchSchema(t *testing.T) {
	for trigger, config := range map[string]any{
		"projects-items":     map[string]any{"project_id": "PVT_example"},
		"security-alerts":    map[string]any{},
		"jira-issue":         map[string]any{"project": "APP"},
		"linear-issue":       map[string]any{"organization-id": "org"},
		"azure-devops-issue": map[string]any{"project_id": "project"},
		"webhooks":           []any{map[string]any{"type": "example.event"}},
	} {
		t.Run(trigger, func(t *testing.T) {
			on := map[string]any{"gateway-dispatch": map[string]any{trigger: config, "repos": []any{"owner/*"}}, "push": map[string]any{}}
			require.NoError(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "enabled.md"))
			for _, dispatch := range []string{"workflow_dispatch", "repository_dispatch"} {
				for _, value := range []any{nil, map[string]any{}, map[string]any{"types": []any{"manual"}}} {
					on[dispatch] = value
					require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "mixed-dispatch.md"))
				}
				delete(on, dispatch)
			}
			for _, flag := range []any{nil, false, true, "true", 1, map[string]any{}, []any{true}} {
				legacy := map[string]any{trigger: config, "webhook-gateway": flag}
				require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": legacy}, "legacy.md"))
			}
		})
	}
	for _, value := range []any{nil, true, false, "projects-items", []any{"security-alerts"}, map[string]any{}, map[string]any{"repos": []any{"owner/repo"}}, map[string]any{"typo": true}, map[string]any{"security-alerts": map[string]any{}, "roles": "write"}} {
		require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": value}}, "invalid-gateway.md"))
	}
	for _, on := range []any{"gateway-dispatch", []any{"push", "gateway-dispatch"}} {
		require.Error(t, ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "shorthand-gateway.md"))
	}
}
