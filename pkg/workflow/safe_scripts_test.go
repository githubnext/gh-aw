//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSafeScriptsConfig verifies parsing of safe-scripts configuration
func TestParseSafeScriptsConfig(t *testing.T) {
	scriptsMap := map[string]any{
		"slack-post-message": map[string]any{
			"name":        "Post Slack Message",
			"description": "Post a message to a Slack channel",
			"inputs": map[string]any{
				"channel": map[string]any{
					"description": "Slack channel name",
					"required":    true,
					"type":        "string",
				},
				"message": map[string]any{
					"description": "Message content",
					"required":    true,
					"type":        "string",
				},
			},
			"script": `async function main(config = {}) {
  return async function handleSlackPostMessage(message, resolvedTemporaryIds) {
    return { success: true };
  };
}
module.exports = { main };`,
		},
	}

	result := parseSafeScriptsConfig(scriptsMap)

	require.NotNil(t, result, "Should return non-nil result")
	require.Len(t, result, 1, "Should have one script")

	script, exists := result["slack-post-message"]
	require.True(t, exists, "Should have slack-post-message script")
	assert.Equal(t, "Post Slack Message", script.Name, "Name should match")
	assert.Equal(t, "Post a message to a Slack channel", script.Description, "Description should match")
	assert.Contains(t, script.Script, "async function main", "Script should contain main function")

	require.Len(t, script.Inputs, 2, "Should have 2 inputs")

	channelInput, ok := script.Inputs["channel"]
	require.True(t, ok, "Should have channel input")
	assert.Equal(t, "string", channelInput.Type, "Channel type should be string")
	assert.True(t, channelInput.Required, "Channel should be required")

	messageInput, ok := script.Inputs["message"]
	require.True(t, ok, "Should have message input")
	assert.Equal(t, "string", messageInput.Type, "Message type should be string")
	assert.True(t, messageInput.Required, "Message should be required")
}

// TestParseSafeScriptsConfigNilMap verifies nil input handling
func TestParseSafeScriptsConfigNilMap(t *testing.T) {
	result := parseSafeScriptsConfig(nil)
	assert.Nil(t, result, "Should return nil for nil map")
}

// TestExtractSafeScriptsFromFrontmatter verifies extraction from frontmatter
func TestExtractSafeScriptsFromFrontmatter(t *testing.T) {
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"scripts": map[string]any{
				"my-handler": map[string]any{
					"description": "A custom handler",
					"script":      "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
				},
			},
		},
	}

	result := extractSafeScriptsFromFrontmatter(frontmatter)

	require.Len(t, result, 1, "Should have one script")
	script, exists := result["my-handler"]
	require.True(t, exists, "Should have my-handler script")
	assert.Equal(t, "A custom handler", script.Description, "Description should match")
}

// TestExtractSafeScriptsFromFrontmatterEmpty verifies empty result when no scripts
func TestExtractSafeScriptsFromFrontmatterEmpty(t *testing.T) {
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{},
		},
	}

	result := extractSafeScriptsFromFrontmatter(frontmatter)
	assert.Empty(t, result, "Should return empty map when no scripts")
}

// TestBuildCustomSafeOutputScriptsJSON verifies JSON generation for script env var
func TestBuildCustomSafeOutputScriptsJSON(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Scripts: map[string]*SafeScriptConfig{
				"my-handler": {
					Description: "Custom handler",
					Script:      "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
				},
			},
		},
	}

	jsonStr := buildCustomSafeOutputScriptsJSON(data)
	require.NotEmpty(t, jsonStr, "Should produce non-empty JSON")

	var mapping map[string]string
	err := json.Unmarshal([]byte(jsonStr), &mapping)
	require.NoError(t, err, "JSON should be valid")

	filename, exists := mapping["my_handler"]
	require.True(t, exists, "Should have my_handler key (normalized with underscore)")
	assert.Equal(t, "safe_output_script_my_handler.cjs", filename, "Filename should match expected pattern")
}

// TestBuildCustomSafeOutputScriptsJSONNormalization verifies name normalization with dashes
func TestBuildCustomSafeOutputScriptsJSONNormalization(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Scripts: map[string]*SafeScriptConfig{
				"slack-post-message": {
					Script: "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
				},
				"notify-team": {
					Script: "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
				},
			},
		},
	}

	jsonStr := buildCustomSafeOutputScriptsJSON(data)
	require.NotEmpty(t, jsonStr, "Should produce non-empty JSON")

	var mapping map[string]string
	err := json.Unmarshal([]byte(jsonStr), &mapping)
	require.NoError(t, err, "JSON should be valid")

	// Check names are normalized to underscores
	assert.Contains(t, mapping, "slack_post_message", "Should normalize dashes to underscores")
	assert.Contains(t, mapping, "notify_team", "Should normalize dashes to underscores")
	assert.Equal(t, "safe_output_script_slack_post_message.cjs", mapping["slack_post_message"], "Filename should match")
	assert.Equal(t, "safe_output_script_notify_team.cjs", mapping["notify_team"], "Filename should match")
}

// TestBuildCustomSafeOutputScriptsJSONEmpty verifies empty output when no scripts
func TestBuildCustomSafeOutputScriptsJSONEmpty(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{},
	}
	assert.Empty(t, buildCustomSafeOutputScriptsJSON(data), "Should return empty for no scripts")

	dataNil := &WorkflowData{SafeOutputs: nil}
	assert.Empty(t, buildCustomSafeOutputScriptsJSON(dataNil), "Should return empty for nil SafeOutputs")
}

// TestGenerateCustomScriptToolDefinition verifies MCP tool definition for scripts
func TestGenerateCustomScriptToolDefinition(t *testing.T) {
	scriptConfig := &SafeScriptConfig{
		Description: "Post a message to Slack",
		Inputs: map[string]*InputDefinition{
			"channel": {
				Description: "Target Slack channel",
				Required:    true,
				Type:        "string",
			},
			"message": {
				Description: "Message text",
				Required:    true,
				Type:        "string",
			},
		},
		Script: "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
	}

	tool := generateCustomScriptToolDefinition("slack_post_message", scriptConfig)

	assert.Equal(t, "slack_post_message", tool["name"], "Tool name should match")
	assert.Equal(t, "Post a message to Slack", tool["description"], "Description should match")

	inputSchema, ok := tool["inputSchema"].(map[string]any)
	require.True(t, ok, "Should have inputSchema")
	assert.Equal(t, "object", inputSchema["type"], "Schema type should be object")
	assert.Equal(t, false, inputSchema["additionalProperties"], "Should not allow additional properties")

	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "Should have properties")
	assert.Len(t, properties, 2, "Should have 2 properties")

	required, ok := inputSchema["required"].([]string)
	require.True(t, ok, "Should have required field")
	assert.Len(t, required, 2, "Should have 2 required fields")
	assert.Contains(t, required, "channel", "Should require channel")
	assert.Contains(t, required, "message", "Should require message")
}

// TestScriptToolsInFilteredJSON verifies scripts appear in the filtered tools JSON
func TestScriptToolsInFilteredJSON(t *testing.T) {
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Scripts: map[string]*SafeScriptConfig{
				"my-custom-handler": {
					Description: "A custom script handler",
					Inputs: map[string]*InputDefinition{
						"target": {
							Description: "Target to process",
							Required:    true,
							Type:        "string",
						},
					},
					Script: "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
				},
			},
		},
	}

	toolsJSON, err := generateFilteredToolsJSON(workflowData, ".github/workflows/test.md")
	require.NoError(t, err, "Should generate tools JSON without error")

	var tools []map[string]any
	err = json.Unmarshal([]byte(toolsJSON), &tools)
	require.NoError(t, err, "Tools JSON should be parseable")

	var customTool map[string]any
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok && name == "my_custom_handler" {
			customTool = tool
			break
		}
	}
	require.NotNil(t, customTool, "Should find my_custom_handler tool in tools JSON")
	assert.Equal(t, "A custom script handler", customTool["description"], "Description should match")

	inputSchema, ok := customTool["inputSchema"].(map[string]any)
	require.True(t, ok, "Should have inputSchema")

	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "Should have properties")
	assert.Contains(t, properties, "target", "Should have target property")
}

// TestBuildCustomScriptFilesStep verifies the generated step writes scripts to files
func TestBuildCustomScriptFilesStep(t *testing.T) {
	scripts := map[string]*SafeScriptConfig{
		"my-handler": {
			Script: "async function main(config) {\n  return async (msg) => ({ success: true });\n}\nmodule.exports = { main };",
		},
	}

	steps := buildCustomScriptFilesStep(scripts)

	require.NotEmpty(t, steps, "Should produce steps")

	fullYAML := strings.Join(steps, "")

	assert.Contains(t, fullYAML, "Setup Safe Output Custom Scripts", "Should have setup step name")
	assert.Contains(t, fullYAML, "safe_output_script_my_handler.cjs", "Should reference the output filename")
	assert.Contains(t, fullYAML, "GH_AW_SAFE_OUTPUT_SCRIPT_MY_HANDLER_EOF", "Should use correct heredoc delimiter")
	assert.Contains(t, fullYAML, "async function main", "Should include script content")
}

// TestBuildCustomScriptFilesStepEmpty verifies nil return for empty scripts
func TestBuildCustomScriptFilesStepEmpty(t *testing.T) {
	steps := buildCustomScriptFilesStep(nil)
	assert.Nil(t, steps, "Should return nil for empty scripts")

	stepsEmpty := buildCustomScriptFilesStep(map[string]*SafeScriptConfig{})
	assert.Nil(t, stepsEmpty, "Should return nil for empty map")
}

// TestHandlerManagerStepIncludesScriptsEnvVar verifies GH_AW_SAFE_OUTPUT_SCRIPTS in handler manager
func TestHandlerManagerStepIncludesScriptsEnvVar(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
			Scripts: map[string]*SafeScriptConfig{
				"my-script": {
					Description: "Custom script",
					Script:      "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
				},
			},
		},
	}

	steps := compiler.buildHandlerManagerStep(workflowData)
	fullYAML := strings.Join(steps, "")

	assert.Contains(t, fullYAML, "GH_AW_SAFE_OUTPUT_SCRIPTS", "Should include GH_AW_SAFE_OUTPUT_SCRIPTS env var")
	assert.Contains(t, fullYAML, "my_script", "Should include normalized script name")
	assert.Contains(t, fullYAML, "safe_output_script_my_script.cjs", "Should include script filename")
}

// TestHandlerManagerStepNoScriptsEnvVar verifies GH_AW_SAFE_OUTPUT_SCRIPTS absent when no scripts
func TestHandlerManagerStepNoScriptsEnvVar(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	steps := compiler.buildHandlerManagerStep(workflowData)
	fullYAML := strings.Join(steps, "")

	assert.NotContains(t, fullYAML, "GH_AW_SAFE_OUTPUT_SCRIPTS", "Should not include GH_AW_SAFE_OUTPUT_SCRIPTS env var when no scripts")
}

// TestSafeOutputsConfigIncludesScripts verifies extractSafeOutputsConfig handles scripts
func TestSafeOutputsConfigIncludesScripts(t *testing.T) {
	compiler := NewCompiler()
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"scripts": map[string]any{
				"post-webhook": map[string]any{
					"name":        "Post Webhook",
					"description": "Post a webhook notification",
					"inputs": map[string]any{
						"url": map[string]any{
							"description": "Webhook URL",
							"required":    true,
							"type":        "string",
						},
					},
					"script": "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
				},
			},
		},
	}

	config := compiler.extractSafeOutputsConfig(frontmatter)

	require.NotNil(t, config, "Should extract config")
	require.Len(t, config.Scripts, 1, "Should have 1 script")

	script, exists := config.Scripts["post-webhook"]
	require.True(t, exists, "Should have post-webhook script")
	assert.Equal(t, "Post Webhook", script.Name, "Name should match")
	assert.Equal(t, "Post a webhook notification", script.Description, "Description should match")
	require.Len(t, script.Inputs, 1, "Should have 1 input")
}

// TestHasAnySafeOutputEnabledWithScripts verifies Scripts are detected as enabled
func TestHasAnySafeOutputEnabledWithScripts(t *testing.T) {
	config := &SafeOutputsConfig{
		Scripts: map[string]*SafeScriptConfig{
			"my-script": {
				Script: "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
			},
		},
	}
	assert.True(t, hasAnySafeOutputEnabled(config), "Should detect scripts as enabled safe outputs")
}

// TestHasNonBuiltinSafeOutputsEnabledWithScripts verifies Scripts count as non-builtin
func TestHasNonBuiltinSafeOutputsEnabledWithScripts(t *testing.T) {
	config := &SafeOutputsConfig{
		Scripts: map[string]*SafeScriptConfig{
			"my-script": {
				Script: "module.exports = { main: async (c) => async (m) => ({ success: true }) };",
			},
		},
	}
	assert.True(t, hasNonBuiltinSafeOutputsEnabled(config), "Scripts should count as non-builtin safe outputs")
}
