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

Inspect the batched webhooks in aw_context.webhook.deliveries.
Items: ${{ toJSON(github.aw.context.items) }}
Event: ${{ github.aw.context.webhook.event }}
`
}

func newWebhookCompiler() *Compiler {
	compiler := NewCompiler()
	compiler.repoConfig = &RepoConfig{GatewayDispatch: true}
	compiler.repoConfigLoaded = true
	return compiler
}

func TestCompileWebhookTriggers(t *testing.T) {
	compiler := newWebhookCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      types: [edited]
`), "webhook-test.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 1)
	var triggers map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(data.On), &triggers))
	on := triggers["on"].(map[string]any)
	assert.NotContains(t, on, "gateway")
	assert.NotContains(t, on, "repos")
	assert.NotContains(t, on, "projects-items")
	assert.NotContains(t, on, "security-alerts")
	assert.NotContains(t, on, "code-scanning-alert")
	assert.NotContains(t, on, "dependabot-alerts")
	assert.NotContains(t, on, "workflow_dispatch")
	assert.NotContains(t, on, "gateway-dispatch")
	assert.NotContains(t, on, "push")
	assert.NotContains(t, on, "webhook")
	assert.NotContains(t, on, "webhook-auth")
	assert.Equal(t, []any{WebhookDispatchEventType("webhook-test")}, on["repository_dispatch"].(map[string]any)["types"])

	output, err := compiler.CompileToYAML(data, "webhook-test.md")
	require.NoError(t, err)
	var compiled map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(output), &compiled))
	compiledOn := compiled["on"].(map[string]any)
	assert.NotContains(t, compiledOn, "gateway-dispatch")
	assert.NotContains(t, compiledOn, "workflow_dispatch")
	assert.Equal(t, map[string]any{"repository_dispatch": map[string]any{"types": []any{WebhookDispatchEventType("webhook-test")}}}, compiledOn)
	assert.Equal(t, []any{WebhookDispatchEventType("webhook-test")}, compiledOn["repository_dispatch"].(map[string]any)["types"])
	preActivation := extractJobSection(output, "pre_activation")
	assert.NotContains(t, preActivation, "check_membership.cjs")
	assert.Contains(t, preActivation, "needs.gateway_provenance.outputs.verified == 'true'")
	assert.Contains(t, extractJobSection(output, "gateway_provenance"), "check_gateway_provenance.cjs")
	activation := extractJobSection(output, "activation")
	assert.Contains(t, activation, "Validate webhook relay")
	assert.Contains(t, activation, "if: "+webhookTriggerExpression(data))
	assert.Contains(t, activation, "GH_AW_WEBHOOK_SUBSCRIPTIONS:")
	assert.NotContains(t, activation, "GH_AW_WEBHOOK_ACTOR")
	assert.NotContains(t, output, "trusted_actors")
	assert.Contains(t, activation, "uses: "+getCachedActionPin("actions/github-script", data))
	assert.Contains(t, activation, "          script: |")
	assert.Contains(t, activation, "validate_webhook.cjs")
	assert.NotContains(t, activation, "run: node")
	assert.NotContains(t, activation, "toJSON(github.event")
	require.Contains(t, activation, "Create prompt with built-in context")
	assert.Less(t, strings.Index(activation, "Validate webhook relay"), strings.Index(activation, "Create prompt with built-in context"))
	agent := extractJobSection(output, "agent")
	assert.Contains(t, agent, "Stage inline webhook event")
	assert.Contains(t, agent, `cp -- "$GITHUB_EVENT_PATH" /tmp/gh-aw/github-event.json`)
	assert.Contains(t, agent, `cp -- "$GITHUB_EVENT_PATH" "${RUNNER_TEMP}/gh-aw/github-event.json"`)
	assert.NotContains(t, activation, `cp -- "$GITHUB_EVENT_PATH"`)
	assert.Contains(t, activation, "Read the projected repository_dispatch event from /tmp/gh-aw/github-event.json")
	assert.Contains(t, activation, "canonical resource URL at payload.url")
	assert.Contains(t, activation, "payload.type array of supported event activities")
	assert.Contains(t, activation, "payload.filters when declared conditions exist")
	assert.Contains(t, activation, "only matched literal values for the declared predicates")
	assert.Contains(t, activation, "Forwarded types are only the declared matching activities")
	assert.Contains(t, activation, "severity is a declared filter literal")
	assert.Contains(t, activation, "exactly one matching declared terminal name")
	assert.Contains(t, activation, "raw source bodies are not forwarded")
	assert.Contains(t, activation, "using provider-authenticated clients so the provider enforces access")
	assert.Contains(t, activation, "never send credentials to arbitrary payload URLs or redirects")
	assert.Contains(t, activation, "without fallback to webhook bodies")
	assert.Contains(t, activation, "authenticated GitHub App client")
	assert.Contains(t, activation, "client_payload.aw_context.webhook.deliveries")
	assert.Contains(t, activation, "context.items")
	assert.Contains(t, activation, "Handle every item")
	assert.Contains(t, activation, "GH_AW_WEBHOOK_TRIGGER:")
	assert.Contains(t, activation, "toJSON(fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').items)")
	assert.Contains(t, activation, "fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').webhook.event")
	assert.NotContains(t, activation, "${{ github.aw.context.")
}

func TestWebhookTriggerLoweringPreservesSource(t *testing.T) {
	var frontmatter map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(`on:
  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      types: [edited]
    dependabot-alerts: {}
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
	assert.NotContains(t, lowered["on"], "gateway")
	assert.NotContains(t, lowered["on"], "gateway-dispatch")
	assert.NotContains(t, lowered["on"], "projects-items")
	assert.NotContains(t, lowered["on"], "security-alerts")
	assert.NotContains(t, lowered["on"], "code-scanning-alert")
	assert.NotContains(t, lowered["on"], "dependabot-alerts")
	assert.NotContains(t, lowered["on"], "projects-item")
	assert.NotContains(t, lowered["on"], "security-alert")
	require.Len(t, data.WebhookSubscriptions, 2)
	assert.Equal(t, "dependabot_alert", data.WebhookSubscriptions[0].Event)
	assert.Equal(t, "projects_v2_item", data.WebhookSubscriptions[1].Event)
}

func TestGatewayDispatchRequiresSubscriptions(t *testing.T) {
	for _, config := range []string{"", "true", "false", "[]", "{}", "{repos: [owner/repo]}"} {
		t.Run(config, func(t *testing.T) {
			_, err := newWebhookCompiler().ParseWorkflowString(webhookWorkflow("on:\n  gateway-dispatch: "+config+"\n"), "empty-gateway.md")
			require.Error(t, err)
		})
	}
}

func TestCompileFirstClassWebhookTriggers(t *testing.T) {
	for name, tt := range map[string]struct {
		trigger string
		event   string
	}{
		"projects-items":     {"  gateway-dispatch:\n    projects-items:\n      project_id: PVT_example\n", "projects_v2_item"},
		"status-transitions": {"  gateway-dispatch:\n    projects-items:\n      project_id: PVT_example\n      status-transitions: [{from: Todo, to: Ready}]\n", "projects_v2_item"},
		"code-scanning":      {"  gateway-dispatch:\n    code-scanning-alert: {}\n", "code_scanning_alert"},
		"dependabot":         {"  gateway-dispatch:\n    dependabot-alerts: {}\n", "dependabot_alert"},
		"multi-repo":         {"  gateway-dispatch:\n    repos: [bar/baz, \"foo/*\"]\n    dependabot-alerts: {}\n", "dependabot_alert"},
	} {
		t.Run(name, func(t *testing.T) {
			compiler := newWebhookCompiler()
			data, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n"+tt.trigger), "first-class.md")
			require.NoError(t, err)
			require.Len(t, data.WebhookSubscriptions, 1)
			assert.Equal(t, tt.event, data.WebhookSubscriptions[0].Event)
			var triggers map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(data.On), &triggers))
			on := triggers["on"].(map[string]any)
			assert.NotContains(t, on, "gateway")
			assert.NotContains(t, on, "projects-item")
			assert.NotContains(t, on, "security-alert")
			assert.NotContains(t, on, "security-alerts")
			assert.NotContains(t, on, "code-scanning-alert")
			assert.NotContains(t, on, "dependabot-alerts")
			assert.NotContains(t, on, "webhook-auth")
			assert.NotContains(t, on, "workflow_dispatch")
			assert.NotContains(t, on, "gateway-dispatch")
			assert.Equal(t, []any{WebhookDispatchEventType("first-class")}, on["repository_dispatch"].(map[string]any)["types"])

			output, err := compiler.CompileToYAML(data, "first-class.md")
			require.NoError(t, err)
			activation := extractJobSection(output, "activation")
			assert.Contains(t, activation, "Validate webhook relay")
			assert.Contains(t, extractJobSection(output, "gateway_provenance"), "check_gateway_provenance.cjs")
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
		"missing project ID": `  gateway-dispatch:
    projects-items: {}`, "nested project ID": `  gateway-dispatch:
    projects-items:
      filters:
        project_id: PVT_example`,
		"removed security type": `  gateway-dispatch:
    dependabot-alerts:
      type: dependabot`,
		"removed security bundle": `  gateway-dispatch:
    security-alerts: {sources: [code-scanning, dependabot]}`,
		"removed sources selector": `  gateway-dispatch:
    dependabot-alerts: {sources: [dependabot]}`,
		"removed general registration": `  webhook:
    not_a_github_event: {}`,
		"removed auth": `  gateway-dispatch:
    projects-items:
      project_id: PVT_example
  webhook-auth:
    actors: ["cao[bot]"]`,
		"invalid action": `  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      types: [not_a_real_action]`,
		"unknown filter": `  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      filters:
        not-a-real-filter: test`,
		"wrong filter type": `  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      filters:
        item_database_id: "7"`,
		"unavailable filter": `  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      types: [created]
      filters:
        field:
          name: Status`,
		"scalar event": `  gateway-dispatch:
    projects-items: true`, "empty webhooks": `  webhook: {}`,
		"unrestricted dispatch": `  repository_dispatch:
  gateway-dispatch:
    projects-items: {project_id: PVT_example}`,
		"wildcard dispatch": `  repository_dispatch:
    types: ["gh-aw-*"]
  gateway-dispatch:
    projects-items: {project_id: PVT_example}`,
		"colliding dispatch": `  repository_dispatch:
    types: ["` + WebhookDispatchEventType("webhook-test") + `"]
  gateway-dispatch:
    projects-items: {project_id: PVT_example}`,
		"security wildcard dispatch": `  repository_dispatch:
    types: ["gh-aw-*"]
  gateway-dispatch:
    dependabot-alerts: {}`,
		"removed secret scanning": `  gateway-dispatch:
    secret-scanning-alert: {}`,
		"mixed removed secret scanning": `  gateway-dispatch:
    code-scanning-alert: {}
    dependabot-alerts: {}
    secret-scanning-alert: {}`,
		"security duplicate spelling": `  gateway-dispatch:
    code-scanning-alert: {}
  webhook:
    code_scanning_alert: {}`,
		"duplicate spelling": `  gateway-dispatch:
    projects-items:
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
  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      status-transitions:
        - {from: Todo, to: Ready}
        - {from: In progress, to: Done}
    code-scanning-alert:
      types: [appeared_in_branch]
      severity: error
    dependabot-alerts:
      types: [dismissed]
      severity: critical
`), "native-selectors.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 3)
	assert.Equal(t, "code_scanning_alert", data.WebhookSubscriptions[0].Event)
	assert.Equal(t, []string{"appeared_in_branch"}, data.WebhookSubscriptions[0].Types)
	assert.Equal(t, []any{"error"}, data.WebhookSubscriptions[0].Conditions[0].Values)
	assert.Equal(t, "dependabot_alert", data.WebhookSubscriptions[1].Event)
	assert.Equal(t, []string{"dismissed"}, data.WebhookSubscriptions[1].Types)
	assert.Equal(t, []any{"critical"}, data.WebhookSubscriptions[1].Conditions[0].Values)
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
	for _, policy := range []string{"", "  roles: write\n"} {
		t.Run(policy, func(t *testing.T) {
			compiler := newWebhookCompiler()
			source := webhookWorkflow("on:\n" + policy + `  gateway-dispatch:
    projects-items:
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
			assert.Contains(t, extractJobSection(output, "gateway_provenance"), "check_gateway_provenance.cjs")
		})
	}
}

func TestWebhookRequiresAppProvenanceRegardlessOfRolesAndBots(t *testing.T) {
	for name, policy := range map[string]string{
		"default": "",
		"roles":   "  roles: [admin]\n",
		"bots":    "  bots: [cao-gateway]\n",
		"all":     "  roles: all\n",
	} {
		t.Run(name, func(t *testing.T) {
			compiler := newWebhookCompiler()
			data, err := compiler.ParseWorkflowString(webhookWorkflow("on:\n  gateway-dispatch:\n    projects-items:\n      project_id: PVT_example\n"+policy), "webhook-test.md")
			if name == "all" {
				require.ErrorContains(t, err, "require on.roles with write access or higher")
				return
			}
			require.NoError(t, err)
			output, err := compiler.CompileToYAML(data, "webhook-test.md")
			require.NoError(t, err)
			preActivation := extractJobSection(output, "pre_activation")
			assert.NotContains(t, preActivation, "check_membership.cjs")
			assert.NotContains(t, preActivation, "GH_AW_ALLOWED_BOTS:")
			assert.Contains(t, preActivation, "needs.gateway_provenance.outputs.verified == 'true'")
			assert.Contains(t, extractJobSection(output, "gateway_provenance"), "check_gateway_provenance.cjs")
			assert.NotContains(t, output, "GH_AW_WEBHOOK_ACTOR")
		})
	}
}

func TestGatewayDispatchRejectsOtherTriggers(t *testing.T) {
	for _, trigger := range []string{
		"workflow_dispatch:", "workflow_dispatch: {}", "workflow_dispatch: {inputs: {message: {type: string}}}",
		"repository_dispatch:", "repository_dispatch: {}", "repository_dispatch: {types: [manual-operation]}",
		"repository_dispatch: {types: [\"" + strings.ToUpper(WebhookDispatchEventType("webhook-test")) + "\"]}",
		"push:", "push: {}", "issues: {types: [opened]}", "pull_request: {}",
		"schedule: daily", "schedule: [{cron: '0 0 * * *'}]", "workflow_call:", "workflow_run: {workflows: ['CI'], types: [completed]}",
		"slash_command: review", "command: review", "label_command: review",
		"future_event:", "future_event: {}",
	} {
		for _, strict := range []string{"true", "false"} {
			t.Run(trigger+"/"+strict, func(t *testing.T) {
				source := webhookWorkflow("strict: " + strict + "\non:\n  " + trigger + "\n  gateway-dispatch:\n    projects-items: {project_id: PVT_example}\n")
				_, err := newWebhookCompiler().ParseWorkflowString(source, "webhook-test.md")
				require.Error(t, err)
				var frontmatter map[string]any
				require.NoError(t, yaml.Unmarshal([]byte("on:\n  "+trigger+"\n  gateway-dispatch:\n    projects-items: {project_id: PVT_example}\n"), &frontmatter))
				_, err = newWebhookCompiler().processWebhookTriggers(frontmatter, &WorkflowData{WorkflowID: "webhook-test"})
				require.ErrorContains(t, err, "on.gateway-dispatch is exclusive")
			})
		}
	}
}

func TestCompileMultipleWebhookSubscriptions(t *testing.T) {
	compiler := newWebhookCompiler()
	data, err := compiler.ParseWorkflowString(webhookWorkflow(`on:
  gateway-dispatch:
    projects-items:
      project_id: PVT_example
      types: [edited]
      filters:
        field:
          name: [Priority, Status]
        project_number: 7
    dependabot-alerts:
      types: [created]
      severity: critical
`), "webhook-test.md")
	require.NoError(t, err)
	require.Len(t, data.WebhookSubscriptions, 2)
	first, second := data.WebhookSubscriptions[1], data.WebhookSubscriptions[0]
	assert.Equal(t, "projects_v2_item", first.Event)
	assert.Equal(t, "dependabot_alert", second.Event)
	assert.Equal(t, first.Dispatch.EventType, second.Dispatch.EventType)
	require.Len(t, second.Conditions, 1)
	assert.Equal(t, "string", second.Conditions[0].ValueType)
	assert.Equal(t, []any{"critical"}, second.Conditions[0].Values)
	require.Len(t, first.Conditions, 3)
	assert.Equal(t, []string{"changes", "field_value", "field_name"}, first.Conditions[0].Path)
	assert.Equal(t, "equals", first.Conditions[0].Operator)
	assert.Equal(t, "string", first.Conditions[0].ValueType)
	assert.Equal(t, []any{"Priority", "Status"}, first.Conditions[0].Values)
	output, err := compiler.CompileToYAML(data, "webhook-test.md")
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(output, "name: Validate webhook relay"))
}
