---
title: Tools Configuration
description: Configure GitHub API tools and AI capabilities available to your agentic workflows, including GitHub tools and Claude-specific integrations.
---

This guide covers the available tools that can be configured in agentic workflows, including GitHub tools and Claude-specific tools.

> **📘 Looking for MCP servers?** See the complete [MCPs](mcps.md) for Model Context Protocol configuration, debugging, and examples.

## Overview

Tools are defined in the frontmatter to specify which GitHub API calls and AI capabilities are available to your workflow:

```yaml
tools:
  github:
    allowed: [create_issue, update_issue]
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

### Default Tools Security Policy

The system automatically includes read-only GitHub tools by default, following the **principle of least privilege**:

```yaml
# Default tools are automatically included (read-only operations only)
# - get_issue, get_pull_request, list_issues, search_repositories
# - get_workflow_run, list_workflow_jobs, download_workflow_run_artifact
# - get_code_scanning_alert, list_secret_scanning_alerts, etc.

tools:
  github:
    # Write operations must be explicitly configured by users
    allowed: [create_issue, add_issue_comment, update_issue]  # ✅ Explicit permissions
```

**Security Policy for Default Tools:**
- ✅ **Included by default**: Read-only operations (`get_*`, `list_*`, `search_*`, `download_*`)
- ❌ **Never included by default**: Write operations (`create_*`, `update_*`, `delete_*`, `add_*`, `remove_*`)
- 🔒 **User control**: Write operations require explicit configuration in workflow's `allowed` list

This ensures workflows have minimal permissions by default while allowing users to explicitly grant additional permissions as needed.

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

- [Commands](commands.md) - CLI commands for workflow management
- [MCPs](mcps.md) - Complete Model Context Protocol setup and usage
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Frontmatter Options](frontmatter.md) - All configuration options
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
