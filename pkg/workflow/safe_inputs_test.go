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
