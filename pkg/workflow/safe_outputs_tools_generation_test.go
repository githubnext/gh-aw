//go:build !integration

package workflow

import (
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

// TestAddRepoParameterIfNeededSpecificTargetRepoNoAllowedRepos tests that repo param is NOT added
// for update_issue when target-repo is a specific repo but allowed-repos is empty.
// The handler automatically routes to the configured target-repo, so the agent doesn't need to
// specify repo in the tool schema.
func TestAddRepoParameterIfNeededSpecificTargetRepoNoAllowedRepos(t *testing.T) {
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
					TargetRepoSlug: "org/target-repo",
				},
			},
		},
	}

	addRepoParameterIfNeeded(tool, "update_issue", safeOutputs)

	inputSchema := tool["inputSchema"].(map[string]any)
	properties := inputSchema["properties"].(map[string]any)

	_, hasRepo := properties["repo"]
	assert.False(t, hasRepo, "repo parameter should NOT be added when target-repo is specific and no allowed-repos")
}

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

// TestGenerateDispatchWorkflowToolRequiredSorted tests that the required array is always sorted.
// This ensures idempotent output regardless of map iteration order.
func TestGenerateDispatchWorkflowToolRequiredSorted(t *testing.T) {
	workflowInputs := map[string]any{
		"tracker_issue": map[string]any{
			"description": "Dashboard issue number to reference",
			"type":        "string",
			"required":    true,
		},
		"flag_key": map[string]any{
			"description": "The LaunchDarkly flag key to clean up",
			"type":        "string",
			"required":    true,
		},
		"optional_param": map[string]any{
			"description": "An optional parameter",
			"type":        "string",
			"required":    false,
		},
	}

	// Run multiple times to catch non-determinism from map iteration
	for i := range 10 {
		tool := generateDispatchWorkflowTool("cleanup-worker", workflowInputs)

		inputSchema, ok := tool["inputSchema"].(map[string]any)
		require.True(t, ok, "inputSchema should be present (iteration %d)", i)

		required, ok := inputSchema["required"].([]string)
		require.True(t, ok, "required should be []string (iteration %d)", i)

		assert.Equal(t, []string{"flag_key", "tracker_issue"}, required,
			"required array should be sorted alphabetically (iteration %d)", i)
	}
}

// TestGenerateFilteredToolsJSONWithStandardOutputs tests that standard safe outputs produce
// the expected tools in the filtered output (regression test for the completeness check).
