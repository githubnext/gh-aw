//go:build !integration

package cli

import (
	"context"
	"encoding/json"
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
  webhook:
    projects_v2_item:
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
	assert.Contains(t, result.workflowData.On, "workflow_dispatch")
}
