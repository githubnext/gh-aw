// Package workflow provides Playwright MCP server configuration and Docker setup.
//
// # Playwright MCP Server
//
// This file handles the configuration and rendering of the Playwright MCP server,
// which provides AI agents with browser automation capabilities through the
// Model Context Protocol (MCP). Playwright enables agents to interact with
// web pages, take screenshots, extract content, and perform accessibility testing.
//
// Two modes are supported, selected via tools.playwright.mode:
//
//   - "cli" (default): Runs @playwright/mcp via npx directly on the runner,
//     without Docker. Simpler setup, no Docker image pull required.
//   - "mcp": Runs the official Microsoft Playwright MCP Docker image
//     (mcr.microsoft.com/playwright/mcp) via the MCP gateway.
//
// # CLI mode
//
// CLI mode invokes `npx -y @playwright/mcp[@version]` as a stdio MCP server.
// The package is downloaded and executed by npx, so no pre-installation is needed.
//
// # MCP (Docker) mode
//
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
//	  playwright:           # cli mode (default)
//	  playwright:
//	    mode: cli           # explicit cli mode (default)
//	    version: "0.0.26"  # optional @playwright/mcp version
//	    args:
//	      - --timeout=30000
//	  playwright:
//	    mode: mcp           # docker-based MCP mode
//	    args:
//	      - --debug
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

// playwrightNpxPackage returns the npx package reference for CLI mode.
// Uses the version from config if specified, otherwise defaults to "latest".
func playwrightNpxPackage(playwrightConfig *PlaywrightToolConfig) string {
	if playwrightConfig != nil && playwrightConfig.Version != "" {
		return "@playwright/mcp@" + playwrightConfig.Version
	}
	return "@playwright/mcp@latest"
}

// renderPlaywrightMCPConfigWithOptions generates the Playwright MCP server configuration with engine-specific options.
// Routes between cli and mcp modes based on playwrightConfig.Mode.
func renderPlaywrightMCPConfigWithOptions(yaml *strings.Builder, playwrightConfig *PlaywrightToolConfig, isLast bool, includeCopilotFields bool, inlineArgs bool, guardPolicies map[string]any) {
	if playwrightConfig.IsCliMode() {
		renderPlaywrightCLIConfigJSON(yaml, playwrightConfig, isLast, includeCopilotFields, inlineArgs, guardPolicies)
	} else {
		renderPlaywrightDockerConfigJSON(yaml, playwrightConfig, isLast, includeCopilotFields, inlineArgs, guardPolicies)
	}
}

// renderPlaywrightCLIConfigJSON generates Playwright CLI mode configuration in JSON format.
// Runs @playwright/mcp via npx as a stdio MCP server without Docker.
func renderPlaywrightCLIConfigJSON(yaml *strings.Builder, playwrightConfig *PlaywrightToolConfig, isLast bool, includeCopilotFields bool, inlineArgs bool, guardPolicies map[string]any) {
	mcpPlaywrightLog.Printf("Rendering Playwright CLI config (JSON): copilot_fields=%t, inline_args=%t", includeCopilotFields, inlineArgs)
	customArgs := getPlaywrightCustomArgs(playwrightConfig)

	// Extract and replace expressions from custom args
	expressions := extractExpressionsFromPlaywrightArgs(customArgs)
	if len(customArgs) > 0 {
		mcpPlaywrightLog.Printf("Applying %d custom Playwright args with %d extracted expressions", len(customArgs), len(expressions))
		customArgs = replaceExpressionsInPlaywrightArgs(customArgs, expressions)
	}

	// Build the full args list: npx package + MCP server flags + custom args
	npxPackage := playwrightNpxPackage(playwrightConfig)
	// --no-sandbox: Required on GitHub Actions runners for Chromium process sandbox
	// --output-dir:  Directory for screenshots and artifacts
	allArgs := append([]string{"-y", npxPackage, "--output-dir", "/tmp/gh-aw/mcp-logs/playwright", "--no-sandbox"}, customArgs...)

	yaml.WriteString("              \"playwright\": {\n")

	// Add type field for Copilot (per MCP Gateway Specification v1.0.0)
	if includeCopilotFields {
		yaml.WriteString("                \"type\": \"stdio\",\n")
	}

	yaml.WriteString("                \"command\": \"npx\",\n")

	// Determine if args field has a trailing comma (guard policies follow) or not
	hasGuardPolicies := len(guardPolicies) > 0
	if inlineArgs {
		yaml.WriteString("                \"args\": [")
		for i, arg := range allArgs {
			if i > 0 {
				yaml.WriteString(", ")
			}
			yaml.WriteString("\"" + arg + "\"")
		}
		if hasGuardPolicies {
			yaml.WriteString("],\n")
		} else {
			yaml.WriteString("]\n")
		}
	} else {
		yaml.WriteString("                \"args\": [\n")
		for i, arg := range allArgs {
			yaml.WriteString("                  \"" + arg + "\"")
			if i < len(allArgs)-1 {
				yaml.WriteString(",")
			}
			yaml.WriteString("\n")
		}
		if hasGuardPolicies {
			yaml.WriteString("                ],\n")
		} else {
			yaml.WriteString("                ]\n")
		}
	}

	if hasGuardPolicies {
		renderGuardPoliciesJSON(yaml, guardPolicies, "                ")
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightDockerConfigJSON generates Playwright Docker/MCP mode configuration in JSON format.
// Per MCP Gateway Specification v1.0.0 section 3.2.1, stdio-based MCP servers MUST be containerized.
// Uses MCP Gateway spec format: container, entrypointArgs, mounts, and args fields.
func renderPlaywrightDockerConfigJSON(yaml *strings.Builder, playwrightConfig *PlaywrightToolConfig, isLast bool, includeCopilotFields bool, inlineArgs bool, guardPolicies map[string]any) {
	mcpPlaywrightLog.Printf("Rendering Playwright MCP config options: copilot_fields=%t, inline_args=%t", includeCopilotFields, inlineArgs)
	customArgs := getPlaywrightCustomArgs(playwrightConfig)

	// Extract all expressions from playwright arguments and replace them with env var references
	expressions := extractExpressionsFromPlaywrightArgs(customArgs)

	// Replace expressions in custom args
	if len(customArgs) > 0 {
		mcpPlaywrightLog.Printf("Applying %d custom Playwright args with %d extracted expressions", len(customArgs), len(expressions))
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
	// When guard policies follow, mounts is not the last field (add trailing comma)
	if len(guardPolicies) > 0 {
		yaml.WriteString("                \"mounts\": [\"/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs:rw\"],\n")
		renderGuardPoliciesJSON(yaml, guardPolicies, "                ")
	} else {
		yaml.WriteString("                \"mounts\": [\"/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs:rw\"]\n")
	}

	// Note: tools field is NOT included here - the converter script adds it back
	// for Copilot. This keeps the gateway config compatible with the schema.

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}
