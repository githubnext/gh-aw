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
	require.NoError(t, os.WriteFile(configPath, []byte(`{"webhook-gateway":true}`), 0o600))
	t.Chdir(root)
}

func TestCompileWebhookRequiresOptIn(t *testing.T) {
	for _, enabled := range []string{"", `{"webhook-gateway":false}`} {
		for _, trigger := range []string{
			"projects-items: {project_id: PVT_example}",
			"security-alerts: {sources: [dependabot]}",
			"repos: [bar/baz, \"foo/*\"]\n  security-alerts: {}",
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
				source := fmt.Sprintf("---\non:\n  webhook-gateway: true\n  %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage the event.\n", trigger)
				require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
				compiler := workflow.NewCompiler(workflow.WithNoEmit(true))
				result := compileWorkflowFile(context.Background(), compiler, path, compileWorkflowFileOptions{noEmit: true, jsonOutput: true})
				require.False(t, result.success)
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), "hosted CAO server")
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), workflow.RepoConfigFileName)
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), `{"webhook-gateway":true}`)
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
  workflow_dispatch:
  bots: [cao-gateway]
  webhook-gateway: true
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
	assert.NotContains(t, result.workflowData.On, "webhook-auth")
	assert.NotContains(t, result.workflowData.On, "projects-item")
	assert.Contains(t, result.workflowData.On, "workflow_dispatch")
	assert.Equal(t, []string{"admin", "maintainer", "write"}, result.workflowData.Roles)
	assert.Equal(t, []string{"cao-gateway"}, result.workflowData.Bots)
	lock, err := os.ReadFile(filepath.Join(dir, "webhook-test.lock.yml"))
	require.NoError(t, err)
	manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(lock))
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, result.validationResult.WebhookSubscriptions, manifest.WebhookSubscriptions)
}

func TestCompileWebhookRejectsSecretScanning(t *testing.T) {
	for _, sources := range []string{"secret-scanning", "[secret-scanning]", "[code-scanning, dependabot, secret-scanning]"} {
		t.Run(sources, func(t *testing.T) {
			dir := t.TempDir()
			enableWebhookTriggers(t, dir)
			path := filepath.Join(dir, "unsupported-secret.md")
			source := fmt.Sprintf("---\non:\n  webhook-gateway: true\n  security-alerts:\n    sources: %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage the alert.\n", sources)
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.False(t, result.success)
			assert.Contains(t, fmt.Sprint(result.validationResult.Errors), "value must be one of 'code-scanning', 'dependabot'")
			assert.Empty(t, result.validationResult.WebhookSubscriptions)
			_, err := os.Stat(filepath.Join(dir, "unsupported-secret.lock.yml"))
			require.True(t, os.IsNotExist(err), "invalid sources must not produce a lock file")
		})
	}
}

func TestCompileWebhookRejectsWeakerRoles(t *testing.T) {
	for _, trigger := range []string{
		"projects-items: {project_id: PVT_example}",
		"security-alerts: {sources: [code-scanning, dependabot]}",
		"repos: [bar/baz, \"foo/*\"]\n  security-alerts: {}",
	} {
		for _, roles := range []string{"all", "read", "triage", "[admin, write, read]"} {
			t.Run(trigger+"/"+roles, func(t *testing.T) {
				dir := t.TempDir()
				enableWebhookTriggers(t, dir)
				path := filepath.Join(dir, "weak-roles.md")
				source := fmt.Sprintf("---\nstrict: false\non:\n  webhook-gateway: true\n  workflow_dispatch:\n  bots: [cao-gateway]\n  roles: %s\n  %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage the event.\n", roles, trigger)
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
		{"code-scanning", "code_scanning_alert", "severity", []string{"alert", "rule", "severity"}, "error"},
		{"dependabot", "dependabot_alert", "severity", []string{"alert", "security_advisory", "severity"}, "critical"},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			dir := testutil.TempDir(t, "security-webhook-compile-*")
			enableWebhookTriggers(t, dir)
			path := filepath.Join(dir, "security-alerts.md")
			selector := "    severity: " + tt.value
			if tt.filter != "severity" {
				selector = fmt.Sprintf("    filters:\n      %s: %s", tt.filter, tt.value)
			}

			require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`---
on:
  webhook-gateway: true
  security-alerts:
    sources: [%s]
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
			assert.NotContains(t, result.workflowData.On, "gateway")
			assert.Equal(t, []string{"admin", "maintainer", "write"}, result.workflowData.Roles)
		})
	}
}

func TestCompileSecurityAlertRepositories(t *testing.T) {
	for _, alias := range []string{"security-alerts"} {
		t.Run(alias, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, "security-repos.md")
			source := fmt.Sprintf("---\non:\n  webhook-gateway: true\n  repos: [bar/baz, \"foo/*\"]\n  %s: {}\npermissions:\n  contents: read\nengine: copilot\n---\nTriage every unique alert.\n", alias)
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.True(t, result.success, "%+v", result.validationResult.Errors)
			subscriptions := result.validationResult.WebhookSubscriptions
			require.Len(t, subscriptions, 2)
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
		`webhook-gateway: true
repos: []
security-alerts: {}`, `webhook-gateway: true
repos: ["*/*"]
security-alerts: {}`, `webhook-gateway: true
repos: ["foo/prefix*"]
security-alerts: {}`, `webhook-gateway: true
repos: [bar/baz]
security-alerts: {sources: [secret-scanning]}`, `webhook-gateway: true
security-alerts: {repos: [bar/baz]}`, `webhook-gateway: true
security-alert: {sources: [dependabot]}`, `webhook-gateway: true
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

func TestCompileWebhookGatewayRejectsObsoleteSyntax(t *testing.T) {
	for _, on := range []string{
		"  projects-item: {project_id: PVT_example}",
		"  projects-items: {project_id: PVT_example}",
		"  security-alert: {sources: [dependabot]}",
		"  security-alerts: {sources: [dependabot]}",
		"  gateway: {}",
		"  gateway: {projects-item: {project_id: PVT_example}}",
		"  gateway: {security-alert: {sources: [dependabot]}}",
		"  gateway: {security-alerts: {}, roles: all}",
		"  gateway: {security-alerts: {}}\n  security-alert: {sources: [dependabot]}",
	} {
		t.Run(on, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, "obsolete-gateway.md")
			source := "---\non:\n" + on + "\npermissions:\n  contents: read\nengine: copilot\n---\nTriage.\n"
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.False(t, result.success)
			assert.NotEmpty(t, result.validationResult.Errors)
			assert.Empty(t, result.validationResult.WebhookSubscriptions)
			_, err := os.Stat(filepath.Join(root, "obsolete-gateway.lock.yml"))
			require.True(t, os.IsNotExist(err), "obsolete declarations must not emit a lock file")
		})
	}

}

func TestCompileWebhookRequiresWorkflowOptIn(t *testing.T) {
	for _, trigger := range []string{
		"projects-items: {project_id: PVT_example}",
		"repos: [bar/baz, \"foo/*\"]\nsecurity-alerts: {}",
	} {
		for _, flag := range []string{"", "  webhook-gateway: false\n", "  webhook-gateway: null\n", "  webhook-gateway: \"true\"\n", "  webhook-gateway: {}\n"} {
			t.Run(trigger+"/"+flag, func(t *testing.T) {
				root := t.TempDir()
				enableWebhookTriggers(t, root)
				path := filepath.Join(root, "workflow-opt-in.md")
				source := "---\nstrict: false\non:\n  workflow_dispatch:\n" + flag + "  " + trigger + "\npermissions:\n  contents: read\nengine: copilot\n---\nTriage.\n"
				require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
				result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
				require.False(t, result.success)
				assert.NotEmpty(t, result.validationResult.Errors)
				assert.Empty(t, result.validationResult.WebhookSubscriptions)
				_, err := os.Stat(filepath.Join(root, "workflow-opt-in.lock.yml"))
				require.True(t, os.IsNotExist(err), "the repository flag must not enable an unapproved workflow")
			})
		}
	}
}
