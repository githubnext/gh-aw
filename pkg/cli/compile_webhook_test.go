//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enableWebhookTriggers(t *testing.T, root string) {
	t.Helper()
	configPath := filepath.Join(root, workflow.RepoConfigFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
	require.NoError(t, os.WriteFile(configPath, []byte(`{"gateway-dispatch":true}`), 0o600))
	t.Chdir(root)
}

func TestCompileWebhookRequiresOptIn(t *testing.T) {
	for _, enabled := range []string{"", `{"gateway-dispatch":false}`} {
		for _, trigger := range []string{
			"projects-items: {project_id: PVT_example}",
			"code-scanning-alert: {}",
			"dependabot-alerts: {}",
			"slash-command: review",
			"label-command: review",
			"repos: [bar/baz, \"foo/*\"]\n  dependabot-alerts: {}",
			"orgs: [github, \"my-*\"]\n  dependabot-alerts: {}",
		} {
			t.Run(enabled+"/"+trigger, func(t *testing.T) {
				root := t.TempDir()
				t.Chdir(root)
				if enabled != "" {
					config := filepath.Join(root, workflow.RepoConfigFileName)
					require.NoError(t, os.MkdirAll(filepath.Dir(config), 0o755))
					require.NoError(t, os.WriteFile(config, []byte(enabled), 0o600))
				}
				path := filepath.Join(root, "cao-required.md")
				source := fmt.Sprintf("---\non:\n  gateway-dispatch:\n    %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage the event.\n", strings.ReplaceAll(trigger, "\n", "\n  "))
				require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
				compiler := workflow.NewCompiler(workflow.WithNoEmit(true))
				result := compileWorkflowFile(context.Background(), compiler, path, compileWorkflowFileOptions{noEmit: true, jsonOutput: true})
				require.False(t, result.success)
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), "hosted CAO server")
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), workflow.RepoConfigFileName)
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), `{"gateway-dispatch":true}`)
			})
		}
	}
}

func TestCompileWebhookJSONContract(t *testing.T) {
	dir := testutil.TempDir(t, "webhook-compile-*")
	enableWebhookTriggers(t, dir)
	path := filepath.Join(dir, "webhook-test.md")
	require.NoError(t, os.WriteFile(path, []byte(`---
on:
  bots: [cao-gateway]
  gateway-dispatch:
    orgs: ["Example-*", GITHUB]
    projects-items:
      project_id: PVT_example
      types: [edited]
      status-transitions: [{from: Todo, to: Ready}]
permissions:
  contents: read
engine: copilot
---
# Webhook workflow

Inspect the batch in aw_context.webhook.deliveries and unique aw_context.items.
`), 0600))
	compiler := workflow.NewCompiler()
	result := compileWorkflowFile(context.Background(), compiler, path, compileWorkflowFileOptions{
		jsonOutput: true,
	})
	require.True(t, result.success, "%+v", result.validationResult.Errors)
	require.True(t, result.validationResult.Valid)
	require.Len(t, result.validationResult.WebhookSubscriptions, 1)
	subscription := result.validationResult.WebhookSubscriptions[0]
	assert.Equal(t, 2, subscription.SchemaVersion)
	assert.Equal(t, "projects_v2_item", subscription.Event)
	assert.Equal(t, []string{"edited"}, subscription.Types)
	assert.Equal(t, []string{"example-*", "github"}, subscription.Orgs)
	assert.Equal(t, "repository_dispatch", subscription.Dispatch.Method)
	assert.Equal(t, workflow.WebhookDispatchEventType("webhook-test"), subscription.Dispatch.EventType)
	assert.Equal(t, "github.event.client_payload.aw_context.webhook.deliveries", subscription.Dispatch.Context.Deliveries)
	assert.Equal(t, "github.event.client_payload.aw_context", subscription.Dispatch.Context.AWContext)
	assert.Equal(t, "aw_context.items", subscription.Dispatch.Context.Items)
	assert.Equal(t, workflow.WebhookBatch{Mode: "same-event", DeduplicateBy: "delivery_id"}, subscription.Dispatch.Batch)
	require.Len(t, subscription.Conditions, 3)
	assert.Equal(t, []string{"changes", "field_value", "field_name"}, subscription.Conditions[0].Path)
	assert.Equal(t, "string", subscription.Conditions[0].ValueType)
	assert.Equal(t, "equals", subscription.Conditions[0].Operator)
	assert.Equal(t, []any{"Status"}, subscription.Conditions[0].Values)
	require.Len(t, subscription.AnyOf, 1)
	require.Len(t, subscription.AnyOf[0], 2)
	assert.Equal(t, []any{"Todo"}, subscription.AnyOf[0][0].Values)
	assert.Equal(t, []any{"Ready"}, subscription.AnyOf[0][1].Values)
	output, err := json.Marshal(result.validationResult)
	require.NoError(t, err)
	assert.Contains(t, string(output), `"webhook_subscriptions":[`)
	assert.Contains(t, string(output), `"conditions":[{"filter":"field.name"`)
	assert.NotContains(t, string(output), `"trusted_actors"`)
	assert.Contains(t, string(output), `"any_of"`)
	assert.NotContains(t, result.workflowData.On, "projects-item")
	assert.NotContains(t, result.workflowData.On, "workflow_dispatch")
	assert.NotContains(t, result.workflowData.On, "gateway-dispatch")
	assert.Equal(t, []string{"admin", "maintainer", "write"}, result.workflowData.Roles)
	assert.Equal(t, []string{"cao-gateway"}, result.workflowData.Bots)
	lock, err := os.ReadFile(filepath.Join(dir, "webhook-test.lock.yml"))
	require.NoError(t, err)
	manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(lock))
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, result.validationResult.WebhookSubscriptions, manifest.WebhookSubscriptions)
}

func TestCompileGatewayCommandJSONContract(t *testing.T) {
	dir := t.TempDir()
	enableWebhookTriggers(t, dir)
	path := filepath.Join(dir, "gateway-commands.md")
	require.NoError(t, os.WriteFile(path, []byte(`---
on:
  gateway-dispatch:
    slash-command:
      name: [review, "fix*"]
      events: [issue_comment, pull_request_comment]
    label-command:
      name: review
      events: issues
permissions:
  contents: read
engine: copilot
---
Notifications: ${{ toJSON(gateway.events) }}
`), 0o600))
	result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
	require.True(t, result.success, "%+v", result.validationResult.Errors)
	require.Len(t, result.validationResult.WebhookSubscriptions, 2)
	for _, subscription := range result.validationResult.WebhookSubscriptions {
		require.NotNil(t, subscription.CommandFilter)
		assert.Empty(t, subscription.Conditions)
		assert.Empty(t, subscription.Provider)
	}
	output, err := json.Marshal(result.validationResult)
	require.NoError(t, err)
	assert.Contains(t, string(output), `"command_filter":{"policy":"slash-command/v1"`)
	assert.Contains(t, string(output), `"remove_label":true`)
	lock, err := os.ReadFile(filepath.Join(dir, "gateway-commands.lock.yml"))
	require.NoError(t, err)
	manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(lock))
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, result.validationResult.WebhookSubscriptions, manifest.WebhookSubscriptions)
	assert.NotContains(t, string(lock), "workflow_dispatch:")
	assert.NotContains(t, string(lock), "check_command_position.cjs")
	assert.Contains(t, string(lock), "toJSON(fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').webhook.deliveries)")
}

func TestCompileWebhookRejectsBundledAndUnsupportedSecurity(t *testing.T) {
	for _, trigger := range []string{
		"secret-scanning-alert: {}", "secret-scanning-alerts: {}", "secret-scanning-scan: {}", "secret-scanning-alert-location: {}",
		"security-alerts: {}", "security-alerts: {sources: [code-scanning, dependabot]}",
		"code-scanning-alert: {sources: [code-scanning]}", "dependabot-alerts: {sources: [dependabot]}",
		"dependabot-alert: {}", "code-scanning-alerts: {}",
	} {
		t.Run(trigger, func(t *testing.T) {
			dir := t.TempDir()
			enableWebhookTriggers(t, dir)
			path := filepath.Join(dir, "unsupported-secret.md")
			source := fmt.Sprintf("---\non:\n  gateway-dispatch:\n    %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage the alert.\n", trigger)
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.False(t, result.success)
			assert.NotEmpty(t, result.validationResult.Errors)
			assert.Empty(t, result.validationResult.WebhookSubscriptions)
			_, err := os.Stat(filepath.Join(dir, "unsupported-secret.lock.yml"))
			require.True(t, os.IsNotExist(err), "invalid security declarations must not produce a lock file")
		})
	}
}

func TestCompileWebhookRejectsWeakerRoles(t *testing.T) {
	for _, trigger := range []string{
		"projects-items: {project_id: PVT_example}",
		"code-scanning-alert: {}",
		"dependabot-alerts: {}",
		"repos: [bar/baz, \"foo/*\"]\n  dependabot-alerts: {}",
		"orgs: [github, \"my-*\"]\n  dependabot-alerts: {}",
	} {
		for _, roles := range []string{"all", "read", "triage", "[admin, write, read]"} {
			t.Run(trigger+"/"+roles, func(t *testing.T) {
				dir := t.TempDir()
				enableWebhookTriggers(t, dir)
				path := filepath.Join(dir, "weak-roles.md")
				source := fmt.Sprintf("---\nstrict: false\non:\n  bots: [cao-gateway]\n  roles: %s\n  gateway-dispatch:\n    %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage the event.\n", roles, strings.ReplaceAll(trigger, "\n", "\n  "))
				require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
				result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
				require.False(t, result.success)
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), "require on.roles with write access or higher")
				assert.Empty(t, result.validationResult.WebhookSubscriptions)
				_, err := os.Stat(filepath.Join(dir, "weak-roles.lock.yml"))
				require.True(t, os.IsNotExist(err), "invalid roles must not produce a lock file")
			})
		}
	}
}

func TestCompileSecurityWebhookJSONContract(t *testing.T) {
	for _, tt := range []struct {
		kind   string
		event  string
		filter string
		path   []string
		value  string
	}{
		{"code-scanning-alert", "code_scanning_alert", "severity", []string{"alert", "rule", "severity"}, "error"},
		{"dependabot-alerts", "dependabot_alert", "severity", []string{"alert", "security_advisory", "severity"}, "critical"},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			dir := testutil.TempDir(t, "security-webhook-compile-*")
			enableWebhookTriggers(t, dir)
			path := filepath.Join(dir, "security-alerts.md")
			selector := "      severity: " + tt.value
			if tt.filter != "severity" {
				selector = fmt.Sprintf("      filters:\n        %s: %s", tt.filter, tt.value)
			}

			require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`---
on:
  gateway-dispatch:
    %s:
      types: [created]
%s
permissions:
  contents: read
engine: copilot
---
Triage the alert without logging secret values.
`, tt.kind, selector)), 0600))
			compiler := workflow.NewCompiler(workflow.WithNoEmit(true))
			result := compileWorkflowFile(context.Background(), compiler, path, compileWorkflowFileOptions{
				noEmit: true, jsonOutput: true,
			})
			require.True(t, result.success, "%+v", result.validationResult.Errors)
			subscriptions := result.validationResult.WebhookSubscriptions
			require.Len(t, subscriptions, 1)
			sub := subscriptions[0]
			assert.Equal(t, tt.event, sub.Event)
			assert.Equal(t, workflow.WebhookDispatchEventType("security-alerts"), sub.Dispatch.EventType)
			assert.Equal(t, []string{"created"}, sub.Types)
			require.Len(t, sub.Conditions, 1)
			assert.Equal(t, tt.path, sub.Conditions[0].Path)
			assert.Equal(t, []any{tt.value}, sub.Conditions[0].Values)
			assert.NotContains(t, result.workflowData.On, "security-alert")
			assert.NotContains(t, result.workflowData.On, tt.kind)
			assert.NotContains(t, result.workflowData.On, "gateway")
			assert.Equal(t, []string{"admin", "maintainer", "write"}, result.workflowData.Roles)
		})
	}
}

func TestCompileSecurityAlertRepositories(t *testing.T) {
	for alias, event := range map[string]string{"code-scanning-alert": "code_scanning_alert", "dependabot-alerts": "dependabot_alert"} {
		t.Run(alias, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, "security-repos.md")
			source := fmt.Sprintf("---\non:\n  gateway-dispatch:\n    repos: [bar/baz, \"foo/*\"]\n    %s: {}\npermissions:\n  contents: read\nengine: copilot\n---\nTriage every unique alert.\n", alias)
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.True(t, result.success, "%+v", result.validationResult.Errors)
			subscriptions := result.validationResult.WebhookSubscriptions
			require.Len(t, subscriptions, 1)
			assert.Equal(t, event, subscriptions[0].Event)
			for _, sub := range subscriptions {
				assert.Equal(t, []string{"bar/baz", "foo/*"}, sub.Repos)
				assert.Equal(t, "${{ github.repository }}", sub.Dispatch.Repository)
				assert.Equal(t, 2, sub.SchemaVersion)
				assert.NotNil(t, sub.Dispatch.PayloadProjection)
			}
			lock, err := os.ReadFile(filepath.Join(root, "security-repos.lock.yml"))
			require.NoError(t, err)
			manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(lock))
			require.NoError(t, err)
			assert.Equal(t, subscriptions, manifest.WebhookSubscriptions)
			assert.Contains(t, string(lock), `"repos":["bar/baz","foo/*"]`)
			assert.NotContains(t, result.workflowData.On, "security-alert")
			assert.Contains(t, result.workflowData.On, "repository_dispatch")
		})
	}
}

func TestCompileSecurityAlertRejectsInvalidRepositories(t *testing.T) {
	for _, trigger := range []string{
		`gateway-dispatch:
  repos: []
  dependabot-alerts: {}`, `gateway-dispatch:
  repos: ["*/*"]
  dependabot-alerts: {}`, `gateway-dispatch:
  repos: ["foo/prefix*"]
  dependabot-alerts: {}`, `gateway-dispatch:
  repos: [bar/baz]
  secret-scanning-alert: {}`, `gateway-dispatch:
  dependabot-alerts: {repos: [bar/baz]}`, `gateway-dispatch:
dependabot-alerts: {}`, `gateway-dispatch:
  projects-items: {project_id: PVT_example, repos: [bar/baz]}`} {
		t.Run(trigger, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, "invalid-repos.md")
			source := fmt.Sprintf("---\non:\n  %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage.\n", strings.ReplaceAll(trigger, "\n", "\n  "))
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.False(t, result.success)
			assert.NotEmpty(t, result.validationResult.Errors)
			assert.Empty(t, result.validationResult.WebhookSubscriptions)
			_, err := os.Stat(filepath.Join(root, "invalid-repos.lock.yml"))
			require.True(t, os.IsNotExist(err), "invalid repository selections must not emit a lock file")
		})
	}
}

func TestCompileWebhookRejectsFlatTriggers(t *testing.T) {
	for _, trigger := range []string{
		"projects-items: {project_id: PVT_example}",
		"code-scanning-alert: {}",
		"dependabot-alerts: {}",
	} {
		t.Run(trigger, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, "workflow-opt-in.md")
			source := "---\nstrict: false\non:\n  workflow_dispatch:\n  " + trigger + "\npermissions:\n  contents: read\nengine: copilot\n---\nTriage.\n"
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.False(t, result.success)
			assert.NotEmpty(t, result.validationResult.Errors)
			assert.Empty(t, result.validationResult.WebhookSubscriptions)
			_, err := os.Stat(filepath.Join(root, "workflow-opt-in.lock.yml"))
			require.True(t, os.IsNotExist(err), "misplaced gateway triggers must not emit a lock file")
		})
	}
}

func TestCompileGatewayDispatchRejectsOtherTriggers(t *testing.T) {
	for _, trigger := range []string{
		"workflow_dispatch:", "workflow_dispatch: {}", "repository_dispatch:", "repository_dispatch: {}",
		"repository_dispatch: {types: [manual]}", `repository_dispatch: {types: ["*"]}`,
		"push:", "push: {}", "issues: {types: [opened]}", "pull_request: {}",
		"schedule: daily", "workflow_call:", "workflow_run: {workflows: ['CI'], types: [completed]}",
		"slash_command: review", "label_command: review", "future_event:",
	} {
		t.Run(trigger, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, "mixed-dispatch.md")
			source := "---\nstrict: false\non:\n  " + trigger + "\n  gateway-dispatch:\n    dependabot-alerts: {}\npermissions:\n  contents: read\nengine: copilot\n---\nTriage.\n"
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.False(t, result.success)
			assert.NotEmpty(t, result.validationResult.Errors)
			assert.Empty(t, result.validationResult.WebhookSubscriptions)
			_, err := os.Stat(filepath.Join(root, "mixed-dispatch.lock.yml"))
			require.True(t, os.IsNotExist(err), "exclusive gateway trigger conflicts must not emit a lock file")
		})
	}

}

func TestCompileWebhookRejectsCustomSubscriptions(t *testing.T) {
	for _, trigger := range []string{
		"webhooks: [{type: example.event}]",
		"dependabot-alerts: {}\nwebhooks: [{type: example.event}]",
	} {
		t.Run(trigger, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, "custom.md")
			source := "---\non:\n  gateway-dispatch:\n    " + strings.ReplaceAll(trigger, "\n", "\n    ") + "\npermissions:\n  contents: read\nengine: copilot\n---\nTriage.\n"
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.False(t, result.success)
			assert.Contains(t, fmt.Sprint(result.validationResult.Errors), "webhooks")
			assert.Empty(t, result.validationResult.WebhookSubscriptions)
			_, err := os.Stat(filepath.Join(root, "custom.lock.yml"))
			require.True(t, os.IsNotExist(err), "custom subscriptions must not emit a lock file")
		})
	}
}
