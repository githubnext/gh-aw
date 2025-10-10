---
"gh-aw": patch
---

Fix: Correct MCP server command in .vscode/mcp.json

The `.vscode/mcp.json` configuration file used an incorrect command `["mcp", "serve"]` to invoke the MCP server. The correct command is `["mcp-server"]`. This fix ensures that AI agents and MCP clients can properly connect to the gh-aw MCP server using the VSCode configuration.

Additionally, added 4 new comprehensive tests in `pkg/cli/mcp_server_test.go` to improve MCP server test coverage, bringing the total to 7 tests.
