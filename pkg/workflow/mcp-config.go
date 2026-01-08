package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

// mcp-config.go provides the shared logger for MCP configuration modules.
// The actual MCP configuration functions have been split into focused modules:
// - mcp_config_types.go: Type definitions and interfaces
// - mcp_config_utils.go: Shared utility functions
// - mcp_config_renderers.go: Core rendering logic
// - mcp_config_builtin_servers.go: Built-in server configurations
// - mcp_config_custom.go: Custom server configuration

var mcpLog = logger.New("workflow:mcp-config")
