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

// TestParseDispatchRepositoryConfig_SingleTool tests parsing a single dispatch_repository tool
func TestParseDispatchRepositoryConfig_SingleTool(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"dispatch_repository": map[string]any{
			"trigger_ci": map[string]any{
				"description": "Trigger CI in another repository",
				"workflow":    "ci.yml",
				"event_type":  "ci_trigger",
				"repository":  "org/target-repo",
				"max":         1,
			},
		},
	}

	config := compiler.parseDispatchRepositoryConfig(outputMap)
	require.NotNil(t, config, "Config should be parsed")
	require.Len(t, config.Tools, 1, "Should have 1 tool")

	tool := config.Tools["trigger_ci"]
	require.NotNil(t, tool, "trigger_ci tool should be present")
	assert.Equal(t, "Trigger CI in another repository", tool.Description)
	assert.Equal(t, "ci.yml", tool.Workflow)
	assert.Equal(t, "ci_trigger", tool.EventType)
	assert.Equal(t, "org/target-repo", tool.Repository)
	assert.Equal(t, strPtr("1"), tool.Max)
}

// TestParseDispatchRepositoryConfig_MultipleTools tests parsing multiple tools
func TestParseDispatchRepositoryConfig_MultipleTools(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"dispatch_repository": map[string]any{
			"trigger_ci": map[string]any{
				"workflow":   "ci.yml",
				"event_type": "ci_trigger",
				"repository": "org/target-repo",
			},
			"notify_service": map[string]any{
				"description": "Notify external service",
				"workflow":    "notify.yml",
				"event_type":  "notify_event",
				"allowed_repositories": []any{
					"org/service-repo",
					"org/backup-repo",
				},
				"inputs": map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "Notification message",
					},
				},
				"max": 2,
			},
		},
	}

	config := compiler.parseDispatchRepositoryConfig(outputMap)
	require.NotNil(t, config, "Config should be parsed")
	require.Len(t, config.Tools, 2, "Should have 2 tools")

	triggerCI := config.Tools["trigger_ci"]
	require.NotNil(t, triggerCI, "trigger_ci should be present")
	assert.Equal(t, "ci.yml", triggerCI.Workflow)
	assert.Equal(t, "ci_trigger", triggerCI.EventType)
	assert.Equal(t, "org/target-repo", triggerCI.Repository)

	notifyService := config.Tools["notify_service"]
	require.NotNil(t, notifyService, "notify_service should be present")
	assert.Equal(t, "notify.yml", notifyService.Workflow)
	assert.Equal(t, "notify_event", notifyService.EventType)
	assert.Equal(t, []string{"org/service-repo", "org/backup-repo"}, notifyService.AllowedRepositories)
	assert.NotNil(t, notifyService.Inputs, "Inputs should be present")
	assert.Equal(t, strPtr("2"), notifyService.Max)
}

// TestParseDispatchRepositoryConfig_DashAlias tests that "dispatch-repository" (dash) also works
func TestParseDispatchRepositoryConfig_DashAlias(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"dispatch-repository": map[string]any{
			"trigger_ci": map[string]any{
				"workflow":   "ci.yml",
				"event_type": "ci_trigger",
				"repository": "org/target-repo",
			},
		},
	}

	config := compiler.parseDispatchRepositoryConfig(outputMap)
	require.NotNil(t, config, "Config should be parsed from dash form")
	require.Len(t, config.Tools, 1, "Should have 1 tool")
}

// TestParseDispatchRepositoryConfig_Absent tests that nil is returned when key is absent
func TestParseDispatchRepositoryConfig_Absent(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"create_issue": map[string]any{},
	}

	config := compiler.parseDispatchRepositoryConfig(outputMap)
	assert.Nil(t, config, "Config should be nil when dispatch_repository is absent")
}

// TestParseDispatchRepositoryConfig_MaxCap tests that max is capped at 50
func TestParseDispatchRepositoryConfig_MaxCap(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	outputMap := map[string]any{
		"dispatch_repository": map[string]any{
			"trigger_ci": map[string]any{
				"workflow":   "ci.yml",
				"event_type": "ci_trigger",
				"repository": "org/target-repo",
				"max":        100,
			},
		},
	}

	config := compiler.parseDispatchRepositoryConfig(outputMap)
	require.NotNil(t, config)
	assert.Equal(t, strPtr("50"), config.Tools["trigger_ci"].Max, "Max should be capped at 50")
}

// TestValidateDispatchRepository_Valid tests that valid config passes validation
func TestValidateDispatchRepository_Valid(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowPath := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"trigger_ci": {
						Workflow:   "ci.yml",
						EventType:  "ci_trigger",
						Repository: "org/target-repo",
					},
				},
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	assert.NoError(t, err, "Validation should succeed for valid config")
}

// TestValidateDispatchRepository_MissingWorkflow tests error when workflow field is missing
func TestValidateDispatchRepository_MissingWorkflow(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowPath := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"trigger_ci": {
						// workflow field is missing
						EventType:  "ci_trigger",
						Repository: "org/target-repo",
					},
				},
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	require.Error(t, err, "Validation should fail when workflow is missing")
	assert.Contains(t, err.Error(), "workflow", "Error should mention workflow field")
}

// TestValidateDispatchRepository_MissingEventType tests error when event_type field is missing
func TestValidateDispatchRepository_MissingEventType(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowPath := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"trigger_ci": {
						Workflow: "ci.yml",
						// event_type is missing
						Repository: "org/target-repo",
					},
				},
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	require.Error(t, err, "Validation should fail when event_type is missing")
	assert.Contains(t, err.Error(), "event_type", "Error should mention event_type field")
}

// TestValidateDispatchRepository_MissingRepository tests error when no repository is specified
func TestValidateDispatchRepository_MissingRepository(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowPath := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"trigger_ci": {
						Workflow:  "ci.yml",
						EventType: "ci_trigger",
						// no repository or allowed_repositories
					},
				},
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	require.Error(t, err, "Validation should fail when no repository is specified")
	assert.Contains(t, err.Error(), "repository", "Error should mention repository")
}

// TestValidateDispatchRepository_AllowedRepositories tests valid config with allowed_repositories
func TestValidateDispatchRepository_AllowedRepositories(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowPath := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"notify_service": {
						Workflow:            "notify.yml",
						EventType:           "notify_event",
						AllowedRepositories: []string{"org/service-repo", "org/backup-repo"},
					},
				},
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	assert.NoError(t, err, "Validation should succeed with allowed_repositories")
}

// TestValidateDispatchRepository_InvalidRepoFormat tests error for malformed repository slug
func TestValidateDispatchRepository_InvalidRepoFormat(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowPath := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"trigger_ci": {
						Workflow:   "ci.yml",
						EventType:  "ci_trigger",
						Repository: "not-a-valid-repo-format", // missing slash
					},
				},
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	require.Error(t, err, "Validation should fail for invalid repository format")
	assert.Contains(t, err.Error(), "invalid", "Error should mention invalid format")
}

// TestValidateDispatchRepository_GitHubExpression tests that GitHub Actions expressions are accepted
func TestValidateDispatchRepository_GitHubExpression(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowPath := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"trigger_ci": {
						Workflow:   "ci.yml",
						EventType:  "ci_trigger",
						Repository: "${{ github.repository }}", // expression
					},
				},
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	assert.NoError(t, err, "GitHub Actions expressions should be accepted without format validation")
}

// TestValidateDispatchRepository_EmptyTools tests error when no tools are defined
func TestValidateDispatchRepository_EmptyTools(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "dispatcher.md")
	err := os.WriteFile(workflowPath, []byte("---\non: issues\n---\ntest"), 0644)
	require.NoError(t, err)

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{}, // empty
			},
		},
	}

	err = compiler.validateDispatchRepository(workflowData, workflowPath)
	require.Error(t, err, "Validation should fail with empty tools map")
	assert.Contains(t, err.Error(), "at least one dispatch tool", "Error should mention tools requirement")
}

// TestValidateDispatchRepository_NilConfig tests that nil config is OK (no-op)
func TestValidateDispatchRepository_NilConfig(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			// DispatchRepository is nil - not configured
		},
	}

	err := compiler.validateDispatchRepository(workflowData, "/tmp/test.md")
	assert.NoError(t, err, "Nil config should not cause an error")
}

// TestGenerateDispatchRepositoryTool tests that tool definitions are generated correctly
func TestGenerateDispatchRepositoryTool(t *testing.T) {
	toolConfig := &DispatchRepositoryToolConfig{
		Description: "Trigger CI in another repository",
		Workflow:    "ci.yml",
		EventType:   "ci_trigger",
		Repository:  "org/target-repo",
		Inputs: map[string]any{
			"environment": map[string]any{
				"type":        "choice",
				"description": "Target environment",
				"options":     []any{"staging", "production"},
				"default":     "staging",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Optional message",
			},
		},
	}

	tool := generateDispatchRepositoryTool("trigger_ci", toolConfig)

	assert.Equal(t, "trigger_ci", tool["name"], "Tool name should match key")
	assert.NotEmpty(t, tool["description"], "Tool should have a description")
	assert.Equal(t, "trigger_ci", tool["_dispatch_repository_tool"], "Should have routing metadata")

	inputSchema, ok := tool["inputSchema"].(map[string]any)
	require.True(t, ok, "Tool should have inputSchema")

	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "inputSchema should have properties")

	assert.Contains(t, properties, "environment", "Should have environment property")
	assert.Contains(t, properties, "message", "Should have message property")

	envProp, ok := properties["environment"].(map[string]any)
	require.True(t, ok, "environment property should be a map")
	assert.Equal(t, "string", envProp["type"])
	assert.Contains(t, envProp, "enum", "choice type should have enum")
	assert.Equal(t, "staging", envProp["default"])
}

// TestGenerateDispatchRepositoryTool_NameNormalization tests underscore normalization
func TestGenerateDispatchRepositoryTool_NameNormalization(t *testing.T) {
	toolConfig := &DispatchRepositoryToolConfig{
		Workflow:   "ci.yml",
		EventType:  "ci_trigger",
		Repository: "org/target-repo",
	}

	tool := generateDispatchRepositoryTool("trigger-ci-workflow", toolConfig)
	assert.Equal(t, "trigger_ci_workflow", tool["name"], "Dashes should be normalized to underscores")
}

// TestDispatchRepositoryConfigSerialization tests that config serializes to JSON correctly
func TestDispatchRepositoryConfigSerialization(t *testing.T) {
	max1 := "1"
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"trigger_ci": {
						Description: "Trigger CI",
						Workflow:    "ci.yml",
						EventType:   "ci_trigger",
						Repository:  "org/target-repo",
						Max:         &max1,
					},
					"notify_service": {
						Workflow:            "notify.yml",
						EventType:           "notify_event",
						AllowedRepositories: []string{"org/service-repo"},
						Max:                 &max1,
					},
				},
			},
		},
	}

	configJSON := generateSafeOutputsConfig(workflowData)
	require.NotEmpty(t, configJSON, "Config JSON should not be empty")

	var config map[string]any
	err := json.Unmarshal([]byte(configJSON), &config)
	require.NoError(t, err, "Config JSON should be valid")

	dispatchRepo, ok := config["dispatch_repository"].(map[string]any)
	require.True(t, ok, "dispatch_repository should be in config")

	tools, ok := dispatchRepo["tools"].(map[string]any)
	require.True(t, ok, "tools should be in dispatch_repository config")

	assert.Contains(t, tools, "trigger_ci", "trigger_ci tool should be in config")
	assert.Contains(t, tools, "notify_service", "notify_service tool should be in config")

	triggerCIConfig, ok := tools["trigger_ci"].(map[string]any)
	require.True(t, ok, "trigger_ci config should be a map")
	assert.Equal(t, "ci.yml", triggerCIConfig["workflow"])
	assert.Equal(t, "ci_trigger", triggerCIConfig["event_type"])
	assert.Equal(t, "org/target-repo", triggerCIConfig["repository"])

	notifyConfig, ok := tools["notify_service"].(map[string]any)
	require.True(t, ok, "notify_service config should be a map")
	assert.Equal(t, "notify.yml", notifyConfig["workflow"])
	allowedRepos, ok := notifyConfig["allowed_repositories"].([]any)
	require.True(t, ok, "allowed_repositories should be present")
	assert.Contains(t, allowedRepos, "org/service-repo")
}

// TestDispatchRepositoryInWorkflowCompilation tests end-to-end compilation
func TestDispatchRepositoryInWorkflowCompilation(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	workflowContent := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch_repository:
    trigger_ci:
      description: Trigger CI in another repository
      workflow: ci.yml
      event_type: ci_trigger
      repository: org/target-repo
      max: 1
---

# Dispatch Repository Workflow

This workflow dispatches repository events.
`
	workflowFile := filepath.Join(awDir, "my-workflow.md")
	err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
	require.NoError(t, err)

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(awDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldDir) }()

	workflowData, err := compiler.ParseWorkflowFile("my-workflow.md")
	require.NoError(t, err, "Should parse workflow successfully")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.DispatchRepository, "DispatchRepository should not be nil")

	assert.Len(t, workflowData.SafeOutputs.DispatchRepository.Tools, 1)

	tool := workflowData.SafeOutputs.DispatchRepository.Tools["trigger_ci"]
	require.NotNil(t, tool)
	assert.Equal(t, "Trigger CI in another repository", tool.Description)
	assert.Equal(t, "ci.yml", tool.Workflow)
	assert.Equal(t, "ci_trigger", tool.EventType)
	assert.Equal(t, "org/target-repo", tool.Repository)
}

// TestDispatchRepositoryValidation_InCompiler tests validation runs during compilation
func TestDispatchRepositoryValidation_InCompiler(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err)

	// Missing workflow field should fail compilation
	workflowContent := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch_repository:
    trigger_ci:
      event_type: ci_trigger
      repository: org/target-repo
---

# Invalid Dispatch Repository Workflow
`
	workflowFile := filepath.Join(awDir, "invalid-workflow.md")
	err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
	require.NoError(t, err)

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldDir) }()

	err = compiler.CompileWorkflow(workflowFile)
	require.Error(t, err, "Compilation should fail due to missing workflow field")
	assert.Contains(t, err.Error(), "dispatch_repository", "Error should mention dispatch_repository")
	assert.Contains(t, err.Error(), "workflow", "Error should mention workflow field")
}
