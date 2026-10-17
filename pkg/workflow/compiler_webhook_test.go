//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func webhookWorkflow(on string) string {
	return "---\n" + on + `
permissions:
  contents: read
engine: copilot
---
# Webhook workflow

Inspect the inline webhook at github.event.client_payload.payload.
`
}

func TestCompileWebhookTriggers(t *testing.T) {
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  workflow_dispatch:
  push:
    branches: [main]
  webhook:
    projects_v2_item:
      types: [edited]
  webhook-auth:
    actors: ["cao[bot]"]
`), "webhook-test.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 1)
	assert.Equal(t, []string{"cao[bot]"}, data.WebhookActors)
	var triggers map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(data.On), &triggers))
	on := triggers["on"].(map[string]any)
	assert.Contains(t, on, "workflow_dispatch")
	assert.Contains(t, on, "push")
	assert.NotContains(t, on, "webhook")
	assert.NotContains(t, on, "webhook-auth")
	assert.Equal(t, []any{WebhookDispatchEventType("webhook-test")}, on["repository_dispatch"].(map[string]any)["types"])

	output, err := compiler.CompileToYAML(data, "webhook-test.md")
	require.NoError(t, err)
	var compiled map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(output), &compiled))
	jobs := compiled["jobs"].(map[string]any)
	for name, raw := range jobs {
		job := raw.(map[string]any)
		assert.Contains(t, job["if"], "github.actor == 'cao[bot]'", "unguarded job %s", name)
		assert.Contains(t, job["if"], "github.event.action != '"+WebhookDispatchEventType("webhook-test")+"'", "job %s must preserve native triggers", name)
	}
	activation := extractJobSection(output, "activation")
	assert.Contains(t, activation, "Validate webhook relay")
	assert.Contains(t, activation, "if: "+webhookTriggerExpression(data))
	assert.Contains(t, activation, "GH_AW_WEBHOOK_SUBSCRIPTIONS:")
	assert.Contains(t, activation, "GH_AW_WEBHOOK_ACTORS:")
	assert.Contains(t, activation, "GH_AW_WEBHOOK_ACTOR: ${{ github.actor }}")
	assert.Contains(t, activation, "validate_webhook.cjs")
	assert.NotContains(t, activation, "toJSON(github.event")
	require.Contains(t, activation, "Create prompt with built-in context")
	assert.Less(t, strings.Index(activation, "Validate webhook relay"), strings.Index(activation, "Create prompt with built-in context"))
	agent := extractJobSection(output, "agent")
	assert.Contains(t, agent, "Stage inline webhook event")
	assert.Contains(t, agent, `cp -- "$GITHUB_EVENT_PATH" /tmp/gh-aw/github-event.json`)
	assert.Contains(t, agent, `cp -- "$GITHUB_EVENT_PATH" "${RUNNER_TEMP}/gh-aw/github-event.json"`)
	assert.NotContains(t, activation, `cp -- "$GITHUB_EVENT_PATH"`)
	assert.Contains(t, activation, "Read the original repository_dispatch event from /tmp/gh-aw/github-event.json")
	assert.Contains(t, activation, "client_payload.payload")
	assert.Contains(t, activation, "GH_AW_WEBHOOK_TRIGGER:")
}

func TestWebhookTriggerLoweringPreservesSource(t *testing.T) {
	var frontmatter map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(`on:
  webhook:
    projects_v2_item: {}
  webhook-auth:
    actors: ["cao[bot]"]
`), &frontmatter))
	original, err := json.Marshal(frontmatter)
	require.NoError(t, err)
	data := &WorkflowData{WorkflowID: "webhook-test"}
	compiler := NewCompiler()
	lowered, err := compiler.processWebhookTriggers(frontmatter, data)
	require.NoError(t, err)
	after, err := json.Marshal(frontmatter)
	require.NoError(t, err)
	assert.Equal(t, string(original), string(after))
	assert.Contains(t, lowered["on"], "repository_dispatch")
	assert.NotContains(t, lowered["on"], "webhook")
}

func TestWebhookRejectsInvalidConfiguration(t *testing.T) {
	tests := map[string]string{
		"unknown event": `  webhook:
    not_a_github_event: {}`,
		"invalid action": `  webhook:
    projects_v2_item:
      types: [not_a_real_action]`,
		"unknown filter": `  webhook:
    projects_v2_item:
      filters:
        not-a-real-filter: test`,
		"wrong filter type": `  webhook:
    projects_v2_item:
      filters:
        item_database_id: "7"`,
		"unavailable filter": `  webhook:
    projects_v2_item:
      types: [created]
      filters:
        field:
          name: Status`,
		"scalar event": `  webhook:
    projects_v2_item: true`,
		"empty webhooks": `  webhook: {}`,
		"unrestricted dispatch": `  repository_dispatch:
  webhook:
    projects_v2_item: {}`,
		"wildcard dispatch": `  repository_dispatch:
    types: ["gh-aw-*"]
  webhook:
    projects_v2_item: {}`,
		"colliding dispatch": `  repository_dispatch:
    types: ["` + WebhookDispatchEventType("webhook-test") + `"]
  webhook:
    projects_v2_item: {}`,
	}
	for name, trigger := range tests {
		t.Run(name, func(t *testing.T) {
			compiler := NewCompiler()
			_, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n"+trigger+`
  webhook-auth:
    actors: ["cao[bot]"]
`), "webhook-test.md")
			require.Error(t, err)
		})
	}
}

func TestWebhookRequiresRelayAuthentication(t *testing.T) {
	for _, auth := range []string{"", "  webhook-auth: {}\n", "  webhook-auth:\n    actors: []\n", "  webhook-auth:\n    actors: ['*']\n"} {
		t.Run(auth, func(t *testing.T) {
			compiler := NewCompiler()
			_, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n  webhook:\n    projects_v2_item: {}\n"+auth), "webhook-test.md")
			require.Error(t, err)
		})
	}
}

func TestWebhookPreservesDisjointDispatchTypes(t *testing.T) {
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  repository_dispatch:
    types: [manual-operation]
  webhook:
    projects_v2_item: {}
  webhook-auth:
    actors: ["cao[bot]"]
`), "webhook-test.md")
	require.NoError(t, err)
	assert.Contains(t, data.On, "manual-operation")
	assert.Contains(t, data.On, WebhookDispatchEventType("webhook-test"))
}

func TestCompileMultipleWebhookSubscriptions(t *testing.T) {
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  webhook:
    projects_v2_item:
      types: [edited]
      filters:
        project_id: PVT_example
        field:
          name: [Priority, Status]
        project_number: 7
    projects_v2:
      types: [edited]
      filters:
        public: false
  webhook-auth:
    actors: ["cao[bot]"]
`), "webhook-test.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 2)
	first, second := data.WebhookSubscriptions[0], data.WebhookSubscriptions[1]
	assert.Equal(t, "projects_v2", first.Event)
	assert.Equal(t, "projects_v2_item", second.Event)
	assert.Equal(t, first.Dispatch.EventType, second.Dispatch.EventType)
	require.Len(t, first.Conditions, 1)
	assert.Equal(t, "boolean", first.Conditions[0].ValueType)
	assert.Equal(t, []any{false}, first.Conditions[0].Values)
	require.Len(t, second.Conditions, 3)
	assert.Equal(t, []string{"changes", "field_value", "field_name"}, second.Conditions[0].Path)
	assert.Equal(t, "equals", second.Conditions[0].Operator)
	assert.Equal(t, "string", second.Conditions[0].ValueType)
	assert.Equal(t, []any{"Priority", "Status"}, second.Conditions[0].Values)
	output, err := compiler.CompileToYAML(data, "webhook-test.md")
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(output, "name: Validate webhook relay"))
}
