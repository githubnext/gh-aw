package workflow

import (
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var fetchLog = logger.New("workflow:fetch")

// AddMCPFetchServerIfNeeded adds the mcp/fetch dockerized MCP server to the tools configuration
// if the engine doesn't have built-in web-fetch support and web-fetch tool is requested
func AddMCPFetchServerIfNeeded(tools map[string]any, engine CodingAgentEngine) (map[string]any, []string) {
	// Check if web-fetch tool is requested
	if _, hasWebFetch := tools["web-fetch"]; !hasWebFetch {
		fetchLog.Print("web-fetch tool not requested, skipping MCP fetch server")
		return tools, nil
	}

	// If the engine already supports web-fetch, no need to add MCP server
	if engine.SupportsWebFetch() {
		fetchLog.Print("Engine has built-in web-fetch support, skipping MCP fetch server")
		return tools, nil
	}

	fetchLog.Print("Adding MCP fetch server for web-fetch tool")

	// Create a copy of the tools map to avoid modifying the original
	updatedTools := make(map[string]any)
	maps.Copy(updatedTools, tools)

	// Remove the web-fetch tool since we'll replace it with an MCP server
	delete(updatedTools, "web-fetch")

	// Add the web-fetch server configuration
	// Note: The "container" key marks this as an MCP server with stdio transport.
	// The actual rendering is done by renderMCPFetchServerConfig() which uses
	// the standardized Docker command format for all engines.
	webFetchConfig := map[string]any{
		"container": "mcp/fetch",
	}

	// Add the web-fetch server to the tools
	updatedTools["web-fetch"] = webFetchConfig

	fetchLog.Print("Successfully added web-fetch MCP server configuration")

	// Return the updated tools and the list of added MCP servers
	return updatedTools, []string{"web-fetch"}
}

// renderMCPFetchServerConfig renders the MCP fetch server configuration
// This is a shared function that can be used by all engines
// includeTools parameter adds "tools": ["*"] field for engines that require it (e.g., Copilot)
func renderMCPFetchServerConfig(yaml *strings.Builder, format string, indent string, isLast bool, includeTools bool) {
	fetchLog.Printf("Rendering MCP fetch server config: format=%s, includeTools=%v", format, includeTools)

	switch format {
	case "json":
		// JSON format (for Claude, Copilot, Custom engines)
		// Use container key per MCP Gateway schema (container-based stdio server)
		yaml.WriteString(indent + "\"web-fetch\": {\n")
		yaml.WriteString(indent + "  \"container\": \"mcp/fetch\"\n")
		if isLast {
			yaml.WriteString(indent + "}\n")
		} else {
			yaml.WriteString(indent + "},\n")
		}
	case "toml":
		// TOML format (for Codex engine)
		// Use container key per MCP Gateway schema (container-based stdio server)
		yaml.WriteString(indent + "\n")
		yaml.WriteString(indent + "[mcp_servers.\"web-fetch\"]\n")
		yaml.WriteString(indent + "container = \"mcp/fetch\"\n")
	}
}
