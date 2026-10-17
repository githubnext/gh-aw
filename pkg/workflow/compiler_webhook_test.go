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
  projects-item:
    types: [edited]
`), "webhook-test.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 1)
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
	preActivation := extractJobSection(output, "pre_activation")
	assert.Contains(t, preActivation, "check_membership.cjs")
	assert.Contains(t, preActivation, `GH_AW_REQUIRED_ROLES: "admin,maintainer,write"`)
	assert.Contains(t, preActivation, "steps.check_membership.outputs.is_team_member")
	activation := extractJobSection(output, "activation")
	assert.Contains(t, activation, "Validate webhook relay")
	assert.Contains(t, activation, "if: "+webhookTriggerExpression(data))
	assert.Contains(t, activation, "GH_AW_WEBHOOK_SUBSCRIPTIONS:")
	assert.NotContains(t, activation, "GH_AW_WEBHOOK_ACTOR")
	assert.NotContains(t, output, "trusted_actors")
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
  projects-item:
    types: [edited]
  security-alert:
    type: dependabot
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
	assert.NotContains(t, lowered["on"], "projects-item")
	assert.NotContains(t, lowered["on"], "security-alert")
	require.Len(t, data.WebhookSubscriptions, 2)
	assert.Equal(t, "dependabot_alert", data.WebhookSubscriptions[0].Event)
	assert.Equal(t, "projects_v2_item", data.WebhookSubscriptions[1].Event)
}

func TestCompileFirstClassWebhookTriggers(t *testing.T) {
	for name, tt := range map[string]struct {
		trigger string
		event   string
	}{
		"projects-item":   {"  projects-item:\n", "projects_v2_item"},
		"code-scanning":   {"  security-alert:\n    type: code-scanning\n", "code_scanning_alert"},
		"dependabot":      {"  security-alert:\n    type: dependabot\n", "dependabot_alert"},
		"secret-scanning": {"  security-alert:\n    type: secret-scanning\n", "secret_scanning_alert"},
	} {
		t.Run(name, func(t *testing.T) {
			compiler := NewCompiler()
			data, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n"+tt.trigger+`
  workflow_dispatch:
  repository_dispatch:
    types: [manual-operation]
`), "first-class.md")
			require.NoError(t, err)
			require.Len(t, data.WebhookSubscriptions, 1)
			assert.Equal(t, tt.event, data.WebhookSubscriptions[0].Event)
			var triggers map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(data.On), &triggers))
			on := triggers["on"].(map[string]any)
			assert.NotContains(t, on, "projects-item")
			assert.NotContains(t, on, "security-alert")
			assert.NotContains(t, on, "webhook-auth")
			assert.Contains(t, on, "workflow_dispatch")
			assert.Equal(t, []any{"manual-operation", WebhookDispatchEventType("first-class")}, on["repository_dispatch"].(map[string]any)["types"])

			output, err := compiler.CompileToYAML(data, "first-class.md")
			require.NoError(t, err)
			activation := extractJobSection(output, "activation")
			assert.Contains(t, activation, "Validate webhook relay")
			assert.Contains(t, extractJobSection(output, "pre_activation"), "check_membership.cjs")
			assert.Contains(t, extractJobSection(output, "agent"), "Stage inline webhook event")
			var compiled map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(output), &compiled))
			job := compiled["jobs"].(map[string]any)["activation"].(map[string]any)
			var subscriptions []WebhookSubscription
			for _, raw := range job["steps"].([]any) {
				step := raw.(map[string]any)
				if step["id"] == "validate-webhook" {
					env := step["env"].(map[string]any)
					require.NoError(t, json.Unmarshal([]byte(env["GH_AW_WEBHOOK_SUBSCRIPTIONS"].(string)), &subscriptions))
				}
			}
			assert.Equal(t, data.WebhookSubscriptions, subscriptions)
		})
	}
}

func TestWebhookRejectsInvalidConfiguration(t *testing.T) {
	tests := map[string]string{
		"removed general registration": `  webhook:
    not_a_github_event: {}`,
		"removed auth": `  projects-item:
  webhook-auth:
    actors: ["cao[bot]"]`,
		"invalid action": `  projects-item:
    types: [not_a_real_action]`,
		"unknown filter": `  projects-item:
    filters:
      not-a-real-filter: test`,
		"wrong filter type": `  projects-item:
    filters:
      item_database_id: "7"`,
		"unavailable filter": `  projects-item:
    types: [created]
    filters:
      field:
        name: Status`,
		"scalar event":   `  projects-item: true`,
		"empty webhooks": `  webhook: {}`,
		"unrestricted dispatch": `  repository_dispatch:
  projects-item: {}`,
		"wildcard dispatch": `  repository_dispatch:
    types: ["gh-aw-*"]
  projects-item: {}`,
		"colliding dispatch": `  repository_dispatch:
    types: ["` + WebhookDispatchEventType("webhook-test") + `"]
  projects-item: {}`,
		"security wildcard dispatch": `  repository_dispatch:
    types: ["gh-aw-*"]
  security-alert:
    type: dependabot`,
		"security duplicate spelling": `  security-alert:
    type: code-scanning
  webhook:
    code_scanning_alert: {}`,
		"duplicate spelling": `  projects-item:
  webhook:
    projects_v2_item: {}`,
	}
	for name, trigger := range tests {
		t.Run(name, func(t *testing.T) {
			compiler := NewCompiler()
			_, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n"+trigger+"\n"), "webhook-test.md")
			require.Error(t, err)
		})
	}
}

func TestWebhookUsesStandardAuthorization(t *testing.T) {
	for name, policy := range map[string]string{
		"default": "",
		"roles":   "  roles: [admin]\n",
		"bots":    "  bots: [cao-gateway]\n",
		"all":     "  roles: all\n",
	} {
		t.Run(name, func(t *testing.T) {
			compiler := NewCompiler()
			data, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n  projects-item:\n"+policy), "webhook-test.md")
			require.NoError(t, err)
			output, err := compiler.CompileToYAML(data, "webhook-test.md")
			require.NoError(t, err)
			if name == "all" {
				assert.NotContains(t, output, "id: check_membership")
			} else {
				preActivation := extractJobSection(output, "pre_activation")
				assert.Contains(t, preActivation, "check_membership.cjs")
				assert.Contains(t, preActivation, "steps.check_membership.outputs.is_team_member")
				if name == "roles" {
					assert.Contains(t, preActivation, `GH_AW_REQUIRED_ROLES: "admin"`)
				}
				if name == "bots" {
					assert.Contains(t, preActivation, `GH_AW_ALLOWED_BOTS: "cao-gateway"`)
				}
			}
			assert.NotContains(t, output, "GH_AW_WEBHOOK_ACTOR")
		})
	}
}

func TestWebhookPreservesDisjointDispatchTypes(t *testing.T) {
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  repository_dispatch:
    types: [manual-operation]
  projects-item: {}
`), "webhook-test.md")
	require.NoError(t, err)
	assert.Contains(t, data.On, "manual-operation")
	assert.Contains(t, data.On, WebhookDispatchEventType("webhook-test"))
}

func TestCompileMultipleWebhookSubscriptions(t *testing.T) {
	compiler := NewCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  projects-item:
    types: [edited]
    filters:
      project_id: PVT_example
      field:
        name: [Priority, Status]
      project_number: 7
  security-alert:
    type: secret-scanning
    types: [created]
    filters:
      publicly_leaked: false
`), "webhook-test.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 2)
	first, second := data.WebhookSubscriptions[0], data.WebhookSubscriptions[1]
	assert.Equal(t, "projects_v2_item", first.Event)
	assert.Equal(t, "secret_scanning_alert", second.Event)
	assert.Equal(t, first.Dispatch.EventType, second.Dispatch.EventType)
	require.Len(t, second.Conditions, 1)
	assert.Equal(t, "boolean", second.Conditions[0].ValueType)
	assert.Equal(t, []any{false}, second.Conditions[0].Values)
	require.Len(t, first.Conditions, 3)
	assert.Equal(t, []string{"changes", "field_value", "field_name"}, first.Conditions[0].Path)
	assert.Equal(t, "equals", first.Conditions[0].Operator)
	assert.Equal(t, "string", first.Conditions[0].ValueType)
	assert.Equal(t, []any{"Priority", "Status"}, first.Conditions[0].Values)
	output, err := compiler.CompileToYAML(data, "webhook-test.md")
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(output, "name: Validate webhook relay"))
}
