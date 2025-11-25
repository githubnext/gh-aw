---
title: Tools
description: Configure GitHub API tools, browser automation, and AI capabilities available to your agentic workflows, including GitHub tools, Playwright, and custom MCP servers.
sidebar:
  order: 700
---

Tools are defined in the frontmatter to specify which GitHub API calls, browser automation, and AI capabilities are available to your workflow:

```yaml wrap
tools:
  edit:
  bash: true
```

Some tools are available by default. All tools declared in imported components are merged into the final workflow.

## Edit Tool (`edit:`)

Allows file editing in the GitHub Actions workspace.

```yaml wrap
tools:
  edit:
```

## Bash Tool (`bash:`)

Allows shell command execution in the GitHub Actions workspace.

```yaml wrap
tools:
  bash:                              # Default safe commands
  bash: []                           # No commands allowed
  bash: ["echo", "ls", "git status"] # Specific commands only
  bash: [":*"]                       # All commands (use with caution)
```

**Configuration:** `bash:` provides default safe commands (`echo`, `ls`, `pwd`, `cat`, `head`, `tail`, `grep`, `wc`, `sort`, `uniq`, `date`). Use `bash: []` to disable, `bash: ["cmd1", "cmd2"]` for specific commands, or `bash: [":*"]` for unrestricted access.

**Wildcards:** Use `:*` for all commands, or `command:*` for specific command families (e.g., `git:*` allows all git operations).

## Web Tools

Enable web content fetching and search capabilities:

```yaml wrap
tools:
  web-fetch:   # Fetch web content
  web-search:  # Search the web (engine-dependent)
```

**Note:** Some engines require third-party MCP servers for web search. See [Using Web Search](/gh-aw/guides/web-search/).

## GitHub Tools (`github:`)

Configure GitHub API operations.

```yaml wrap
tools:
  github:                                      # Default read-only access
  github:
    toolsets: [repos, issues, pull_requests]   # Toolset groups (recommended)
    allowed: [create_issue, update_issue]      # Or specific permissions
    mode: remote                               # "local" (Docker) or "remote" (hosted)
    read-only: true                            # Read-only operations
    github-token: "${{ secrets.CUSTOM_PAT }}"  # Custom token
```

### GitHub Toolsets

:::tip[Prefer Toolsets Over Individual Tools]
Use `toolsets:` to enable groups of related tools instead of listing individual tools with `allowed:`. Toolsets reduce configuration verbosity and ensure complete functionality.
:::

Enable specific GitHub API groups to improve tool selection and reduce context size:

```yaml wrap
tools:
  github:
    toolsets: [repos, issues, pull_requests, actions]
```

**Available Toolsets**: `context`, `repos`, `issues`, `pull_requests`, `users`, `actions`, `code_security`, `discussions`, `labels`, `notifications`, `orgs`, `projects`, `gists`, `search`, `dependabot`, `experiments`, `secret_protection`, `security_advisories`, `stargazers`

**Default**: `context`, `repos`, `issues`, `pull_requests`, `users`

**Common Combinations**:
- Read-only: `[default]` or `[context, repos]`
- Issue/PR management: `[default, discussions]`
- CI/CD: `[default, actions]`
- Security: `[default, code_security, secret_protection]`
- Full access: `[all]`

#### Toolset Contents

Common toolsets include:
- **context**: User/team info (`get_me`, `get_teams`, `get_team_members`)
- **repos**: Repository operations (`get_repository`, `get_file_contents`, `search_code`, `list_commits`, releases)
- **issues**: Issue management (`list_issues`, `create_issue`, `update_issue`, `search_issues`, comments, reactions)
- **pull_requests**: PR operations (`list_pull_requests`, `get_pull_request`, `create_pull_request`, `search_pull_requests`)
- **actions**: Workflows and runs (`list_workflows`, `list_workflow_runs`, `get_workflow_run`, artifacts)
- **code_security**: Security scanning alerts
- **discussions**: GitHub Discussions
- **labels**: Label management

Combine `toolsets:` with `allowed:` to further restrict available tools within enabled toolsets. Toolsets work in both local (Docker) and remote (hosted) modes.

### Modes and Restrictions

**Remote Mode**: Use the hosted GitHub MCP server for faster startup without Docker. Requires setting `GH_AW_GITHUB_TOKEN` secret (standard `GITHUB_TOKEN` not supported):

```yaml wrap
tools:
  github:
    mode: remote  # Default is "local" (Docker)
```

Setup: `gh secret set GH_AW_GITHUB_TOKEN -a actions --body "<your-pat>"`

**Read-Only Mode**: Restrict to read-only operations (default behavior unless write operations configured):

```yaml wrap
tools:
  github:
    read-only: true
```

**Lockdown Mode**: Limit content from public repositories to items authored by users with push access. Private repositories and collaborator-owned content remain unaffected:

```yaml wrap
tools:
  github:
    lockdown: true
```

Lockdown mode filters issue comments, sub-issues, and PR content to prevent exposure of potentially untrusted content from public repositories. Useful for security-sensitive workflows processing public repository data.

## Playwright Tool (`playwright:`)

Enables browser automation using containerized Playwright with domain-based access control:

```yaml wrap
tools:
  playwright:
    allowed_domains: ["defaults", "github", "*.custom.com"]  # Domain access
    version: "1.56.1"  # Optional: pin version (defaults to 1.56.1)
```

**Version Pinning**: Defaults to version 1.56.1 for stability. Set `version: "latest"` to use the latest version or specify a different version number.

**Domain Access**: Uses same ecosystem bundles as `network:` configuration (`defaults`, `github`, `node`, `python`, etc.). Default is `["localhost", "127.0.0.1"]` for security. Supports wildcards like `*.example.com`.

## Custom MCP Servers (`mcp-servers:`)

Integrate custom Model Context Protocol servers for third-party services, APIs, or specialized tools:

```yaml wrap
mcp-servers:
  slack:
    command: "npx"  # or "node", "python"
    args: ["-y", "@slack/mcp-server"]
    env:
      SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
    allowed: ["send_message", "get_channel_history"]

  notion:
    container: "mcp/notion"  # Docker alternative
    env:
      NOTION_API_TOKEN: "${{ secrets.NOTION_API_TOKEN }}"
    allowed: ["search_pages", "get_page"]

  datadog:
    url: "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"  # HTTP alternative
    headers:
      DD_API_KEY: "${{ secrets.DD_API_KEY }}"
    allowed: ["search_datadog_dashboards"]
```

**Configuration Options**:
- `command:` + `args:` - Process-based MCP server (Node.js, Python, etc.)
- `container:` - Docker container image
- `url:` + `headers:` - HTTP endpoint with authentication
- `env:` - Environment variables for the MCP server
- `allowed:` - Restrict to specific tool names

MCP servers run in isolated environments with controlled network access. See [MCPs Guide](/gh-aw/guides/mcps/) for detailed setup instructions.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
- [Network Permissions](/gh-aw/reference/network/) - Network access control for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Complete Model Context Protocol setup and usage
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
