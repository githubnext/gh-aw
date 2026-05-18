package workflow

import (
	"fmt"
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
var internalMCPServerNames = map[string]bool{
	"github": true, // GitHub MCP server is handled differently and should not be CLI-mounted
}

// getMCPCLIServerNames returns the sorted list of MCP server names that will be
// mounted as CLI tools.
//
// Infrastructure servers (safeoutputs, mcpscripts) are always CLI-mounted when
// configured, regardless of whether cli-proxy is enabled. This ensures that
// workflows using engine.command can call safeoutputs/mcpscripts as shell
// commands inside the AWF chroot without needing cli-proxy: true.
//
// User-facing MCP servers (playwright, custom, etc.) are only mounted as CLIs
// when tools.cli-proxy is true.
//
// Returns nil when no servers are eligible for mounting.
// The GitHub MCP server is excluded (handled differently).
func getMCPCLIServerNames(data *WorkflowData) []string {
	if data == nil {
		return nil
	}

	var servers []string

	// User-facing MCP servers require cli-proxy: true.
	if data.ParsedTools != nil && data.ParsedTools.CLIProxy {
		mcpCLIMountLog.Print("cli-proxy enabled, collecting user-facing CLI server names")

		// Collect user-facing standard MCP tools from ParsedTools
		if data.ParsedTools.Playwright != nil {
			// In CLI mode, playwright is installed as @playwright/cli via npm and is NOT
			// an MCP server, so it must not appear in the CLI-mounted servers list.
			if !isPlaywrightCLIMode(data.ParsedTools) {
				servers = append(servers, "playwright")
			}
		}
		if _, hasQMD := data.ParsedTools.Custom["qmd"]; hasQMD {
			servers = append(servers, "qmd")
		}
		if data.ParsedTools.AgenticWorkflows != nil && data.ParsedTools.AgenticWorkflows.Enabled {
			// The gateway and manifest use "agenticworkflows" (no hyphen) as the server ID.
			servers = append(servers, constants.AgenticWorkflowsMCPServerID.String())
		}
		// Include custom MCP servers (not in the internal list)
		for name, toolConfig := range data.ParsedTools.Custom {
			if !internalMCPServerNames[name] && name != "qmd" {
				mcpConfigMap := mcpServerConfigToMap(toolConfig)
				if hasMcp, _ := hasMCPConfig(mcpConfigMap); hasMcp {
					if !slices.Contains(servers, name) {
						servers = append(servers, name)
					}
				}
			}
		}
	}

	// Infrastructure servers (safeoutputs, mcpscripts) are always CLI-mounted when
	// configured. This allows workflows with engine.command to call safeoutputs or
	// mcpscripts as shell commands inside the AWF/Copilot chroot without requiring
	// cli-proxy: true. The PATH setup in GetMCPCLIPathSetup ensures the bin directory
	// is reachable inside the sandbox.
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

func buildCLIWorkflowDataForMounts(workflowData *WorkflowData, tools *ToolsConfig, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig) *WorkflowData {
	if workflowData == nil {
		workflowData = &WorkflowData{}
	}

	copied := *workflowData
	if copied.ParsedTools == nil {
		copied.ParsedTools = tools
	}
	if copied.SafeOutputs == nil {
		copied.SafeOutputs = safeOutputs
	}
	if copied.MCPScripts == nil {
		copied.MCPScripts = mcpScripts
	}

	return &copied
}

// hasBashRestrictedAllowlist reports true when tools contains an explicit, finite bash
// command allowlist (not a wildcard). Used to decide whether implicit commands must be
// injected for auto-allowed tools such as playwright-cli.
func hasBashRestrictedAllowlist(tools *ToolsConfig) bool {
	if tools == nil || tools.Bash == nil {
		return false
	}
	if tools.Bash.AllowedCommands == nil || len(tools.Bash.AllowedCommands) == 0 {
		return false
	}
	for _, cmd := range tools.Bash.AllowedCommands {
		if cmd == "*" || cmd == ":*" {
			return false
		}
	}
	return true
}

func getMountedCLIServerNamesIfBashRestricted(workflowData *WorkflowData, tools *ToolsConfig, safeOutputs *SafeOutputsConfig, mcpScripts *MCPScriptsConfig) []string {
	if !hasBashRestrictedAllowlist(tools) {
		return nil
	}
	return getMCPCLIServerNames(buildCLIWorkflowDataForMounts(workflowData, tools, safeOutputs, mcpScripts))
}

func withMountedCLIShellCommandsInRestrictedBash(workflowData *WorkflowData) *ToolsConfig {
	if workflowData == nil {
		return nil
	}
	if workflowData.ParsedTools == nil {
		return workflowData.ParsedTools
	}

	hasRestrictedBash := hasBashRestrictedAllowlist(workflowData.ParsedTools)
	servers := getMountedCLIServerNamesIfBashRestricted(workflowData, workflowData.ParsedTools, workflowData.SafeOutputs, workflowData.MCPScripts)
	needsPlaywrightCLI := isPlaywrightCLIMode(workflowData.ParsedTools) && hasRestrictedBash
	needsGitHubCLI := isGitHubCLIModeEnabled(workflowData) && hasRestrictedBash

	if len(servers) == 0 && !needsPlaywrightCLI && !needsGitHubCLI {
		return workflowData.ParsedTools
	}

	if workflowData.ParsedTools.Bash == nil || len(workflowData.ParsedTools.Bash.AllowedCommands) == 0 {
		return workflowData.ParsedTools
	}

	// Make a shallow copy of ToolsConfig with a new Bash to avoid mutating the original.
	copied := *workflowData.ParsedTools
	augmentedBash := append([]string(nil), workflowData.ParsedTools.Bash.AllowedCommands...)

	for _, server := range servers {
		command := server + ":*"
		if !slices.Contains(augmentedBash, command) {
			augmentedBash = append(augmentedBash, command)
		}
	}

	// When playwright is configured in CLI mode, playwright-cli must be executable.
	// Automatically add it to the restricted bash allowlist so the agent can invoke it.
	if needsPlaywrightCLI {
		const playwrightCLICommand = "playwright-cli:*"
		if !slices.Contains(augmentedBash, playwrightCLICommand) {
			augmentedBash = append(augmentedBash, playwrightCLICommand)
		}
	}

	// When GitHub CLI mode is enabled (tools.github.mode: gh-proxy), GitHub access
	// happens through the pre-authenticated gh binary. Ensure restricted bash
	// allowlists can invoke it.
	if needsGitHubCLI {
		const ghCLICommand = "gh:*"
		if !slices.Contains(augmentedBash, ghCLICommand) {
			augmentedBash = append(augmentedBash, ghCLICommand)
		}
	}

	copied.Bash = &BashToolConfig{AllowedCommands: augmentedBash}
	return &copied
}

// getMCPCLIExcludeFromAgentConfig returns the sorted list of MCP server names that
// should be excluded from the agent's MCP config (because they are CLI-only).
//
// Only excludes servers when tools.cli-proxy is explicitly enabled. Infrastructure
// servers (safeoutputs, mcpscripts) are CLI-mounted even without cli-proxy (so the
// agent can call them as shell commands), but they remain in the agent's MCP config
// unless cli-proxy is explicitly enabled. This preserves existing agent behaviour
// for workflows that use safeoutputs via MCP rather than via the CLI wrapper.
func getMCPCLIExcludeFromAgentConfig(data *WorkflowData) []string {
	if data == nil || data.ParsedTools == nil || !data.ParsedTools.CLIProxy {
		return nil
	}
	return getMCPCLIServerNames(data)
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
