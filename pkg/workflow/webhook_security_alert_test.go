package workflow

import (
	"encoding/json"
	"fmt"
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
	}
	require.Len(t, securityAlertEvents, len(tests))
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			config := map[string]any{
				"sources": []any{tt.kind},
				"types":   []any{"created"},
			}
			if tt.filter == "severity" {
				config["severity"] = tt.value
			} else {
				config["filters"] = map[string]any{tt.filter: tt.value}
			}
			on := map[string]any{
				"security-alert": config,
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
			after, err := json.Marshal(on)
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after))
			require.NoError(t, ValidateEventTypes(map[string]any{"on": on}))
			assert.Equal(t, []string{tt.event}, NormalizeGitHubAppWebhookEvents(on))

			on["security-alert"] = map[string]any{"sources": tt.kind}
			all, err := ParseWebhookSubscriptions(on, "security-agent")
			require.NoError(t, err)
			catalog, err := WebhookCatalog()
			require.NoError(t, err)
			require.Len(t, all, 1)
			assert.Equal(t, tt.event, all[0].Event)
			assert.ElementsMatch(t, catalog[tt.event].Actions, all[0].Types)
			assert.Empty(t, all[0].Conditions)
			assert.True(t, matchWebhookContract(all[0], map[string]any{"action": "created"}))
			assert.False(t, matchWebhookContract(all[0], map[string]any{"action": "completed"}))

		})
	}
}

func TestSecurityAlertRejectsInvalidConfiguration(t *testing.T) {
	for name, tt := range map[string]struct {
		config any
		want   string
	}{
		"null":                     {nil, "must be an object"},
		"scalar":                   {"dependabot", "must be an object"},
		"null sources":             {map[string]any{"sources": nil}, "on.security-alert.sources"},
		"empty sources":            {map[string]any{"sources": []any{}}, "on.security-alert.sources"},
		"blank source":             {map[string]any{"sources": " "}, "on.security-alert.sources"},
		"unknown source":           {map[string]any{"sources": "security_advisory"}, "on.security-alert.sources"},
		"known and unknown source": {map[string]any{"sources": []any{"dependabot", "secret-scanning-scan"}}, "on.security-alert.sources"},
		"removed secret source":    {map[string]any{"sources": "secret-scanning"}, "must select only code-scanning or dependabot"},
		"removed secret list":      {map[string]any{"sources": []any{"secret-scanning"}}, "must select only code-scanning or dependabot"},
		"mixed removed source":     {map[string]any{"sources": []any{"code-scanning", "dependabot", "secret-scanning"}}, "must select only code-scanning or dependabot"},
		"mixed sources":            {map[string]any{"sources": []any{"dependabot", true}}, "on.security-alert.sources"},
		"scan event":               {map[string]any{"sources": "secret-scanning-scan"}, "on.security-alert.sources"},
		"location event": {map[string]any{
			"sources": "secret-scanning-alert-location",
		}, "on.security-alert.sources"},
		"removed type":                   {map[string]any{"sources": "dependabot", "type": "dependabot"}, "unsupported on.security-alert property"},
		"unsupported property":           {map[string]any{"sources": "dependabot", "typo": true}, "unsupported on.security-alert property"},
		"wrong action":                   {map[string]any{"sources": "dependabot", "types": []any{"completed"}}, "unsupported activity type"},
		"wrong source action":            {map[string]any{"sources": "code-scanning", "types": []any{"publicly_leaked"}}, "unsupported activity type"},
		"empty actions":                  {map[string]any{"sources": "dependabot", "types": []any{}}, "non-empty list"},
		"unsupported filter":             {map[string]any{"sources": "dependabot", "filters": map[string]any{"not-a-filter": true}}, "unsupported filter"},
		"nested severity":                {map[string]any{"sources": "dependabot", "filters": map[string]any{"severity": "high"}}, "use on.security-alert.severity beside types"},
		"wrong severity type":            {map[string]any{"sources": "code-scanning", "severity": true}, "non-empty string"},
		"empty severity":                 {map[string]any{"sources": "dependabot", "severity": ""}, "non-empty string"},
		"invalid severity":               {map[string]any{"sources": "dependabot", "severity": "error"}, "not one of"},
		"incompatible source severities": {map[string]any{"sources": []any{"code-scanning", "dependabot"}, "severity": "critical"}, "not one of"},
		"secret material":                {map[string]any{"sources": "dependabot", "filters": map[string]any{"secret": "not-a-secret"}}, "unsupported filter"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseWebhookSubscriptions(map[string]any{
				"security-alert": tt.config,
			}, "security-agent")
			require.ErrorContains(t, err, tt.want)
			assert.Contains(t, err.Error(), "on.security-alert")
		})
	}

	for _, shorthand := range []any{"security-alert", []any{"push", "security-alert"}, "security-alerts", []any{"push", "security-alerts"}} {
		_, err := ParseWebhookSubscriptions(shorthand, "security-agent")
		require.ErrorContains(t, err, "requires mapping syntax")
	}
}

func TestSecurityAlertRepositorySelectors(t *testing.T) {
	for _, name := range securityAlertTriggerNames {
		t.Run(name, func(t *testing.T) {
			config := map[string]any{"repos": []any{"Foo/*", "bar/baz", "BAR/BAZ", "foo/*"}}
			on := map[string]any{name: config}
			before, err := json.Marshal(on)
			require.NoError(t, err)
			subscriptions, err := ParseWebhookSubscriptions(on, "multi-repo")
			require.NoError(t, err)
			require.Len(t, subscriptions, 2, "omitted sources selects both supported alert sources")
			for _, subscription := range subscriptions {
				assert.Equal(t, []string{"bar/baz", "foo/*"}, subscription.Repos)
				assert.Equal(t, "${{ github.repository }}", subscription.Dispatch.Repository)
				assert.Empty(t, subscription.Conditions, "repository selectors are source metadata, not payload equality predicates")
			}
			assert.Equal(t, []string{"code_scanning_alert", "dependabot_alert"}, NormalizeGitHubAppWebhookEvents(on))
			require.NoError(t, ValidateEventTypes(map[string]any{"on": on}))
			after, err := json.Marshal(on)
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after), "parsing must preserve source frontmatter")
		})
	}
}

func TestSecurityAlertRejectsInvalidRepositorySelectors(t *testing.T) {
	for _, name := range securityAlertTriggerNames {
		for _, repos := range []any{
			nil, "bar/baz", true, []any{}, []string{}, []any{7}, []any{"bar/baz", true},
			[]any{""}, []any{" "}, []any{"*"}, []any{"*/*"}, []any{"*/baz"},
			[]any{"bar/**"}, []any{"bar/ba*"}, []any{"bar/?az"}, []any{"bar/[baz]"},
			[]any{"bar"}, []any{"/baz"}, []any{"bar/"}, []any{"bar/baz/other"},
			[]any{"https://github.com/bar/baz"}, []any{"${{ github.repository }}"},
			[]any{"bar/baz "}, []any{" bar/baz"}, []any{"bar/baz\n"}, []any{"bar/."}, []any{"bar/.."},
		} {
			t.Run(name+"/"+fmt.Sprint(repos), func(t *testing.T) {
				_, err := ParseWebhookSubscriptions(map[string]any{name: map[string]any{"repos": repos}}, "invalid-repos")
				require.ErrorContains(t, err, "on."+name+".repos")
				assert.Contains(t, err.Error(), "owner/repo or owner/*")
			})
		}
	}
}

func TestSecurityAlertAliasesAndOmittedSelectors(t *testing.T) {
	_, err := ParseWebhookSubscriptions(map[string]any{"security-alert": map[string]any{}, "security-alerts": map[string]any{}}, "duplicate")
	require.ErrorContains(t, err, "configure only one")
	for _, name := range securityAlertTriggerNames {
		subscriptions, err := ParseWebhookSubscriptions(map[string]any{name: map[string]any{}}, "defaults")
		require.NoError(t, err)
		require.Len(t, subscriptions, 2)
		for _, subscription := range subscriptions {
			assert.Nil(t, subscription.Repos, "omission does not introduce a new selector or implicit wildcard")
			encoded, err := json.Marshal(subscription)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), `"repos"`)
		}
		subscriptions, err = ParseWebhookSubscriptions(map[string]any{name: map[string]any{"sources": "dependabot", "repos": []string{"bar/baz"}}}, "explicit-source")
		require.NoError(t, err)
		require.Len(t, subscriptions, 1)
		assert.Equal(t, "dependabot_alert", subscriptions[0].Event)
		assert.Equal(t, []string{"bar/baz"}, subscriptions[0].Repos)
		_, err = ParseWebhookSubscriptions(map[string]any{name: map[string]any{"repos": []string{"bar/baz"}, "sources": "secret-scanning"}}, "secret")
		require.ErrorContains(t, err, "on."+name+".sources")
	}
}

func TestSecurityAlertMultipleSources(t *testing.T) {
	on := map[string]any{"security-alert": map[string]any{
		"sources": []any{"dependabot", "code-scanning"},
		"types":   []any{"created"},
	}}
	subscriptions, err := ParseWebhookSubscriptions(on, "security-agent")
	require.NoError(t, err)
	require.Len(t, subscriptions, 2)
	expected := []string{"code_scanning_alert", "dependabot_alert"}
	for i, subscription := range subscriptions {
		assert.Equal(t, expected[i], subscription.Event)
		assert.Equal(t, []string{"created"}, subscription.Types)
		assert.Equal(t, subscriptions[0].Dispatch.EventType, subscription.Dispatch.EventType)
	}
	assert.Equal(t, expected, NormalizeGitHubAppWebhookEvents(on))
}

func TestSecurityAlertSeverityAlternatives(t *testing.T) {
	subscriptions, err := ParseWebhookSubscriptions(map[string]any{"security-alert": map[string]any{
		"sources": []any{"dependabot"}, "types": []any{"created"}, "severity": []any{"high", "critical"},
	}}, "security-agent")
	require.NoError(t, err)
	require.Len(t, subscriptions, 1)
	require.Len(t, subscriptions[0].Conditions, 1)
	assert.Equal(t, []any{"critical", "high"}, subscriptions[0].Conditions[0].Values)
	for _, tt := range []struct {
		severity string
		match    bool
	}{{"critical", true}, {"high", true}, {"low", false}, {"HIGH", false}} {
		payload := map[string]any{"action": "created", "alert": map[string]any{
			"security_advisory": map[string]any{"severity": tt.severity},
		}}
		assert.Equal(t, tt.match, matchWebhookContract(subscriptions[0], payload))
	}
}
