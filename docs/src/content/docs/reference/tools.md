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

Enables or disables specific GitHub API groups to improve tool selection and reduce context size.

```yaml
tools:
  github:
    toolset: [repos, issues, pull_requests, actions]
```

**Available Toolsets**: `context` (recommended), `actions`, `code_security`, `dependabot`, `discussions`, `experiments`, `gists`, `issues`, `labels`, `notifications`, `orgs`, `projects`, `pull_requests`, `repos`, `secret_protection`, `security_advisories`, `stargazers`, `users`

Default: `toolset: [all]` enables all toolsets. Currently supported in local (Docker) mode only.

### GitHub Remote Mode

Uses the hosted GitHub MCP server at `https://api.githubcopilot.com/mcp/` for faster startup without Docker.

```yaml
tools:
  github:
    mode: remote
    allowed: [list_issues, create_issue]
```

**Setup**: Create a Personal Access Token and set the `GITHUB_MCP_TOKEN` secret:

```bash
gh secret set GITHUB_MCP_TOKEN -a actions --body "<your-github-pat>"
```

**Note**: Remote mode requires `GITHUB_MCP_TOKEN` (standard `GITHUB_TOKEN` is not supported).

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

## Other MCP Servers

Use `mcp-servers:` for custom MCP server configuration:

```yaml
mcp-servers:
  custom-api:
    command: "node"
    args: ["custom-mcp-server.js"]
    env:
      API_KEY: "${{ secrets.CUSTOM_API_KEY }}"

tools:
  github:
    allowed: [create_issue, update_issue]
```

MCP servers run alongside the AI engine in isolated environments with controlled network access.

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All frontmatter configuration options
- [Network Permissions](/gh-aw/reference/network/) - Network access control for AI engines
- [MCPs](/gh-aw/guides/mcps/) - Complete Model Context Protocol setup and usage
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
