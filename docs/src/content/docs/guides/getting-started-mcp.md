---
title: Getting Started with MCP
description: Learn how to integrate Model Context Protocol (MCP) servers with your agentic workflows to connect AI agents to GitHub, databases, and external services.
sidebar:
  order: 2
---

This guide walks you through integrating [Model Context Protocol](/gh-aw/reference/glossary/#mcp-model-context-protocol) (MCP) servers with GitHub Agentic Workflows, from your first configuration to advanced patterns.

## What is MCP?

[Model Context Protocol](/gh-aw/reference/glossary/#mcp-model-context-protocol) (MCP) is a standardized protocol that enables agents to connect to external tools, databases, and APIs. [MCP servers](/gh-aw/reference/glossary/#mcp-server) act as specialized adapters, giving agents access to GitHub, web search, databases, and third-party services like Notion, Slack, and Datadog.

## Quick Start

Get your first MCP integration running in under 5 minutes.

### Step 1: Add GitHub Tools

Create a workflow file at `.github/workflows/my-workflow.md`:

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [default]
---

# Issue Analysis Agent

Analyze the issue and provide a summary of similar existing issues.
```

The `toolsets: [default]` configuration gives your agentic workflow access to repository, issue, and pull request tools.

### Step 2: Compile and Test

Compile the workflow to generate the GitHub Actions YAML:

```bash
gh aw compile my-workflow
```

Verify the MCP configuration:

```bash
gh aw mcp inspect my-workflow
```

You now have a working MCP integration. The agent can read issues, search repositories, and access pull request information.

## Configuration Patterns

### Toolsets Pattern (Recommended)

Use `toolsets:` to enable groups of related GitHub tools:

```yaml wrap
tools:
  github:
    toolsets: [default]  # Expands to: context, repos, issues, pull_requests (action-friendly)
```

Toolsets remain stable across MCP server versions, while individual tool names may change. See [Available Toolsets](#available-toolsets) for the full list.

### Allowed Pattern (Custom MCP Servers)

Use `allowed:` when configuring custom (non-GitHub) MCP servers:

```yaml wrap
mcp-servers:
  notion:
    container: "mcp/notion"
    allowed: ["search_pages", "get_page"]
```

## GitHub MCP Server

The GitHub MCP server is built into agentic workflows and provides comprehensive access to GitHub's API.

### Available Toolsets

| Toolset | Description | Tools |
|---------|-------------|-------|
| `context` | User and team information | `get_teams`, `get_team_members` |
| `repos` | Repository operations | `get_repository`, `get_file_contents`, `list_commits` |
| `issues` | Issue management | `list_issues`, `create_issue`, `update_issue` |
| `pull_requests` | PR operations | `list_pull_requests`, `create_pull_request` |
| `actions` | Workflow runs and artifacts | `list_workflows`, `list_workflow_runs` |
| `discussions` | GitHub Discussions | `list_discussions`, `create_discussion` |
| `code_security` | Security alerts | `list_code_scanning_alerts` |
| `users` | User profiles | `get_me`, `get_user`, `list_users` |

The `default` toolset includes: `context`, `repos`, `issues`, `pull_requests`. When used in workflows, `[default]` expands to action-friendly toolsets that work with GitHub Actions tokens. Note: The `users` toolset is not included by default as GitHub Actions tokens do not support user operations.

### Operating Modes

Remote mode (`mode: remote`) connects to a hosted server for faster startup with no Docker required. Local mode (`mode: local`) runs in Docker, enabling version pinning for offline or restricted environments. See [Remote vs Local Mode](/gh-aw/reference/github-tools/#remote-vs-local-mode).

The GitHub MCP server always operates read-only. Write operations are handled through [safe outputs](/gh-aw/reference/safe-outputs/), which run in a separate permission-controlled job.

## MCP Registry

The GitHub MCP registry provides a centralized catalog of MCP servers.

### Adding Servers

```bash
# Browse available MCP servers
gh aw mcp add

# Add a specific server
gh aw mcp add my-workflow makenotion/notion-mcp-server

# Add with custom tool ID
gh aw mcp add my-workflow makenotion/notion-mcp-server --tool-id my-notion
```

The command searches the registry, adds the server configuration, and recompiles the workflow.

Reference registry servers directly in your workflow:

```yaml wrap
mcp-servers:
  markitdown:
    registry: https://api.mcp.github.com/v0/servers/microsoft/markitdown
    container: "ghcr.io/microsoft/markitdown"
    allowed: ["*"]
```

The `registry` field provides metadata for tooling while the `container` or `command` fields specify how to run the server.

### Using a Custom Registry

For enterprise or private registries:

```bash
gh aw mcp add my-workflow server-name --registry https://custom.registry.com/v1
```

## Custom MCP Servers

Configure third-party MCP servers using commands, Docker containers, or HTTP endpoints:

```yaml wrap
mcp-servers:
  # Command-based (stdio)
  markitdown:
    command: "npx"
    args: ["-y", "markitdown-mcp"]
    allowed: ["*"]

  # Docker container
  ast-grep:
    container: "mcp/ast-grep:latest"
    allowed: ["*"]

  # HTTP endpoint with auth
  slack:
    url: "https://api.slack.com/mcp"
    env:
      SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
    network:
      allowed: ["api.slack.com"]  # Optional egress restrictions
    allowed: ["send_message", "get_channel_history"]
```

## Practical Examples

### Example 1: Basic Issue Triage

```aw wrap
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [default]
safe-outputs:
  add-comment:
---

# Issue Triage Agent

Analyze issue #${{ github.event.issue.number }} and add a comment with category, related issues, and suggested labels.
```

### Example 2: Multi-Service Integration

```aw wrap
---
on: weekly on sunday
permissions:
  contents: read
  security-events: read
  discussions: write
tools:
  github:
    toolsets: [default, code_security, discussions]
safe-outputs:
  create-discussion:
    category: "Security"
    title-prefix: "[security-scan] "
---

# Security Audit Agent

Review code scanning alerts and create weekly security discussions with findings.
```

## Debugging MCP Configurations

Inspect configured servers and available tools:

```bash
# View all MCP servers
gh aw mcp inspect my-workflow

# Get detailed server information
gh aw mcp inspect my-workflow --server github --verbose

# List available tools
gh aw mcp list-tools github my-workflow

# Validate configuration
gh aw compile my-workflow --validate --strict
```

## Troubleshooting

**Tool not found:** Run `gh aw mcp inspect my-workflow` to verify available tools. Ensure the correct toolset is enabled or that tool names in `allowed:` match exactly.

**Authentication errors:** Verify the secret exists in repository settings and has required scopes.

**Connection failures:** Check URL syntax for HTTP servers, network configuration for containers, and verify Docker images exist.

**Validation errors:** Check YAML syntax, ensure `toolsets:` uses array format (`[default]` not `default`), and verify `allowed:` is an array.

## Next Steps

- [Using MCPs](/gh-aw/guides/mcps/) - Complete MCP configuration reference
- [Tools Reference](/gh-aw/reference/tools/) - All available tools and options
- [Security Guide](/gh-aw/introduction/architecture/) - MCP security best practices
- [CLI Commands](/gh-aw/setup/cli/) - Full CLI documentation including `mcp` commands
- [Imports](/gh-aw/reference/imports/) - Shared MCP configurations in `.github/workflows/shared/mcp/`
