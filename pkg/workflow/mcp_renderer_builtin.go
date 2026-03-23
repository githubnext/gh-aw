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
// qmd runs as a containerized stdio MCP server started by the gateway, with the
// pre-built index and embedding models mounted from the host via Actions cache.
func (r *MCPConfigRendererUnified) RenderQmdMCP(yaml *strings.Builder, qmdTool any) {
	mcpRendererLog.Printf("Rendering qmd MCP: format=%s, inline_args=%t", r.options.Format, r.options.InlineArgs)

	if r.options.Format == "toml" {
		r.renderQmdTOML(yaml)
		return
	}

	// JSON format
	renderQmdMCPConfigWithOptions(yaml, r.options.IsLast, r.options.IncludeCopilotFields, r.options.InlineArgs)
}

// renderQmdTOML generates qmd MCP configuration in TOML format using a containerized stdio server.
// The gateway starts the container, mounting the pre-built index (/tmp/gh-aw/qmd-index/) and
// embedding models (${HOME}/.cache/qmd/) from the host. INDEX_PATH and HOME env vars are
// forwarded to the container so qmd and node-llama-cpp locate the correct files.
func (r *MCPConfigRendererUnified) renderQmdTOML(yaml *strings.Builder) {
	mcpRendererBuiltinLog.Print("Rendering qmd MCP in TOML format (container stdio)")

	version := string(constants.DefaultQmdVersion)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.qmd]\n")
	yaml.WriteString("          container = \"node:24\"\n")
	yaml.WriteString("          entrypoint = \"npx\"\n")
	yaml.WriteString("          entrypointArgs = [\n")
	yaml.WriteString("            \"--yes\",\n")
	yaml.WriteString("            \"--package\",\n")
	yaml.WriteString("            \"@tobilu/qmd@" + version + "\",\n")
	yaml.WriteString("            \"qmd\",\n")
	yaml.WriteString("            \"mcp\",\n")
	yaml.WriteString("          ]\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"--network\",\n")
	yaml.WriteString("            \"host\",\n")
	yaml.WriteString("          ]\n")
	// Mount the qmd index (under /tmp/gh-aw/) and the embedding models cache.
	// The node-llama-cpp binary cache is not mounted; the container downloads the
	// appropriate prebuilt binary for its own OS on first use.
	yaml.WriteString("          mounts = [\n")
	yaml.WriteString("            \"/tmp/gh-aw:/tmp/gh-aw:rw\",\n")
	yaml.WriteString("            \"${HOME}/.cache/qmd:${HOME}/.cache/qmd:rw\",\n")
	yaml.WriteString("          ]\n")
	// Forward INDEX_PATH (location of the SQLite index) and HOME (so node-llama-cpp
	// and qmd resolve ~/.cache/ paths correctly inside the container).
	// NODE_LLAMA_CPP_GPU is forwarded so GPU probing can be disabled on CPU-only runners.
	// Use \${VAR} so the shell heredoc does not expand them; the gateway resolves them.
	// HOME is not in the gateway env so it is expanded by the heredoc shell instead.
	yaml.WriteString("          env = { \"INDEX_PATH\" = \"\\${INDEX_PATH}\", \"HOME\" = \"${HOME}\", \"NODE_LLAMA_CPP_GPU\" = \"\\${NODE_LLAMA_CPP_GPU}\" }\n")
}

// renderQmdMCPConfigWithOptions generates the qmd MCP server configuration in JSON format.
// qmd uses a containerized stdio server started by the MCP gateway, with mounts for
// the pre-built index and embedding models.
func renderQmdMCPConfigWithOptions(yaml *strings.Builder, isLast bool, includeCopilotFields bool, inlineArgs bool) {
	version := string(constants.DefaultQmdVersion)
	qmdArgs := []string{"--yes", "--package", "@tobilu/qmd@" + version, "qmd", "mcp"}
	dockerArgs := []string{"--network", "host"}
	mounts := []string{"/tmp/gh-aw:/tmp/gh-aw:rw", "${HOME}/.cache/qmd:${HOME}/.cache/qmd:rw"}
	// env uses \${VAR} so the heredoc shell does not expand INDEX_PATH and NODE_LLAMA_CPP_GPU;
	// the gateway resolves them from its own environment (passed via -e flags in DOCKER_COMMAND).
	// HOME is not in the gateway env, so ${HOME} is expanded by the heredoc shell to /home/runner.
	envValues := map[string]string{
		"INDEX_PATH":         "\\${INDEX_PATH}",
		"HOME":               "${HOME}",
		"NODE_LLAMA_CPP_GPU": "\\${NODE_LLAMA_CPP_GPU}",
	}
	envKeys := sortedMapKeys(envValues)

	yaml.WriteString("              \"qmd\": {\n")

	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	yaml.WriteString("                \"container\": \"node:24\",\n")
	yaml.WriteString("                \"entrypoint\": \"npx\",\n")

	if inlineArgs {
		// Entrypoint args inline
		yaml.WriteString("                \"entrypointArgs\": [")
		for i, arg := range qmdArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		yaml.WriteString("],\n")
		// Docker args inline
		yaml.WriteString("                \"args\": [")
		for i, arg := range dockerArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		yaml.WriteString("],\n")
		// Mounts inline
		yaml.WriteString("                \"mounts\": [")
		for i, m := range mounts {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + m + "\"")
		}
		yaml.WriteString("],\n")
		// Env object inline
		yaml.WriteString("                \"env\": {")
		for i, key := range envKeys {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + key + "\": \"" + envValues[key] + "\"")
		}
		yaml.WriteString("}\n")
	} else {
		// Entrypoint args multi-line
		yaml.WriteString("                \"entrypointArgs\": [\n")
		for i, arg := range qmdArgs {
			if i < len(qmdArgs)-1 {
				yaml.WriteString("                  \"" + arg + "\",\n")
			} else {
				yaml.WriteString("                  \"" + arg + "\"\n")
			}
		}
		yaml.WriteString("                ],\n")
		// Docker args multi-line
		yaml.WriteString("                \"args\": [\n")
		for i, arg := range dockerArgs {
			if i < len(dockerArgs)-1 {
				yaml.WriteString("                  \"" + arg + "\",\n")
			} else {
				yaml.WriteString("                  \"" + arg + "\"\n")
			}
		}
		yaml.WriteString("                ],\n")
		// Mounts multi-line
		yaml.WriteString("                \"mounts\": [\n")
		for i, m := range mounts {
			if i < len(mounts)-1 {
				yaml.WriteString("                  \"" + m + "\",\n")
			} else {
				yaml.WriteString("                  \"" + m + "\"\n")
			}
		}
		yaml.WriteString("                ],\n")
		// Env object multi-line
		yaml.WriteString("                \"env\": {\n")
		for i, key := range envKeys {
			if i < len(envKeys)-1 {
				yaml.WriteString("                  \"" + key + "\": \"" + envValues[key] + "\",\n")
			} else {
				yaml.WriteString("                  \"" + key + "\": \"" + envValues[key] + "\"\n")
			}
		}
		yaml.WriteString("                }\n")
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// RenderSerenaMCP generates Serena MCP server configuration
func (r *MCPConfigRendererUnified) RenderSerenaMCP(yaml *strings.Builder, serenaTool any) {
	mcpRendererLog.Printf("Rendering Serena MCP: format=%s, inline_args=%t", r.options.Format, r.options.InlineArgs)

	if r.options.Format == "toml" {
		r.renderSerenaTOML(yaml, serenaTool)
		// Add guard policies for TOML format as a separate section
		if len(r.options.WriteSinkGuardPolicies) > 0 {
			mcpRendererLog.Print("Adding guard-policies to serena TOML (derived from GitHub guard-policy)")
			renderGuardPoliciesToml(yaml, r.options.WriteSinkGuardPolicies, "serena")
		}
		return
	}

	// JSON format
	renderSerenaMCPConfigWithOptions(yaml, serenaTool, r.options.IsLast, r.options.IncludeCopilotFields, r.options.InlineArgs, r.options.WriteSinkGuardPolicies)
}

// renderSerenaTOML generates Serena MCP configuration in TOML format
// Supports two modes:
// - "docker" (default): Uses Docker container with stdio transport
// - "local": Uses local uvx with HTTP transport
func (r *MCPConfigRendererUnified) renderSerenaTOML(yaml *strings.Builder, serenaTool any) {
	mcpRendererBuiltinLog.Print("Rendering Serena MCP in TOML format")
	customArgs := getSerenaCustomArgs(serenaTool)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.serena]\n")

	// Docker mode: use stdio transport (default)
	// Select the appropriate Serena container based on requested languages
	containerImage := selectSerenaContainer(serenaTool)
	yaml.WriteString("          container = \"" + containerImage + ":latest\"\n")

	// Docker runtime args (--network host for network access)
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"--network\",\n")
	yaml.WriteString("            \"host\",\n")
	yaml.WriteString("          ]\n")

	// Serena entrypoint
	yaml.WriteString("          entrypoint = \"serena\"\n")

	// Entrypoint args for Serena MCP server
	yaml.WriteString("          entrypointArgs = [\n")
	yaml.WriteString("            \"start-mcp-server\",\n")
	yaml.WriteString("            \"--context\",\n")
	yaml.WriteString("            \"codex\",\n")
	yaml.WriteString("            \"--project\",\n")
	// Security: Use GITHUB_WORKSPACE environment variable instead of template expansion to prevent template injection
	yaml.WriteString("            \"${GITHUB_WORKSPACE}\"")

	// Append custom args if present
	for _, arg := range customArgs {
		yaml.WriteString(",\n")
		yaml.WriteString("            " + strconv.Quote(arg))
	}

	yaml.WriteString("\n")
	yaml.WriteString("          ]\n")

	// Add volume mount for workspace access
	// Security: Use GITHUB_WORKSPACE environment variable instead of template expansion to prevent template injection
	yaml.WriteString("          mounts = [\"${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw\"]\n")
}

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
