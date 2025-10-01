package workflow

import (
	"strings"
)

// AddMCPFetchServerIfNeeded adds the mcp/fetch dockerized MCP server to the tools configuration
// if the engine doesn't have built-in web-fetch support and web-fetch tool is requested
func AddMCPFetchServerIfNeeded(tools map[string]any, engine CodingAgentEngine) (map[string]any, []string) {
	// Check if web-fetch tool is requested
	if _, hasWebFetch := tools["web-fetch"]; !hasWebFetch {
		return tools, nil
	}

	// If the engine already supports web-fetch, no need to add MCP server
	if engine.SupportsWebFetch() {
		return tools, nil
	}

	// Create a copy of the tools map to avoid modifying the original
	updatedTools := make(map[string]any)
	for key, value := range tools {
		updatedTools[key] = value
	}

	// Remove the web-fetch tool since we'll replace it with an MCP server
	delete(updatedTools, "web-fetch")

	// Add the mcp/fetch server configuration
	mcpFetchConfig := map[string]any{
		"container": "ghcr.io/modelcontextprotocol/servers/fetch:latest",
		"allowed":   []any{"*"}, // Allow all fetch operations by default
	}

	// Add the mcp/fetch server to the tools
	updatedTools["mcp/fetch"] = mcpFetchConfig

	// Return the updated tools and the list of added MCP servers
	return updatedTools, []string{"mcp/fetch"}
}

// renderMCPFetchServerConfig renders the MCP fetch server configuration
// This is a shared function that can be used by all engines
func renderMCPFetchServerConfig(yaml *strings.Builder, format string, indent string, isLast bool) {
	if format == "json" {
		// JSON format (for Claude, Custom engines)
		yaml.WriteString(indent + "\"mcp/fetch\": {\n")
		yaml.WriteString(indent + "  \"command\": \"docker\",\n")
		yaml.WriteString(indent + "  \"args\": [\n")
		yaml.WriteString(indent + "    \"run\",\n")
		yaml.WriteString(indent + "    \"-i\",\n")
		yaml.WriteString(indent + "    \"--rm\",\n")
		yaml.WriteString(indent + "    \"ghcr.io/modelcontextprotocol/servers/fetch:latest\"\n")
		yaml.WriteString(indent + "  ]\n")
		if isLast {
			yaml.WriteString(indent + "}\n")
		} else {
			yaml.WriteString(indent + "},\n")
		}
	} else if format == "toml" {
		// TOML format (for Codex engine)
		yaml.WriteString(indent + "\n")
		yaml.WriteString(indent + "[mcp_servers.\"mcp/fetch\"]\n")
		yaml.WriteString(indent + "command = \"docker\"\n")
		yaml.WriteString(indent + "args = [\n")
		yaml.WriteString(indent + "  \"run\",\n")
		yaml.WriteString(indent + "  \"-i\",\n")
		yaml.WriteString(indent + "  \"--rm\",\n")
		yaml.WriteString(indent + "  \"ghcr.io/modelcontextprotocol/servers/fetch:latest\"\n")
		yaml.WriteString(indent + "]\n")
	}
}
