// Package workflow provides GitHub Actions setup step generation for MCP servers.
//
// # MCP Setup Generator
//
// This file generates the complete setup sequence for MCP servers in GitHub Actions
// workflows. It orchestrates the initialization of all MCP tools including built-in
// servers (GitHub, Playwright, safe-outputs, mcp-scripts) and custom HTTP/stdio
// MCP servers.
//
// Key responsibilities:
//   - Identifying and collecting MCP tools from workflow configuration
//   - Generating Docker image download steps
//   - Installing gh-aw extension for agentic-workflows tool
//   - Setting up safe-outputs MCP server (config, API key, HTTP server)
//   - Setting up mcp-scripts MCP server (config, tool files, HTTP server)
//   - Starting the MCP gateway with proper environment variables
//   - Rendering MCP configuration for the selected AI engine
//
// Setup sequence:
//  1. Download required Docker images
//  2. Install gh-aw extension (if agentic-workflows enabled)
//  3. Write safe-outputs config.json (may contain template expressions; kept small)
//  4. Write safe-outputs tools.json and validation.json (large, no template expressions)
//  5. Generate and start safe-outputs HTTP server
//  6. Setup mcp-scripts config and tool files (JavaScript, Python, Shell, Go)
//  7. Generate and start mcp-scripts HTTP server
//  8. Start MCP Gateway with all environment variables

// 10. Render engine-specific MCP configuration
//
// MCP tools supported:
//   - github: GitHub API access via MCP (local Docker or remote hosted)
//   - playwright: Browser automation with Playwright
//   - safe-outputs: Controlled output storage for AI agents
//   - mcp-scripts: Custom tool execution with secret passthrough
//   - cache-memory: Memory/knowledge base management
//   - agentic-workflows: Workflow execution via gh-aw
//   - Custom HTTP/stdio MCP servers
//
// Gateway modes:
//   - Enabled (default): MCP servers run through gateway proxy
//   - Disabled (sandbox: false): Direct MCP server communication
//
// Related files:
//   - mcp_gateway_config.go: Gateway configuration management
//   - mcp_environment.go: Environment variable collection
//   - mcp_renderer.go: MCP configuration YAML rendering
//   - safe_outputs.go: Safe outputs server configuration
//   - mcp_scripts.go: MCP Scripts server configuration
//
// Example workflow setup:
//   - Download Docker images
//   - Write safe-outputs config to ${RUNNER_TEMP}/gh-aw/safeoutputs/
//   - Start safe-outputs HTTP server on port 3001
//   - Write mcp-scripts config to ${RUNNER_TEMP}/gh-aw/mcp-scripts/
//   - Start mcp-scripts HTTP server on port 3000
//   - Start MCP Gateway (default port 8080)
//   - Render MCP config based on engine (copilot/claude/codex/custom)
package workflow

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var mcpSetupGeneratorLog = logger.New("workflow:mcp_setup_generator")

// generateMCPSetup generates the MCP server configuration setup
func (c *Compiler) generateMCPSetup(yaml *strings.Builder, tools map[string]any, engine CodingAgentEngine, workflowData *WorkflowData) error {
	mcpSetupGeneratorLog.Print("Generating MCP server configuration setup")
	// Collect tools that need MCP server configuration
	var mcpTools []string

	// Check if workflowData is valid before accessing its fields
	if workflowData == nil {
		return nil
	}

	workflowTools := workflowData.Tools

	for toolName, toolValue := range workflowTools {
		// Skip if the tool is explicitly disabled (set to false)
		if toolValue == false {
			continue
		}
		// When cli-proxy is enabled, agents use the pre-authenticated gh CLI for GitHub
		// reads instead of the GitHub MCP server. Skip so it is not configured with the gateway.
		if toolName == "github" && isFeatureEnabled(constants.CliProxyFeatureFlag, workflowData) {
			mcpSetupGeneratorLog.Print("Skipping GitHub MCP server registration: cli-proxy feature flag is enabled")
			continue
		}
		// Standard MCP tools
		if toolName == "github" || toolName == "playwright" || toolName == "cache-memory" || toolName == "agentic-workflows" {
			mcpTools = append(mcpTools, toolName)
		} else if mcpConfig, ok := toolValue.(map[string]any); ok {
			// Check if it's explicitly marked as MCP type in the new format
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				mcpTools = append(mcpTools, toolName)
			}
		}
	}

	// Check if safe-outputs is enabled and add to MCP tools
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		mcpTools = append(mcpTools, "safe-outputs")
	}

	// Check if mcp-scripts is configured and feature flag is enabled, add to MCP tools
	if IsMCPScriptsEnabled(workflowData.MCPScripts, workflowData) {
		mcpTools = append(mcpTools, "mcp-scripts")
	}

	// Populate dispatch-workflow file mappings before generating config
	// This ensures workflow_files is available in the config.json
	populateDispatchWorkflowFiles(workflowData, c.markdownPath)

	// Populate call-workflow file mappings before generating config
	// This ensures workflow_files is available in the config.json
	populateCallWorkflowFiles(workflowData, c.markdownPath)

	// Generate safe-outputs configuration once to avoid duplicate computation
	var safeOutputConfig string
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		var err error
		safeOutputConfig, err = generateSafeOutputsConfig(workflowData)
		if err != nil {
			return fmt.Errorf("failed to generate safe outputs config: %w", err)
		}
	}

	// Sort tools to ensure stable code generation
	sort.Strings(mcpTools)

	if mcpSetupGeneratorLog.Enabled() {
		mcpSetupGeneratorLog.Printf("Collected %d MCP tools: %v", len(mcpTools), mcpTools)
	}

	// Ensure MCP gateway config has defaults set before collecting Docker images
	ensureDefaultMCPGatewayConfig(workflowData)

	// Collect all Docker images that will be used and generate download step
	dockerImages := collectDockerImages(tools, workflowData, c.actionMode)
	generateDownloadDockerImagesStep(yaml, dockerImages)

	// If no MCP tools, no configuration needed
	if len(mcpTools) == 0 {
		mcpSetupGeneratorLog.Print("No MCP tools configured, skipping MCP setup")
		return nil
	}

	// Install gh-aw extension if agentic-workflows tool is enabled
	hasAgenticWorkflows := slices.Contains(mcpTools, "agentic-workflows")
	if hasAgenticWorkflows {
		if hasSharedGhAwImport(workflowData.ImportedFiles) {
			mcpSetupGeneratorLog.Print("Skipping gh-aw extension installation step (provided by shared/mcp/gh-aw.md import)")
		} else {
			generateGhAwExtensionInstallStep(yaml)
		}
	}

	if err := c.generateSafeOutputsMCPSetup(yaml, workflowData, safeOutputConfig); err != nil {
		return err
	}

	if err := c.generateMCPScriptsMCPSetup(yaml, workflowData); err != nil {
		return err
	}

	return c.generateMCPGatewaySetup(yaml, tools, engine, workflowData, mcpTools, hasAgenticWorkflows)
}
