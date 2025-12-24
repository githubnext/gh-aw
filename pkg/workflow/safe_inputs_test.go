package workflow

import (
	"strings"
	"testing"
)

func TestParseSafeInputs(t *testing.T) {
	tests := []struct {
		name          string
		frontmatter   map[string]any
		expectedTools int
		expectedNil   bool
	}{
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			expectedNil: true,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			expectedNil: true,
		},
		{
			name: "single javascript tool",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"search-issues": map[string]any{
						"description": "Search for issues",
						"script":      "return 'hello';",
						"inputs": map[string]any{
							"query": map[string]any{
								"type":        "string",
								"description": "Search query",
								"required":    true,
							},
						},
					},
				},
			},
			expectedTools: 1,
		},
		{
			name: "single shell tool",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"echo-message": map[string]any{
						"description": "Echo a message",
						"run":         "echo $INPUT_MESSAGE",
						"inputs": map[string]any{
							"message": map[string]any{
								"type":        "string",
								"description": "Message to echo",
								"default":     "Hello",
							},
						},
					},
				},
			},
			expectedTools: 1,
		},
		{
			name: "single python tool",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"analyze-data": map[string]any{
						"description": "Analyze data with Python",
						"py":          "import json\nprint(json.dumps({'result': 'success'}))",
						"inputs": map[string]any{
							"data": map[string]any{
								"type":        "string",
								"description": "Data to analyze",
								"required":    true,
							},
						},
					},
				},
			},
			expectedTools: 1,
		},
		{
			name: "multiple tools",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"tool1": map[string]any{
						"description": "Tool 1",
						"script":      "return 1;",
					},
					"tool2": map[string]any{
						"description": "Tool 2",
						"run":         "echo 2",
					},
				},
			},
			expectedTools: 2,
		},
		{
			name: "tool with env secrets",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"api-call": map[string]any{
						"description": "Call API",
						"script":      "return fetch(url);",
						"env": map[string]any{
							"API_KEY": "${{ secrets.API_KEY }}",
						},
					},
				},
			},
			expectedTools: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSafeInputs(tt.frontmatter)

			if tt.expectedNil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			if len(result.Tools) != tt.expectedTools {
				t.Errorf("Expected %d tools, got %d", tt.expectedTools, len(result.Tools))
			}
		})
	}
}

func TestHasSafeInputs(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeInputsConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty tools",
			config:   &SafeInputsConfig{Tools: map[string]*SafeInputToolConfig{}},
			expected: false,
		},
		{
			name: "with tools",
			config: &SafeInputsConfig{
				Tools: map[string]*SafeInputToolConfig{
					"test": {Name: "test", Description: "Test tool"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSafeInputs(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsSafeInputsEnabled(t *testing.T) {
	// Test config with tools
	configWithTools := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"test": {Name: "test", Description: "Test tool"},
		},
	}

	tests := []struct {
		name         string
		config       *SafeInputsConfig
		workflowData *WorkflowData
		expected     bool
	}{
		{
			name:         "nil config - not enabled",
			config:       nil,
			workflowData: nil,
			expected:     false,
		},
		{
			name:         "empty tools - not enabled",
			config:       &SafeInputsConfig{Tools: map[string]*SafeInputToolConfig{}},
			workflowData: nil,
			expected:     false,
		},
		{
			name:         "with tools - enabled by default",
			config:       configWithTools,
			workflowData: nil,
			expected:     true,
		},
		{
			name:   "with tools and feature flag enabled - enabled (backward compat)",
			config: configWithTools,
			workflowData: &WorkflowData{
				Features: map[string]any{"safe-inputs": true},
			},
			expected: true,
		},
		{
			name:   "with tools and feature flag disabled - still enabled (feature flag ignored)",
			config: configWithTools,
			workflowData: &WorkflowData{
				Features: map[string]any{"safe-inputs": false},
			},
			expected: true,
		},
		{
			name:   "with tools and other features - enabled",
			config: configWithTools,
			workflowData: &WorkflowData{
				Features: map[string]any{"other-feature": true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeInputsEnabled(tt.config, tt.workflowData)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsSafeInputsEnabledWithEnv(t *testing.T) {
	// Test config with tools
	configWithTools := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"test": {Name: "test", Description: "Test tool"},
		},
	}

	// Safe-inputs are enabled by default when configured, environment variable no longer needed
	t.Run("with tools - enabled regardless of GH_AW_FEATURES", func(t *testing.T) {
		t.Setenv("GH_AW_FEATURES", "safe-inputs")
		result := IsSafeInputsEnabled(configWithTools, nil)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})

	t.Run("with tools and GH_AW_FEATURES=other - still enabled", func(t *testing.T) {
		t.Setenv("GH_AW_FEATURES", "other")
		result := IsSafeInputsEnabled(configWithTools, nil)
		if !result {
			t.Errorf("Expected true, got false")
		}
	})
}

// TestParseSafeInputsAndExtractSafeInputsConfigConsistency verifies that ParseSafeInputs
// and extractSafeInputsConfig produce identical results for the same input.
// This ensures both functions use the shared helper correctly.
func TestParseSafeInputsAndExtractSafeInputsConfigConsistency(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
	}{
		{
			name:        "nil frontmatter",
			frontmatter: nil,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
		},
		{
			name: "single tool with all fields",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"search-issues": map[string]any{
						"description": "Search for issues",
						"script":      "return 'hello';",
						"inputs": map[string]any{
							"query": map[string]any{
								"type":        "string",
								"description": "Search query",
								"required":    true,
								"default":     "test",
							},
						},
						"env": map[string]any{
							"API_KEY": "${{ secrets.API_KEY }}",
						},
					},
				},
			},
		},
		{
			name: "multiple tools with different types",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"js-tool": map[string]any{
						"description": "JavaScript tool",
						"script":      "return 1;",
					},
					"shell-tool": map[string]any{
						"description": "Shell tool",
						"run":         "echo hello",
					},
					"python-tool": map[string]any{
						"description": "Python tool",
						"py":          "print('hello')",
					},
				},
			},
		},
		{
			name: "tool with complex inputs",
			frontmatter: map[string]any{
				"safe-inputs": map[string]any{
					"complex-tool": map[string]any{
						"description": "Complex tool",
						"script":      "return inputs;",
						"inputs": map[string]any{
							"string-param": map[string]any{
								"type":        "string",
								"description": "A string parameter",
							},
							"number-param": map[string]any{
								"type":        "number",
								"description": "A number parameter",
								"default":     42,
							},
							"bool-param": map[string]any{
								"type":        "boolean",
								"description": "A boolean parameter",
								"required":    true,
							},
						},
					},
				},
			},
		},
	}

	compiler := &Compiler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result1 := ParseSafeInputs(tt.frontmatter)
			result2 := compiler.extractSafeInputsConfig(tt.frontmatter)

			// Both should be nil or both non-nil
			if (result1 == nil) != (result2 == nil) {
				t.Errorf("Inconsistent nil results: ParseSafeInputs=%v, extractSafeInputsConfig=%v", result1 == nil, result2 == nil)
				return
			}

			if result1 == nil {
				return
			}

			// Compare number of tools
			if len(result1.Tools) != len(result2.Tools) {
				t.Errorf("Different number of tools: ParseSafeInputs=%d, extractSafeInputsConfig=%d", len(result1.Tools), len(result2.Tools))
				return
			}

			// Compare each tool
			for toolName, tool1 := range result1.Tools {
				tool2, exists := result2.Tools[toolName]
				if !exists {
					t.Errorf("Tool %s not found in extractSafeInputsConfig result", toolName)
					continue
				}

				if tool1.Name != tool2.Name {
					t.Errorf("Tool %s: Name mismatch: %s vs %s", toolName, tool1.Name, tool2.Name)
				}
				if tool1.Description != tool2.Description {
					t.Errorf("Tool %s: Description mismatch: %s vs %s", toolName, tool1.Description, tool2.Description)
				}
				if tool1.Script != tool2.Script {
					t.Errorf("Tool %s: Script mismatch: %s vs %s", toolName, tool1.Script, tool2.Script)
				}
				if tool1.Run != tool2.Run {
					t.Errorf("Tool %s: Run mismatch: %s vs %s", toolName, tool1.Run, tool2.Run)
				}
				if tool1.Py != tool2.Py {
					t.Errorf("Tool %s: Py mismatch: %s vs %s", toolName, tool1.Py, tool2.Py)
				}

				// Compare inputs
				if len(tool1.Inputs) != len(tool2.Inputs) {
					t.Errorf("Tool %s: Different number of inputs: %d vs %d", toolName, len(tool1.Inputs), len(tool2.Inputs))
					continue
				}

				for inputName, input1 := range tool1.Inputs {
					input2, exists := tool2.Inputs[inputName]
					if !exists {
						t.Errorf("Tool %s: Input %s not found in extractSafeInputsConfig result", toolName, inputName)
						continue
					}

					if input1.Type != input2.Type {
						t.Errorf("Tool %s, Input %s: Type mismatch: %s vs %s", toolName, inputName, input1.Type, input2.Type)
					}
					if input1.Description != input2.Description {
						t.Errorf("Tool %s, Input %s: Description mismatch: %s vs %s", toolName, inputName, input1.Description, input2.Description)
					}
					if input1.Required != input2.Required {
						t.Errorf("Tool %s, Input %s: Required mismatch: %v vs %v", toolName, inputName, input1.Required, input2.Required)
					}
					// Compare defaults (handle nil case)
					if (input1.Default == nil) != (input2.Default == nil) {
						t.Errorf("Tool %s, Input %s: Default nil mismatch: %v vs %v", toolName, inputName, input1.Default, input2.Default)
					}
				}

				// Compare env
				if len(tool1.Env) != len(tool2.Env) {
					t.Errorf("Tool %s: Different number of env vars: %d vs %d", toolName, len(tool1.Env), len(tool2.Env))
					continue
				}

				for envName, envValue1 := range tool1.Env {
					envValue2, exists := tool2.Env[envName]
					if !exists {
						t.Errorf("Tool %s: Env %s not found in extractSafeInputsConfig result", toolName, envName)
						continue
					}
					if envValue1 != envValue2 {
						t.Errorf("Tool %s, Env %s: Value mismatch: %s vs %s", toolName, envName, envValue1, envValue2)
					}
				}
			}
		})
	}
}

func TestGetSafeInputsEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		config      *SafeInputsConfig
		expectedLen int
		contains    []string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectedLen: 0,
		},
		{
			name: "tool with env",
			config: &SafeInputsConfig{
				Tools: map[string]*SafeInputToolConfig{
					"test": {
						Name: "test",
						Env: map[string]string{
							"API_KEY": "${{ secrets.API_KEY }}",
							"TOKEN":   "${{ secrets.TOKEN }}",
						},
					},
				},
			},
			expectedLen: 2,
			contains:    []string{"API_KEY", "TOKEN"},
		},
		{
			name: "multiple tools with shared env",
			config: &SafeInputsConfig{
				Tools: map[string]*SafeInputToolConfig{
					"tool1": {
						Name: "tool1",
						Env:  map[string]string{"API_KEY": "key1"},
					},
					"tool2": {
						Name: "tool2",
						Env:  map[string]string{"API_KEY": "key2"},
					},
				},
			},
			expectedLen: 1, // Should deduplicate
			contains:    []string{"API_KEY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSafeInputsEnvVars(tt.config)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected %d env vars, got %d: %v", tt.expectedLen, len(result), result)
			}

			for _, expected := range tt.contains {
				found := false
				for _, v := range result {
					if v == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to contain %s, got %v", expected, result)
				}
			}
		})
	}
}

func TestCollectSafeInputsSecrets(t *testing.T) {
	tests := []struct {
		name        string
		config      *SafeInputsConfig
		expectedLen int
	}{
		{
			name:        "nil config",
			config:      nil,
			expectedLen: 0,
		},
		{
			name: "tool with secrets",
			config: &SafeInputsConfig{
				Tools: map[string]*SafeInputToolConfig{
					"test": {
						Name: "test",
						Env: map[string]string{
							"API_KEY": "${{ secrets.API_KEY }}",
						},
					},
				},
			},
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectSafeInputsSecrets(tt.config)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected %d secrets, got %d", tt.expectedLen, len(result))
			}
		})
	}
}

func TestGenerateSafeInputsMCPServerScript(t *testing.T) {
	config := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"search-issues": {
				Name:        "search-issues",
				Description: "Search for issues in the repository",
				Script:      "return 'hello';",
				Inputs: map[string]*SafeInputParam{
					"query": {
						Type:        "string",
						Description: "Search query",
						Required:    true,
					},
				},
			},
			"echo-message": {
				Name:        "echo-message",
				Description: "Echo a message",
				Run:         "echo $INPUT_MESSAGE",
				Inputs: map[string]*SafeInputParam{
					"message": {
						Type:        "string",
						Description: "Message to echo",
						Default:     "Hello",
					},
				},
			},
			"analyze-data": {
				Name:        "analyze-data",
				Description: "Analyze data with Python",
				Py:          "import json\nprint(json.dumps({'result': 'success'}))",
				Inputs: map[string]*SafeInputParam{
					"data": {
						Type:        "string",
						Description: "Data to analyze",
						Required:    true,
					},
				},
			},
		},
	}

	// Test the entry point script
	script := generateSafeInputsMCPServerScript(config)

	// Check for HTTP server entry point structure
	if !strings.Contains(script, "safe_inputs_mcp_server_http.cjs") {
		t.Error("Script should reference the HTTP MCP server module")
	}

	if !strings.Contains(script, "startHttpServer") {
		t.Error("Script should use startHttpServer function")
	}

	if !strings.Contains(script, "tools.json") {
		t.Error("Script should reference tools.json configuration file")
	}

	if !strings.Contains(script, "/tmp/gh-aw/safe-inputs/logs") {
		t.Error("Script should specify log directory")
	}

	if !strings.Contains(script, "GH_AW_SAFE_INPUTS_PORT") {
		t.Error("Script should reference GH_AW_SAFE_INPUTS_PORT environment variable")
	}

	if !strings.Contains(script, "GH_AW_SAFE_INPUTS_API_KEY") {
		t.Error("Script should reference GH_AW_SAFE_INPUTS_API_KEY environment variable")
	}

	// Test the tools configuration JSON
	toolsJSON := generateSafeInputsToolsConfig(config)

	if !strings.Contains(toolsJSON, `"serverName": "safeinputs"`) {
		t.Error("Tools config should contain server name 'safeinputs'")
	}

	if !strings.Contains(toolsJSON, `"name": "search-issues"`) {
		t.Error("Tools config should contain search-issues tool")
	}

	if !strings.Contains(toolsJSON, `"name": "echo-message"`) {
		t.Error("Tools config should contain echo-message tool")
	}

	if !strings.Contains(toolsJSON, `"name": "analyze-data"`) {
		t.Error("Tools config should contain analyze-data tool")
	}

	// Check for JavaScript tool handler
	if !strings.Contains(toolsJSON, `"handler": "search-issues.cjs"`) {
		t.Error("Tools config should reference JavaScript tool handler file")
	}

	// Check for shell tool handler
	if !strings.Contains(toolsJSON, `"handler": "echo-message.sh"`) {
		t.Error("Tools config should reference shell script handler file")
	}

	// Check for Python tool handler
	if !strings.Contains(toolsJSON, `"handler": "analyze-data.py"`) {
		t.Error("Tools config should reference Python script handler file")
	}

	// Check for input schema
	if !strings.Contains(toolsJSON, `"description": "Search query"`) {
		t.Error("Tools config should contain input descriptions")
	}

	if !strings.Contains(toolsJSON, `"required"`) {
		t.Error("Tools config should contain required fields array")
	}
}

func TestGenerateSafeInputsToolsConfigWithEnv(t *testing.T) {
	config := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"github-query": {
				Name:        "github-query",
				Description: "Query GitHub with authentication",
				Run:         "gh repo view $INPUT_REPO",
				Inputs: map[string]*SafeInputParam{
					"repo": {
						Type:     "string",
						Required: true,
					},
				},
				Env: map[string]string{
					"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
					"API_KEY":  "${{ secrets.API_KEY }}",
				},
			},
		},
	}

	toolsJSON := generateSafeInputsToolsConfig(config)

	// Verify that env field is present in tools.json
	if !strings.Contains(toolsJSON, `"env"`) {
		t.Error("Tools config should contain env field")
	}

	// Verify that env contains environment variable names (not secrets or $ prefixes)
	// The values should be just the variable names like "GH_TOKEN": "GH_TOKEN"
	if !strings.Contains(toolsJSON, `"GH_TOKEN": "GH_TOKEN"`) {
		t.Error("Tools config should contain GH_TOKEN env variable name")
	}

	if !strings.Contains(toolsJSON, `"API_KEY": "API_KEY"`) {
		t.Error("Tools config should contain API_KEY env variable name")
	}

	// Verify that actual secret expressions are NOT in tools.json
	if strings.Contains(toolsJSON, "secrets.GITHUB_TOKEN") {
		t.Error("Tools config should NOT contain secret expressions")
	}

	if strings.Contains(toolsJSON, "secrets.API_KEY") {
		t.Error("Tools config should NOT contain secret expressions")
	}

	// Verify that $ prefix is not used (which might suggest variable expansion)
	if strings.Contains(toolsJSON, `"$GH_TOKEN"`) {
		t.Error("Tools config should NOT contain $ prefix in env values")
	}
}

func TestGenerateSafeInputJavaScriptToolScript(t *testing.T) {
	config := &SafeInputToolConfig{
		Name:        "test-tool",
		Description: "A test tool",
		Script:      "return inputs.value * 2;",
		Inputs: map[string]*SafeInputParam{
			"value": {
				Type:        "number",
				Description: "Value to double",
			},
		},
	}

	script := generateSafeInputJavaScriptToolScript(config)

	if !strings.Contains(script, "test-tool") {
		t.Error("Script should contain tool name")
	}

	if !strings.Contains(script, "A test tool") {
		t.Error("Script should contain description")
	}

	if !strings.Contains(script, "return inputs.value * 2;") {
		t.Error("Script should contain the tool script")
	}

	if !strings.Contains(script, "module.exports") {
		t.Error("Script should export execute function")
	}
}

func TestGenerateSafeInputShellToolScript(t *testing.T) {
	config := &SafeInputToolConfig{
		Name:        "test-shell",
		Description: "A shell test tool",
		Run:         "echo $INPUT_MESSAGE",
	}

	script := generateSafeInputShellToolScript(config)

	if !strings.Contains(script, "#!/bin/bash") {
		t.Error("Script should have bash shebang")
	}

	if !strings.Contains(script, "test-shell") {
		t.Error("Script should contain tool name")
	}

	if !strings.Contains(script, "set -euo pipefail") {
		t.Error("Script should have strict mode")
	}

	if !strings.Contains(script, "echo $INPUT_MESSAGE") {
		t.Error("Script should contain the run command")
	}
}

func TestGenerateSafeInputPythonToolScript(t *testing.T) {
	config := &SafeInputToolConfig{
		Name:        "test-python",
		Description: "A Python test tool",
		Py:          "result = {'message': 'Hello from Python'}\nprint(json.dumps(result))",
		Inputs: map[string]*SafeInputParam{
			"message": {
				Type:        "string",
				Description: "Message to process",
			},
			"count": {
				Type:        "number",
				Description: "Number of times",
			},
		},
	}

	script := generateSafeInputPythonToolScript(config)

	if !strings.Contains(script, "#!/usr/bin/env python3") {
		t.Error("Script should have python3 shebang")
	}

	if !strings.Contains(script, "test-python") {
		t.Error("Script should contain tool name")
	}

	if !strings.Contains(script, "import json") {
		t.Error("Script should import json module")
	}

	if !strings.Contains(script, "import sys") {
		t.Error("Script should import sys module")
	}

	if !strings.Contains(script, "inputs = json.loads(sys.stdin.read())") {
		t.Error("Script should parse inputs from stdin")
	}

	if !strings.Contains(script, "result = {'message': 'Hello from Python'}") {
		t.Error("Script should contain the Python code")
	}

	// Check for input parameter documentation
	if !strings.Contains(script, "# message = inputs.get('message'") {
		t.Error("Script should document message parameter access")
	}

	if !strings.Contains(script, "# count = inputs.get('count'") {
		t.Error("Script should document count parameter access")
	}
}

// TestSafeInputsStableCodeGeneration verifies that code generation produces stable, deterministic output
// when called multiple times with the same input. This ensures tools and inputs are sorted properly.
func TestSafeInputsStableCodeGeneration(t *testing.T) {
	// Create a config with multiple tools and inputs to ensure sorting is tested
	config := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"zebra-tool": {
				Name:        "zebra-tool",
				Description: "A tool that starts with Z",
				Run:         "echo zebra",
				Inputs: map[string]*SafeInputParam{
					"zebra-input": {Type: "string", Description: "Zebra input"},
					"alpha-input": {Type: "number", Description: "Alpha input"},
					"beta-input":  {Type: "boolean", Description: "Beta input"},
				},
				Env: map[string]string{
					"ZEBRA_SECRET": "${{ secrets.ZEBRA }}",
					"ALPHA_SECRET": "${{ secrets.ALPHA }}",
				},
			},
			"alpha-tool": {
				Name:        "alpha-tool",
				Description: "A tool that starts with A",
				Script:      "return 'alpha';",
				Inputs: map[string]*SafeInputParam{
					"charlie-param": {Type: "string", Description: "Charlie param"},
					"alpha-param":   {Type: "string", Description: "Alpha param"},
				},
			},
			"middle-tool": {
				Name:        "middle-tool",
				Description: "A tool in the middle",
				Run:         "echo middle",
			},
		},
	}

	// Generate the entry point script multiple times and verify identical output
	iterations := 10
	entryScripts := make([]string, iterations)

	for i := 0; i < iterations; i++ {
		entryScripts[i] = generateSafeInputsMCPServerScript(config)
	}

	// All entry point script iterations should produce identical output
	for i := 1; i < iterations; i++ {
		if entryScripts[i] != entryScripts[0] {
			t.Errorf("generateSafeInputsMCPServerScript produced different output on iteration %d", i+1)
		}
	}

	// Generate the tools config JSON multiple times and verify identical output
	toolsConfigs := make([]string, iterations)

	for i := 0; i < iterations; i++ {
		toolsConfigs[i] = generateSafeInputsToolsConfig(config)
	}

	// All tools config iterations should produce identical output
	for i := 1; i < iterations; i++ {
		if toolsConfigs[i] != toolsConfigs[0] {
			t.Errorf("generateSafeInputsToolsConfig produced different output on iteration %d", i+1)
			// Find first difference for debugging
			for j := 0; j < len(toolsConfigs[0]) && j < len(toolsConfigs[i]); j++ {
				if toolsConfigs[0][j] != toolsConfigs[i][j] {
					start := j - 50
					if start < 0 {
						start = 0
					}
					end := j + 50
					if end > len(toolsConfigs[0]) {
						end = len(toolsConfigs[0])
					}
					if end > len(toolsConfigs[i]) {
						end = len(toolsConfigs[i])
					}
					t.Errorf("First difference at position %d:\n  Expected: %q\n  Got: %q", j, toolsConfigs[0][start:end], toolsConfigs[i][start:end])
					break
				}
			}
		}
	}

	// Verify tools appear in sorted order in tools.json (alpha-tool before middle-tool before zebra-tool)
	alphaPos := strings.Index(toolsConfigs[0], `"name": "alpha-tool"`)
	middlePos := strings.Index(toolsConfigs[0], `"name": "middle-tool"`)
	zebraPos := strings.Index(toolsConfigs[0], `"name": "zebra-tool"`)

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Error("Tools config should contain all tools")
	}

	if alphaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("Tools should be sorted alphabetically: alpha(%d) < middle(%d) < zebra(%d)", alphaPos, middlePos, zebraPos)
	}

	// Test JavaScript tool script stability
	jsScripts := make([]string, iterations)
	for i := 0; i < iterations; i++ {
		jsScripts[i] = generateSafeInputJavaScriptToolScript(config.Tools["alpha-tool"])
	}

	for i := 1; i < iterations; i++ {
		if jsScripts[i] != jsScripts[0] {
			t.Errorf("generateSafeInputJavaScriptToolScript produced different output on iteration %d", i+1)
		}
	}

	// Verify inputs in JSDoc are sorted
	alphaParamPos := strings.Index(jsScripts[0], "inputs.alpha-param")
	charlieParamPos := strings.Index(jsScripts[0], "inputs.charlie-param")

	if alphaParamPos == -1 || charlieParamPos == -1 {
		t.Error("JavaScript script should contain all input parameters in JSDoc")
	}

	if alphaParamPos >= charlieParamPos {
		t.Errorf("Input parameters should be sorted alphabetically in JSDoc: alpha(%d) < charlie(%d)", alphaParamPos, charlieParamPos)
	}

	// Test collectSafeInputsSecrets stability
	secretResults := make([]map[string]string, iterations)
	for i := 0; i < iterations; i++ {
		secretResults[i] = collectSafeInputsSecrets(config)
	}

	// All iterations should produce same key set
	for i := 1; i < iterations; i++ {
		if len(secretResults[i]) != len(secretResults[0]) {
			t.Errorf("collectSafeInputsSecrets produced different number of secrets on iteration %d", i+1)
		}
		for key, val := range secretResults[0] {
			if secretResults[i][key] != val {
				t.Errorf("collectSafeInputsSecrets produced different value for key %s on iteration %d", key, i+1)
			}
		}
	}

	// Test getSafeInputsEnvVars stability
	envResults := make([][]string, iterations)
	for i := 0; i < iterations; i++ {
		envResults[i] = getSafeInputsEnvVars(config)
	}

	for i := 1; i < iterations; i++ {
		if len(envResults[i]) != len(envResults[0]) {
			t.Errorf("getSafeInputsEnvVars produced different number of env vars on iteration %d", i+1)
		}
		for j := range envResults[0] {
			if envResults[i][j] != envResults[0][j] {
				t.Errorf("getSafeInputsEnvVars produced different value at position %d on iteration %d: expected %s, got %s",
					j, i+1, envResults[0][j], envResults[i][j])
			}
		}
	}
}
