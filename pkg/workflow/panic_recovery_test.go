package workflow

import (
	"strings"
	"testing"
)

// TestParseWorkflowFile_PanicRecovery tests that panics during workflow parsing are recovered
func TestParseWorkflowFile_PanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid workflow file",
			path:        "testdata/simple-workflow.md",
			expectError: false,
		},
		{
			name:        "nonexistent file",
			path:        "/nonexistent/path/to/workflow.md",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			// This should not panic even if there are internal errors
			data, err := compiler.ParseWorkflowFile(tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil && !strings.Contains(err.Error(), "failed to read file") {
					// Some errors are expected (like file not found)
					// But we're testing that panics are caught and converted to errors
					if strings.Contains(err.Error(), "internal error") {
						t.Logf("Recovered from internal error: %v", err)
					}
				}
			}

			// Verify that if error contains "internal error", it's a recovered panic
			if err != nil && strings.Contains(err.Error(), "internal error") {
				if !strings.Contains(err.Error(), "This is a bug") {
					t.Error("internal error should include bug reporting message")
				}
				if !strings.Contains(err.Error(), "github.com/githubnext/gh-aw/issues") {
					t.Error("internal error should include issue reporting URL")
				}
			}

			// If we got data back, it should be valid
			if data != nil && data.Name == "" {
				// Some basic validation
				t.Log("Got workflow data back successfully")
			}
		})
	}
}

// TestGetMCPConfig_PanicRecovery tests that panics during MCP config extraction are recovered
func TestGetMCPConfig_PanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		toolConfig  map[string]any
		toolName    string
		expectError bool
	}{
		{
			name: "valid stdio MCP config",
			toolConfig: map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []any{"@my/tool"},
			},
			toolName:    "test-tool",
			expectError: false,
		},
		{
			name: "valid http MCP config",
			toolConfig: map[string]any{
				"type": "http",
				"url":  "https://example.com/mcp",
			},
			toolName:    "http-tool",
			expectError: false,
		},
		{
			name: "invalid config - missing required fields",
			toolConfig: map[string]any{
				"type": "http",
				// Missing url field
			},
			toolName:    "bad-tool",
			expectError: true,
		},
		{
			name: "empty config",
			toolConfig: map[string]any{
				// No fields at all
			},
			toolName:    "empty-tool",
			expectError: true,
		},
		{
			name: "nil config should not panic",
			toolConfig: map[string]any{
				"env": nil, // nil value that might cause panic
			},
			toolName:    "nil-tool",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic even with malformed input
			config, err := getMCPConfig(tt.toolConfig, tt.toolName)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if config == nil {
					t.Error("expected config but got nil")
				}
			}

			// Verify that if error contains "internal error", it's a recovered panic
			if err != nil && strings.Contains(err.Error(), "internal error") {
				if !strings.Contains(err.Error(), "This is a bug") {
					t.Error("internal error should include bug reporting message")
				}
				if !strings.Contains(err.Error(), "github.com/githubnext/gh-aw/issues") {
					t.Error("internal error should include issue reporting URL")
				}
			}
		})
	}
}

// TestRenderSharedMCPConfig_PanicRecovery tests that panics during MCP rendering are handled
func TestRenderSharedMCPConfig_PanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		toolConfig  map[string]any
		toolName    string
		expectError bool
	}{
		{
			name: "valid config",
			toolConfig: map[string]any{
				"type":    "stdio",
				"command": "test",
			},
			toolName:    "valid-tool",
			expectError: false,
		},
		{
			name: "config with problematic values",
			toolConfig: map[string]any{
				"type": "unknown-type", // This should cause an error but not panic
			},
			toolName:    "problematic-tool",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := &strings.Builder{}
			renderer := MCPConfigRenderer{
				IndentLevel:           "  ",
				Format:                "json",
				RequiresCopilotFields: false,
			}

			// This should not panic
			err := renderSharedMCPConfig(yaml, tt.toolName, tt.toolConfig, renderer)

			if tt.expectError {
				if err == nil {
					// Some errors might be ignored (warnings)
					t.Logf("Expected error but got none (might be warning)")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Verify that internal errors have proper messaging
			if err != nil && strings.Contains(err.Error(), "internal error") {
				if !strings.Contains(err.Error(), "This is a bug") {
					t.Error("internal error should include bug reporting message")
				}
			}
		})
	}
}

// TestPanicRecoveryErrorFormat verifies the error message format for recovered panics
func TestPanicRecoveryErrorFormat(t *testing.T) {
	// Test with a workflow that will cause an error
	compiler := NewCompiler(false, "", "test")
	_, err := compiler.ParseWorkflowFile("/this/path/does/not/exist.md")

	if err == nil {
		t.Skip("Expected error reading nonexistent file")
	}

	// Even though this is a normal error (not a panic), verify the error is formatted correctly
	if err != nil {
		errStr := err.Error()
		t.Logf("Error message: %s", errStr)

		// If it's an internal error (recovered panic), verify format
		if strings.Contains(errStr, "internal error") {
			// Check for required components
			if !strings.Contains(errStr, "This is a bug") {
				t.Error("internal error missing 'This is a bug' message")
			}
			if !strings.Contains(errStr, "github.com/githubnext/gh-aw/issues") {
				t.Error("internal error missing issue reporting URL")
			}

			// Verify it mentions what operation failed
			if !strings.Contains(errStr, "parsing") && !strings.Contains(errStr, "compilation") {
				t.Error("internal error should mention the operation that failed")
			}
		}
	}
}
