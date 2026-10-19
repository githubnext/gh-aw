//go:build !integration

package workflow

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayCommands(t *testing.T) {
	for _, trigger := range []string{
		"slash-command:",
		"slash-command: review",
		`slash-command: {name: [review, "fix*", "*"], events: [issue_comment, pull_request_comment], strategy: centralized}`,
		`slash-command: {events: ["*"]}`,
		`label-command: {}`,
		`label-command: {name: review, names: [review, deploy], events: [issues, pull_request], remove_label: true, strategy: decentralized}`,
	} {
		t.Run(trigger, func(t *testing.T) {
			compiler := newWebhookCompiler()
			data, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n  gateway-dispatch:\n    orgs: [Github]\n    repos: [github/repo]\n    "+trigger+"\n"), "gateway-review.md")
			require.NoError(t, err)
			require.Len(t, data.WebhookSubscriptions, 1)
			subscription := data.WebhookSubscriptions[0]
			require.NotNil(t, subscription.CommandFilter)
			assert.NotEmpty(t, subscription.CommandFilter.Names)
			assert.NotEmpty(t, subscription.CommandFilter.Events)
			assert.Equal(t, []string{"github"}, subscription.Orgs)
			assert.Equal(t, []string{"github/repo"}, subscription.Repos)
			assert.Empty(t, subscription.Conditions)
			assert.Empty(t, data.Command)
			assert.Empty(t, data.LabelCommand)
			output, err := compiler.CompileToYAML(data, "gateway-review.md")
			require.NoError(t, err)
			var compiled map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(output), &compiled))
			assert.Equal(t, map[string]any{"repository_dispatch": map[string]any{"types": []any{WebhookDispatchEventType("gateway-review")}}}, compiled["on"])
			assert.NotContains(t, output, "check_command_position.cjs")
			assert.NotContains(t, output, "route_slash_command.cjs")
		})
	}
}

func TestGatewayCommandConfiguration(t *testing.T) {
	var on map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(`gateway-dispatch:
  slash-command:
    name: [review, "fix*"]
    events: [pull_request_comment]
    strategy: centralized
  label-command:
    name: review
    names: [deploy, review]
    events: [discussion]
`), &on))
	subscriptions, err := ParseWebhookSubscriptions(on, "gateway-review")
	require.NoError(t, err)
	require.Len(t, subscriptions, 2)
	assert.Equal(t, []string{"issue_comment"}, subscriptions[0].NativeEvents)
	assert.Equal(t, []string{"created", "edited"}, subscriptions[0].Types)
	assert.Equal(t, &WebhookCommandFilter{Policy: "slash-command/v1", Names: []string{"fix*", "review"}, Events: []string{"pull_request_comment"}, Strategy: "centralized"}, subscriptions[0].CommandFilter)
	assert.Equal(t, []string{"deploy", "review"}, subscriptions[1].CommandFilter.Names)
	assert.True(t, *subscriptions[1].CommandFilter.RemoveLabel)
	assert.Equal(t, []string{"discussion", "issue_comment"}, NormalizeGitHubAppWebhookEvents(on))
	assert.Equal(t, map[string]string{"pull_requests": "read", "discussions": "write"}, ComputeGitHubAppManifestWebhookPermissions(on))
	subscriptions, err = ParseWebhookSubscriptions(map[string]any{"gateway-dispatch": map[string]any{"slash-command": nil}}, "my-workflow")
	require.NoError(t, err)
	assert.Equal(t, []string{"my-workflow"}, subscriptions[0].CommandFilter.Names)
	assert.Equal(t, []string{"created", "edited", "opened", "reopened"}, subscriptions[0].Types)
}

func TestGatewayCommandsRejectInvalidConfiguration(t *testing.T) {
	for _, trigger := range []string{
		"slash-command: true", "slash-command: []", "slash-command: {name: []}",
		`slash-command: {name: "/review"}`, `slash-command: {name: "fi*x"}`,
		`slash-command: {name: "${{ github.actor }}"}`, `slash-command: {events: [push]}`,
		`slash-command: {events: ["*", push]}`, `slash-command: {events: []}`,
		`slash-command: {names: [review]}`, `slash-command: {placeholder: context}`,
		`slash-command: {remove_label: true}`, `slash-command: {strategy: decentralized}`,
		"label-command:", `label-command: {name: []}`, `label-command: {names: []}`,
		`label-command: {events: issue_comment}`, `label-command: {remove_label: "true"}`,
		`label-command: {remove_label: false}`,
		`label-command: {strategy: centralized}`, `label-command: {unknown: true}`,
	} {
		t.Run(trigger, func(t *testing.T) {
			_, err := newWebhookCompiler().ParseWorkflowString(webhookWorkflow("on:\n  gateway-dispatch:\n    "+trigger+"\n"), "commands.md")
			require.Error(t, err)
			var on map[string]any
			require.NoError(t, yaml.Unmarshal([]byte("gateway-dispatch:\n  "+trigger+"\n"), &on))
			_, err = ParseWebhookSubscriptions(on, "commands")
			require.Error(t, err)
		})
	}
	for _, native := range []string{"slash_command: review", "label_command: review", "command: review"} {
		var on map[string]any
		require.NoError(t, yaml.Unmarshal([]byte(native+"\ngateway-dispatch:\n  slash-command: review\n"), &on))
		_, err := ParseWebhookSubscriptions(on, "commands")
		require.ErrorContains(t, err, "on.gateway-dispatch is exclusive")
	}
}

func TestGatewayEventsPrompt(t *testing.T) {
	compiler := newWebhookCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n  gateway-dispatch:\n    slash-command: review\n")+`
All notifications: ${{ toJSON(gateway.events) }}
First URL: ${{ gateway.events[0].payload.url }}
Matching labels: ${{ toJSON(gateway.events.*.payload.filters.label) }}
`, "gateway-events.md")
	require.NoError(t, err)
	output, err := compiler.CompileToYAML(data, "gateway-events.md")
	require.NoError(t, err)
	assert.Contains(t, output, "toJSON(fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').webhook.deliveries)")
	assert.Contains(t, output, "fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').webhook.deliveries[0].payload.url")
	assert.Contains(t, output, "toJSON(fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').webhook.deliveries.*.payload.filters.label)")
	assert.NotContains(t, output, "${{ gateway.events")
}
