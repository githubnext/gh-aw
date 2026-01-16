# MCP Server Usage Patterns

This guide provides best practices for configuring and using Model Context Protocol (MCP) servers in agentic workflows.

## When to Use MCP Servers

Consider using MCP servers when a task benefits from:

- **Reusable capabilities**: External functionality that could be shared across workflows
- **Complex integrations**: APIs or services requiring specialized logic
- **Custom tools**: Domain-specific tools not available in built-in toolsets
- **Stateful operations**: Tools that maintain state or sessions

## MCP Server Configuration

### Basic Configuration

MCP servers are configured in the top-level `mcp-servers:` block:

```yaml
mcp-servers:
  my-custom-server:
    command: "node"
    args: ["path/to/mcp-server.js"]
    allowed:
      - custom_function_1
      - custom_function_2
```

### Configuration Fields

- **`command`**: The executable to run (e.g., `node`, `python`, `npx`)
- **`args`**: Array of command-line arguments
- **`allowed`**: List of tool names from the MCP server that should be available to the agent
- **`env`** (optional): Environment variables for the MCP server

### Example Configurations

**Node.js MCP Server**:
```yaml
mcp-servers:
  notion:
    command: "node"
    args: ["./mcp-servers/notion-server.js"]
    allowed:
      - get_page
      - create_page
      - search_database
```

**Python MCP Server**:
```yaml
mcp-servers:
  data-analysis:
    command: "python"
    args: ["-m", "mcp_servers.data_analysis"]
    env:
      API_KEY: "${{ secrets.DATA_API_KEY }}"
    allowed:
      - query_data
      - generate_chart
```

**NPX Package**:
```yaml
mcp-servers:
  filesystem:
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
    allowed:
      - read_file
      - write_file
      - list_directory
```

## Inspecting MCP Servers

Use `gh aw mcp` commands to analyze configured MCP servers:

```bash
# List all workflows with MCP servers
gh aw mcp list

# Inspect MCP servers in a specific workflow
gh aw mcp inspect workflow-name

# View specific server details
gh aw mcp inspect workflow-name --server my-custom-server

# View specific tool details
gh aw mcp inspect workflow-name --tool custom_function_1

# Launch web-based inspector
gh aw mcp inspect --inspector
```

## Best Practices

### 1. Use Allowed Lists

Always specify an `allowed:` list to restrict which tools the agent can access:

```yaml
mcp-servers:
  github-advanced:
    command: "npx"
    args: ["-y", "@github/mcp-server"]
    allowed:
      - search_code
      - get_repository
      # Only allow read operations, not mutations
```

### 2. Prefer Safe Outputs for Write Operations

When an MCP server provides write operations to GitHub or external services, consider using `safe-outputs` instead to ensure proper validation:

```yaml
# Instead of allowing MCP server mutations directly
safe-outputs:
  add-comment:
    max: 1
  create-issue:
    enabled: true
```

### 3. Environment Variables for Secrets

Use GitHub secrets for sensitive configuration:

```yaml
mcp-servers:
  slack:
    command: "node"
    args: ["./mcp-servers/slack.js"]
    env:
      SLACK_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
    allowed:
      - post_message
      - list_channels
```

### 4. Install Dependencies in Steps

If your MCP server requires installation or setup, add steps before the agent job:

```yaml
steps:
  - name: Install MCP server dependencies
    run: |
      npm install -g mcp-server-package
      # or: pip install mcp-server-package
```

### 5. Document Tool Purpose

Add comments explaining what each MCP server provides:

```yaml
mcp-servers:
  # Provides access to internal wiki and documentation
  confluence:
    command: "node"
    args: ["./mcp-servers/confluence.js"]
    allowed:
      - search_pages
      - get_page_content
```

## Common MCP Server Patterns

### API Integration Pattern

For integrating with external APIs:

```yaml
mcp-servers:
  api-client:
    command: "node"
    args: ["./mcp-servers/api-client.js"]
    env:
      API_URL: "https://api.example.com"
      API_TOKEN: "${{ secrets.API_TOKEN }}"
    allowed:
      - fetch_data
      - validate_response
```

### Database Query Pattern

For database access:

```yaml
mcp-servers:
  database:
    command: "python"
    args: ["-m", "mcp_servers.database"]
    env:
      DATABASE_URL: "${{ secrets.DATABASE_URL }}"
    allowed:
      - query
      - get_schema
```

### File System Pattern

For file system operations:

```yaml
mcp-servers:
  filesystem:
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "./workspace"]
    allowed:
      - read_file
      - list_directory
      # Exclude write operations for safety
```

## Shared MCP Components

For reusable MCP servers across multiple workflows, consider creating shared workflow components. See `.github/aw/create-shared-agentic-workflow.md` for guidance on wrapping MCP servers as shared components.

## Troubleshooting

### Common Issues

1. **Tool not available**: Verify the tool name in `allowed:` matches the tool name exposed by the MCP server
2. **Server fails to start**: Check `command` and `args` are correct, and dependencies are installed
3. **Authentication errors**: Verify environment variables and secrets are properly configured
4. **Permission denied**: Ensure the MCP server executable has proper permissions

### Debugging Tips

- Use `gh aw mcp inspect` to verify server configuration
- Check workflow logs for MCP server startup errors
- Test MCP servers locally before using in workflows
- Use the web inspector (`gh aw mcp inspect --inspector`) for interactive debugging

## Summary

- Use MCP servers for reusable, complex, or custom capabilities
- Always specify `allowed:` lists to restrict tool access
- Store sensitive data in GitHub secrets
- Install dependencies in workflow steps
- Use safe outputs for write operations
- Inspect and debug with `gh aw mcp` commands
