package main

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/cli"
	"github.com/githubnext/gh-aw/pkg/console"
)

// Build-time variables
var (
	version = "dev"
)

func main() {
	// Set version info
	cli.SetVersionInfo(version)

	// Create the mcp-gateway command
	cmd := cli.NewMCPGatewayCommand()

	// Update command usage to reflect standalone binary
	cmd.Use = "awmg"
	cmd.Short = "MCP Gateway - Aggregate multiple MCP servers into a single HTTP gateway"
	cmd.Long = `awmg (Agentic Workflows MCP Gateway) - Aggregate multiple MCP servers into a single HTTP gateway.

The gateway:
- Integrates by default with the sandbox.mcp extension point
- Imports Claude/Copilot/Codex MCP server JSON configuration
- Starts each MCP server and mounts an MCP client on each
- Mounts an HTTP MCP server that acts as a gateway to the MCP clients
- Supports most MCP gestures through the go-MCP SDK
- Provides extensive logging to file in the MCP log folder

Configuration can be provided via:
1. --config flag(s) pointing to JSON config file(s) (can be specified multiple times)
2. stdin (reads JSON configuration from standard input)

Multiple config files are merged in order, with later files overriding earlier ones.

Configuration format:
{
  "mcpServers": {
    "server-name": {
      "command": "command",
      "args": ["arg1", "arg2"],
      "env": {"KEY": "value"}
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "optional-key"
  }
}

Examples:
  awmg --config config.json                    # From single file
  awmg --config base.json --config override.json # From multiple files (merged)
  awmg --port 8080                             # From stdin
  echo '{"mcpServers":{...}}' | awmg           # Pipe config
  awmg --config config.json --log-dir /tmp/logs # Custom log dir`

	// Add version flag
	cmd.Version = version
	cmd.SetVersionTemplate("awmg version {{.Version}}\n")

	// Execute command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatErrorMessage(err.Error()))
		os.Exit(1)
	}
}
