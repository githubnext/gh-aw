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

func TestCompileWebhookJSONContract(t *testing.T) {
	dir := testutil.TempDir(t, "webhook-compile-*")
	path := filepath.Join(dir, "webhook-test.md")
	require.NoError(t, os.WriteFile(path, []byte(`---
on:
  workflow_dispatch:
  projects-item:
    types: [edited]
    filters:
      field:
        name: Status
  webhook-auth:
    actors: ["cao[bot]"]
permissions:
  contents: read
engine: copilot
---
# Webhook workflow

Inspect github.event.client_payload.payload.
`), 0600))
	compiler := workflow.NewCompiler(workflow.WithNoEmit(true))
	result := compileWorkflowFile(context.Background(), compiler, path, compileWorkflowFileOptions{
		noEmit: true, jsonOutput: true,
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
	assert.Equal(t, []string{"cao[bot]"}, subscription.Dispatch.TrustedActors)
	assert.Equal(t, "github.event.client_payload.payload", subscription.Dispatch.Context.Payload)
	require.Len(t, subscription.Conditions, 1)
	assert.Equal(t, []string{"changes", "field_value", "field_name"}, subscription.Conditions[0].Path)
	assert.Equal(t, "string", subscription.Conditions[0].ValueType)
	assert.Equal(t, "equals", subscription.Conditions[0].Operator)
	assert.Equal(t, []any{"Status"}, subscription.Conditions[0].Values)
	output, err := json.Marshal(result.validationResult)
	require.NoError(t, err)
	assert.Contains(t, string(output), `"webhook_subscriptions":[`)
	assert.Contains(t, string(output), `"conditions":[{"filter":"field.name"`)
	assert.NotContains(t, result.workflowData.On, "webhook-auth")
	assert.NotContains(t, result.workflowData.On, "projects-item")
	assert.Contains(t, result.workflowData.On, "workflow_dispatch")
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
			path := filepath.Join(dir, "security-alerts.md")
			require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf(`---
on:
  security-alert:
    type: %s
    types: [created]
    filters:
      %s: %s
  webhook-auth:
    actors: ["cao[bot]"]
permissions:
  contents: read
engine: copilot
---
Triage the alert without logging secret values.
`, tt.kind, tt.filter, tt.value)), 0600))
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
			assert.Equal(t, []string{"cao[bot]"}, sub.Dispatch.TrustedActors)
			assert.Equal(t, []string{"created"}, sub.Types)
			require.Len(t, sub.Conditions, 1)
			assert.Equal(t, tt.path, sub.Conditions[0].Path)
			assert.Equal(t, []any{tt.value}, sub.Conditions[0].Values)
			assert.NotContains(t, result.workflowData.On, "security-alert")
		})
	}
}
