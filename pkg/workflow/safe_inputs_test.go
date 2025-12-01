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
		},
	}

	script := generateSafeInputsMCPServerScript(config)

	// Check for basic MCP server structure
	if !strings.Contains(script, "safeinputs") {
		t.Error("Script should contain server name 'safeinputs'")
	}

	// Check for tool registration
	if !strings.Contains(script, `registerTool("search-issues"`) {
		t.Error("Script should register search-issues tool")
	}

	if !strings.Contains(script, `registerTool("echo-message"`) {
		t.Error("Script should register echo-message tool")
	}

	// Check for JavaScript tool handler
	if !strings.Contains(script, "search-issues.cjs") {
		t.Error("Script should reference JavaScript tool file")
	}

	// Check for shell tool handler
	if !strings.Contains(script, "echo-message.sh") {
		t.Error("Script should reference shell script file")
	}

	// Check for MCP methods
	if !strings.Contains(script, "tools/list") {
		t.Error("Script should handle tools/list method")
	}

	if !strings.Contains(script, "tools/call") {
		t.Error("Script should handle tools/call method")
	}

	// Check for large output handling
	if !strings.Contains(script, "LARGE_OUTPUT_THRESHOLD") {
		t.Error("Script should contain large output threshold constant")
	}

	if !strings.Contains(script, "/tmp/gh-aw/safe-inputs/calls") {
		t.Error("Script should contain calls directory path")
	}

	if !strings.Contains(script, "handleLargeOutput") {
		t.Error("Script should contain handleLargeOutput function")
	}

	// Check for structured response fields
	if !strings.Contains(script, "status") {
		t.Error("Script should contain status field in structured response")
	}

	if !strings.Contains(script, "file_path") {
		t.Error("Script should contain file_path field in structured response")
	}

	if !strings.Contains(script, "file_size_bytes") {
		t.Error("Script should contain file_size_bytes field in structured response")
	}

	// Check for JSON schema extraction with jq
	if !strings.Contains(script, "extractJsonSchema") {
		t.Error("Script should contain extractJsonSchema function")
	}

	if !strings.Contains(script, "json_schema_preview") {
		t.Error("Script should contain json_schema_preview field for JSON output")
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

	// Generate the script multiple times and verify identical output
	iterations := 10
	scripts := make([]string, iterations)

	for i := 0; i < iterations; i++ {
		scripts[i] = generateSafeInputsMCPServerScript(config)
	}

	// All iterations should produce identical output
	for i := 1; i < iterations; i++ {
		if scripts[i] != scripts[0] {
			t.Errorf("generateSafeInputsMCPServerScript produced different output on iteration %d", i+1)
			// Find first difference for debugging
			for j := 0; j < len(scripts[0]) && j < len(scripts[i]); j++ {
				if scripts[0][j] != scripts[i][j] {
					start := j - 50
					if start < 0 {
						start = 0
					}
					end := j + 50
					if end > len(scripts[0]) {
						end = len(scripts[0])
					}
					if end > len(scripts[i]) {
						end = len(scripts[i])
					}
					t.Errorf("First difference at position %d:\n  Expected: %q\n  Got: %q", j, scripts[0][start:end], scripts[i][start:end])
					break
				}
			}
		}
	}

	// Verify tools appear in sorted order (alpha-tool before middle-tool before zebra-tool)
	alphaPos := strings.Index(scripts[0], `registerTool("alpha-tool"`)
	middlePos := strings.Index(scripts[0], `registerTool("middle-tool"`)
	zebraPos := strings.Index(scripts[0], `registerTool("zebra-tool"`)

	if alphaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Error("Script should contain all tools")
	}

	if !(alphaPos < middlePos && middlePos < zebraPos) {
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

	if !(alphaParamPos < charlieParamPos) {
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
