//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileProviderIssueJSONContract(t *testing.T) {
	for _, alias := range []string{"jira-issue", "linear-issue", "azure-devops-issue"} {
		t.Run(alias, func(t *testing.T) {
			root := t.TempDir()
			enableWebhookTriggers(t, root)
			path := filepath.Join(root, alias+".md")
			source := "---\non:\n  webhook-gateway: true\n  " + alias + ":\n    types: [opened, labeled]\npermissions:\n  contents: read\nengine: copilot\n---\nInspect every issue in github.aw.context.items.\n"
			require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
			result := compileWorkflowFile(context.Background(), workflow.NewCompiler(), path, compileWorkflowFileOptions{jsonOutput: true})
			require.True(t, result.success, "%+v", result.validationResult.Errors)
			require.Len(t, result.validationResult.WebhookSubscriptions, 1)
			sub := result.validationResult.WebhookSubscriptions[0]
			require.NotNil(t, sub.ActivityNormalization)
			assert.NotEmpty(t, sub.Provider)
			assert.NotEmpty(t, sub.NativeEvents)
			output, err := json.Marshal(result.validationResult)
			require.NoError(t, err)
			assert.Contains(t, string(output), `"activity_normalization":`)
			lock, err := os.ReadFile(filepath.Join(root, alias+".lock.yml"))
			require.NoError(t, err)
			manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(lock))
			require.NoError(t, err)
			require.Len(t, manifest.WebhookSubscriptions, 1)
			assert.Equal(t, sub, manifest.WebhookSubscriptions[0])
		})
	}
}
