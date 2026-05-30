//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractWorkflowCallSecretsFromParsed_NoOn tests with empty workflow map
func TestExtractWorkflowCallSecretsFromParsed_NoOn(t *testing.T) {
	result := extractWorkflowCallSecretsFromParsed(map[string]any{})
	assert.Nil(t, result, "Should return nil when no on: section")
}

// TestExtractWorkflowCallSecretsFromParsed_NoWorkflowCall tests without workflow_call trigger
func TestExtractWorkflowCallSecretsFromParsed_NoWorkflowCall(t *testing.T) {
	workflow := map[string]any{
		"on": map[string]any{
			"push": map[string]any{},
		},
	}
	result := extractWorkflowCallSecretsFromParsed(workflow)
	assert.Nil(t, result, "Should return nil when no workflow_call trigger")
}

// TestExtractWorkflowCallSecretsFromParsed_NoSecrets tests workflow_call without secrets section
func TestExtractWorkflowCallSecretsFromParsed_NoSecrets(t *testing.T) {
	workflow := map[string]any{
		"on": map[string]any{
			"workflow_call": map[string]any{
				"inputs": map[string]any{
					"payload": map[string]any{"type": "string"},
				},
			},
		},
	}
	result := extractWorkflowCallSecretsFromParsed(workflow)
	assert.Nil(t, result, "Should return nil when no secrets section")
}

// TestExtractWorkflowCallSecretsFromParsed_WithSecrets tests extraction of declared secrets
func TestExtractWorkflowCallSecretsFromParsed_WithSecrets(t *testing.T) {
	workflow := map[string]any{
		"on": map[string]any{
			"workflow_call": map[string]any{
				"secrets": map[string]any{
					"MY_TOKEN":      map[string]any{"required": false},
					"ANOTHER_TOKEN": map[string]any{"required": true},
				},
			},
		},
	}
	result := extractWorkflowCallSecretsFromParsed(workflow)
	require.Len(t, result, 2, "Should return two secrets")
	assert.Equal(t, []string{"ANOTHER_TOKEN", "MY_TOKEN"}, result, "Should return sorted secret names")
}

// TestExtractCallWorkflowSecrets_NoFiles tests fallback when no compiled file exists
func TestExtractCallWorkflowSecrets_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	gatewayMD := filepath.Join(workflowsDir, "gateway.md")
	require.NoError(t, os.WriteFile(gatewayMD, []byte("# test"), 0644))

	// Worker doesn't exist at all
	result, err := extractCallWorkflowSecrets("nonexistent-worker", gatewayMD)
	require.NoError(t, err, "Should not error when no file found")
	assert.Nil(t, result, "Should return nil when no compiled file found")
}

// TestExtractCallWorkflowSecrets_LockFileNoSecrets tests lock file without on.workflow_call.secrets
func TestExtractCallWorkflowSecrets_LockFileNoSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	lockContent := `name: Worker
"on":
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo done
`
	err := os.WriteFile(filepath.Join(workflowsDir, "worker.lock.yml"), []byte(lockContent), 0644)
	require.NoError(t, err)

	gatewayMD := filepath.Join(workflowsDir, "gateway.md")
	require.NoError(t, os.WriteFile(gatewayMD, []byte("# test"), 0644))

	result, err := extractCallWorkflowSecrets("worker", gatewayMD)
	require.NoError(t, err, "Should not error for lock file without secrets")
	assert.Nil(t, result, "Should return nil when no secrets declared")
}

// TestExtractCallWorkflowSecrets_LockFileWithSecrets tests lock file with on.workflow_call.secrets
func TestExtractCallWorkflowSecrets_LockFileWithSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	lockContent := `name: Worker
"on":
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
    secrets:
      GH_AW_GITHUB_TOKEN:
        required: false
      COPILOT_GITHUB_TOKEN:
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo done
`
	err := os.WriteFile(filepath.Join(workflowsDir, "worker.lock.yml"), []byte(lockContent), 0644)
	require.NoError(t, err)

	gatewayMD := filepath.Join(workflowsDir, "gateway.md")
	require.NoError(t, os.WriteFile(gatewayMD, []byte("# test"), 0644))

	result, err := extractCallWorkflowSecrets("worker", gatewayMD)
	require.NoError(t, err, "Should not error for lock file with secrets")
	require.Len(t, result, 2, "Should return two secrets")
	assert.Equal(t, []string{"COPILOT_GITHUB_TOKEN", "GH_AW_GITHUB_TOKEN"}, result,
		"Should return sorted secret names")
}

// TestExtractCallWorkflowSecrets_LockFilePrecedesYML tests that .lock.yml takes precedence over .yml
func TestExtractCallWorkflowSecrets_LockFilePrecedesYML(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	lockContent := `name: Worker
"on":
  workflow_call:
    secrets:
      FROM_LOCK:
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
`
	ymlContent := `name: Worker
"on":
  workflow_call:
    secrets:
      FROM_YML:
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
`
	err := os.WriteFile(filepath.Join(workflowsDir, "worker.lock.yml"), []byte(lockContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(workflowsDir, "worker.yml"), []byte(ymlContent), 0644)
	require.NoError(t, err)

	gatewayMD := filepath.Join(workflowsDir, "gateway.md")
	require.NoError(t, os.WriteFile(gatewayMD, []byte("# test"), 0644))

	result, err := extractCallWorkflowSecrets("worker", gatewayMD)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "FROM_LOCK", result[0], "Should use .lock.yml over .yml")
}
