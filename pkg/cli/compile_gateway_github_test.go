//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileGatewayGitHubNotificationsJSONContract(t *testing.T) {
	root := t.TempDir()
	enableWebhookTriggers(t, root)
	triggers := []string{
		"issue", "pull-request", "discussion", "issue-comment", "pull-request-comment",
		"pull-request-review", "pull-request-review-comment", "pull-request-review-thread", "discussion-comment", "commit-comment",
	}
	source := "---\non:\n  gateway-dispatch:\n    repos: [github/*]\n    orgs: [github]\n    " + strings.Join(triggers, ":\n    ") + ":"
	source += "\npermissions:\n  contents: read\nengine: copilot\n---\nNotifications: ${{ toJSON(gateway.events) }}\n"
	path := filepath.Join(root, "github-notifications.md")
	require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
	require.True(t, result.success, "%+v", result.validationResult.Errors)
	output, err := formatValidationOutput([]ValidationResult{result.validationResult})
	require.NoError(t, err)
	var results []ValidationResult
	require.NoError(t, json.Unmarshal([]byte(output), &results))
	require.Len(t, results, 1)
	assert.Equal(t, result.validationResult, results[0])
	require.Len(t, result.validationResult.WebhookSubscriptions, 10)
	for _, subscription := range result.validationResult.WebhookSubscriptions {
		assert.Len(t, subscription.NativeEvents, 1)
		assert.NotEmpty(t, subscription.Types)
		assert.Equal(t, []string{"github/*"}, subscription.Repos)
		assert.Equal(t, []string{"github"}, subscription.Orgs)
		assert.Nil(t, subscription.CommandFilter)
		assert.Empty(t, subscription.Provider)
	}
	lock, err := os.ReadFile(filepath.Join(root, "github-notifications.lock.yml"))
	require.NoError(t, err)
	manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(lock))
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Equal(t, result.validationResult.WebhookSubscriptions, manifest.WebhookSubscriptions)
	assert.NotContains(t, string(lock), "workflow_dispatch:")
	assert.NotContains(t, string(lock), "check_command_position.cjs")
	assert.Contains(t, string(lock), "toJSON(fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').webhook.deliveries)")
}
