package workflow

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var mcpCLIMountLog = logger.New("workflow:mcp_cli_mount")

// mcp_cli_mount.go generates a workflow step that mounts MCP servers as local CLI tools
// and produces the prompt section that informs the agent about these tools.
//
// After the MCP gateway is started, this step runs mount_mcp_as_cli.cjs via
// actions/github-script which:
//   - Reads the CLI manifest saved by start_mcp_gateway.cjs
//   - Queries each server for its tools/list via JSON-RPC
//   - Writes a standalone CLI wrapper script for each server to ${RUNNER_TEMP}/gh-aw/mcp-cli/bin/
//   - Locks the bin directory (chmod 555) so the agent cannot modify the scripts
//   - Adds the directory to PATH via core.addPath()

// internalMCPServerNames lists the MCP servers that are internal infrastructure and
// should not be exposed as user-facing CLI tools.
// Note: safeoutputs and mcpscripts are NOT excluded — they are always CLI-mounted
// (regardless of mount-as-clis setting), as they provide safe-output and script tools
// that the agent should invoke via CLI wrappers.
var internalMCPServerNames = map[string]bool{
	"github": true, // GitHub MCP server is handled differently and should not be CLI-mounted
}

// alwaysCLIMountedServers lists MCP servers that are always CLI-mounted when enabled,
// regardless of the mount-as-clis setting. These servers remain available as MCP tools
// too (dual access), but the prompt instructs the agent to prefer the CLI wrappers.
var alwaysCLIMountedServers = map[string]bool{
	"safeoutputs": true,
	"mcpscripts":  true,
}

// getMCPCLIServerNames returns the sorted list of MCP server names that will be
// mounted as CLI tools. It includes:
//   - safeoutputs and mcpscripts ALWAYS (when enabled), regardless of mount-as-clis
//   - standard MCP tools (playwright, etc.) and custom MCP servers when mount-as-clis is true
//
// The entire feature is gated behind the mcp-cli feature flag. Without the flag,
// this function returns nil and code generation remains unchanged.
// The GitHub MCP server is excluded (handled differently).
func getMCPCLIServerNames(data *WorkflowData) []string {
	if data == nil {
		return nil
	}

	// The entire MCP CLI mounting feature is gated behind the mcp-cli feature flag.
	// Without the feature flag, code generation remains unchanged regardless of
	// the mount-as-clis setting.
	if !isFeatureEnabled(constants.MCPCLIFeatureFlag, data) {
		mcpCLIMountLog.Print("mcp-cli feature flag not enabled, skipping CLI mount generation")
		return nil
	}
	mcpCLIMountLog.Print("mcp-cli feature flag enabled, collecting CLI server names")

	var servers []string

	// When mount-as-clis is enabled, include all user-facing standard MCP tools
	// and custom MCP servers.
	if data.ParsedTools != nil && data.ParsedTools.MountAsCLIs {
		// Collect user-facing standard MCP tools from the raw Tools map
		for toolName, toolValue := range data.Tools {
			if toolValue == false {
				continue
			}
			// Only include tools that have MCP servers (skip bash, web-fetch, web-search, edit, cache-memory, etc.)
			// Note: "github" is excluded — it is handled differently and should not be CLI-mounted.
			switch toolName {
			case "playwright", "qmd":
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
	}

	// Always include safeoutputs and mcpscripts when they are enabled,
	// regardless of mount-as-clis setting. These servers use their gateway
	// server-ID form (no hyphens) so the CLI wrapper names match the manifest entries.
	if HasSafeOutputsEnabled(data.SafeOutputs) && !slices.Contains(servers, constants.SafeOutputsMCPServerID.String()) {
		servers = append(servers, constants.SafeOutputsMCPServerID.String())
	}
	if IsMCPScriptsEnabled(data.MCPScripts) && !slices.Contains(servers, constants.MCPScriptsMCPServerID.String()) {
		servers = append(servers, constants.MCPScriptsMCPServerID.String())
	}

	if len(servers) == 0 {
		mcpCLIMountLog.Print("No MCP CLI servers configured")
		return nil
	}

	sort.Strings(servers)
	mcpCLIMountLog.Printf("MCP CLI servers selected: %v", servers)
	return servers
}

func buildCLIWorkflowDataForMounts(workflowData *WorkflowData, tools map[string]any, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig) *WorkflowData {
	if workflowData == nil {
		workflowData = &WorkflowData{}
	}

	copied := *workflowData
	if copied.Tools == nil {
		copied.Tools = tools
	}
	if copied.SafeOutputs == nil {
		copied.SafeOutputs = safeOutputs
	}
	if copied.MCPScripts == nil {
		copied.MCPScripts = mcpScripts
	}
	if copied.ParsedTools == nil && copied.Tools != nil {
		copied.ParsedTools = NewTools(copied.Tools)
	}
	// Some call paths may not provide WorkflowData.Features (e.g. direct unit calls).
	// When mount-as-clis is explicitly enabled in tools config, synthesize the feature
	// flag so mounted MCP CLI server names can still be derived consistently.
	if copied.Features == nil && copied.ParsedTools != nil && copied.ParsedTools.MountAsCLIs {
		copied.Features = map[string]any{
			string(constants.MCPCLIFeatureFlag): true,
		}
	}

	return &copied
}

func getMountedCLIServerNamesIfBashRestricted(workflowData *WorkflowData, tools map[string]any, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig) []string {
	if tools == nil {
		return nil
	}
	bashConfig, hasBash := tools["bash"]
	if !hasBash {
		return nil
	}
	bashCommands, ok := bashConfig.([]any)
	if !ok || len(bashCommands) == 0 {
		return nil
	}
	for _, cmd := range bashCommands {
		if cmdStr, ok := cmd.(string); ok && (cmdStr == "*" || cmdStr == ":*") {
			return nil
		}
	}
	return getMCPCLIServerNames(buildCLIWorkflowDataForMounts(workflowData, tools, safeOutputs, mcpScripts))
}

func withMountedCLIShellCommandsInRestrictedBash(workflowData *WorkflowData) map[string]any {
	if workflowData == nil {
		return nil
	}
	if workflowData.Tools == nil {
		return workflowData.Tools
	}

	servers := getMountedCLIServerNamesIfBashRestricted(workflowData, workflowData.Tools, workflowData.SafeOutputs, workflowData.MCPScripts)
	if len(servers) == 0 {
		return workflowData.Tools
	}

	bashCommands, ok := workflowData.Tools["bash"].([]any)
	if !ok || len(bashCommands) == 0 {
		return workflowData.Tools
	}

	copiedTools := make(map[string]any, len(workflowData.Tools))
	// A shallow copy is sufficient because we only replace the top-level "bash"
	// value with a newly allocated slice and do not mutate nested map/slice values.
	maps.Copy(copiedTools, workflowData.Tools)

	augmentedBash := append([]any(nil), bashCommands...)
	for _, server := range servers {
		command := server + ":*"
		exists := false
		for _, allowed := range augmentedBash {
			if allowedStr, ok := allowed.(string); ok && allowedStr == command {
				exists = true
				break
			}
		}
		if !exists {
			augmentedBash = append(augmentedBash, command)
		}
	}

	copiedTools["bash"] = augmentedBash
	return copiedTools
}

// getMCPCLIExcludeFromAgentConfig returns the sorted list of MCP server names that
// should be excluded from the agent's MCP config (because they are CLI-only).
// safeoutputs and mcpscripts are NOT excluded — they remain available as both
// MCP tools and CLI commands (dual access). The prompt instructs the agent to
// prefer the CLI wrappers for these servers.
func getMCPCLIExcludeFromAgentConfig(data *WorkflowData) []string {
	allCLI := getMCPCLIServerNames(data)
	if len(allCLI) == 0 {
		return nil
	}

	var exclude []string
	for _, name := range allCLI {
		if !alwaysCLIMountedServers[name] {
			exclude = append(exclude, name)
		}
	}

	if len(exclude) == 0 {
		return nil
	}

	sort.Strings(exclude)
	return exclude
}

// generateMCPCLIMountStep generates the "Mount MCP servers as CLIs" workflow step.
// This step runs after the MCP gateway is started and creates executable CLI wrapper
// scripts for each MCP server in a read-only directory on $PATH.
func (c *Compiler) generateMCPCLIMountStep(yaml *strings.Builder, data *WorkflowData) {
	servers := getMCPCLIServerNames(data)
	if len(servers) == 0 {
		return
	}
	mcpCLIMountLog.Printf("Generating MCP CLI mount step for %d servers: %v", len(servers), servers)

	yaml.WriteString("      - name: Mount MCP servers as CLIs\n")
	yaml.WriteString("        id: mount-mcp-clis\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          MCP_GATEWAY_API_KEY: ${{ steps.start-mcp-gateway.outputs.gateway-api-key }}\n")
	yaml.WriteString("          MCP_GATEWAY_DOMAIN: ${{ steps.start-mcp-gateway.outputs.gateway-domain }}\n")
	yaml.WriteString("          MCP_GATEWAY_PORT: ${{ steps.start-mcp-gateway.outputs.gateway-port }}\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getActionPin("actions/github-script"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	yaml.WriteString("            const { main } = require('" + SetupActionDestination + "/mount_mcp_as_cli.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// GetMCPCLIPathSetup returns a shell command that adds the MCP CLI bin directory
// to PATH inside the AWF container. This ensures CLI-mounted MCP servers are
// accessible as shell commands even though sudo's secure_path may strip the
// core.addPath() additions from $GITHUB_PATH.
//
// Returns an empty string if no MCP CLIs are configured, so callers can safely
// chain it with && without introducing empty commands.
func GetMCPCLIPathSetup(data *WorkflowData) string {
	if getMCPCLIServerNames(data) == nil {
		return ""
	}
	return `export PATH="${RUNNER_TEMP}/gh-aw/mcp-cli/bin:$PATH"`
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
