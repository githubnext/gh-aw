//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"
)

// TestMCPScriptsTimeoutParsing tests that timeout is correctly parsed from frontmatter
func TestMCPScriptsTimeoutParsing(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     map[string]any
		toolName        string
		expectedTimeout int
	}{
		{
			name: "default timeout when not specified",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"test-tool": map[string]any{
						"description": "Test tool",
						"script":      "return 'hello';",
					},
				},
			},
			toolName:        "test-tool",
			expectedTimeout: 60, // Default timeout
		},
		{
			name: "explicit timeout as integer",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"slow-tool": map[string]any{
						"description": "Slow tool",
						"script":      "return 'slow';",
						"timeout":     120,
					},
				},
			},
			toolName:        "slow-tool",
			expectedTimeout: 120,
		},
		{
			name: "explicit timeout as float",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"fast-tool": map[string]any{
						"description": "Fast tool",
						"run":         "echo 'fast'",
						"timeout":     30.0,
					},
				},
			},
			toolName:        "fast-tool",
			expectedTimeout: 30,
		},
		{
			name: "timeout for shell script",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"shell-tool": map[string]any{
						"description": "Shell tool",
						"run":         "sleep 5",
						"timeout":     10,
					},
				},
			},
			toolName:        "shell-tool",
			expectedTimeout: 10,
		},
		{
			name: "timeout for python script",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"python-tool": map[string]any{
						"description": "Python tool",
						"py":          "print('hello')",
						"timeout":     45,
					},
				},
			},
			toolName:        "python-tool",
			expectedTimeout: 45,
		},
		{
			name: "timeout for go script",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"go-tool": map[string]any{
						"description": "Go tool",
						"go":          "fmt.Println(\"hello\")",
						"timeout":     90,
					},
				},
			},
			toolName:        "go-tool",
			expectedTimeout: 90,
		},
		{
			name: "timeout as valid numeric string",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"string-tool": map[string]any{
						"description": "String timeout tool",
						"script":      "return 'ok';",
						"timeout":     "120",
					},
				},
			},
			toolName:        "string-tool",
			expectedTimeout: 120,
		},
		{
			name: "invalid string timeout falls back to default",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"bad-timeout-tool": map[string]any{
						"description": "Bad timeout tool",
						"script":      "return 'ok';",
						"timeout":     "not-a-number",
					},
				},
			},
			toolName:        "bad-timeout-tool",
			expectedTimeout: 60, // Default timeout
		},
		{
			name: "duration minutes string timeout is parsed",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"duration-tool": map[string]any{
						"description": "Duration string timeout tool",
						"script":      "return 'ok';",
						"timeout":     "5m",
					},
				},
			},
			toolName:        "duration-tool",
			expectedTimeout: 300, // 5 * 60
		},
		{
			name: "duration hours string timeout is parsed",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"hour-tool": map[string]any{
						"description": "Hour timeout tool",
						"script":      "return 'ok';",
						"timeout":     "1h",
					},
				},
			},
			toolName:        "hour-tool",
			expectedTimeout: 3600, // 1 * 3600
		},
		{
			name: "duration seconds string timeout is parsed",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"sec-tool": map[string]any{
						"description": "Seconds timeout tool",
						"script":      "return 'ok';",
						"timeout":     "30s",
					},
				},
			},
			toolName:        "sec-tool",
			expectedTimeout: 30,
		},
		{
			name: "compound duration string timeout is parsed",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"compound-tool": map[string]any{
						"description": "Compound duration timeout tool",
						"script":      "return 'ok';",
						"timeout":     "1h30m",
					},
				},
			},
			toolName:        "compound-tool",
			expectedTimeout: 5400, // 90 * 60
		},
		{
			name: "empty string timeout falls back to default",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"empty-timeout-tool": map[string]any{
						"description": "Empty string timeout tool",
						"script":      "return 'ok';",
						"timeout":     "",
					},
				},
			},
			toolName:        "empty-timeout-tool",
			expectedTimeout: 60, // Default timeout
		},
		{
			name: "string timeout with whitespace is parsed",
			frontmatter: map[string]any{
				"mcp-scripts": map[string]any{
					"ws-tool": map[string]any{
						"description": "Whitespace timeout tool",
						"script":      "return 'ok';",
						"timeout":     " 120 ",
					},
				},
			},
			toolName:        "ws-tool",
			expectedTimeout: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := (&Compiler{}).extractMCPScriptsConfig(tt.frontmatter)
			if config == nil {
				t.Fatalf("Expected config, got nil")
			}

			tool, exists := config.Tools[tt.toolName]
			if !exists {
				t.Fatalf("Expected tool %s to exist", tt.toolName)
			}

			if tool.Timeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %d, got %d", tt.expectedTimeout, tool.Timeout)
			}
		})
	}
}

// TestMCPScriptsTimeoutInJSON tests that timeout is included in the generated tools.json
func TestMCPScriptsTimeoutInJSON(t *testing.T) {
	config := &MCPScriptsConfig{
		Tools: map[string]*MCPScriptToolConfig{
			"fast-tool": {
				Name:        "fast-tool",
				Description: "Fast tool",
				Script:      "return 'fast';",
				Timeout:     30,
			},
			"slow-tool": {
				Name:        "slow-tool",
				Description: "Slow tool",
				Run:         "echo 'slow'",
				Timeout:     120,
			},
			"default-tool": {
				Name:        "default-tool",
				Description: "Default timeout tool",
				Py:          "print('default')",
				Timeout:     60,
			},
			"go-tool": {
				Name:        "go-tool",
				Description: "Go timeout tool",
				Go:          "fmt.Println(\"hello\")",
				Timeout:     180,
			},
		},
	}

	jsonStr := GenerateMCPScriptsToolsConfig(config)

	// Parse the JSON to verify structure
	var parsedConfig MCPScriptsConfigJSON
	if err := json.Unmarshal([]byte(jsonStr), &parsedConfig); err != nil {
		t.Fatalf("Failed to parse generated JSON: %v", err)
	}

	// Verify timeouts are present
	toolTimeouts := make(map[string]int)
	for _, tool := range parsedConfig.Tools {
		toolTimeouts[tool.Name] = tool.Timeout
	}

	expected := map[string]int{
		"fast-tool":    30,
		"slow-tool":    120,
		"default-tool": 60,
		"go-tool":      180,
	}

	for toolName, expectedTimeout := range expected {
		actualTimeout, exists := toolTimeouts[toolName]
		if !exists {
			t.Errorf("Tool %s not found in generated JSON", toolName)
			continue
		}
		if actualTimeout != expectedTimeout {
			t.Errorf("Tool %s: expected timeout %d, got %d", toolName, expectedTimeout, actualTimeout)
		}
	}
}

// TestMCPScriptsMergePreservesTimeout tests that timeout is preserved when merging configs
func TestMCPScriptsMergePreservesTimeout(t *testing.T) {
	compiler := &Compiler{}

	// Main config with one tool
	main := &MCPScriptsConfig{
		Tools: map[string]*MCPScriptToolConfig{
			"main-tool": {
				Name:        "main-tool",
				Description: "Main tool",
				Script:      "return 'main';",
				Timeout:     90,
			},
		},
	}

	// Imported config with a different tool
	importedJSON := `{
		"imported-tool": {
			"description": "Imported tool",
			"run": "echo 'imported'",
			"timeout": 45
		}
	}`

	merged := compiler.mergeMCPScripts(main, []string{importedJSON})

	// Verify main tool timeout is preserved
	if merged.Tools["main-tool"].Timeout != 90 {
		t.Errorf("Expected main-tool timeout 90, got %d", merged.Tools["main-tool"].Timeout)
	}

	// Verify imported tool timeout is set
	if merged.Tools["imported-tool"].Timeout != 45 {
		t.Errorf("Expected imported-tool timeout 45, got %d", merged.Tools["imported-tool"].Timeout)
	}
}

// TestMCPScriptsDefaultTimeoutWhenMerging tests that default timeout is used when not specified in imported config
func TestMCPScriptsDefaultTimeoutWhenMerging(t *testing.T) {
	compiler := &Compiler{}

	main := &MCPScriptsConfig{
		Tools: make(map[string]*MCPScriptToolConfig),
	}

	// Imported config without timeout specified
	importedJSON := `{
		"imported-tool": {
			"description": "Imported tool without timeout",
			"script": "return 'imported';"
		}
	}`

	merged := compiler.mergeMCPScripts(main, []string{importedJSON})

	// Verify default timeout is used
	if merged.Tools["imported-tool"].Timeout != 60 {
		t.Errorf("Expected default timeout 60, got %d", merged.Tools["imported-tool"].Timeout)
	}
}

// TestMCPScriptsMergeStringTimeout tests that string timeouts are parsed correctly when merging
func TestMCPScriptsMergeStringTimeout(t *testing.T) {
	compiler := &Compiler{}

	main := &MCPScriptsConfig{
		Tools: make(map[string]*MCPScriptToolConfig),
	}

	// Imported config with valid numeric string timeout
	validImportedJSON := `{
		"valid-string-timeout": {
			"description": "Tool with numeric string timeout",
			"script": "return 'ok';",
			"timeout": "120"
		}
	}`

	merged := compiler.mergeMCPScripts(main, []string{validImportedJSON})

	// Verify valid numeric string timeout is parsed correctly
	if merged.Tools["valid-string-timeout"].Timeout != 120 {
		t.Errorf("Expected timeout 120, got %d", merged.Tools["valid-string-timeout"].Timeout)
	}

	// Imported config with invalid string timeout
	invalidImportedJSON := `{
		"invalid-string-timeout": {
			"description": "Tool with invalid string timeout",
			"script": "return 'ok';",
			"timeout": "not-a-number"
		}
	}`

	merged2 := compiler.mergeMCPScripts(main, []string{invalidImportedJSON})

	// Verify invalid string timeout falls back to default (60s)
	if merged2.Tools["invalid-string-timeout"].Timeout != 60 {
		t.Errorf("Expected default timeout 60, got %d", merged2.Tools["invalid-string-timeout"].Timeout)
	}

	// Imported config with duration-style string timeout ("5m").
	// time.ParseDuration now converts this to 300s.
	durationImportedJSON := `{
		"duration-string-timeout": {
			"description": "Tool with duration-style string timeout",
			"script": "return 'ok';",
			"timeout": "5m"
		}
	}`

	merged3 := compiler.mergeMCPScripts(main, []string{durationImportedJSON})

	// Verify "5m" is correctly parsed as 300 seconds
	if merged3.Tools["duration-string-timeout"].Timeout != 300 {
		t.Errorf("Expected timeout 300 for \"5m\", got %d", merged3.Tools["duration-string-timeout"].Timeout)
	}
}

// TestParseTimeoutString is a unit test for the parseTimeoutString helper.
func TestParseTimeoutString(t *testing.T) {
	tests := []struct {
		input    string
		wantSecs int
		wantOk   bool
	}{
		// Plain integers
		{"120", 120, true},
		{"0", 0, true},
		{" 120 ", 120, true}, // leading/trailing whitespace

		// Go duration strings
		{"30s", 30, true},
		{"6m", 360, true},
		{"1h", 3600, true},
		{"1h30m", 5400, true},
		{"2h30m10s", 9010, true},

		// Invalid
		{"", 0, false},
		{"   ", 0, false},
		{"not-a-number", 0, false},
		{"5 m", 0, false}, // space inside duration is not valid
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotSecs, gotOk := parseTimeoutString(tt.input)
			if gotOk != tt.wantOk {
				t.Errorf("parseTimeoutString(%q) ok = %v, want %v", tt.input, gotOk, tt.wantOk)
			}
			if gotOk && gotSecs != tt.wantSecs {
				t.Errorf("parseTimeoutString(%q) = %d, want %d", tt.input, gotSecs, tt.wantSecs)
			}
		})
	}
}
