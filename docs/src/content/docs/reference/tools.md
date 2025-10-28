---
title: Tools
description: Configure GitHub API tools, browser automation, and AI capabilities available to your agentic workflows, including GitHub tools, Playwright, and custom MCP servers.
sidebar:
  order: 700
---

This guide covers the available tools that can be configured in agentic workflows, including GitHub tools, Playwright browser automation, custom MCP servers, and neutral tools.

## Overview

Tools are defined in the frontmatter to specify which GitHub API calls, browser automation, and AI capabilities are available to your workflow:

```yaml
tools:
  edit:
  bash: true
```

Some tools are available by default. All tools declared in imported components are merged into the final workflow.

## Edit Tool (`edit:`)

Allows file editing in the GitHub Actions workspace.

```yaml
tools:
  edit:
```

## Bash Tool (`bash:`)

Allows shell command execution in the GitHub Actions workspace.

```yaml
tools:
  bash:                              # Default safe commands
  bash: []                           # No commands allowed
  bash: ["echo", "ls", "git status"] # Specific commands only
  bash: [":*"]                       # All commands (use with caution)
```

**Configuration:**

- `bash:` or `bash: null` → Default safe commands: `echo`, `ls`, `pwd`, `cat`, `head`, `tail`, `grep`, `wc`, `sort`, `uniq`, `date`
- `bash: []` → No bash access
- `bash: ["cmd1", "cmd2"]` → Only specified commands
- `bash: [":*"]` or `bash: ["*"]` → All commands (unrestricted)

**Wildcards:**

```yaml
bash: [":*"]                      # All bash commands
bash: ["git:*"]                   # All git commands
bash: ["npm:*", "echo", "ls"]     # Mix of wildcards and specific commands
```

- `:*` or `*`: All commands (Copilot uses `--allow-all-tools`; refused in strict mode)
- `command:*`: All invocations of a specific command (e.g., `git:*` allows `git add`, `git commit`, etc.)

## Web Fetch Tool (`web-fetch:`)

Enables web content fetching.

```yaml
tools:
  web-fetch:
```

## Web Search Tool (`web-search:`)

Enables web search (if supported by the AI engine).

```yaml
tools:
  web-search:
```

:::note
Some engines (like Copilot) require third-party MCP servers for web search. See [Using Web Search](/gh-aw/guides/web-search/).
:::

## GitHub Tools (`github:`)

Configure GitHub API operations.

```yaml
tools:
  github:                                      # Default read-only access

  # OR with specific configuration:
  github:
    allowed: [create_issue, update_issue]      # Specific permissions
    mode: remote                               # "local" (Docker) or "remote" (hosted)
    version: "latest"                          # MCP server version (local only)
    args: ["--verbose", "--debug"]             # Additional arguments (local only)
    read-only: true                            # Read-only operations
    github-token: "${{ secrets.CUSTOM_PAT }}"  # Custom token
    toolset: [repos, issues, pull_requests]    # Toolset groups
```

### GitHub Toolsets

:::tip[Prefer Toolsets Over Individual Tools]
Use `toolset:` to enable groups of related tools instead of listing individual tools with `allowed:`. Toolsets provide better organization, reduce configuration verbosity, and ensure you get all related functionality.
:::

Enables or disables specific GitHub API groups to improve tool selection and reduce context size.

```yaml
tools:
  github:
    toolset: [repos, issues, pull_requests, actions]
```

**Available Toolsets**: `context` (recommended), `actions`, `code_security`, `dependabot`, `discussions`, `experiments`, `gists`, `issues`, `labels`, `notifications`, `orgs`, `projects`, `pull_requests`, `repos`, `secret_protection`, `security_advisories`, `stargazers`, `users`, `search`

**Default Toolsets** (when `toolset:` is not specified): `context`, `repos`, `issues`, `pull_requests`, `users`

**Recommended Combinations**:
- **Read-only workflows**: `toolset: [default]` or `toolset: [context, repos]`
- **Issue/PR/Discussion management**: `toolset: [default, discussions]` 
- **CI/CD workflows**: `toolset: [default, actions]`
- **Security workflows**: `toolset: [default, code_security, secret_protection]`
- **Full access**: `toolset: [all]`

#### Tool-to-Toolset Mapping

When migrating from `allowed:` to `toolset:`, use this mapping to identify the correct toolset for your tools:

| Tool Name | Toolset | Description |
|-----------|---------|-------------|
| `get_me`, `get_teams`, `get_team_members` | `context` | User and environment context |
| `get_repository`, `get_file_contents`, `search_code`, `list_commits`, `get_commit` | `repos` | Repository operations |
| `issue_read`, `list_issues`, `create_issue`, `update_issue`, `search_issues`, `add_reaction` | `issues` | Issue management |
| `pull_request_read`, `list_pull_requests`, `get_pull_request`, `create_pull_request`, `search_pull_requests` | `pull_requests` | Pull request operations |
| `list_workflows`, `list_workflow_runs`, `get_workflow_run`, `download_workflow_run_artifact` | `actions` | GitHub Actions/CI/CD |
| `list_code_scanning_alerts`, `get_code_scanning_alert`, `create_code_scanning_alert` | `code_security` | Code security scanning |
| `list_discussions`, `create_discussion` | `discussions` | GitHub Discussions |
| `get_label`, `list_labels`, `create_label` | `labels` | Label management |
| `list_notifications`, `mark_notifications_read` | `notifications` | Notifications |
| `get_user`, `list_users` | `users` | User profiles |
| `get_organization`, `list_organizations` | `orgs` | Organization management |
| `create_gist`, `list_gists` | `gists` | Gist operations |
| `get_latest_release`, `list_releases` | `repos` | Release management (part of repos) |
| `create_issue_comment` | `issues` | Issue comments (part of issues) |

**Example Migration**:

Before (using `allowed:`):
```yaml
tools:
  github:
    allowed:
      - list_pull_requests
      - get_pull_request
      - pull_request_read
      - list_commits
      - get_commit
      - get_file_contents
```

After (using `toolset:`):
```yaml
tools:
  github:
    toolset: [default]  # Includes repos and pull_requests
```

Or for more specific control:
```yaml
tools:
  github:
    toolset: [repos, pull_requests]
```

:::note
Both `toolset:` and `allowed:` can be used together. When specified, `allowed:` further restricts which tools are available within the enabled toolsets.
:::

**Supported Modes**: Toolsets are supported in both local (Docker) and remote (hosted) modes.

### GitHub Remote Mode

Uses the hosted GitHub MCP server at `https://api.githubcopilot.com/mcp/` for faster startup without Docker.

```yaml
tools:
  github:
    mode: remote
    allowed: [list_issues, create_issue]
```

**Setup**: Create a Personal Access Token and set the `GH_AW_GITHUB_TOKEN` secret:

```bash
gh secret set GH_AW_GITHUB_TOKEN -a actions --body "<your-github-pat>"
```

**Note**: Remote mode requires `GH_AW_GITHUB_TOKEN` (standard `GITHUB_TOKEN` is not supported).

### GitHub Read-Only Mode

Restricts GitHub API to read-only operations.

```yaml
tools:
  github:
    read-only: true
```

Default: `github:` provides read-only access.

## Playwright Tool (`playwright:`)

Enables browser automation using containerized Playwright.

```yaml
tools:
  playwright:
    version: "latest"                      # Playwright version
    args: ["--browser", "chromium"]        # Additional arguments
    allowed_domains: ["defaults", "github", "*.custom.com"]  # Domain access
```

**Domain Access**: Uses same ecosystem bundles as `network:` configuration (`defaults`, `github`, `node`, `python`, etc.). Default: `["localhost", "127.0.0.1"]` for security.

```yaml
playwright:
  allowed_domains:
    - "defaults"         # Basic infrastructure
    - "github"          # GitHub domains
    - "*.example.com"   # Custom wildcard
```

## Custom MCP Servers (`mcp-servers:`)

Use `mcp-servers:` to integrate custom Model Context Protocol servers for third-party services, APIs, or specialized tools.

### Basic Configuration

**npx-based MCP server:**
```yaml
mcp-servers:
  custom-api:
    command: "npx"
    args: ["-y", "@company/custom-mcp-server"]
    env:
      API_KEY: "${{ secrets.CUSTOM_API_KEY }}"
```

**Node.js script:**
```yaml
mcp-servers:
  analytics:
    command: "node"
    args: ["scripts/analytics-mcp-server.js"]
    env:
      ANALYTICS_TOKEN: "${{ secrets.ANALYTICS_TOKEN }}"
```

**Python MCP server:**
```yaml
mcp-servers:
  data-processor:
    command: "python"
    args: ["-m", "data_processor.mcp_server"]
    env:
      DATABASE_URL: "${{ secrets.DATABASE_URL }}"
      API_TIMEOUT: "30"
```

**Docker container:**
```yaml
mcp-servers:
  notion:
    container: "mcp/notion"
    env:
      NOTION_API_TOKEN: "${{ secrets.NOTION_API_TOKEN }}"
    allowed:
      - "search_pages"
      - "get_page"
      - "query_database"
```

**HTTP MCP server:**
```yaml
mcp-servers:
  datadog:
    url: "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"
    headers:
      DD_API_KEY: "${{ secrets.DD_API_KEY }}"
      DD_APPLICATION_KEY: "${{ secrets.DD_APPLICATION_KEY }}"
    allowed:
      - "search_datadog_dashboards"
      - "get_datadog_metric"
```

### Configuration Fields

- **`command:`** - Executable command (e.g., `"node"`, `"python"`, `"npx"`)
- **`args:`** - Command arguments as array of strings
- **`env:`** - Environment variables for the MCP server process
- **`container:`** - Docker container image (alternative to `command`)
- **`url:`** - HTTP endpoint for remote MCP servers (alternative to `command`)
- **`headers:`** - HTTP headers for authentication (used with `url`)
- **`allowed:`** - List of allowed tool names from the MCP server

### Complete Example

Combining GitHub tools with custom MCP servers:

```yaml
tools:
  github:
    allowed: [create_issue, update_issue]

mcp-servers:
  slack:
    command: "npx"
    args: ["-y", "@slack/mcp-server"]
    env:
      SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
      SLACK_TEAM_ID: "${{ secrets.SLACK_TEAM_ID }}"
    allowed:
      - "send_message"
      - "get_channel_history"
  
  notion:
    container: "mcp/notion"
    env:
      NOTION_API_TOKEN: "${{ secrets.NOTION_API_TOKEN }}"
    allowed:
      - "search_pages"
      - "get_page"
```

MCP servers run alongside the AI engine in isolated environments with controlled network access. See [MCPs Guide](/gh-aw/guides/mcps/) for detailed setup instructions and examples.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
- [Network Permissions](/gh-aw/reference/network/) - Network access control for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Complete Model Context Protocol setup and usage
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
