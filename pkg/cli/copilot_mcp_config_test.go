package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestEnsureCopilotMCPConfig(t *testing.T) {
	tests := []struct {
		name            string
		existingConfig  *CopilotMCPConfig
		verbose         bool
		wantErr         bool
		validateContent func(*testing.T, []byte)
	}{
		{
			name:    "creates new copilot-mcp-config.json",
			verbose: false,
			wantErr: false,
			validateContent: func(t *testing.T, content []byte) {
				var config CopilotMCPConfig
				if err := json.Unmarshal(content, &config); err != nil {
					t.Fatalf("Failed to parse config: %v", err)
				}

				if _, exists := config.MCPServers["github-agentic-workflows"]; !exists {
					t.Error("Expected config to contain 'github-agentic-workflows' server")
				}

				server := config.MCPServers["github-agentic-workflows"]
				if server.Command != "gh" {
					t.Errorf("Expected command to be 'gh', got %s", server.Command)
				}
				if len(server.Args) != 2 || server.Args[0] != "aw" || server.Args[1] != "mcp-server" {
					t.Errorf("Expected args to be ['aw', 'mcp-server'], got %v", server.Args)
				}
			},
		},
		{
			name: "skips update when server already configured identically",
			existingConfig: &CopilotMCPConfig{
				MCPServers: map[string]CopilotMCPServerConfig{
					"github-agentic-workflows": {
						Command: "gh",
						Args:    []string{"aw", "mcp-server"},
					},
				},
			},
			verbose: true,
			wantErr: false,
			validateContent: func(t *testing.T, content []byte) {
				var config CopilotMCPConfig
				if err := json.Unmarshal(content, &config); err != nil {
					t.Fatalf("Failed to parse config: %v", err)
				}

				if len(config.MCPServers) != 1 {
					t.Errorf("Expected exactly 1 server, got %d", len(config.MCPServers))
				}
			},
		},
		{
			name: "updates when server exists but configuration differs",
			existingConfig: &CopilotMCPConfig{
				MCPServers: map[string]CopilotMCPServerConfig{
					"github-agentic-workflows": {
						Command: "gh",
						Args:    []string{"aw", "old-command"},
					},
				},
			},
			verbose: false,
			wantErr: false,
			validateContent: func(t *testing.T, content []byte) {
				var config CopilotMCPConfig
				if err := json.Unmarshal(content, &config); err != nil {
					t.Fatalf("Failed to parse config: %v", err)
				}

				server := config.MCPServers["github-agentic-workflows"]
				if server.Args[1] != "mcp-server" {
					t.Errorf("Expected args[1] to be updated to 'mcp-server', got %s", server.Args[1])
				}
			},
		},
		{
			name: "preserves other servers when updating",
			existingConfig: &CopilotMCPConfig{
				MCPServers: map[string]CopilotMCPServerConfig{
					"other-server": {
						Command: "other-command",
						Args:    []string{"arg1"},
					},
				},
			},
			verbose: false,
			wantErr: false,
			validateContent: func(t *testing.T, content []byte) {
				var config CopilotMCPConfig
				if err := json.Unmarshal(content, &config); err != nil {
					t.Fatalf("Failed to parse config: %v", err)
				}

				if len(config.MCPServers) != 2 {
					t.Errorf("Expected 2 servers, got %d", len(config.MCPServers))
				}

				if _, exists := config.MCPServers["other-server"]; !exists {
					t.Error("Expected existing 'other-server' to be preserved")
				}

				if _, exists := config.MCPServers["github-agentic-workflows"]; !exists {
					t.Error("Expected 'github-agentic-workflows' server to be added")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Create existing config if specified
			if tt.existingConfig != nil {
				workflowsDir := filepath.Join(".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				data, err := json.MarshalIndent(tt.existingConfig, "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal existing config: %v", err)
				}

				configPath := filepath.Join(workflowsDir, "copilot-mcp-config.json")
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Fatalf("Failed to write existing config: %v", err)
				}
			}

			// Call the function
			err = ensureCopilotMCPConfig(tt.verbose)

			if (err != nil) != tt.wantErr {
				t.Errorf("ensureCopilotMCPConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify the file was created/updated
			configPath := filepath.Join(".github", "workflows", "copilot-mcp-config.json")
			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read copilot-mcp-config.json: %v", err)
			}

			// Run custom validation if provided
			if tt.validateContent != nil {
				tt.validateContent(t, content)
			}
		})
	}
}

func TestEnsureCopilotMCPConfigFileLocation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	err = ensureCopilotMCPConfig(false)
	if err != nil {
		t.Fatalf("ensureCopilotMCPConfig() failed: %v", err)
	}

	// Verify file is in .github/workflows directory
	configPath := filepath.Join(".github", "workflows", "copilot-mcp-config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected copilot-mcp-config.json to be created in .github/workflows directory")
	}
}

func TestEnsureCopilotMCPConfigJSONFormat(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	err = ensureCopilotMCPConfig(false)
	if err != nil {
		t.Fatalf("ensureCopilotMCPConfig() failed: %v", err)
	}

	// Verify JSON structure
	configPath := filepath.Join(".github", "workflows", "copilot-mcp-config.json")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config CopilotMCPConfig
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify structure matches expected format
	if config.MCPServers == nil {
		t.Error("Expected mcpServers field to exist")
	}

	// Verify required fields are present
	server, exists := config.MCPServers["github-agentic-workflows"]
	if !exists {
		t.Fatal("Expected github-agentic-workflows server to exist")
	}

	if server.Command == "" {
		t.Error("Expected command field to be non-empty")
	}

	if server.Args == nil || len(server.Args) == 0 {
		t.Error("Expected args field to be non-empty")
	}
}

func TestEnsureCopilotMCPConfigIdempotent(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Call first time
	err = ensureCopilotMCPConfig(false)
	if err != nil {
		t.Fatalf("ensureCopilotMCPConfig() failed on first call: %v", err)
	}

	configPath := filepath.Join(".github", "workflows", "copilot-mcp-config.json")
	content1, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after first call: %v", err)
	}

	// Call second time
	err = ensureCopilotMCPConfig(false)
	if err != nil {
		t.Fatalf("ensureCopilotMCPConfig() failed on second call: %v", err)
	}

	content2, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after second call: %v", err)
	}

	// Content should be identical
	if string(content1) != string(content2) {
		t.Error("Expected config to remain unchanged on second call")
	}
}

func TestCopilotMCPConfigStructMarshaling(t *testing.T) {
	t.Parallel()

	config := CopilotMCPConfig{
		MCPServers: map[string]CopilotMCPServerConfig{
			"test-server": {
				Command: "test-command",
				Args:    []string{"arg1", "arg2"},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(&config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal back
	var unmarshaledConfig CopilotMCPConfig
	if err := json.Unmarshal(data, &unmarshaledConfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify structure
	server, exists := unmarshaledConfig.MCPServers["test-server"]
	if !exists {
		t.Fatal("Expected 'test-server' to exist")
	}

	if server.Command != "test-command" {
		t.Errorf("Expected command 'test-command', got %q", server.Command)
	}

	if len(server.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(server.Args))
	}
}

func TestEnsureCopilotMCPConfigFilePermissions(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	err = ensureCopilotMCPConfig(false)
	if err != nil {
		t.Fatalf("ensureCopilotMCPConfig() failed: %v", err)
	}

	// Check file permissions
	configPath := filepath.Join(".github", "workflows", "copilot-mcp-config.json")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat copilot-mcp-config.json: %v", err)
	}

	// Verify file is readable and writable
	mode := info.Mode()
	if mode.Perm()&0600 != 0600 {
		t.Errorf("Expected file to have at least 0600 permissions, got %o", mode.Perm())
	}
}
