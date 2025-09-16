package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPRef(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "mcp-ref-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a .vscode directory and mcp.json file
	vscodeDirPath := filepath.Join(tmpDir, ".vscode")
	err = os.MkdirAll(vscodeDirPath, 0755)
	if err != nil {
		t.Fatal(err)
	}

	mcpJSON := `{
  "servers": {
    "my-tool": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
      "env": {
        "NODE_ENV": "production"
      }
    },
    "github-server": {
      "command": "docker",
      "args": ["run", "--rm", "-i", "-e", "GITHUB_TOKEN", "ghcr.io/github/github-mcp-server"],
      "env": {
        "GITHUB_TOKEN": "ghp_xxx"
      }
    }
  }
}`

	mcpJSONPath := filepath.Join(vscodeDirPath, "mcp.json")
	err = os.WriteFile(mcpJSONPath, []byte(mcpJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		serverName  string
		expectedCmd string
		expectedErr bool
	}{
		{
			name:        "valid server - my-tool",
			serverName:  "my-tool",
			expectedCmd: "npx",
			expectedErr: false,
		},
		{
			name:        "valid server - github-server",
			serverName:  "github-server",
			expectedCmd: "docker",
			expectedErr: false,
		},
		{
			name:        "invalid server",
			serverName:  "non-existent",
			expectedCmd: "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := loadVSCodeMCPConfig(tmpDir, tt.serverName)
			
			if tt.expectedErr {
				if err == nil {
					t.Errorf("Expected error for server '%s', but got none", tt.serverName)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for server '%s': %v", tt.serverName, err)
				return
			}

			if config["command"] != tt.expectedCmd {
				t.Errorf("Expected command '%s', got '%s'", tt.expectedCmd, config["command"])
			}

			if config["type"] != "stdio" {
				t.Errorf("Expected type 'stdio', got '%s'", config["type"])
			}
		})
	}
}

func TestValidateMCPRef(t *testing.T) {
	tests := []struct {
		name       string
		toolConfig map[string]any
		expectErr  bool
		errMsg     string
	}{
		{
			name:       "no mcp-ref",
			toolConfig: map[string]any{},
			expectErr:  false,
		},
		{
			name: "valid mcp-ref vscode",
			toolConfig: map[string]any{
				"mcp-ref": "vscode",
			},
			expectErr: false,
		},
		{
			name: "invalid mcp-ref value",
			toolConfig: map[string]any{
				"mcp-ref": "invalid",
			},
			expectErr: true,
			errMsg:    "unsupported 'mcp-ref' value",
		},
		{
			name: "mcp-ref not string",
			toolConfig: map[string]any{
				"mcp-ref": 123,
			},
			expectErr: true,
			errMsg:    "must be a string",
		},
		{
			name: "mcp-ref with inputs (should fail)",
			toolConfig: map[string]any{
				"mcp-ref": "vscode",
				"inputs":  map[string]any{"key": "value"},
			},
			expectErr: true,
			errMsg:    "cannot specify 'inputs'",
		},
		{
			name: "mcp-ref with mcp section (should fail)",
			toolConfig: map[string]any{
				"mcp-ref": "vscode",
				"mcp":     map[string]any{"type": "stdio"},
			},
			expectErr: true,
			errMsg:    "cannot specify both 'mcp-ref' and 'mcp'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPRef("test-tool", tt.toolConfig)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message '%s' doesn't contain expected text '%s'", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHasMCPConfigWithMCPRef(t *testing.T) {
	tests := []struct {
		name        string
		toolConfig  map[string]any
		expectMCP   bool
		expectedType string
	}{
		{
			name:         "no mcp config",
			toolConfig:   map[string]any{},
			expectMCP:    false,
			expectedType: "",
		},
		{
			name: "mcp-ref vscode",
			toolConfig: map[string]any{
				"mcp-ref": "vscode",
			},
			expectMCP:    true,
			expectedType: "stdio",
		},
		{
			name: "regular mcp config",
			toolConfig: map[string]any{
				"mcp": map[string]any{
					"type": "http",
				},
			},
			expectMCP:    true,
			expectedType: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasMCP, mcpType := hasMCPConfig(tt.toolConfig)
			
			if hasMCP != tt.expectMCP {
				t.Errorf("Expected hasMCP=%v, got %v", tt.expectMCP, hasMCP)
			}
			
			if mcpType != tt.expectedType {
				t.Errorf("Expected type='%s', got '%s'", tt.expectedType, mcpType)
			}
		})
	}
}