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
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("2")},
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
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("3")},
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

// TestGenerateCustomJobToolDefinition tests that generateCustomJobToolDefinition produces
// valid MCP tool definitions from SafeJobConfig input definitions.
func TestGenerateCustomJobToolDefinition(t *testing.T) {
	tests := []struct {
		name      string
		jobName   string
		jobConfig *SafeJobConfig
		check     func(t *testing.T, result map[string]any)
	}{
		{
			name:    "basic string input",
			jobName: "my_job",
			jobConfig: &SafeJobConfig{
				Description: "A test job",
				Inputs: map[string]*InputDefinition{
					"title": {
						Type:        "string",
						Description: "The title",
						Required:    true,
					},
				},
			},
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "my_job", result["name"], "name should match job name")
				assert.Equal(t, "A test job", result["description"], "description should be included")
				schema, ok := result["inputSchema"].(map[string]any)
				require.True(t, ok, "inputSchema should be a map")
				assert.Equal(t, "object", schema["type"], "schema type should be object")
				assert.Equal(t, false, schema["additionalProperties"], "additionalProperties should be false")
				props, ok := schema["properties"].(map[string]any)
				require.True(t, ok, "properties should be a map")
				titleProp, ok := props["title"].(map[string]any)
				require.True(t, ok, "title property should exist")
				assert.Equal(t, "string", titleProp["type"], "title type should be string")
				assert.Equal(t, "The title", titleProp["description"], "title description should be set")
				required, ok := schema["required"].([]string)
				require.True(t, ok, "required should be a []string")
				assert.Contains(t, required, "title", "title should be required")
			},
		},
		{
			name:    "boolean input",
			jobName: "bool_job",
			jobConfig: &SafeJobConfig{
				Inputs: map[string]*InputDefinition{
					"flag": {
						Type:     "boolean",
						Required: false,
					},
				},
			},
			check: func(t *testing.T, result map[string]any) {
				schema := result["inputSchema"].(map[string]any)
				props := schema["properties"].(map[string]any)
				flagProp := props["flag"].(map[string]any)
				assert.Equal(t, "boolean", flagProp["type"], "flag type should be boolean")
				assert.Nil(t, schema["required"], "required should be absent when no required fields")
			},
		},
		{
			name:    "number input",
			jobName: "num_job",
			jobConfig: &SafeJobConfig{
				Inputs: map[string]*InputDefinition{
					"count": {
						Type:     "number",
						Required: true,
					},
				},
			},
			check: func(t *testing.T, result map[string]any) {
				schema := result["inputSchema"].(map[string]any)
				props := schema["properties"].(map[string]any)
				countProp := props["count"].(map[string]any)
				assert.Equal(t, "number", countProp["type"], "count type should be number")
			},
		},
		{
			name:    "choice input with enum",
			jobName: "choice_job",
			jobConfig: &SafeJobConfig{
				Inputs: map[string]*InputDefinition{
					"color": {
						Type:    "choice",
						Options: []string{"red", "green", "blue"},
					},
				},
			},
			check: func(t *testing.T, result map[string]any) {
				schema := result["inputSchema"].(map[string]any)
				props := schema["properties"].(map[string]any)
				colorProp := props["color"].(map[string]any)
				assert.Equal(t, "string", colorProp["type"], "choice type should map to string")
				assert.Equal(t, []string{"red", "green", "blue"}, colorProp["enum"], "enum options should be set")
			},
		},
		{
			name:    "no inputs",
			jobName: "empty_job",
			jobConfig: &SafeJobConfig{
				Description: "No inputs",
			},
			check: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "empty_job", result["name"], "name should match")
				schema := result["inputSchema"].(map[string]any)
				props := schema["properties"].(map[string]any)
				assert.Empty(t, props, "properties should be empty")
				assert.Nil(t, schema["required"], "required should be absent")
			},
		},
		{
			name:    "no description uses default",
			jobName: "nodesc_job",
			jobConfig: &SafeJobConfig{
				Inputs: map[string]*InputDefinition{
					"x": {Type: "string"},
				},
			},
			check: func(t *testing.T, result map[string]any) {
				desc, hasDesc := result["description"]
				assert.True(t, hasDesc, "description should be present (default is added)")
				assert.Contains(t, desc.(string), "nodesc_job", "default description should include job name")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateCustomJobToolDefinition(tt.jobName, tt.jobConfig)
			require.NotNil(t, result, "result should not be nil")
			tt.check(t, result)
		})
	}
}

// TestGenerateCustomJobToolDefinitionJSONSerializable verifies that the output of
// generateCustomJobToolDefinition can be marshaled to valid JSON.
func TestGenerateCustomJobToolDefinitionJSONSerializable(t *testing.T) {
	jobConfig := &SafeJobConfig{
		Description: "Run deployment",
		Inputs: map[string]*InputDefinition{
			"env": {
				Type:        "choice",
				Description: "Target environment",
				Required:    true,
				Options:     []string{"staging", "production"},
			},
			"dry_run": {
				Type:     "boolean",
				Required: false,
			},
		},
	}

	result := generateCustomJobToolDefinition("deploy", jobConfig)
	data, err := json.Marshal(result)
	require.NoError(t, err, "result should be JSON serializable")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed), "JSON should be parseable back")
	assert.Equal(t, "deploy", parsed["name"], "name should round-trip through JSON")
}

// TestGenerateSafeOutputsConfigAddLabelsBlocked tests that the blocked field is included
// in config.json for add_labels.
func TestGenerateSafeOutputsConfigAddLabelsBlocked(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			AddLabels: &AddLabelsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					Target:         "*",
					TargetRepoSlug: "microsoft/vscode",
				},
				Allowed: []string{"bug", "enhancement"},
				Blocked: []string{"[*]*", "~spam", "stale", "triage-needed"},
			},
		},
	}

	result := generateSafeOutputsConfig(data)
	require.NotEmpty(t, result, "Expected non-empty config")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "Result must be valid JSON")

	addLabelsConfig, ok := parsed["add_labels"].(map[string]any)
	require.True(t, ok, "Expected add_labels key in config")

	blocked, ok := addLabelsConfig["blocked"]
	require.True(t, ok, "Expected blocked field in add_labels config")
	blockedSlice, ok := blocked.([]any)
	require.True(t, ok, "Blocked should be an array")
	assert.Len(t, blockedSlice, 4, "Should have 4 blocked patterns")
	assert.Equal(t, "[*]*", blockedSlice[0], "First blocked pattern should match")
	assert.Equal(t, "~spam", blockedSlice[1], "Second blocked pattern should match")
	assert.Equal(t, "stale", blockedSlice[2], "Third blocked pattern should match")
	assert.Equal(t, "triage-needed", blockedSlice[3], "Fourth blocked pattern should match")
}

// TestGenerateSafeOutputsConfigCreatePullRequestTargetRepo tests that target-repo
// and related cross-repo fields are included in config.json for create_pull_request.
func TestGenerateSafeOutputsConfigCreatePullRequestTargetRepo(t *testing.T) {
	falseVal := false
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				TargetRepoSlug:       "caido/proxy-frontend",
				AllowedRepos:         []string{"caido/other-repo"},
				BaseBranch:           "dev",
				Draft:                strPtr("true"),
				Reviewers:            []string{"corb3nik"},
				TitlePrefix:          "[refactor] ",
				FallbackAsIssue:      &falseVal,
			},
		},
	}

	result := generateSafeOutputsConfig(data)
	require.NotEmpty(t, result, "Expected non-empty config")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "Result must be valid JSON")

	prConfig, ok := parsed["create_pull_request"].(map[string]any)
	require.True(t, ok, "Expected create_pull_request key in config")

	assert.Equal(t, "caido/proxy-frontend", prConfig["target-repo"], "target-repo should be set")

	allowedRepos, ok := prConfig["allowed_repos"].([]any)
	require.True(t, ok, "allowed_repos should be an array")
	assert.Len(t, allowedRepos, 1, "Should have 1 allowed repo")
	assert.Equal(t, "caido/other-repo", allowedRepos[0], "allowed_repos should match")

	assert.Equal(t, "dev", prConfig["base_branch"], "base_branch should be set")
	assert.Equal(t, true, prConfig["draft"], "draft should be true")

	reviewers, ok := prConfig["reviewers"].([]any)
	require.True(t, ok, "reviewers should be an array")
	assert.Len(t, reviewers, 1, "Should have 1 reviewer")
	assert.Equal(t, "corb3nik", reviewers[0], "reviewer should match")

	assert.Equal(t, "[refactor] ", prConfig["title_prefix"], "title_prefix should be set")
	assert.Equal(t, false, prConfig["fallback_as_issue"], "fallback_as_issue should be false")
}

// TestGenerateSafeOutputsConfigCreatePullRequestBackwardCompat tests that config without
// target-repo still works correctly (backward compatibility).
func TestGenerateSafeOutputsConfigCreatePullRequestBackwardCompat(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("2")},
				AllowedLabels:        []string{"bug"},
				AllowEmpty:           strPtr("true"),
				AutoMerge:            strPtr("true"),
				Expires:              24,
			},
		},
	}

	result := generateSafeOutputsConfig(data)
	require.NotEmpty(t, result, "Expected non-empty config")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed), "Result must be valid JSON")

	prConfig, ok := parsed["create_pull_request"].(map[string]any)
	require.True(t, ok, "Expected create_pull_request key in config")

	assert.InDelta(t, float64(2), prConfig["max"], 0.0001, "max should be 2")
	assert.Equal(t, true, prConfig["allow_empty"], "allow_empty should be true")
	assert.Equal(t, true, prConfig["auto_merge"], "auto_merge should be true")
	assert.InDelta(t, float64(24), prConfig["expires"], 0.0001, "expires should be 24")

	allowedLabels, ok := prConfig["allowed_labels"].([]any)
	require.True(t, ok, "allowed_labels should be an array")
	assert.Len(t, allowedLabels, 1, "Should have 1 allowed label")
	assert.Equal(t, "bug", allowedLabels[0], "allowed_labels should match")

	// target-repo and allowed_repos should not be present when not configured
	_, hasTargetRepo := prConfig["target-repo"]
	assert.False(t, hasTargetRepo, "target-repo should not be present when not configured")
	_, hasAllowedRepos := prConfig["allowed_repos"]
	assert.False(t, hasAllowedRepos, "allowed_repos should not be present when not configured")
}
