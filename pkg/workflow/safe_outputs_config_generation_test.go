//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateSafeOutputsConfigDispatchWorkflow tests that generateSafeOutputsConfig correctly
// includes dispatch_workflow configuration with workflow_files mapping.
func TestGenerateSafeOutputsConfigDispatchWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	ciWorkflow := `name: CI
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "ci.lock.yml"), []byte(ciWorkflow), 0644),
		"Failed to write ci workflow")

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchWorkflow: &DispatchWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 2},
				Workflows:            []string{"ci"},
				WorkflowFiles: map[string]string{
					"ci": ".lock.yml",
				},
			},
		},
	}

	result := generateSafeOutputsConfig(data)
	require.NotEmpty(t, result, "Expected non-empty config")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "Result must be valid JSON")

	dispatchConfig, ok := parsed["dispatch_workflow"].(map[string]any)
	require.True(t, ok, "Expected dispatch_workflow key in config")

	assert.InDelta(t, float64(2), dispatchConfig["max"], 0.0001, "Max should be 2")

	workflowFiles, ok := dispatchConfig["workflow_files"].(map[string]any)
	require.True(t, ok, "Expected workflow_files in dispatch_workflow config")
	assert.Equal(t, ".lock.yml", workflowFiles["ci"], "ci should map to .lock.yml")
}

// TestGenerateSafeOutputsConfigMissingToolWithIssue tests the missing_tool config with create_issue enabled.
func TestGenerateSafeOutputsConfigMissingToolWithIssue(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			MissingTool: &MissingToolConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 3},
				CreateIssue:          true,
				TitlePrefix:          "[Missing Tool] ",
				Labels:               []string{"bug"},
			},
		},
	}

	result := generateSafeOutputsConfig(data)
	require.NotEmpty(t, result, "Expected non-empty config")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "Result must be valid JSON")

	_, hasMissingTool := parsed["missing_tool"]
	assert.True(t, hasMissingTool, "Expected missing_tool key in config")

	createMissingIssue, hasCreateMissingIssue := parsed["create_missing_tool_issue"].(map[string]any)
	require.True(t, hasCreateMissingIssue, "Expected create_missing_tool_issue key in config")
	assert.Equal(t, "[Missing Tool] ", createMissingIssue["title_prefix"], "title_prefix should match")
	assert.InDelta(t, float64(1), createMissingIssue["max"], 0.0001, "max for issue creation should be 1")
}

// TestGenerateSafeOutputsConfigMentions tests the mentions configuration generation.
func TestGenerateSafeOutputsConfigMentions(t *testing.T) {
	enabled := true
	allowTeamMembers := false
	max := 5

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Mentions: &MentionsConfig{
				Enabled:          &enabled,
				AllowTeamMembers: &allowTeamMembers,
				Max:              &max,
				Allowed:          []string{"user1", "user2"},
			},
		},
	}

	result := generateSafeOutputsConfig(data)
	require.NotEmpty(t, result, "Expected non-empty config")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "Result must be valid JSON")

	mentions, ok := parsed["mentions"].(map[string]any)
	require.True(t, ok, "Expected mentions key in config")
	assert.Equal(t, true, mentions["enabled"], "enabled should be true")
	assert.Equal(t, false, mentions["allowTeamMembers"], "allowTeamMembers should be false")
	assert.InDelta(t, float64(5), mentions["max"], 0.0001, "max should be 5")
}

// TestPopulateDispatchWorkflowFilesNoSafeOutputs tests that the function handles nil SafeOutputs gracefully.
func TestPopulateDispatchWorkflowFilesNoSafeOutputs(t *testing.T) {
	data := &WorkflowData{SafeOutputs: nil}
	// Should not panic
	populateDispatchWorkflowFiles(data, "/some/path")
}

// TestPopulateDispatchWorkflowFilesNoWorkflows tests that the function handles empty Workflows list gracefully.
func TestPopulateDispatchWorkflowFilesNoWorkflows(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchWorkflow: &DispatchWorkflowConfig{
				Workflows: []string{},
			},
		},
	}
	// Should not panic or modify anything
	populateDispatchWorkflowFiles(data, "/some/path")
	assert.Nil(t, data.SafeOutputs.DispatchWorkflow.WorkflowFiles, "WorkflowFiles should remain nil")
}

// TestPopulateDispatchWorkflowFilesFindsLockFile tests that .lock.yml is preferred over .yml.
func TestPopulateDispatchWorkflowFilesFindsLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows dir")

	// Create both .yml and .lock.yml files
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "deploy.yml"), []byte("name: deploy\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "deploy.lock.yml"), []byte("name: deploy\n"), 0644))

	markdownPath := filepath.Join(tmpDir, ".github", "aw", "test.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(markdownPath), 0755))

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchWorkflow: &DispatchWorkflowConfig{
				Workflows: []string{"deploy"},
			},
		},
	}

	populateDispatchWorkflowFiles(data, markdownPath)

	require.NotNil(t, data.SafeOutputs.DispatchWorkflow.WorkflowFiles, "WorkflowFiles should be populated")
	assert.Equal(t, ".lock.yml", data.SafeOutputs.DispatchWorkflow.WorkflowFiles["deploy"],
		"Should prefer .lock.yml over .yml")
}
