package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpConfigLog = logger.New("cli:mcp_config_file")

// mcpConfigFilePath is the path to the MCP configuration file used by GitHub Copilot CLI.
// GitHub Copilot CLI reads .mcp.json from the repository root.
const mcpConfigFilePath = ".mcp.json"

// VSCodeMCPServer represents a single MCP server configuration for VSCode mcp.json
type VSCodeMCPServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	CWD     string   `json:"cwd,omitempty"`
}

// MCPConfig represents the structure of .mcp.json for Claude Code.
type MCPConfig struct {
	MCPServers map[string]VSCodeMCPServer `json:"mcpServers,omitempty"`
	// Servers is a legacy key kept for backward-compatible reads of existing .mcp.json files.
	Servers map[string]VSCodeMCPServer `json:"servers,omitempty"`
}

// ensureMCPConfig creates .mcp.json with gh-aw MCP server configuration
// If the file already exists, it renders console instructions instead of editing
func ensureMCPConfig(verbose bool) error {
	mcpConfigLog.Print("Creating .mcp.json")

	mcpConfigPath := mcpConfigFilePath

	// Add or update gh-aw MCP server configuration
	ghAwServerName := "github-agentic-workflows"
	ghAwConfig := VSCodeMCPServer{
		Command: "gh",
		Args:    []string{"aw", "mcp-server"},
	}

	// Check if file already exists
	if data, err := os.ReadFile(mcpConfigPath); err == nil {
		mcpConfigLog.Printf("File already exists: %s", mcpConfigPath)

		// Parse existing config
		var config MCPConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing mcp.json: %w", err)
		}

		// Check if the server is already configured correctly
		if existingConfig, exists := config.MCPServers[ghAwServerName]; exists {
			existingJSON, _ := json.Marshal(existingConfig)
			newJSON, _ := json.Marshal(ghAwConfig)
			if string(existingJSON) == string(newJSON) {
				mcpConfigLog.Print("Configuration is identical, skipping")
				if verbose {
					fmt.Fprintf(os.Stderr, "MCP server '%s' already configured in %s\n", ghAwServerName, mcpConfigPath)
				}
				return nil
			}
		}

		// File exists but needs update - render instructions instead of editing
		mcpConfigLog.Print("File exists, rendering update instructions instead of editing")
		renderMCPConfigUpdateInstructions(mcpConfigPath, ghAwServerName, ghAwConfig)
		return nil
	}

	// File doesn't exist - create it
	mcpConfigLog.Print("No existing config found, creating new one")
	config := MCPConfig{
		MCPServers: make(map[string]VSCodeMCPServer),
	}
	config.MCPServers[ghAwServerName] = ghAwConfig

	// Write config file with proper indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mcp.json: %w", err)
	}

	if err := os.WriteFile(mcpConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write mcp.json: %w", err)
	}
	mcpConfigLog.Printf("Created new file: %s", mcpConfigPath)

	return nil
}

// renderMCPConfigUpdateInstructions renders console instructions for updating .mcp.json
func renderMCPConfigUpdateInstructions(filePath, serverName string, serverConfig VSCodeMCPServer) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%s %s\n",
		"ℹ",
		"Existing file detected: "+filePath)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "To enable GitHub Copilot Agent MCP server integration, please add the following")
	fmt.Fprintln(os.Stderr, "to the \"mcpServers\" section of your .mcp.json file:")
	fmt.Fprintln(os.Stderr)

	// Generate the JSON to add
	serverJSON, _ := json.MarshalIndent(map[string]VSCodeMCPServer{
		serverName: serverConfig,
	}, "", "  ")

	fmt.Fprintln(os.Stderr, string(serverJSON))
	fmt.Fprintln(os.Stderr)
}
