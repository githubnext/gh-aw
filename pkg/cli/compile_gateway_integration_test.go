//go:build integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileGatewayDispatchIntegration(t *testing.T) {
	for _, entry := range []struct {
		path   string
		events []string
	}{
		{"pkg/cli/workflows/test-gateway-github-notifications.md", []string{
			"issues", "pull_request", "discussion", "issue_comment", "pull_request_comment",
			"pull_request_review", "pull_request_review_comment", "pull_request_review_thread", "discussion_comment", "commit_comment",
		}},
		{"pkg/cli/workflows/test-gateway-resource-events.md", []string{"projects_v2_item", "code_scanning_alert", "dependabot_alert", "jira_issue", "linear_issue", "azure_devops_issue", "webhook"}},
		{"pkg/cli/workflows/test-gateway-commands.md", []string{"slash_command", "label_command"}},
		{".github/workflows/smoke-security-alerts.md", []string{"code_scanning_alert", "dependabot_alert"}},
	} {
		t.Run(filepath.Base(entry.path), func(t *testing.T) {
			setup := setupIntegrationTest(t)
			defer setup.cleanup()
			name := strings.TrimSuffix(filepath.Base(entry.path), ".md")
			source, err := os.ReadFile(filepath.Join(projectRoot, entry.path))
			require.NoError(t, err)
			path := filepath.Join(setup.workflowsDir, name+".md")
			require.NoError(t, os.WriteFile(path, source, 0o600))
			config := filepath.Join(setup.tempDir, workflow.RepoConfigFileName)
			require.NoError(t, os.WriteFile(config, []byte(`{"webhook-gateway":true}`), 0o600))
			var stderr bytes.Buffer
			cmd := exec.Command(setup.binaryPath, "compile", "--json", path)
			cmd.Stderr = &stderr
			output, err := cmd.Output()
			require.NoError(t, err, "%s\n%s", output, stderr.String())
			var results []ValidationResult
			require.NoError(t, json.Unmarshal(output, &results))
			require.Len(t, results, 1)
			require.True(t, results[0].Valid, "%+v", results[0].Errors)
			lock, err := os.ReadFile(filepath.Join(setup.workflowsDir, name+".lock.yml"))
			require.NoError(t, err)
			assertGatewayDispatchIntegrationLock(t, name, lock, results[0], entry.events)
		})
	}
}

func assertGatewayDispatchIntegrationLock(t *testing.T, name string, lock []byte, result ValidationResult, events []string) {
	t.Helper()
	route := workflow.WebhookDispatchEventType(name)
	var compiled struct {
		On   map[string]any `yaml:"on"`
		Jobs map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(lock, &compiled))
	require.Equal(t, map[string]any{
		"repository_dispatch": map[string]any{"types": []any{route}},
	}, compiled.On, "gateway-dispatch must lower to only its workflow-specific repository_dispatch route")
	require.NotEmpty(t, compiled.Jobs)
	if name == "smoke-security-alerts" {
		permissions := compiled.Jobs["agent"].Permissions
		assert.Equal(t, "write", permissions["copilot-requests"])
		assert.Equal(t, "read", permissions["security-events"])
		assert.Equal(t, "read", permissions["vulnerability-alerts"])
	}
	manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(lock))
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.Equal(t, result.WebhookSubscriptions, manifest.WebhookSubscriptions)
	actualEvents := make([]string, 0, len(manifest.WebhookSubscriptions))
	for _, subscription := range manifest.WebhookSubscriptions {
		actualEvents = append(actualEvents, subscription.Event)
		assert.Equal(t, "repository_dispatch", subscription.Dispatch.Method)
		assert.Equal(t, route, subscription.Dispatch.EventType)
		assert.Equal(t, 2, subscription.SchemaVersion)
		assert.Equal(t, "resource-url-types-filters/v2", subscription.Dispatch.PayloadProjection.Policy)
	}
	assert.ElementsMatch(t, events, actualEvents)
	assert.NotContains(t, string(lock), "check_command_position.cjs")
	assert.NotContains(t, string(lock), "workflow_dispatch:")
	assert.Contains(t, string(lock), "validate_webhook.cjs")
	assert.Contains(t, string(lock), "toJSON(fromJSON(steps.generate_aw_info.outputs.aw_context || '{}').webhook.deliveries)")
}
