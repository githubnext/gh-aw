//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	require.NoError(t, os.WriteFile(configPath, []byte(`{"webhook-triggers":true}`), 0o600))
	t.Chdir(root)
}

func TestCompileWebhookRequiresOptIn(t *testing.T) {
	for _, enabled := range []string{"", `{"webhook-triggers":false}`} {
		for _, trigger := range []string{"projects-item: {project_id: PVT_example}", "security-alert: {sources: [dependabot]}"} {
			t.Run(enabled+"/"+trigger, func(t *testing.T) {
				root := t.TempDir()
				t.Chdir(root)
				if enabled != "" {
					config := filepath.Join(root, workflow.RepoConfigFileName)
					require.NoError(t, os.MkdirAll(filepath.Dir(config), 0o755))
					require.NoError(t, os.WriteFile(config, []byte(enabled), 0o600))
				}
				path := filepath.Join(root, "cao-required.md")
				source := fmt.Sprintf("---\non:\n  %s\npermissions:\n  contents: read\nengine: copilot\n---\nTriage the event.\n", trigger)
				require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
				compiler := workflow.NewCompiler(workflow.WithNoEmit(true))
				result := compileWorkflowFile(context.Background(), compiler, path, compileWorkflowFileOptions{noEmit: true, jsonOutput: true})
				require.False(t, result.success)
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), "hosted CAO server")
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), workflow.RepoConfigFileName)
				assert.Contains(t, fmt.Sprint(result.validationResult.Errors), `{"webhook-triggers":true}`)
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
  projects-item:
    project_id: PVT_example
    types: [edited]
    status-transitions: [{from: Todo, to: Ready}]
permissions:
  contents: read
engine: copilot
---
# Webhook workflow

Inspect github.event.client_payload.payload.
`), 0600))
	compiler := workflow.NewCompiler()
	result := compileWorkflowFile(context.Background(), compiler, path, compileWorkflowFileOptions{
		jsonOutput: true,
	})
	require.True(t, result.success, "%+v", result.validationResult.Errors)
	require.True(t, result.validationResult.Valid)
	require.Len(t, result.validationResult.WebhookSubscriptions, 1)
	subscription := result.validationResult.WebhookSubscriptions[0]
	assert.Equal(t, 1, subscription.SchemaVersion)
	assert.Equal(t, "projects_v2_item", subscription.Event)
	assert.Equal(t, []string{"edited"}, subscription.Types)
	assert.Equal(t, "repository_dispatch", subscription.Dispatch.Method)
	assert.Equal(t, workflow.WebhookDispatchEventType("webhook-test"), subscription.Dispatch.EventType)
	assert.Equal(t, "github.event.client_payload.payload", subscription.Dispatch.Context.Payload)
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
		{"secret-scanning", "secret_scanning_alert", "secret_type", []string{"alert", "secret_type"}, "github_personal_access_token"},
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
  security-alert:
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
			assert.Equal(t, []string{"admin", "maintainer", "write"}, result.workflowData.Roles)
		})
	}
}
