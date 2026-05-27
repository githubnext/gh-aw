// Package workflow provides built-in MCP server configuration rendering.
//
// # Built-in MCP Servers
//
// This file implements rendering functions for gh-aw's built-in MCP servers:
// safe-outputs, agentic-workflows, and their variations. These servers provide
// core functionality for AI agent workflows including controlled output storage,
// workflow execution, and memory management.
//
// Key responsibilities:
//   - Rendering safe-outputs MCP server configuration (HTTP transport)
//   - Rendering agentic-workflows MCP server configuration (stdio transport)
//   - Engine-specific format handling (JSON vs TOML)
//   - Managing HTTP server endpoints and authentication
//   - Configuring Docker containers for stdio servers
//   - Handling environment variable passthrough
//
// Built-in MCP servers:
//
// 1. Safe-outputs MCP server:
//   - Transport: HTTP (runs on host, accessed via HTTP)
//   - Port: 3001 (configurable via GH_AW_SAFE_OUTPUTS_PORT)
//   - Authentication: API key in Authorization header
//   - Purpose: Provides controlled storage for AI agent outputs
//   - Tools: add_issue_comment, create_issue, update_issue, upload_asset, etc.
//
// 2. Agentic-workflows MCP server:
//   - Transport: stdio (runs in Docker container)
//   - Container: Alpine Linux with gh-aw binary mounted (or localhost/gh-aw:dev in dev mode)
//   - Entrypoint: ${RUNNER_TEMP}/gh-aw/gh-aw mcp-server (release mode) or container default (dev mode)
//   - Network: Enabled via --network host for GitHub API access (api.github.com)
//   - Purpose: Enables workflow compilation, validation, and execution via gh aw CLI
//   - Tools: compile, validate, list, status, audit, logs, add, update, fix
//
// HTTP vs stdio transport:
// - HTTP: Server runs on host, accessible via HTTP URL with authentication
// - stdio: Server runs in Docker container, communicates via stdin/stdout
//
// Engine compatibility:
// The renderer supports multiple output formats:
//   - JSON (Copilot, Claude, Custom): JSON-like MCP configuration
//   - TOML (Codex): TOML-like MCP configuration
//
// Copilot-specific features:
// When IncludeCopilotFields is true, the renderer adds:
//   - "type" field: Specifies transport type (http or stdio)
//   - Backslash-escaped variables: \${VAR} for MCP passthrough
//
// Safe-outputs configuration:
// Safe-outputs runs as an HTTP server and requires:
//   - Port and API key from step outputs
//   - Config files: config.json, tools.json, validation.json
//   - Environment variables for feature configuration
//
// The HTTP URL uses either:
//   - host.docker.internal: When agent runs in firewall container
//   - localhost: When agent firewall is disabled (sandbox.agent.disabled)
//
// Agentic-workflows configuration:
// Agentic-workflows runs in a stdio container and requires:
//   - Mounted gh-aw binary from ${RUNNER_TEMP}/gh-aw (release mode) or baked into image (dev mode)
//   - Mounted gh CLI binary for GitHub API access (release mode) or baked into image (dev mode)
//   - Mounted workspace for workflow files
//   - Mounted temp directory for logs
//   - GITHUB_TOKEN for GitHub API access
//   - Network access enabled via --network host for api.github.com
//
// Related files:
//   - mcp_renderer.go: Main renderer that calls these functions
//   - mcp_setup_generator.go: Generates setup steps for these servers
//   - safe_outputs.go: Safe-outputs configuration and validation
//   - mcp_scripts.go: MCP Scripts configuration (similar pattern)
//
// Example safe-outputs config:
//
//	{
//	  "safe_outputs": {
//	    "type": "http",
//	    "url": "http://host.docker.internal:$GH_AW_SAFE_OUTPUTS_PORT",
//	    "headers": {
//	      "Authorization": "$GH_AW_SAFE_OUTPUTS_API_KEY"
//	    }
//	  }
//	}
//
// Example agentic-workflows config:
//
//	{
//	  "agenticworkflows": {
//	    "type": "stdio",
//	    "container": "alpine:3.20",
//	    "entrypoint": "${RUNNER_TEMP}/gh-aw/gh-aw",
//	    "entrypointArgs": ["mcp-server"],
//	    "mounts": ["${RUNNER_TEMP}/gh-aw:${RUNNER_TEMP}/gh-aw:ro", ...],
//	    "env": {
//	      "GITHUB_TOKEN": "$GITHUB_TOKEN"
//	    }
//	  }
//	}
package workflow
