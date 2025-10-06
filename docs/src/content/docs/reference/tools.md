---
title: Tools Configuration
description: Configure GitHub API tools, browser automation, and AI capabilities available to your agentic workflows, including GitHub tools, Playwright, and custom MCP servers.
sidebar:
  order: 5
---

This guide covers the available tools that can be configured in agentic workflows, including GitHub tools, Playwright browser automation, custom MCP servers, and engine-specific tools.

## Overview

Tools are defined in the frontmatter to specify which GitHub API calls, browser automation, and AI capabilities are available to your workflow:

```yaml
tools:
  github:
    allowed: [create_issue, update_issue]
  playwright:
    allowed_domains: ["github.com", "*.example.com"]
  edit:
  bash: true  # Uses default safe commands
```

All tools declared in included components are merged into the final workflow.

> [!TIP]
> You can discover and inspect the tools available for Agentic Workflows by running: <br/>
> `gh aw mcp list <workflow-file>` - Quick overview of MCP servers <br/>
> `gh aw mcp inspect <workflow-file>` - Detailed inspection with connection testing

## GitHub Tools (`github:`)

Configure which GitHub API operations are allowed for your workflow.

```yaml
tools:
  github:
    # Uses default GitHub API access with workflow permissions
```

or the extended form:

```yaml
tools:
  github:
    allowed: [create_issue, update_issue, add_issue_comment]  # Optional: specific permissions
    version: "latest"                          # Optional: MCP server version
    args: ["--verbose", "--debug"]            # Optional: additional command-line arguments
    read-only: true                           # Optional: restrict to read-only operations
```

### GitHub Read-Only Mode

The `read-only` flag restricts the GitHub MCP server to read-only operations, preventing any modifications to repositories, issues, pull requests, etc.

```yaml
tools:
  github:
    read-only: true
```

When `read-only: true` is specified, the GitHub MCP server runs with the `GITHUB_READ_ONLY` environment variable set, which enables read-only mode at the server level.

**Default behavior**: When the GitHub tool is specified without any configuration (just `github:` with no properties), the default behavior provides read-only access with all read-only tools available.

### GitHub Args Configuration

The `args` field allows you to pass additional command-line arguments to the GitHub MCP server:

```yaml
tools:
  github:
    args: ["--custom-flag", "--verbose"]
```

Arguments are appended to the generated MCP server command and properly escaped for special characters including spaces, quotes, and backslashes.

The system automatically includes comprehensive default read-only GitHub tools. These defaults are merged with your custom `allowed` tools, providing comprehensive repository access.

**Default Read-Only Tools**:

**Actions**: `download_workflow_run_artifact`, `get_job_logs`, `get_workflow_run`, `list_workflows`

**Issues & PRs**: `get_issue`, `get_pull_request`, `list_issues`, `list_pull_requests`, `search_issues`

**Repository**: `get_commit`, `get_file_contents`, `list_branches`, `list_commits`, `search_code`

**Security**: `get_code_scanning_alert`, `list_secret_scanning_alerts`, `get_dependabot_alert`

**Users & Organizations**: `search_users`, `search_orgs`, `get_me`

## Playwright Tool (`playwright:`)

Enable browser automation and web testing capabilities using containerized Playwright:

```yaml
tools:
  playwright:
    allowed_domains: ["github.com", "*.example.com"]
```

### Playwright Configuration Options

```yaml
tools:
  playwright:
    version: "latest"                    # Optional: Playwright version
    allowed_domains: ["defaults", "github", "*.custom.com"]  # Domain access control
```

### Playwright Args Configuration

The `args` field allows you to pass additional command-line arguments to the Playwright MCP server:

```yaml
tools:
  playwright:
    args: ["--browser", "chromium"]
```

Common use cases include custom flags for debugging or testing scenarios.

**Note**: Only Chromium browser is supported.

Arguments are appended to the generated MCP server command and properly escaped for special characters.

### Domain Configuration

The `allowed_domains` field supports the same ecosystem bundle resolution as the top-level `network:` configuration, with **localhost-only** as the default for enhanced security:

**Ecosystem Bundle Examples:**
```yaml
tools:
  playwright:
    allowed_domains: 
      - "defaults"              # Basic infrastructure domains
      - "github"               # GitHub domains (github.com, api.github.com, etc.)
      - "node"                 # Node.js ecosystem
      - "python"               # Python ecosystem
      - "*.example.com"        # Custom domain with wildcard
```

**Security Model:**
- **Default**: `["localhost", "127.0.0.1"]` - localhost access only
- **Ecosystem bundles**: Use same identifiers as `network:` configuration
- **Custom domains**: Support exact matches and wildcard patterns
- **Containerized execution**: Isolated Docker environment for security

**Available Ecosystem Identifiers:**
Same as `network:` configuration: `defaults`, `github`, `node`, `python`, `containers`, `java`, `rust`, `playwright`, etc.

## MCP Server Integration

Use the dedicated `mcp-servers:` section for MCP server configuration:

```yaml
# In your workflow frontmatter
mcp-servers:
  custom-api:
    command: "node"
    args: ["custom-mcp-server.js"]
    env:
      API_KEY: "${{ secrets.CUSTOM_API_KEY }}"

# Separate tools section for built-in capabilities
tools:
  github:
    allowed: [create_issue, update_issue]
  playwright:
    allowed_domains: ["github.com"]
```

### Adding MCP Servers from Registry

The easiest way to add MCP servers is from the GitHub MCP registry:

```bash
# List available MCP servers
gh aw mcp add

# Add a specific server to your workflow  
gh aw mcp add my-workflow makenotion/notion-mcp-server
```

See the [MCP Integration Guide](/gh-aw/guides/mcps/) for comprehensive MCP server setup and configuration.

**MCP Server Execution:**
- MCP servers run alongside the AI engine to provide specialized capabilities
- Each server provides specific tools (APIs, database access, etc.)
- Servers run in isolated environments with controlled access
- Domain restrictions apply to network-enabled servers

## Neutral Tools (`edit:`, `web-fetch:`, `web-search:`, `bash:`)

```yaml
tools:
  edit:        # File editing capabilities
  web-fetch:    # Web content fetching
  web-search:   # Web search capabilities
  bash: true   # Default safe commands (echo, ls, pwd, cat, head, tail, grep, wc, sort, uniq, date)
  # bash: ["echo", "ls", "git status"]  # Or specify custom commands
```

**Note:** Some engines (like Copilot) don't have built-in `web-search` support. You can add web search using third-party MCP servers instead. See the [Web Search with MCP guide](/gh-aw/guides/web-search/) for options.

### Bash Command Configuration

The bash tool provides access to shell commands with different levels of control and security.

#### Basic Configuration Options

```yaml
tools:
  # Default commands: Provides common safe commands (echo, ls, pwd, cat, etc.)
  bash: true

  # Specific commands: Only allow specified commands
  bash: ["echo", "ls", "git status", "npm test"]

  # No commands: Bash tool enabled but no commands allowed
  bash: []

  # All commands: Unrestricted bash access (use with caution)
  bash: [":*"]
```

#### Default Bash Commands

When `bash: true` or `bash: null` is specified, the system automatically provides these safe, read-only commands:

**File Operations**: `ls`, `pwd`, `cat`, `head`, `tail`  
**Text Processing**: `grep`, `sort`, `uniq`, `wc`  
**Basic Utilities**: `echo`, `date`

These defaults ensure consistent behavior across Claude and Copilot engines while maintaining security.

#### Configuration Behavior

- **`bash: true`** → Adds default safe commands (`echo`, `ls`, `pwd`, `cat`, `head`, `tail`, `grep`, `wc`, `sort`, `uniq`, `date`)
- **`bash: null`** → Adds default commands (only if no git commands needed for safe outputs)  
- **`bash: []`** → No bash commands allowed (empty array preserved)
- **`bash: ["cmd1", "cmd2"]`** → Only specified commands allowed
- **`bash: [":*"]`** → All bash commands allowed (unrestricted access)

#### Wildcards and Advanced Patterns

```yaml
tools:
  bash: [":*"]                    # Allow ALL bash commands - use with caution
  bash: ["git:*"]                 # Allow all git commands with any arguments
  bash: ["npm:*", "echo", "ls"]   # Mix of wildcards and specific commands
```

**Wildcard Options:**
- **`:*`**: Allows **all bash commands** without restriction
- **`command:*`**: Allows **all invocations of a specific command** with any arguments

#### Security Considerations

**Default Commands**: The default command set is designed to be safe and read-only, suitable for most workflows without security concerns.

**Custom Commands**: When specifying custom commands, ensure they don't provide unintended access to sensitive operations.

**Wildcard Access**: Using `:*` allows unrestricted bash access. Use only in trusted environments or when comprehensive shell access is required.

**Git Integration**: When safe outputs require git commands (e.g., `create-pull-request`), the system automatically adds necessary git commands like `git add`, `git commit`, etc.

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
- [Network Permissions](/gh-aw/reference/network/) - Network access control for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Complete Model Context Protocol setup and usage
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Include Directives](/gh-aw/reference/include-directives/) - Modularizing workflows with includes
