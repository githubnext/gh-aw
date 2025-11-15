package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpRendererLog = logger.New("workflow:mcp_renderer")

// MCPRendererOptions contains configuration options for the unified MCP renderer
type MCPRendererOptions struct {
	// IncludeCopilotFields indicates if the engine requires "type" and "tools" fields (true for copilot engine)
	IncludeCopilotFields bool
	// InlineArgs indicates if args should be rendered inline (true for copilot) or multi-line (false for claude/custom)
	InlineArgs bool
	// Format specifies the output format ("json" for JSON-like, "toml" for TOML-like)
	Format string
	// IsLast indicates if this is the last server in the configuration (affects trailing comma)
	IsLast bool
}

// MCPConfigRendererUnified provides unified rendering methods for MCP configurations
// across different engines (Claude, Copilot, Codex, Custom)
type MCPConfigRendererUnified struct {
	options MCPRendererOptions
}

// NewMCPConfigRenderer creates a new unified MCP config renderer with the specified options
func NewMCPConfigRenderer(opts MCPRendererOptions) *MCPConfigRendererUnified {
	mcpRendererLog.Printf("Creating MCP renderer: format=%s, copilot_fields=%t, inline_args=%t, is_last=%t",
		opts.Format, opts.IncludeCopilotFields, opts.InlineArgs, opts.IsLast)
	return &MCPConfigRendererUnified{
		options: opts,
	}
}

// RenderGitHubMCP generates the GitHub MCP server configuration
// Supports both local (Docker) and remote (hosted) modes
func (r *MCPConfigRendererUnified) RenderGitHubMCP(yaml *strings.Builder, githubTool any, workflowData *WorkflowData) {
	githubType := getGitHubType(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	toolsets := getGitHubToolsets(githubTool)

	mcpRendererLog.Printf("Rendering GitHub MCP: type=%s, read_only=%t, toolsets=%v, format=%s",
		githubType, readOnly, toolsets, r.options.Format)

	if r.options.Format == "toml" {
		// TOML format doesn't support GitHub MCP yet
		return
	}

	yaml.WriteString("              \"github\": {\n")

	// Check if remote mode is enabled (type: remote)
	if githubType == "remote" {
		// Determine authorization value based on engine requirements
		// Copilot uses MCP passthrough syntax: "Bearer \${GITHUB_PERSONAL_ACCESS_TOKEN}"
		// Other engines use shell variable: "Bearer $GITHUB_MCP_SERVER_TOKEN"
		authValue := "Bearer $GITHUB_MCP_SERVER_TOKEN"
		if r.options.IncludeCopilotFields {
			authValue = "Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}"
		}
		
		RenderGitHubMCPRemoteConfig(yaml, GitHubMCPRemoteOptions{
			ReadOnly:           readOnly,
			Toolsets:           toolsets,
			AuthorizationValue: authValue,
			IncludeToolsField:  r.options.IncludeCopilotFields,
			AllowedTools:       getGitHubAllowedTools(githubTool),
			IncludeEnvSection:  r.options.IncludeCopilotFields,
		})
	} else {
		// Local mode - use Docker-based GitHub MCP server (default)
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)

		RenderGitHubMCPDockerConfig(yaml, GitHubMCPDockerOptions{
			ReadOnly:           readOnly,
			Toolsets:           toolsets,
			DockerImageVersion: githubDockerImageVersion,
			CustomArgs:         customArgs,
			IncludeTypeField:   r.options.IncludeCopilotFields,
			AllowedTools:       getGitHubAllowedTools(githubTool),
			EffectiveToken:     "", // Token passed via env
		})
	}

	if r.options.IsLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// RenderPlaywrightMCP generates the Playwright MCP server configuration
func (r *MCPConfigRendererUnified) RenderPlaywrightMCP(yaml *strings.Builder, playwrightTool any) {
	mcpRendererLog.Printf("Rendering Playwright MCP: format=%s, inline_args=%t", r.options.Format, r.options.InlineArgs)

	if r.options.Format == "toml" {
		r.renderPlaywrightTOML(yaml, playwrightTool)
		return
	}

	// JSON format
	renderPlaywrightMCPConfigWithOptions(yaml, playwrightTool, r.options.IsLast, r.options.IncludeCopilotFields, r.options.InlineArgs)
}

// renderPlaywrightTOML generates Playwright MCP configuration in TOML format
func (r *MCPConfigRendererUnified) renderPlaywrightTOML(yaml *strings.Builder, playwrightTool any) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.playwright]\n")
	yaml.WriteString("          command = \"npx\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"@playwright/mcp@latest\",\n")
	yaml.WriteString("            \"--output-dir\",\n")
	yaml.WriteString("            \"/tmp/gh-aw/mcp-logs/playwright\"")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("            \"--allowed-origins\",\n")
		yaml.WriteString("            \"" + strings.Join(args.AllowedDomains, ";") + "\"")
	}

	// Append custom args if present
	writeArgsToYAML(yaml, customArgs, "            ")

	yaml.WriteString("\n")
	yaml.WriteString("          ]\n")
}

// RenderSafeOutputsMCP generates the Safe Outputs MCP server configuration
func (r *MCPConfigRendererUnified) RenderSafeOutputsMCP(yaml *strings.Builder) {
	mcpRendererLog.Printf("Rendering Safe Outputs MCP: format=%s", r.options.Format)

	if r.options.Format == "toml" {
		r.renderSafeOutputsTOML(yaml)
		return
	}

	// JSON format
	renderSafeOutputsMCPConfigWithOptions(yaml, r.options.IsLast, r.options.IncludeCopilotFields)
}

// renderSafeOutputsTOML generates Safe Outputs MCP configuration in TOML format
func (r *MCPConfigRendererUnified) renderSafeOutputsTOML(yaml *strings.Builder) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers." + constants.SafeOutputsMCPServerID + "]\n")
	yaml.WriteString("          command = \"node\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"/tmp/gh-aw/safeoutputs/mcp-server.cjs\",\n")
	yaml.WriteString("          ]\n")
	yaml.WriteString("          env_vars = [\"GH_AW_SAFE_OUTPUTS\", \"GH_AW_ASSETS_BRANCH\", \"GH_AW_ASSETS_MAX_SIZE_KB\", \"GH_AW_ASSETS_ALLOWED_EXTS\", \"GITHUB_REPOSITORY\", \"GITHUB_SERVER_URL\"]\n")
}

// RenderAgenticWorkflowsMCP generates the Agentic Workflows MCP server configuration
func (r *MCPConfigRendererUnified) RenderAgenticWorkflowsMCP(yaml *strings.Builder) {
	mcpRendererLog.Printf("Rendering Agentic Workflows MCP: format=%s", r.options.Format)

	if r.options.Format == "toml" {
		r.renderAgenticWorkflowsTOML(yaml)
		return
	}

	// JSON format
	renderAgenticWorkflowsMCPConfigWithOptions(yaml, r.options.IsLast, r.options.IncludeCopilotFields)
}

// renderAgenticWorkflowsTOML generates Agentic Workflows MCP configuration in TOML format
func (r *MCPConfigRendererUnified) renderAgenticWorkflowsTOML(yaml *strings.Builder) {
	yaml.WriteString("          \n")
	yaml.WriteString("          [mcp_servers.agentic_workflows]\n")
	yaml.WriteString("          command = \"gh\"\n")
	yaml.WriteString("          args = [\n")
	yaml.WriteString("            \"aw\",\n")
	yaml.WriteString("            \"mcp-server\",\n")
	yaml.WriteString("          ]\n")
	yaml.WriteString("          env_vars = [\"GITHUB_TOKEN\"]\n")
}
