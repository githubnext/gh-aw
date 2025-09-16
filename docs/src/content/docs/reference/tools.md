---
title: Tools Configuration
description: Configure GitHub API tools, browser automation, and AI capabilities available to your agentic workflows, including GitHub tools, Playwright, and custom MCP servers.
sidebar:
  order: 5
---

This guide covers the available tools that can be configured in agentic workflows, including GitHub tools, Playwright browser automation, custom MCP servers, and Claude-specific tools.

> **📘 Looking for MCP servers?** See the complete [MCPs](../guides/mcps.md) for Model Context Protocol configuration, debugging, and examples.

## Overview

Tools are defined in the frontmatter to specify which GitHub API calls, browser automation, and AI capabilities are available to your workflow:

```yaml
tools:
  github:
    allowed: [create_issue, update_issue]
  playwright:
    allowed_domains: ["github.com", "*.example.com"]
  edit:
  bash: ["echo", "ls", "git status"]
```

All tools declared in included components are merged into the final workflow.

> [!TIP]
> You can inspect the tools available for an Agentic Workflow by running <br/>
> `gh aw mcp-inspect <workflow-file>`

## GitHub Tools (`github:`)

Configure which GitHub API operations are allowed for your workflow.

### Basic Configuration

```yaml
tools:
  github:
    # Uses default GitHub API access with workflow permissions
```

### Extended Configuration

```yaml
tools:
  github:
    allowed: [create_issue, update_issue, add_issue_comment]  # Optional: specific permissions
    docker_image_version: "latest"                          # Optional: MCP server version
```

### GitHub Tools Overview

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
    docker_image_version: "latest"                    # Optional: Playwright Docker image version
    allowed_domains: ["defaults", "github", "*.custom.com"]  # Domain access control
```

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

## Custom MCP Tools

Add custom Model Context Protocol servers for specialized integrations:

```yaml
tools:
  custom-api:
    mcp:
      command: "node"
      args: ["custom-mcp-server.js"]
      env:
        API_KEY: "${{ secrets.CUSTOM_API_KEY }}"
```

**Tool Execution:**
- Tools are configured as MCP servers that run alongside the AI engine
- Each tool provides specific capabilities (APIs, browser automation, etc.)
- Tools run in isolated environments with controlled access
- Domain restrictions apply to network-enabled tools like Playwright

## Neutral Tools (`edit:`, `web-fetch:`, `web-search:`, `bash:`)

Available when using `engine: claude` (it is the default engine). Configure Claude-specific capabilities and tools.

```yaml
tools:
  edit:        # File editing capabilities
  web-fetch:    # Web content fetching
  web-search:   # Web search capabilities
  bash: ["echo", "ls", "git status"]  # Allowed bash commands
```

### Bash Command Configuration

```yaml
tools:
  bash: ["echo", "ls", "git", "npm", "python"]
```

#### Bash Wildcards

```yaml
tools:
  bash: [":*"]  # Allow ALL bash commands - use with caution
```

**Wildcard Options:**
- **`:*`**: Allows **all bash commands** without restriction
- **`prefix:*`**: Allows **all commands starting with prefix**

**Security Note**: Using `:*` allows unrestricted bash access. Use only in trusted environments.

### Default Claude Tools

When using `engine: claude` with a `github` tool, these tools are automatically added:

- **`Task`**: Task management and workflow coordination
- **`Glob`**: File pattern matching and globbing operations  
- **`Grep`**: Text search and pattern matching within files
- **`LS`**: Directory listing and file system navigation
- **`Read`**: File reading operations
- **`NotebookRead`**: Jupyter notebook reading capabilities

No explicit declaration needed - automatically included with Claude + GitHub configuration.

### Complete Claude Example

```yaml
tools:
  github:
    allowed: [get_issue, add_issue_comment]
  edit:
  web-fetch:
  bash: ["echo", "ls", "git", "npm test"]
```


## Security Considerations

### Bash Command Restrictions
```yaml
tools:
  bash: ["echo", "ls", "git status"]        # ✅ Restricted set
  # bash: [":*"]                           # ⚠️  Unrestricted - use carefully
```

### Tool Permissions
```yaml
tools:
  github:
    allowed: [get_issue, add_issue_comment]     # ✅ Minimal required permissions
    # allowed: ["*"]                           # ⚠️  Broad access - review carefully
```

## Related Documentation

- [Frontmatter Options](../reference/frontmatter.md) - All frontmatter configuration options
- [Network Permissions](../reference/network.md) - Network access control for AI engines
- [MCPs](../guides/mcps.md) - Complete Model Context Protocol setup and usage
- [CLI Commands](../tools/cli.md) - CLI commands for workflow management
- [Workflow Structure](../reference/workflow-structure.md) - Directory layout and organization
- [Include Directives](../reference/include-directives.md) - Modularizing workflows with includes
