# 🔧 Tools Configuration

This guide covers the available tools that can be configured in agentic workflows, including GitHub tools and Claude-specific tools.

> **📘 Looking for MCP servers?** See the complete [MCPs](mcps.md) for Model Context Protocol configuration, debugging, and examples.

## Overview

Tools are defined in the frontmatter to specify which GitHub API calls and AI capabilities are available to your workflow:

```yaml
tools:
  github:
    allowed: [create_issue, update_issue]
  claude:
    allowed:
      Edit:
      Bash: ["echo", "ls", "git status"]
```

All tools declared in included components are merged into the final workflow.

## GitHub Tools (`github:`)

Configure which GitHub API operations are allowed for your workflow.

### Basic Configuration

```yaml
tools:
  github:
    allowed: [create_issue, update_issue, add_issue_comment]
```

### GitHub Tools Overview

The system automatically includes comprehensive default read-only GitHub tools. These defaults are merged with your custom `allowed` tools, providing comprehensive repository access.

**Default Read-Only Tools**:

**Actions**: `download_workflow_run_artifact`, `get_job_logs`, `get_workflow_run`, `list_workflows`

**Issues & PRs**: `get_issue`, `get_pull_request`, `list_issues`, `list_pull_requests`, `search_issues`

**Repository**: `get_commit`, `get_file_contents`, `list_branches`, `list_commits`, `search_code`

**Security**: `get_code_scanning_alert`, `list_secret_scanning_alerts`, `get_dependabot_alert`

**Users & Organizations**: `search_users`, `search_orgs`, `get_me`

## Claude Tools (`claude:`)

Available when using `engine: claude`. Configure Claude-specific capabilities and tools.

### Basic Claude Tools

```yaml
tools:
  claude:
    allowed:
      Edit:        # File editing capabilities
      MultiEdit:   # Multi-file editing
      Write:       # File writing
      NotebookEdit: # Jupyter notebook editing
      WebFetch:    # Web content fetching
      WebSearch:   # Web search capabilities
      Bash: ["echo", "ls", "git status"]  # Allowed bash commands
```

### Bash Command Configuration

```yaml
tools:
  claude:
    allowed:
      Bash: ["echo", "ls", "git", "npm", "python"]
```

#### Bash Wildcards

```yaml
tools:
  claude:
    allowed:
      Bash:
        allowed: [":*"]  # Allow ALL bash commands - use with caution
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
  claude:
    allowed:
      Edit:
      Write:
      WebFetch:
      Bash: ["echo", "ls", "git", "npm test"]
```

## Engine Compatibility

### Claude Engine
- ✅ GitHub tools
- ✅ Claude-specific tools
- ✅ Custom MCP tools (see [MCP Guide](mcps.md))

### Codex Engine
- ✅ GitHub tools
- ❌ Claude-specific tools (ignored)
- ✅ Custom MCP tools (stdio only, see [MCP Guide](mcps.md))


## Security Considerations

### Bash Command Restrictions
```yaml
tools:
  claude:
    allowed:
      Bash: ["echo", "ls", "git status"]        # ✅ Restricted set
      # Bash: [":*"]                           # ⚠️  Unrestricted - use carefully
```

### Tool Permissions
```yaml
tools:
  github:
    allowed: [get_issue, add_issue_comment]     # ✅ Minimal required permissions
    # allowed: ["*"]                           # ⚠️  Broad access - review carefully
```

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [MCPs](mcps.md) - Complete Model Context Protocol setup and usage
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Frontmatter Options](frontmatter.md) - All configuration options
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
