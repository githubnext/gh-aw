// Package workflow provides Playwright MCP server configuration and Docker setup.
//
// # Playwright MCP Server
//
// This file handles the configuration and rendering of the Playwright MCP server,
// which provides AI agents with browser automation capabilities through the
// Model Context Protocol (MCP). Playwright enables agents to interact with
// web pages, take screenshots, extract content, and perform accessibility testing.
//
// Key responsibilities:
//   - Generating Playwright MCP server configuration
//   - Managing Docker container setup for Playwright
//   - Processing custom Playwright arguments
//   - Rendering configuration for different AI engines
//
// Container configuration:
// Playwright runs in a Docker container using the official Microsoft Playwright
// MCP image (mcr.microsoft.com/playwright/mcp). The container is configured with:
//   - --init flag for proper signal handling
//   - --network host for network access
//   - --security-opt seccomp=unconfined for Chromium sandbox compatibility
//   - --ipc=host for shared memory access required by Chromium
//   - Volume mounts for log storage
//   - Output directory for screenshots and artifacts
//
// GitHub Actions compatibility:
// The security flags are required for Chromium to function properly on GitHub Actions
// runners. Without these flags, Playwright initialization fails with "EOF" error because
// Chromium crashes during startup due to sandbox constraints.
//
// Network access:
// Network egress for Playwright is controlled by the workflow firewall (network.allowed).
// Use the top-level network configuration to specify allowed domains.
//
// Engine compatibility:
// The renderer supports multiple AI engines with engine-specific formatting:
//   - Copilot: Includes "type" field, inline args
//   - Claude/Custom: Multi-line args, simplified format
//   - All engines: Same core configuration structure
//
// Related files:
//   - mcp_playwright_config.go: Playwright configuration types and parsing
//   - mcp_renderer.go: Main MCP renderer that calls this function
//   - mcp_setup_generator.go: Includes Playwright in setup sequence
//
// Example configuration:
//
//	tools:
//	  playwright:
//	    version: v1.41.0
//	    args:
//	      - --debug
//	      - --timeout=30000
//	network:
//	  allowed:
//	    - github.com
//	    - api.github.com
package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpPlaywrightLog = logger.New("workflow:mcp_config_playwright_renderer")

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses Docker container to launch Playwright MCP for consistent browser environment
// This is a shared function used by both Claude and Custom engines
func renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightConfig *PlaywrightToolConfig, isLast bool) {
	mcpPlaywrightLog.Print("Rendering Playwright MCP configuration")
	renderPlaywrightMCPConfigWithOptions(yaml, playwrightConfig, isLast, false, false)
}

// renderPlaywrightMCPConfigWithOptions generates the Playwright MCP server configuration with engine-specific options
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypointArgs, mounts, and args fields.
func renderPlaywrightMCPConfigWithOptions(yaml *strings.Builder, playwrightConfig *PlaywrightToolConfig, isLast bool, includeCopilotFields bool, inlineArgs bool) {
	customArgs := getPlaywrightCustomArgs(playwrightConfig)

	// Extract all expressions from playwright arguments and replace them with env var references
	expressions := extractExpressionsFromPlaywrightArgs(customArgs)

	// Replace expressions in custom args
	if len(customArgs) > 0 {
		customArgs = replaceExpressionsInPlaywrightArgs(customArgs, expressions)
	}

	// Use official Playwright MCP Docker image (no version tag - only one image)
	playwrightImage := "mcr.microsoft.com/playwright/mcp"

	yaml.WriteString("              \"playwright\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0, use "stdio" for containerized servers)
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	// MCP Gateway spec fields for containerized stdio servers
	yaml.WriteString("                \"container\": \"" + playwrightImage + "\",\n")

	// Docker runtime args (goes before container image in docker run command)
	// These are additional flags for docker run like --init and --network
	// Add security-opt and ipc flags for Chromium browser compatibility in GitHub Actions
	// --security-opt seccomp=unconfined: Required for Chromium sandbox to function properly
	// --ipc=host: Provides shared memory access required by Chromium
	dockerArgs := []string{"--init", "--network", "host", "--security-opt", "seccomp=unconfined", "--ipc=host"}
	if inlineArgs {
		yaml.WriteString("                \"args\": [")
		for i, arg := range dockerArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		yaml.WriteString("],\n")
	} else {
		yaml.WriteString("                \"args\": [\n")
		for i, arg := range dockerArgs {
			yaml.WriteString("                  \"" + arg + "\"")
			if i < len(dockerArgs)-1 {
				yaml.WriteString(",")
			}
			yaml.WriteString("\n")
		}
		yaml.WriteString("                ],\n")
	}

	// Build entrypoint args for Playwright MCP server (goes after container image)
	// --no-sandbox: Disables Chromium's process sandbox, which otherwise
	// creates a network namespace for renderer processes that cannot reach localhost.
	// This is required for screenshot workflows that serve docs on localhost.
	// Note: as of @playwright/mcp v0.0.26+, --no-sandbox is a direct top-level flag.
	entrypointArgs := []string{"--output-dir", "/tmp/gh-aw/mcp-logs/playwright", "--no-sandbox"}
	// Append custom args if present
	if len(customArgs) > 0 {
		entrypointArgs = append(entrypointArgs, customArgs...)
	}

	// Render entrypointArgs
	if inlineArgs {
		yaml.WriteString("                \"entrypointArgs\": [")
		for i, arg := range entrypointArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		yaml.WriteString("],\n")
	} else {
		yaml.WriteString("                \"entrypointArgs\": [\n")
		for i, arg := range entrypointArgs {
			yaml.WriteString("                  \"" + arg + "\"")
			if i < len(entrypointArgs)-1 {
				yaml.WriteString(",")
			}
			yaml.WriteString("\n")
		}
		yaml.WriteString("                ],\n")
	}

	// Add volume mounts
	yaml.WriteString("                \"mounts\": [\"/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs:rw\"]\n")

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}
