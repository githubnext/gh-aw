package workflow

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// mcp_cli_mount.go generates a workflow step that mounts MCP servers as local CLI tools
// and produces the prompt section that informs the agent about these tools.
//
// After the MCP gateway is started, this step runs mount_mcp_as_cli.cjs via
// actions/github-script which:
//   - Reads the CLI manifest saved by start_mcp_gateway.sh
//   - Queries each server for its tools/list via JSON-RPC
//   - Writes a standalone CLI wrapper script for each server to ${RUNNER_TEMP}/gh-aw/mcp-cli/bin/
//   - Locks the bin directory (chmod 555) so the agent cannot modify the scripts
//   - Adds the directory to PATH via core.addPath()

// internalMCPServerNames lists the MCP servers that are internal infrastructure and
// should not be exposed as user-facing CLI tools.
// Include both config-key and rendered server-ID variants where they differ.
var internalMCPServerNames = map[string]bool{
	"safeoutputs": true,
	"mcp-scripts": true,
	"mcpscripts":  true,
}

// getMCPCLIServerNames returns the sorted list of MCP server names that will be
// mounted as CLI tools. It includes standard MCP tools (github, playwright, etc.)
// and custom MCP servers, but excludes internal infrastructure servers.
// Returns nil if tools.mount-as-clis is not set to true.
func getMCPCLIServerNames(data *WorkflowData) []string {
	if data == nil {
		return nil
	}

	// Only mount if tools.mount-as-clis: true is set.
	// Also returns nil when tools configuration is missing entirely.
	if data.ParsedTools == nil || !data.ParsedTools.MountAsCLIs {
		return nil
	}

	var servers []string

	// Collect user-facing standard MCP tools from the raw Tools map
	for toolName, toolValue := range data.Tools {
		if toolValue == false {
			continue
		}
		// Only include tools that have MCP servers (skip bash, web-fetch, web-search, edit, cache-memory, etc.)
		switch toolName {
		case "github", "playwright", "qmd":
			servers = append(servers, toolName)
		case "agentic-workflows":
			// The gateway and manifest use "agenticworkflows" (no hyphen) as the server ID.
			// Using the gateway ID here ensures GH_AW_MCP_CLI_SERVERS matches the manifest entries.
			servers = append(servers, constants.AgenticWorkflowsMCPServerID.String())
		default:
			// Include custom MCP servers (not in the internal list)
			if !internalMCPServerNames[toolName] {
				if mcpConfig, ok := toolValue.(map[string]any); ok {
					if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
						servers = append(servers, toolName)
					}
				}
			}
		}
	}

	// Also check ParsedTools.Custom for custom MCP servers
	if data.ParsedTools != nil {
		for name := range data.ParsedTools.Custom {
			if !internalMCPServerNames[name] && !slices.Contains(servers, name) {
				servers = append(servers, name)
			}
		}
	}

	sort.Strings(servers)
	return servers
}

// generateMCPCLIMountStep generates the "Mount MCP servers as CLIs" workflow step.
// This step runs after the MCP gateway is started and creates executable CLI wrapper
// scripts for each MCP server in a read-only directory on $PATH.
func (c *Compiler) generateMCPCLIMountStep(yaml *strings.Builder, data *WorkflowData) {
	servers := getMCPCLIServerNames(data)
	if len(servers) == 0 {
		return
	}

	yaml.WriteString("      - name: Mount MCP servers as CLIs\n")
	yaml.WriteString("        id: mount-mcp-clis\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          MCP_GATEWAY_API_KEY: ${{ steps.start-mcp-gateway.outputs.gateway-api-key }}\n")
	yaml.WriteString("          MCP_GATEWAY_DOMAIN: ${{ steps.start-mcp-gateway.outputs.gateway-domain }}\n")
	yaml.WriteString("          MCP_GATEWAY_PORT: ${{ steps.start-mcp-gateway.outputs.gateway-port }}\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	yaml.WriteString("            const { main } = require('" + SetupActionDestination + "/mount_mcp_as_cli.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// buildMCPCLIPromptSection returns a PromptSection describing the CLI tools available
// to the agent, or nil if there are no servers to mount.
// The prompt is loaded from actions/setup/md/mcp_cli_tools_prompt.md at runtime,
// with the __GH_AW_MCP_CLI_SERVERS_LIST__ placeholder substituted by the substitution step.
func buildMCPCLIPromptSection(data *WorkflowData) *PromptSection {
	servers := getMCPCLIServerNames(data)
	if len(servers) == 0 {
		return nil
	}

	// Build the human-readable list of servers with example usage
	var listLines []string
	for _, name := range servers {
		listLines = append(listLines, fmt.Sprintf("- `%s` — run `%s --help` to see available tools", name, name))
	}
	serversList := strings.Join(listLines, "\n")

	return &PromptSection{
		Content: mcpCLIToolsPromptFile,
		IsFile:  true,
		EnvVars: map[string]string{
			"GH_AW_MCP_CLI_SERVERS_LIST": serversList,
		},
	}
}
