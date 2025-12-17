package parser

import (
	"strings"
	"testing"
)

func TestValidateMCPConfigWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		mcpConfig   map[string]any
		toolName    string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid stdio MCP config with command",
			mcpConfig: map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []string{"-y", "@modelcontextprotocol/server-memory"},
			},
			toolName: "memory",
			wantErr:  false,
		},
		{
			name: "valid http MCP config with url",
			mcpConfig: map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
			},
			toolName: "api-server",
			wantErr:  false,
		},
		{
			name: "invalid: empty string for command field",
			mcpConfig: map[string]any{
				"type":    "stdio",
				"command": "",
			},
			toolName:    "test-tool",
			wantErr:     true,
			errContains: "minLength",
		},
		{
			name: "invalid: empty string for url field",
			mcpConfig: map[string]any{
				"type": "http",
				"url":  "",
			},
			toolName:    "test-tool",
			wantErr:     true,
			errContains: "minLength",
		},
		{
			name: "valid stdio MCP config with container",
			mcpConfig: map[string]any{
				"type":      "stdio",
				"container": "ghcr.io/modelcontextprotocol/server-memory",
			},
			toolName: "memory",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPConfigWithSchema(tt.mcpConfig, tt.toolName)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMCPConfigWithSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error message should contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// TestGetSafeOutputTypeKeys tests extracting safe output type keys from the embedded schema
func TestGetSafeOutputTypeKeys(t *testing.T) {
	keys, err := GetSafeOutputTypeKeys()
	if err != nil {
		t.Fatalf("GetSafeOutputTypeKeys() returned error: %v", err)
	}

	// Should return multiple keys
	if len(keys) == 0 {
		t.Error("GetSafeOutputTypeKeys() returned empty list")
	}

	// Should include known safe output types
	expectedKeys := []string{
		"create-issue",
		"add-comment",
		"create-discussion",
		"create-pull-request",
		"update-issue",
	}

	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	for _, expected := range expectedKeys {
		if !keySet[expected] {
			t.Errorf("GetSafeOutputTypeKeys() missing expected key: %s", expected)
		}
	}

	// Should NOT include meta-configuration fields
	metaFields := []string{
		"allowed-domains",
		"staged",
		"env",
		"github-token",
		"app",
		"max-patch-size",
		"jobs",
		"runs-on",
		"messages",
	}

	for _, meta := range metaFields {
		if keySet[meta] {
			t.Errorf("GetSafeOutputTypeKeys() should not include meta field: %s", meta)
		}
	}

	// Keys should be sorted
	for i := 1; i < len(keys); i++ {
		if keys[i-1] > keys[i] {
			t.Errorf("GetSafeOutputTypeKeys() keys are not sorted: %s > %s", keys[i-1], keys[i])
		}
	}
}
