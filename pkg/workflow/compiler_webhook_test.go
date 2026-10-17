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

func newWebhookCompiler() *Compiler {
	compiler := NewCompiler()
	compiler.repoConfig = &RepoConfig{CAO: &CAOConfig{Triggers: true}}
	compiler.repoConfigLoaded = true
	return compiler
}

func TestCompileWebhookTriggers(t *testing.T) {
	compiler := newWebhookCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  workflow_dispatch:
  push:
    branches: [main]
  projects-item:
    project_id: PVT_example
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
    project_id: PVT_example
    types: [edited]
  security-alert:
    sources: [dependabot]
`), &frontmatter))
	original, err := json.Marshal(frontmatter)
	require.NoError(t, err)
	data := &WorkflowData{WorkflowID: "webhook-test"}
	compiler := newWebhookCompiler()
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
		"projects-item":      {"  projects-item:\n    project_id: PVT_example\n", "projects_v2_item"},
		"status-transitions": {"  projects-item:\n    project_id: PVT_example\n    status-transitions: [{from: Todo, to: Ready}]\n", "projects_v2_item"},
		"code-scanning":      {"  security-alert:\n    sources: [code-scanning]\n", "code_scanning_alert"},
		"dependabot":         {"  security-alert:\n    sources: [dependabot]\n", "dependabot_alert"},
		"secret-scanning":    {"  security-alert:\n    sources: [secret-scanning]\n", "secret_scanning_alert"},
	} {
		t.Run(name, func(t *testing.T) {
			compiler := newWebhookCompiler()
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
		"missing project ID": `  projects-item: {}`,
		"nested project ID": `  projects-item:
    filters:
      project_id: PVT_example`,
		"removed security type": `  security-alert:
    type: dependabot`,
		"removed general registration": `  webhook:
    not_a_github_event: {}`,
		"removed auth": `  projects-item:
    project_id: PVT_example
  webhook-auth:
    actors: ["cao[bot]"]`,
		"invalid action": `  projects-item:
    project_id: PVT_example
    types: [not_a_real_action]`,
		"unknown filter": `  projects-item:
    project_id: PVT_example
    filters:
      not-a-real-filter: test`,
		"wrong filter type": `  projects-item:
    project_id: PVT_example
    filters:
      item_database_id: "7"`,
		"unavailable filter": `  projects-item:
    project_id: PVT_example
    types: [created]
    filters:
      field:
        name: Status`,
		"scalar event":   `  projects-item: true`,
		"empty webhooks": `  webhook: {}`,
		"unrestricted dispatch": `  repository_dispatch:
  projects-item: {project_id: PVT_example}`,
		"wildcard dispatch": `  repository_dispatch:
    types: ["gh-aw-*"]
  projects-item: {project_id: PVT_example}`,
		"colliding dispatch": `  repository_dispatch:
    types: ["` + WebhookDispatchEventType("webhook-test") + `"]
  projects-item: {project_id: PVT_example}`,
		"security wildcard dispatch": `  repository_dispatch:
    types: ["gh-aw-*"]
  security-alert:
    sources: [dependabot]`,
		"security duplicate spelling": `  security-alert:
    sources: [code-scanning]
  webhook:
    code_scanning_alert: {}`,
		"duplicate spelling": `  projects-item:
    project_id: PVT_example
  webhook:
    projects_v2_item: {}`,
	}

	for name, trigger := range tests {
		t.Run(name, func(t *testing.T) {
			compiler := newWebhookCompiler()
			_, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n"+trigger+"\n"), "webhook-test.md")
			require.Error(t, err)
		})
	}
}

func TestCompileWebhookNativeSelectors(t *testing.T) {
	compiler := newWebhookCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  projects-item:
    project_id: PVT_example
    status-transitions:
      - {from: Todo, to: Ready}
      - {from: In progress, to: Done}
  security-alert:
    sources: [code-scanning, dependabot, secret-scanning]
    types: [created]
`), "native-selectors.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 4)
	project := data.WebhookSubscriptions[2]
	assert.Equal(t, "projects_v2_item", project.Event)
	assert.Equal(t, []string{"edited"}, project.Types)
	require.Len(t, project.AnyOf, 2)
	output, err := compiler.CompileToYAML(data, "native-selectors.md")
	require.NoError(t, err)
	assert.Contains(t, output, "any_of")
	assert.NotContains(t, output, "check_project_status_transition")
	assert.NotContains(t, output, "status_transition_ok")
	assert.NotContains(t, output, "GH_AW_PROJECT_STATUS_CONFIG")
	assert.NotContains(t, data.On, "status-transitions")
	assert.NotContains(t, data.On, "security-alert")
	assert.Equal(t, 1, strings.Count(output, "name: Validate webhook relay"))
}

func TestWebhookStatusTransitionsDoNotAddAdmissionSteps(t *testing.T) {
	for _, policy := range []string{"", "  roles: all\n"} {
		t.Run(policy, func(t *testing.T) {
			compiler := newWebhookCompiler()
			source := webhookWorkflow("on:\n" + policy + `  projects-item:
    project_id: PVT_example
    status-transitions:
      - {from: Todo, to: Ready}
      - {from: In progress, to: Done}
`)
			data, err := compiler.ParseWorkflowString(source, "cao-status.md")
			require.NoError(t, err)
			require.Len(t, data.WebhookSubscriptions, 1)
			require.Len(t, data.WebhookSubscriptions[0].AnyOf, 2)
			output, err := compiler.CompileToYAML(data, "cao-status.md")
			require.NoError(t, err)
			assert.Contains(t, output, "any_of")
			assert.NotContains(t, output, "check_project_status_transition")
			assert.NotContains(t, output, "status_transition_ok")
			assert.NotContains(t, output, "GH_AW_PROJECT_STATUS_CONFIG")
			if policy != "" {
				assert.NotContains(t, output, "\n  pre_activation:\n")
			} else {
				assert.Contains(t, extractJobSection(output, "pre_activation"), "check_membership.cjs")
			}
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
			compiler := newWebhookCompiler()
			data, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n  projects-item:\n    project_id: PVT_example\n"+policy), "webhook-test.md")
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
	compiler := newWebhookCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  repository_dispatch:
    types: [manual-operation]
  projects-item: {project_id: PVT_example}
`), "webhook-test.md")
	require.NoError(t, err)
	assert.Contains(t, data.On, "manual-operation")
	assert.Contains(t, data.On, WebhookDispatchEventType("webhook-test"))
}

func TestCompileMultipleWebhookSubscriptions(t *testing.T) {
	compiler := newWebhookCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  projects-item:
    project_id: PVT_example
    types: [edited]
    filters:
      field:
        name: [Priority, Status]
      project_number: 7
  security-alert:
    sources: [secret-scanning]
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
