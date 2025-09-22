# MCP Configuration for VS Code

This repository includes an `mcps.json` file that configures the GitHub Agentic Workflows CLI as an MCP server for use with VS Code and other MCP clients.

## Configuration File

The `mcps.json` file contains several pre-configured MCP server instances:

### Available Configurations

1. **`gh-aw`** - Full server with all tools enabled
   - Exposes all 8 CLI tools: compile, logs, mcp_inspect, mcp_list, mcp_add, run, enable, disable

2. **`gh-aw-compile-only`** - Development-focused server
   - Only exposes: compile, logs
   - Ideal for development workflows

3. **`gh-aw-workflow-mgmt`** - Workflow management server  
   - Only exposes: run, enable, disable
   - Focused on workflow execution and management

4. **`gh-aw-mcp-tools`** - MCP management server
   - Only exposes: mcp_inspect, mcp_list, mcp_add
   - For managing and inspecting MCP configurations

## Usage with VS Code

1. Ensure the `gh-aw` binary is built:
   ```bash
   make build
   # or
   go build -o gh-aw ./cmd/gh-aw
   ```

2. Copy or symlink the `mcps.json` file to your VS Code MCP configuration directory, or reference it in your VS Code settings.

3. Configure VS Code to use the MCP servers from this file.

4. The servers will be available as MCP tools in compatible VS Code extensions.

## Custom Configuration

You can modify the `mcps.json` file to create custom tool combinations:

```json
{
  "mcpServers": {
    "my-custom-gh-aw": {
      "command": "./gh-aw",
      "args": ["mcp", "serve", "--allowed-tools", "compile,run,enable"],
      "cwd": ".",
      "env": {
        "PATH": "${PATH}"
      }
    }
  }
}
```

## Available Tools

- **compile** - Compile markdown workflow files to YAML
- **logs** - Download and analyze agentic workflow logs
- **mcp_inspect** - Inspect MCP servers and list available tools  
- **mcp_list** - List MCP servers defined in agentic workflows
- **mcp_add** - Add MCP tools to agentic workflows
- **run** - Run agentic workflows on GitHub Actions
- **enable** - Enable workflows
- **disable** - Disable workflows

## Testing the Configuration

You can test an MCP server configuration manually:

```bash
# Test the basic configuration
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"clientInfo": {"name": "test", "version": "1.0.0"}}}' | ./gh-aw mcp serve

# Test with filtered tools
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"clientInfo": {"name": "test", "version": "1.0.0"}}}' | ./gh-aw mcp serve --allowed-tools compile,logs
```