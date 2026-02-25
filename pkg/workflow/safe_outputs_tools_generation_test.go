//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateCustomJobToolDefinitionBasic tests the basic structure of a generated custom tool.
func TestGenerateCustomJobToolDefinitionBasic(t *testing.T) {
	jobConfig := &SafeJobConfig{
		Description: "My custom job",
		Inputs: map[string]*InputDefinition{
			"env": {
				Type:        "choice",
				Description: "Environment to deploy to",
				Options:     []string{"staging", "production"},
				Required:    true,
			},
			"message": {
				Type:        "string",
				Description: "Optional message",
			},
		},
	}

	tool := generateCustomJobToolDefinition("deploy_app", jobConfig)

	assert.Equal(t, "deploy_app", tool["name"], "Tool name should match")
	assert.Equal(t, "My custom job", tool["description"], "Description should match")

	inputSchema, ok := tool["inputSchema"].(map[string]any)
	require.True(t, ok, "inputSchema should be present")
	assert.Equal(t, "object", inputSchema["type"], "inputSchema type should be object")
	assert.Equal(t, false, inputSchema["additionalProperties"], "additionalProperties should be false")

	required, ok := inputSchema["required"].([]string)
	require.True(t, ok, "required should be a []string")
	assert.Contains(t, required, "env", "env should be required")

	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "properties should be present")

	envProp, ok := properties["env"].(map[string]any)
	require.True(t, ok, "env property should exist")
	assert.Equal(t, "string", envProp["type"], "choice type maps to string")
	assert.Equal(t, []string{"staging", "production"}, envProp["enum"], "enum values should match")
}

// TestGenerateCustomJobToolDefinitionDefaultDescription tests that a default description is used when none provided.
func TestGenerateCustomJobToolDefinitionDefaultDescription(t *testing.T) {
	jobConfig := &SafeJobConfig{}
	tool := generateCustomJobToolDefinition("my_job", jobConfig)
	assert.Equal(t, "Execute the my_job custom job", tool["description"], "Default description should be set")
}

// TestGenerateCustomJobToolDefinitionBooleanInput tests boolean input type mapping.
func TestGenerateCustomJobToolDefinitionBooleanInput(t *testing.T) {
	jobConfig := &SafeJobConfig{
		Inputs: map[string]*InputDefinition{
			"dry_run": {
				Type:     "boolean",
				Required: false,
			},
		},
	}

	tool := generateCustomJobToolDefinition("run_job", jobConfig)
	inputSchema := tool["inputSchema"].(map[string]any)
	properties := inputSchema["properties"].(map[string]any)

	dryRunProp, ok := properties["dry_run"].(map[string]any)
	require.True(t, ok, "dry_run property should exist")
	assert.Equal(t, "boolean", dryRunProp["type"], "boolean type should map to boolean")
}

// TestAddRepoParameterIfNeededCreatesIssueWithRepos tests that repo param is added for create_issue
// when allowed_repos is configured.
func TestAddRepoParameterIfNeededCreatesIssueWithRepos(t *testing.T) {
	tool := map[string]any{
		"name": "create_issue",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{"type": "string"},
			},
		},
	}

	safeOutputs := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			AllowedRepos:   []string{"org/repo1", "org/repo2"},
			TargetRepoSlug: "org/repo1",
		},
	}

	addRepoParameterIfNeeded(tool, "create_issue", safeOutputs)

	inputSchema := tool["inputSchema"].(map[string]any)
	properties := inputSchema["properties"].(map[string]any)

	repoProp, ok := properties["repo"].(map[string]any)
	require.True(t, ok, "repo property should be added")
	assert.Equal(t, "string", repoProp["type"], "repo type should be string")
	assert.Contains(t, repoProp["description"].(string), "org/repo1", "description should include default repo")
}

// TestAddRepoParameterIfNeededNoAllowedRepos tests that repo param is NOT added when no allowed_repos.
func TestAddRepoParameterIfNeededNoAllowedRepos(t *testing.T) {
	tool := map[string]any{
		"name": "create_issue",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{"type": "string"},
			},
		},
	}

	safeOutputs := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{},
	}

	addRepoParameterIfNeeded(tool, "create_issue", safeOutputs)

	inputSchema := tool["inputSchema"].(map[string]any)
	properties := inputSchema["properties"].(map[string]any)

	_, hasRepo := properties["repo"]
	assert.False(t, hasRepo, "repo property should NOT be added when no allowed_repos")
}

// TestAddRepoParameterIfNeededWildcardTargetRepo tests that repo param is added for update_issue
// when target-repo is "*" (wildcard), even without allowed-repos.
func TestAddRepoParameterIfNeededWildcardTargetRepo(t *testing.T) {
	tool := map[string]any{
		"name": "update_issue",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{"type": "string"},
			},
		},
	}

	safeOutputs := &SafeOutputsConfig{
		UpdateIssues: &UpdateIssuesConfig{
			UpdateEntityConfig: UpdateEntityConfig{
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					TargetRepoSlug: "*",
				},
			},
		},
	}

	addRepoParameterIfNeeded(tool, "update_issue", safeOutputs)

	inputSchema := tool["inputSchema"].(map[string]any)
	properties := inputSchema["properties"].(map[string]any)

	repoProp, ok := properties["repo"].(map[string]any)
	require.True(t, ok, "repo property should be added when target-repo is wildcard")
	assert.Equal(t, "string", repoProp["type"], "repo type should be string")
	assert.Contains(t, repoProp["description"].(string), "Any repository can be targeted", "description should indicate any repo allowed")
}

// TestGenerateFilteredToolsJSONUpdateIssueWithWildcardTargetRepo tests that update_issue tool
// is generated in tools.json when target-repo is "*" (wildcard).
func TestGenerateFilteredToolsJSONUpdateIssueWithWildcardTargetRepo(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			UpdateIssues: &UpdateIssuesConfig{
				UpdateEntityConfig: UpdateEntityConfig{
					SafeOutputTargetConfig: SafeOutputTargetConfig{
						TargetRepoSlug: "*",
					},
				},
			},
		},
	}

	result, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "generateFilteredToolsJSON should not error")

	var tools []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &tools), "Result should be valid JSON")

	// Find update_issue tool
	var updateIssueTool map[string]any
	for _, tool := range tools {
		if tool["name"] == "update_issue" {
			updateIssueTool = tool
			break
		}
	}
	require.NotNil(t, updateIssueTool, "update_issue tool should be present when target-repo is wildcard")

	// Verify repo parameter is in the schema
	inputSchema, ok := updateIssueTool["inputSchema"].(map[string]any)
	require.True(t, ok, "inputSchema should be present")
	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "properties should be present")
	_, hasRepo := properties["repo"]
	assert.True(t, hasRepo, "repo parameter should be added to update_issue when target-repo is wildcard")
}

// TestParseUpdateIssuesConfigWithWildcardTargetRepo tests that parseUpdateIssuesConfig
// returns a non-nil config when target-repo is "*".
func TestParseUpdateIssuesConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"update-issue": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parseUpdateIssuesConfig(outputMap)
	require.NotNil(t, result, "parseUpdateIssuesConfig should return non-nil for wildcard target-repo")
	assert.Equal(t, "*", result.TargetRepoSlug, "TargetRepoSlug should be '*'")
}

// TestGenerateDispatchWorkflowToolBasic tests basic dispatch workflow tool generation.
func TestGenerateDispatchWorkflowToolBasic(t *testing.T) {
	workflowInputs := map[string]any{
		"environment": map[string]any{
			"description": "Target environment",
			"type":        "choice",
			"options":     []any{"staging", "production"},
			"required":    true,
		},
	}

	tool := generateDispatchWorkflowTool("deploy-app", workflowInputs)

	assert.Equal(t, "deploy_app", tool["name"], "Tool name should be normalized")
	assert.Equal(t, "deploy-app", tool["_workflow_name"], "Internal workflow name should be preserved")
	assert.Contains(t, tool["description"].(string), "deploy-app", "Description should mention workflow name")

	inputSchema, ok := tool["inputSchema"].(map[string]any)
	require.True(t, ok, "inputSchema should be present")

	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "properties should be present")

	envProp, ok := properties["environment"].(map[string]any)
	require.True(t, ok, "environment property should exist")
	assert.Equal(t, "string", envProp["type"], "choice maps to string")
	assert.Equal(t, []any{"staging", "production"}, envProp["enum"], "enum values should match")
}

// TestGenerateDispatchWorkflowToolEmptyInputs tests dispatch workflow tool with no inputs.
func TestGenerateDispatchWorkflowToolEmptyInputs(t *testing.T) {
	tool := generateDispatchWorkflowTool("simple-workflow", make(map[string]any))

	assert.Equal(t, "simple_workflow", tool["name"], "Name should be normalized")

	inputSchema := tool["inputSchema"].(map[string]any)
	properties := inputSchema["properties"].(map[string]any)
	assert.Empty(t, properties, "Properties should be empty for workflow with no inputs")

	_, hasRequired := inputSchema["required"]
	assert.False(t, hasRequired, "required field should not be present when no required inputs")
}

// TestCheckAllEnabledToolsPresentAllMatch tests that no error is returned when all enabled tools are present.
func TestCheckAllEnabledToolsPresentAllMatch(t *testing.T) {
	enabledTools := map[string]bool{
		"create_issue": true,
		"add_comment":  true,
	}
	filteredTools := []map[string]any{
		{"name": "create_issue"},
		{"name": "add_comment"},
	}

	err := checkAllEnabledToolsPresent(enabledTools, filteredTools)
	assert.NoError(t, err, "should not error when all enabled tools are present")
}

// TestCheckAllEnabledToolsPresentMissing tests that an error is returned when a tool is missing.
func TestCheckAllEnabledToolsPresentMissing(t *testing.T) {
	enabledTools := map[string]bool{
		"create_issue":     true,
		"nonexistent_tool": true,
	}
	filteredTools := []map[string]any{
		{"name": "create_issue"},
	}

	err := checkAllEnabledToolsPresent(enabledTools, filteredTools)
	require.Error(t, err, "should error when an enabled tool is missing from filtered tools")
	assert.Contains(t, err.Error(), "nonexistent_tool", "error should mention the missing tool")
	assert.Contains(t, err.Error(), "compiler error", "error should be labelled as a compiler error")
	assert.Contains(t, err.Error(), "report this issue to the developer", "error should instruct user to report")
}

// TestCheckAllEnabledToolsPresentEmpty tests that no error is returned when there are no enabled tools.
func TestCheckAllEnabledToolsPresentEmpty(t *testing.T) {
	err := checkAllEnabledToolsPresent(map[string]bool{}, []map[string]any{})
	assert.NoError(t, err, "should not error with empty inputs")
}

// TestCheckAllEnabledToolsPresentMultipleMissing tests that multiple missing tools are all reported.
func TestCheckAllEnabledToolsPresentMultipleMissing(t *testing.T) {
	enabledTools := map[string]bool{
		"tool_a": true,
		"tool_b": true,
		"tool_c": true,
	}
	filteredTools := []map[string]any{
		{"name": "tool_a"},
	}

	err := checkAllEnabledToolsPresent(enabledTools, filteredTools)
	require.Error(t, err, "should error when multiple enabled tools are missing")
	assert.Contains(t, err.Error(), "tool_b", "error should mention tool_b")
	assert.Contains(t, err.Error(), "tool_c", "error should mention tool_c")
}

// TestGenerateFilteredToolsJSONWithStandardOutputs tests that standard safe outputs produce
// the expected tools in the filtered output (regression test for the completeness check).
func TestGenerateFilteredToolsJSONWithStandardOutputs(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
			MissingData:  &MissingDataConfig{},
			// MissingTool is the built-in safe-output type that lets the AI report that a
			// requested tool is unavailable; it is distinct from "a tool definition that is missing".
			MissingTool: &MissingToolConfig{},
			// NoOp (null) is the built-in fallback safe-output type for transparency.
			NoOp: &NoOpConfig{},
		},
	}

	result, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "should not error when all standard tools are present in static JSON")

	var tools []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &tools), "result should be valid JSON")

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok {
			toolNames[name] = true
		}
	}
	assert.True(t, toolNames["create_issue"], "create_issue should be present")
	assert.True(t, toolNames["missing_data"], "missing_data should be present")
	assert.True(t, toolNames["missing_tool"], "missing_tool should be present")
	assert.True(t, toolNames["noop"], "noop should be present")
}

// TestGenerateFilteredToolsJSONCustomJobs tests that custom job tools are included in filtered output.
func TestGenerateFilteredToolsJSONCustomJobs(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Jobs: map[string]*SafeJobConfig{
				"my_job": {
					Description: "My custom job",
					Inputs: map[string]*InputDefinition{
						"input1": {
							Type:        "string",
							Description: "First input",
							Required:    true,
						},
					},
				},
			},
		},
	}

	result, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "generateFilteredToolsJSON should not error")

	var tools []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &tools), "Result should be valid JSON")

	require.Len(t, tools, 1, "Should have exactly 1 tool")
	assert.Equal(t, "my_job", tools[0]["name"], "Tool name should match job name")
	assert.Equal(t, "My custom job", tools[0]["description"], "Description should match")
}

// TestGenerateFilteredToolsJSONSortsCustomJobs tests that custom jobs are sorted deterministically.
func TestGenerateFilteredToolsJSONSortsCustomJobs(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Jobs: map[string]*SafeJobConfig{
				"z_job": {Description: "Z job"},
				"a_job": {Description: "A job"},
				"m_job": {Description: "M job"},
			},
		},
	}

	result, err := generateFilteredToolsJSON(data, ".github/workflows/test.md")
	require.NoError(t, err, "generateFilteredToolsJSON should not error")

	var tools []map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &tools), "Result should be valid JSON")

	require.Len(t, tools, 3, "Should have 3 tools")
	assert.Equal(t, "a_job", tools[0]["name"], "First tool should be a_job (alphabetical)")
	assert.Equal(t, "m_job", tools[1]["name"], "Second tool should be m_job")
	assert.Equal(t, "z_job", tools[2]["name"], "Third tool should be z_job")
}
