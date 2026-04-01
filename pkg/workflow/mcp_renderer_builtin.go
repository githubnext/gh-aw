package workflow

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var mcpRendererBuiltinLog = logger.New("workflow:mcp_renderer_builtin")

// RenderPlaywrightMCP generates the Playwright MCP server configuration
func (r *MCPConfigRendererUnified) RenderPlaywrightMCP(yaml *strings.Builder, playwrightTool any) {
	mcpRendererLog.Printf("Rendering Playwright MCP: format=%s, inline_args=%t", r.options.Format, r.options.InlineArgs)

	// Parse playwright tool configuration to strongly-typed struct
	playwrightConfig := parsePlaywrightTool(playwrightTool)

	if r.options.Format == "toml" {
		r.renderPlaywrightTOML(yaml, playwrightConfig)
		// Add guard policies for TOML format as a separate section
		if len(r.options.WriteSinkGuardPolicies) > 0 {
			mcpRendererLog.Print("Adding guard-policies to playwright TOML (derived from GitHub guard-policy)")
			renderGuardPoliciesToml(yaml, r.options.WriteSinkGuardPolicies, "playwright")
		}
		return
	}

	// JSON format
	renderPlaywrightMCPConfigWithOptions(yaml, playwrightConfig, r.options.IsLast, r.options.IncludeCopilotFields, r.options.InlineArgs, r.options.WriteSinkGuardPolicies)
}

// renderPlaywrightTOML generates Playwright MCP configuration in TOML format
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypointArgs, mounts, and args fields.
func (r *MCPConfigRendererUnified) renderPlaywrightTOML(yaml *strings.Builder, playwrightConfig *PlaywrightToolConfig) {
	mcpRendererBuiltinLog.Print("Rendering Playwright MCP in TOML format")
	customArgs := getPlaywrightCustomArgs(playwrightConfig)

	// Use official Playwright MCP Docker image (no version tag - only one image)
	playwrightImage := "mcr.microsoft.com/playwright/mcp"

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.playwright]\n")
	yaml.WriteString("          container = \"" + playwrightImage + "\"\n")

	// Docker runtime args (goes before container image in docker run command)
	// Add security-opt and ipc flags for Chromium browser compatibility in GitHub Actions
	// --security-opt seccomp=unconfined: Required for Chromium sandbox to function properly
	// --ipc=host: Provides shared memory access required by Chromium
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"--init\",\n")
	yaml.WriteString("            \"--network\",\n")
	yaml.WriteString("            \"host\",\n")
	yaml.WriteString("            \"--security-opt\",\n")
	yaml.WriteString("            \"seccomp=unconfined\",\n")
	yaml.WriteString("            \"--ipc=host\",\n")
	yaml.WriteString("          ]\n")

	// Entrypoint args for Playwright MCP server (goes after container image)
	yaml.WriteString("          entrypointArgs = [\n")
	yaml.WriteString("            \"--output-dir\",\n")
	yaml.WriteString("            \"/tmp/gh-aw/mcp-logs/playwright\"")

	// Append custom args if present
	writeArgsToYAML(yaml, customArgs, "            ")

	yaml.WriteString("\n")
	yaml.WriteString("          ]\n")

	// Add volume mounts
	yaml.WriteString("          mounts = [\"/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs:rw\"]\n")
}

// RenderQmdMCP generates the qmd documentation search MCP server configuration.
// qmd runs natively on the host VM with HTTP transport, started by the
// "Start QMD MCP Server" step before the gateway, so the gateway connects via HTTP.
// Using HTTP transport avoids node-llama-cpp's direct process.stdout writes (dot-progress
// during model loading) from corrupting the stdio JSON-RPC stream.
func (r *MCPConfigRendererUnified) RenderQmdMCP(yaml *strings.Builder, qmdTool any, workflowData *WorkflowData) {
	mcpRendererLog.Printf("Rendering qmd MCP: format=%s, inline_args=%t", r.options.Format, r.options.InlineArgs)

	if r.options.Format == "toml" {
		r.renderQmdTOML(yaml, workflowData)
		// Add guard policies for TOML format as a separate section
		if len(r.options.WriteSinkGuardPolicies) > 0 {
			mcpRendererLog.Print("Adding guard-policies to qmd TOML (derived from GitHub guard-policy)")
			renderGuardPoliciesToml(yaml, r.options.WriteSinkGuardPolicies, "qmd")
		}
		return
	}

	// JSON format
	renderQmdMCPConfigWithOptions(yaml, r.options.IsLast, r.options.IncludeCopilotFields, r.options.WriteSinkGuardPolicies, workflowData)
}

// resolveQmdHost returns the hostname the gateway should use to reach the qmd HTTP server.
// qmd runs natively on the host VM, so port DefaultQmdMCPPort is bound on the host network.
// When the agent sandbox is enabled (default), the gateway runs inside a Docker container
// with its own network namespace and must reach the host via host.docker.internal.
// When the agent sandbox is disabled (agent.disabled: true), the gateway also runs on
// the host, so localhost is sufficient.
func resolveQmdHost(workflowData *WorkflowData) string {
	if workflowData != nil && workflowData.SandboxConfig != nil &&
		workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled {
		return "localhost"
	}
	return "host.docker.internal"
}

// qmdMCPURL returns the full HTTP MCP URL for the qmd server.
func qmdMCPURL(workflowData *WorkflowData) string {
	host := resolveQmdHost(workflowData)
	return "http://" + host + ":" + strconv.Itoa(constants.DefaultQmdMCPPort) + "/mcp"
}

// renderQmdTOML generates qmd MCP configuration in TOML format using HTTP transport.
// qmd is started natively before the gateway (see generateQmdStartStep),
// and the gateway connects to the qmd HTTP MCP server at DefaultQmdMCPPort/mcp.
func (r *MCPConfigRendererUnified) renderQmdTOML(yaml *strings.Builder, workflowData *WorkflowData) {
	mcpRendererBuiltinLog.Print("Rendering qmd MCP in TOML format (HTTP transport)")

	url := qmdMCPURL(workflowData)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.qmd]\n")
	yaml.WriteString("          type = \"http\"\n")
	yaml.WriteString("          url = \"" + url + "\"\n")
}

// renderQmdMCPConfigWithOptions generates the qmd MCP server configuration in JSON format.
// qmd uses HTTP transport (server started before the gateway), so only the URL is needed.
func renderQmdMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool, guardPolicies map[string]any, workflowData *WorkflowData) {
	url := qmdMCPURL(workflowData)

	yaml.WriteString("              \"qmd\": {\n")
	yaml.WriteString("                \"type\": \"http\",\n")

	if len(guardPolicies) > 0 {
		yaml.WriteString("                \"url\": \"" + url + "\",\n")
		renderGuardPoliciesJSON(yaml, guardPolicies, "                ")
	} else {
		yaml.WriteString("                \"url\": \"" + url + "\"\n")
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// RenderSerenaMCP generates Serena MCP server configuration
// RenderSafeOutputsMCP generates the Safe Outputs MCP server configuration
func (r *MCPConfigRendererUnified) RenderSafeOutputsMCP(yaml *strings.Builder, workflowData *WorkflowData) {
	mcpRendererLog.Printf("Rendering Safe Outputs MCP: format=%s", r.options.Format)

	if r.options.Format == "toml" {
		r.renderSafeOutputsTOML(yaml, workflowData)
		return
	}

	// JSON format
	renderSafeOutputsMCPConfigWithOptions(yaml, r.options.IsLast, r.options.IncludeCopilotFields, workflowData)
}

// renderSafeOutputsTOML generates Safe Outputs MCP configuration in TOML format
// Now uses HTTP transport instead of stdio, similar to mcp-scripts
func (r *MCPConfigRendererUnified) renderSafeOutputsTOML(yaml *strings.Builder, workflowData *WorkflowData) {
	// Determine host based on whether agent is disabled
	host := "host.docker.internal"
	if workflowData != nil && workflowData.SandboxConfig != nil && workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled {
		// When agent is disabled (no firewall), use localhost instead of host.docker.internal
		host = "localhost"
	}

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + constants.SafeOutputsMCPServerID.String() + "]\n")
	yaml.WriteString("          type = \"http\"\n")
	yaml.WriteString("          url = \"http://" + host + ":$GH_AW_SAFE_OUTPUTS_PORT\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + constants.SafeOutputsMCPServerID.String() + ".headers]\n")
	yaml.WriteString("          Authorization = \"$GH_AW_SAFE_OUTPUTS_API_KEY\"\n")

	// Check if GitHub tool has guard-policies configured (or auto-lockdown will run)
	// If so, generate a linked write-sink guard-policy for safeoutputs
	guardPolicies := deriveWriteSinkGuardPolicyFromWorkflow(workflowData)
	if len(guardPolicies) > 0 {
		mcpRendererLog.Print("Adding guard-policies to safeoutputs TOML (derived from GitHub guard-policy or auto-lockdown detection)")
		// Render guard-policies in TOML format
		renderGuardPoliciesToml(yaml, guardPolicies, constants.SafeOutputsMCPServerID.String())
	}
}

// RenderMCPScriptsMCP generates the MCP Scripts server configuration
func (r *MCPConfigRendererUnified) RenderMCPScriptsMCP(yaml *strings.Builder, mcpScripts *MCPScriptsConfig, workflowData *WorkflowData) {
	mcpRendererLog.Printf("Rendering MCP Scripts: format=%s", r.options.Format)

	if r.options.Format == "toml" {
		r.renderMCPScriptsTOML(yaml, mcpScripts, workflowData)
		// Add guard policies for TOML format as a separate section
		if len(r.options.WriteSinkGuardPolicies) > 0 {
			mcpRendererLog.Print("Adding guard-policies to mcp-scripts TOML (derived from GitHub guard-policy)")
			renderGuardPoliciesToml(yaml, r.options.WriteSinkGuardPolicies, constants.MCPScriptsMCPServerID.String())
		}
		return
	}

	// JSON format
	renderMCPScriptsMCPConfigWithOptions(yaml, mcpScripts, r.options.IsLast, r.options.IncludeCopilotFields, workflowData, r.options.WriteSinkGuardPolicies)
}

// renderMCPScriptsTOML generates MCP Scripts configuration in TOML format
// Uses HTTP transport exclusively
func (r *MCPConfigRendererUnified) renderMCPScriptsTOML(yaml *strings.Builder, mcpScripts *MCPScriptsConfig, workflowData *WorkflowData) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + constants.MCPScriptsMCPServerID.String() + "]\n")
	yaml.WriteString("          type = \"http\"\n")

	// Determine host based on whether agent is disabled
	host := "host.docker.internal"
	if workflowData != nil && workflowData.SandboxConfig != nil && workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled {
		// When agent is disabled (no firewall), use localhost instead of host.docker.internal
		host = "localhost"
		mcpRendererLog.Print("Using localhost for mcp-scripts (agent disabled)")
	} else {
		mcpRendererLog.Print("Using host.docker.internal for mcp-scripts (agent enabled)")
	}

	yaml.WriteString("          url = \"http://" + host + ":$GH_AW_MCP_SCRIPTS_PORT\"\n")
	yaml.WriteString("          headers = { Authorization = \"$GH_AW_MCP_SCRIPTS_API_KEY\" }\n")
	// Note: env_vars is not supported for HTTP transport in MCP configuration
	// Environment variables are passed via the workflow job's env: section instead
}

// RenderAgenticWorkflowsMCP generates the Agentic Workflows MCP server configuration
func (r *MCPConfigRendererUnified) RenderAgenticWorkflowsMCP(yaml *strings.Builder) {
	mcpRendererLog.Printf("Rendering Agentic Workflows MCP: format=%s, action_mode=%s", r.options.Format, r.options.ActionMode)

	if r.options.Format == "toml" {
		r.renderAgenticWorkflowsTOML(yaml)
		// Add guard policies for TOML format as a separate section
		if len(r.options.WriteSinkGuardPolicies) > 0 {
			mcpRendererLog.Print("Adding guard-policies to agentic-workflows TOML (derived from GitHub guard-policy)")
			renderGuardPoliciesToml(yaml, r.options.WriteSinkGuardPolicies, constants.AgenticWorkflowsMCPServerID.String())
		}
		return
	}

	// JSON format
	renderAgenticWorkflowsMCPConfigWithOptions(yaml, r.options.IsLast, r.options.IncludeCopilotFields, r.options.ActionMode, r.options.WriteSinkGuardPolicies)
}

// renderAgenticWorkflowsTOML generates Agentic Workflows MCP configuration in TOML format
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
func (r *MCPConfigRendererUnified) renderAgenticWorkflowsTOML(yaml *strings.Builder) {
	mcpRendererBuiltinLog.Printf("Rendering Agentic Workflows MCP in TOML format: action_mode=%s", r.options.ActionMode)
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + constants.AgenticWorkflowsMCPServerID.String() + "]\n")

	containerImage := constants.DefaultAlpineImage
	var entrypoint string
	var entrypointArgs []string
	var mounts []string

	if r.options.ActionMode.IsDev() {
		// Dev mode: Use locally built Docker image which includes gh-aw binary and gh CLI
		// The Dockerfile sets ENTRYPOINT ["gh-aw"] and CMD ["mcp-server", "--validate-actor"]
		// So we don't need to specify entrypoint or entrypointArgs
		containerImage = constants.DevModeGhAwImage
		entrypoint = ""      // Use container's default ENTRYPOINT
		entrypointArgs = nil // Use container's default CMD
		// Only mount workspace and temp directory - binary and gh CLI are in the image
		mounts = []string{constants.DefaultWorkspaceMount, constants.DefaultTmpGhAwMount}
	} else {
		// Release mode: Use minimal Alpine image with mounted binaries
		entrypoint = "${RUNNER_TEMP}/gh-aw/gh-aw"
		entrypointArgs = []string{"mcp-server", "--validate-actor"}
		// Mount gh-aw binary, gh CLI binary, workspace, and temp directory
		mounts = []string{constants.DefaultGhAwMount, constants.DefaultGhBinaryMount, constants.DefaultWorkspaceMount, constants.DefaultTmpGhAwMount}
	}

	yaml.WriteString("          container = \"" + containerImage + "\"\n")

	// Only write entrypoint if it's specified (release mode)
	// In dev mode, use the container's default ENTRYPOINT
	if entrypoint != "" {
		yaml.WriteString("          entrypoint = \"" + entrypoint + "\"\n")
	}

	// Only write entrypointArgs if specified (release mode)
	// In dev mode, use the container's default CMD
	if entrypointArgs != nil {
		yaml.WriteString("          entrypointArgs = [")
		for i, arg := range entrypointArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		yaml.WriteString("]\n")
	}

	// Write mounts
	yaml.WriteString("          mounts = [")
	for i, mount := range mounts {
		if i > 0 {
			yaml.WriteString(", ")
		}
		yaml.WriteString("\"" + mount + "\"")
	}
	yaml.WriteString("]\n")

	yaml.WriteString("          env_vars = [\"DEBUG\", \"GH_TOKEN\", \"GITHUB_TOKEN\", \"GITHUB_ACTOR\", \"GITHUB_REPOSITORY\"]\n")
}
