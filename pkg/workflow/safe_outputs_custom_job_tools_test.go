package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomJobToolsInToolsJSON verifies that custom safe-output jobs
// are properly included in the tools.json file with correct MCP tool schema
func TestCustomJobToolsInToolsJSON(t *testing.T) {
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Jobs: map[string]*SafeJobConfig{
				"test_environment": {
					Description: "A test job with choice input",
					Inputs: map[string]*InputDefinition{
						"environment": {
							Description: "Target environment",
							Required:    true,
							Type:        "choice",
							Options:     []string{"staging", "production"},
						},
						"test_type": {
							Description: "Type of test to run",
							Required:    true,
							Type:        "choice",
							Options:     []string{"smoke", "integration", "e2e"},
						},
					},
					Output: "Environment test completed successfully",
				},
			},
		},
	}

	// Generate the tools JSON
	toolsJSON, err := generateFilteredToolsJSON(workflowData)
	require.NoError(t, err, "Should generate tools JSON")
	
	// Parse the JSON
	var tools []map[string]any
	err = json.Unmarshal([]byte(toolsJSON), &tools)
	require.NoError(t, err, "Should parse tools JSON: %s", toolsJSON)
	
	// Find the test_environment tool
	var testEnvTool map[string]any
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok && name == "test_environment" {
			testEnvTool = tool
			break
		}
	}
	
	require.NotNil(t, testEnvTool, "Should find test_environment tool in tools.json")
	
	// Verify the tool structure
	assert.Equal(t, "test_environment", testEnvTool["name"], "Tool name should be test_environment")
	assert.Equal(t, "A test job with choice input", testEnvTool["description"], "Description should match")
	
	// Verify the input schema
	inputSchema, ok := testEnvTool["inputSchema"].(map[string]any)
	require.True(t, ok, "Should have inputSchema")
	
	assert.Equal(t, "object", inputSchema["type"], "Schema type should be object")
	assert.Equal(t, false, inputSchema["additionalProperties"], "Should not allow additional properties")
	
	// Verify properties
	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "Should have properties")
	
	// Check environment property
	envProp, ok := properties["environment"].(map[string]any)
	require.True(t, ok, "Should have environment property")
	assert.Equal(t, "string", envProp["type"], "Environment type should be string (choice)")
	assert.Equal(t, "Target environment", envProp["description"], "Environment description should match")
	
	envEnum, ok := envProp["enum"].([]any)
	require.True(t, ok, "Should have enum for environment")
	assert.Equal(t, 2, len(envEnum), "Should have 2 options")
	assert.Contains(t, envEnum, "staging", "Should contain staging option")
	assert.Contains(t, envEnum, "production", "Should contain production option")
	
	// Check test_type property
	testTypeProp, ok := properties["test_type"].(map[string]any)
	require.True(t, ok, "Should have test_type property")
	assert.Equal(t, "string", testTypeProp["type"], "Test type should be string (choice)")
	assert.Equal(t, "Type of test to run", testTypeProp["description"], "Test type description should match")
	
	testTypeEnum, ok := testTypeProp["enum"].([]any)
	require.True(t, ok, "Should have enum for test_type")
	assert.Equal(t, 3, len(testTypeEnum), "Should have 3 options")
	assert.Contains(t, testTypeEnum, "smoke", "Should contain smoke option")
	assert.Contains(t, testTypeEnum, "integration", "Should contain integration option")
	assert.Contains(t, testTypeEnum, "e2e", "Should contain e2e option")
	
	// Verify required fields
	required, ok := inputSchema["required"].([]any)
	require.True(t, ok, "Should have required array")
	assert.Equal(t, 2, len(required), "Should have 2 required fields")
	assert.Contains(t, required, "environment", "Environment should be required")
	assert.Contains(t, required, "test_type", "Test type should be required")
}

// TestCustomJobToolsWithDifferentInputTypes verifies that custom jobs
// with different input types are correctly converted to JSON Schema
func TestCustomJobToolsWithDifferentInputTypes(t *testing.T) {
	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			Jobs: map[string]*SafeJobConfig{
				"multi_input_job": {
					Description: "Job with multiple input types",
					Inputs: map[string]*InputDefinition{
						"name": {
							Description: "User name",
							Required:    true,
							Type:        "string",
							Default:     "Alice",
						},
						"count": {
							Description: "Number of items",
							Required:    false,
							Type:        "number",
							Default:     10,
						},
						"enabled": {
							Description: "Enable feature",
							Required:    true,
							Type:        "boolean",
						},
						"mode": {
							Description: "Operation mode",
							Required:    false,
							Type:        "choice",
							Options:     []string{"fast", "slow", "medium"},
							Default:     "medium",
						},
					},
				},
			},
		},
	}

	// Generate the tools JSON
	toolsJSON, err := generateFilteredToolsJSON(workflowData)
	require.NoError(t, err, "Should generate tools JSON")
	
	// Parse the JSON
	var tools []map[string]any
	err = json.Unmarshal([]byte(toolsJSON), &tools)
	require.NoError(t, err, "Should parse tools JSON")
	
	// Find the multi_input_job tool
	var jobTool map[string]any
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok && name == "multi_input_job" {
			jobTool = tool
			break
		}
	}
	
	require.NotNil(t, jobTool, "Should find multi_input_job tool in tools.json")
	
	// Verify the input schema
	inputSchema, ok := jobTool["inputSchema"].(map[string]any)
	require.True(t, ok, "Should have inputSchema")
	
	properties, ok := inputSchema["properties"].(map[string]any)
	require.True(t, ok, "Should have properties")
	
	// Check string type
	nameProp, ok := properties["name"].(map[string]any)
	require.True(t, ok, "Should have name property")
	assert.Equal(t, "string", nameProp["type"], "Name should be string type")
	assert.Equal(t, "Alice", nameProp["default"], "Name should have default value")
	
	// Check number type
	countProp, ok := properties["count"].(map[string]any)
	require.True(t, ok, "Should have count property")
	assert.Equal(t, "number", countProp["type"], "Count should be number type")
	// Note: JSON numbers are float64 after unmarshal
	assert.Equal(t, float64(10), countProp["default"], "Count should have default value")
	
	// Check boolean type
	enabledProp, ok := properties["enabled"].(map[string]any)
	require.True(t, ok, "Should have enabled property")
	assert.Equal(t, "boolean", enabledProp["type"], "Enabled should be boolean type")
	
	// Check choice type
	modeProp, ok := properties["mode"].(map[string]any)
	require.True(t, ok, "Should have mode property")
	assert.Equal(t, "string", modeProp["type"], "Mode should be string type (choice)")
	assert.Equal(t, "medium", modeProp["default"], "Mode should have default value")
	
	modeEnum, ok := modeProp["enum"].([]any)
	require.True(t, ok, "Should have enum for mode")
	assert.Equal(t, 3, len(modeEnum), "Should have 3 options")
	
	// Verify required fields
	required, ok := inputSchema["required"].([]any)
	require.True(t, ok, "Should have required array")
	assert.Equal(t, 2, len(required), "Should have 2 required fields")
	assert.Contains(t, required, "name", "Name should be required")
	assert.Contains(t, required, "enabled", "Enabled should be required")
	assert.NotContains(t, required, "count", "Count should not be required")
	assert.NotContains(t, required, "mode", "Mode should not be required")
}
