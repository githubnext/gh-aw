//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/actions/setup"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayGitHubNotifications(t *testing.T) {
	catalog, err := WebhookCatalog()
	require.NoError(t, err)
	var shared struct {
		Events map[string]struct {
			NativeEvent string `json:"native_event"`
		} `json:"events"`
	}
	require.NoError(t, json.Unmarshal(setup.GitHubNotificationWebhooks, &shared))
	require.Len(t, shared.Events, len(githubNotificationTriggers))
	for trigger, event := range githubNotificationTriggers {
		t.Run(trigger, func(t *testing.T) {
			definition, exists := catalog[event]
			require.True(t, exists)
			require.NotEmpty(t, shared.Events[event].NativeEvent)
			on := map[string]any{"gateway-dispatch": map[string]any{trigger: nil, "repos": []any{"Org/*"}, "orgs": []any{"OR*"}}}
			require.NoError(t, parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": on}, "notify.md"))
			subscriptions, err := ParseWebhookSubscriptions(on, "notify")
			require.NoError(t, err)
			require.Len(t, subscriptions, 1)
			subscription := subscriptions[0]
			assert.Equal(t, event, subscription.Event)
			assert.ElementsMatch(t, definition.Actions, subscription.Types)
			assert.Equal(t, []string{shared.Events[event].NativeEvent}, subscription.NativeEvents)
			assert.Equal(t, []string{"org/*"}, subscription.Repos)
			assert.Equal(t, []string{"or*"}, subscription.Orgs)
			assert.Empty(t, subscription.Conditions)
			assert.Nil(t, subscription.CommandFilter)
			assert.Nil(t, subscription.ActivityNormalization)
			assert.Equal(t, []string{shared.Events[event].NativeEvent}, NormalizeGitHubAppWebhookEvents(on))
			key, supported := GitHubAppManifestPermissionKey(githubNotificationPermission(event))
			require.True(t, supported)
			assert.Equal(t, map[string]string{key: "read"}, ComputeGitHubAppManifestWebhookPermissions(on))
			for _, action := range definition.Actions {
				config := map[string]any{"types": []any{action}, "filters": map[string]any{"sender": "reviewer"}}
				on["gateway-dispatch"].(map[string]any)[trigger] = config
				one, err := ParseWebhookSubscriptions(on, "notify")
				require.NoError(t, err, action)
				assert.Equal(t, []string{action}, one[0].Types)
				assert.Equal(t, []string{"sender", "login"}, one[0].Conditions[0].Path)
			}
		})
	}
}

func TestCompileGatewayGitHubNotifications(t *testing.T) {
	for trigger := range githubNotificationTriggers {
		t.Run(trigger, func(t *testing.T) {
			compiler := newWebhookCompiler()
			source := webhookWorkflow("on:\n  gateway-dispatch:\n    " + trigger + ":\n")
			data, err := compiler.ParseWorkflowString(source, "notify.md")
			require.NoError(t, err)
			compiled, err := compiler.CompileToYAML(data, "notify.md")
			require.NoError(t, err)
			var output map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(compiled), &output))
			assert.Equal(t, map[string]any{"repository_dispatch": map[string]any{"types": []any{WebhookDispatchEventType("notify")}}}, output["on"])
			assert.NotContains(t, compiled, "check_command_position.cjs")
			assert.Empty(t, data.Command)
			assert.Empty(t, data.LabelCommand)
		})
	}
}

func TestGatewayGitHubNotificationFilterBoundaries(t *testing.T) {
	for _, config := range []any{
		true, "opened", []any{"opened"},
		map[string]any{"types": []any{"not-an-action"}},
		map[string]any{"types": "opened"},
		map[string]any{"types": nil},
		map[string]any{"types": []any{}},
		map[string]any{"branches": []any{"main"}},
		map[string]any{"filters": map[string]any{"body": "private"}},
		map[string]any{"filters": map[string]any{"draft": "false"}},
		map[string]any{"types": []any{"opened"}, "filters": map[string]any{"label": "ready"}},
	} {
		_, err := ParseWebhookSubscriptions(map[string]any{"gateway-dispatch": map[string]any{"pull-request": config}}, "invalid")
		require.Error(t, err, "%#v", config)
	}
	on := map[string]any{"gateway-dispatch": map[string]any{"pull-request": map[string]any{
		"types": []any{"closed"}, "filters": map[string]any{"merged": true, "base": "main", "labels": []any{"ready"}},
	}}}
	subscriptions, err := ParseWebhookSubscriptions(on, "merge")
	require.NoError(t, err)
	require.Len(t, subscriptions[0].Conditions, 3)
	assert.Equal(t, []string{"pull_request", "base", "ref"}, subscriptions[0].Conditions[0].Path)
	assert.Equal(t, []string{"pull_request", "labels", "*", "name"}, subscriptions[0].Conditions[1].Path)
	assert.Equal(t, "boolean", subscriptions[0].Conditions[2].ValueType)
	assert.Equal(t, []any{true}, subscriptions[0].Conditions[2].Values)
}

func TestGatewayNotificationsDoNotDowngradeCommandPermissions(t *testing.T) {
	on := map[string]any{"gateway-dispatch": map[string]any{
		"issue": nil, "pull-request": nil, "discussion": nil,
		"label-command": map[string]any{"name": "review"},
	}}
	assert.Equal(t, map[string]string{"issues": "write", "pull_requests": "write", "discussions": "write"}, ComputeGitHubAppManifestWebhookPermissions(on))
	_, err := NewCompiler().ParseWorkflowString(webhookWorkflow("on:\n  discussion: {types: [created]}\n"), "native.md")
	require.NoError(t, err, "native discussion workflows must remain independent")
}
