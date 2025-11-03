package workflow

import (
	"strings"
	"testing"
)

func TestSerializeToTOML(t *testing.T) {
	tests := []struct {
		name           string
		config         *TOMLConfig
		indent         string
		expectedSubstr []string
	}{
		{
			name: "basic configuration with history and github server",
			config: &TOMLConfig{
				History: HistoryConfig{
					Persistence: "none",
				},
				MCPServers: map[string]MCPServerConfig{
					"github": {
						Command:           "docker",
						Args:              []string{"run", "-i", "--rm"},
						UserAgent:         "test-workflow",
						StartupTimeoutSec: 120,
						ToolTimeoutSec:    60,
						Env: map[string]string{
							"GITHUB_PERSONAL_ACCESS_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
						},
					},
				},
			},
			indent: "  ",
			expectedSubstr: []string{
				"[history]",
				"persistence = \"none\"",
				"[mcp_servers.github]",
				"command = \"docker\"",
				"args = [",
				"\"run\"",
				"\"--rm\"",
				"]",
				"user_agent = \"test-workflow\"",
				"startup_timeout_sec = 120",
				"tool_timeout_sec = 60",
				"env.GITHUB_PERSONAL_ACCESS_TOKEN = \"${{ secrets.GITHUB_TOKEN }}\"",
			},
		},
		{
			name: "multiple servers configuration",
			config: &TOMLConfig{
				History: HistoryConfig{
					Persistence: "none",
				},
				MCPServers: map[string]MCPServerConfig{
					"github": {
						Command: "docker",
						Args:    []string{"run", "-i"},
					},
					"safeoutputs": {
						Command: "node",
						Args:    []string{"/tmp/gh-aw/safeoutputs/mcp-server.cjs"},
						Env: map[string]string{
							"GH_AW_SAFE_OUTPUTS": "${{ env.GH_AW_SAFE_OUTPUTS }}",
						},
						UseInlineEnv: true,
					},
				},
			},
			indent: "",
			expectedSubstr: []string{
				"[history]",
				"[mcp_servers.github]",
				"[mcp_servers.safeoutputs]",
				"command = \"node\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SerializeToTOML(tt.config, tt.indent)
			if err != nil {
				t.Fatalf("SerializeToTOML failed: %v", err)
			}

			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(result, substr) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", substr, result)
				}
			}
		})
	}
}

func TestBuildTOMLConfig(t *testing.T) {
	config := BuildTOMLConfig()

	if config.History.Persistence != "none" {
		t.Errorf("Expected history.persistence to be 'none', got %q", config.History.Persistence)
	}

	if config.MCPServers == nil {
		t.Error("Expected MCPServers to be initialized, got nil")
	}
}

func TestAddMCPServer(t *testing.T) {
	config := BuildTOMLConfig()

	// Add a server with unsorted environment variables
	serverConfig := MCPServerConfig{
		Command: "test",
		Env: map[string]string{
			"Z_VAR": "z",
			"A_VAR": "a",
			"M_VAR": "m",
		},
	}

	config.AddMCPServer("test-server", serverConfig)

	// Verify server was added
	if _, ok := config.MCPServers["test-server"]; !ok {
		t.Error("Expected test-server to be added to MCPServers")
	}

	// Verify environment variables are preserved (order in map doesn't matter for access)
	server := config.MCPServers["test-server"]
	if server.Env["Z_VAR"] != "z" || server.Env["A_VAR"] != "a" || server.Env["M_VAR"] != "m" {
		t.Errorf("Environment variables not preserved correctly: %v", server.Env)
	}
}

func TestTOMLIndentation(t *testing.T) {
	config := &TOMLConfig{
		History: HistoryConfig{
			Persistence: "none",
		},
		MCPServers: map[string]MCPServerConfig{
			"test": {
				Command: "echo",
			},
		},
	}

	// Test with custom indentation
	result, err := SerializeToTOML(config, "          ")
	if err != nil {
		t.Fatalf("SerializeToTOML failed: %v", err)
	}

	// Check that non-empty lines have indentation
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if len(line) > 0 && line != "\n" {
			if !strings.HasPrefix(line, "          ") {
				t.Errorf("Expected line to start with indentation, got: %q", line)
			}
		}
	}
}
