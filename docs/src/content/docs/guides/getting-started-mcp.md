---
title: Getting Started with MCP
description: Learn how to integrate Model Context Protocol (MCP) servers with your agentic workflows to connect AI agents to GitHub, databases, and external services.
sidebar:
  order: 2
---

This guide walks you through integrating MCP servers with GitHub Agentic Workflows, from your first configuration to advanced patterns.

## What is MCP?

Model Context Protocol (MCP) is a standardized protocol that enables AI agents to securely connect to external tools, databases, and APIs. In agentic workflows, MCP servers provide the "hands and eyes" that allow AI to:

- Access GitHub repositories, issues, and pull requests
- Query databases and external APIs
- Search the web and fetch content
- Interact with third-party services like Notion, Slack, and Datadog

Think of MCP servers as specialized adapters—each one connects your AI agent to a specific set of capabilities.

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

The `toolsets: [default]` configuration gives your agent access to repository, issue, and pull request tools.

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

Agentic workflows support two patterns for configuring MCP tools. Understanding when to use each pattern ensures stable, maintainable workflows.

### Toolsets Pattern (Recommended)

Use `toolsets:` to enable groups of related GitHub tools:

```yaml wrap
tools:
  github:
    toolsets: [default]  # Includes repos, issues, pull_requests, users, context
```

Toolsets are the recommended approach because individual tool names may change between MCP server versions, but toolsets remain stable.

**Common toolset combinations:**

| Use Case | Toolsets |
|----------|----------|
| General workflows | `[default]` |
| Issue/PR management | `[default, discussions]` |
| CI/CD workflows | `[default, actions]` |
| Security scanning | `[default, code_security]` |
| Full access | `[all]` |

### Allowed Pattern (Custom MCP Servers)

Use `allowed:` when configuring custom (non-GitHub) MCP servers:

```yaml wrap
mcp-servers:
  notion:
    container: "mcp/notion"
    allowed: ["search_pages", "get_page"]
```

:::caution
For GitHub tools, always use `toolsets:` instead of `allowed:`. The `allowed:` pattern for GitHub tools is deprecated because tool names may change between versions.
:::

## GitHub MCP Server

The GitHub MCP server is built into agentic workflows and provides comprehensive access to GitHub's API.

### Available Toolsets

| Toolset | Description | Tools |
|---------|-------------|-------|
| `context` | User and team information | `get_me`, `get_teams`, `get_team_members` |
| `repos` | Repository operations | `get_repository`, `get_file_contents`, `list_commits` |
| `issues` | Issue management | `list_issues`, `create_issue`, `update_issue` |
| `pull_requests` | PR operations | `list_pull_requests`, `create_pull_request` |
| `actions` | Workflow runs and artifacts | `list_workflows`, `list_workflow_runs` |
| `discussions` | GitHub Discussions | `list_discussions`, `create_discussion` |
| `code_security` | Security alerts | `list_code_scanning_alerts` |
| `users` | User profiles | `get_user`, `list_users` |

The `default` toolset includes: `context`, `repos`, `issues`, `pull_requests`, `users`.

### Operating Modes

GitHub MCP supports two modes. Choose based on your requirements:

**Remote Mode (Recommended):**
```yaml wrap
tools:
  github:
    mode: remote
    toolsets: [default]
```
Remote mode connects to the hosted GitHub MCP server with faster startup and no Docker requirement.

**Local Mode (Docker-based):**
```yaml wrap
tools:
  github:
    mode: local
    toolsets: [default]
    version: "sha-09deac4"
```
Local mode runs the MCP server in a Docker container, useful for pinning specific versions or offline environments.

### Authentication

Token precedence (highest to lowest):

1. `github-token` field in configuration
2. `GH_AW_GITHUB_TOKEN` secret
3. `GITHUB_TOKEN` (default)

Custom token configuration:

```yaml wrap
tools:
  github:
    github-token: "${{ secrets.CUSTOM_PAT }}"
    toolsets: [default]
```

### Read-Only Mode

Restrict operations to read-only for security-sensitive workflows:

```yaml wrap
tools:
  github:
    read-only: true
    toolsets: [repos, issues]
```

## Custom MCP Servers

Extend your workflows with third-party MCP servers for services like databases, APIs, and specialized tools.

### Server Types

**Command-based (stdio):**
```yaml wrap
mcp-servers:
  markitdown:
    command: "npx"
    args: ["-y", "markitdown-mcp"]
    allowed: ["*"]
```

**Docker containers:**
```yaml wrap
mcp-servers:
  ast-grep:
    container: "mcp/ast-grep:latest"
    allowed: ["*"]
```

**HTTP endpoints:**
```yaml wrap
mcp-servers:
  microsoftdocs:
    url: "https://learn.microsoft.com/api/mcp"
    allowed: ["*"]
```

### Environment Variables

Pass secrets and configuration to MCP servers:

```yaml wrap
mcp-servers:
  slack:
    command: "npx"
    args: ["-y", "@slack/mcp-server"]
    env:
      SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
    allowed: ["send_message", "get_channel_history"]
```

### Network Restrictions

Limit egress for containerized MCP servers:

```yaml wrap
mcp-servers:
  custom-tool:
    container: "mcp/custom-tool"
    network:
      allowed: ["api.example.com"]
    allowed: ["*"]
```

## Practical Examples

These examples demonstrate progressively complex MCP configurations.

### Example 1: Basic Issue Triage

A simple workflow using default GitHub toolsets:

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

Analyze issue #${{ github.event.issue.number }} and add a helpful comment with:
- Category classification
- Related issues
- Suggested labels
```

### Example 2: PR Review with Actions Data

Access GitHub Actions data for CI-aware reviews:

```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  pull-requests: read
  actions: read
tools:
  github:
    toolsets: [default, actions]
safe-outputs:
  add-comment:
---

# PR Review Agent

Review pull request #${{ github.event.pull_request.number }}:
1. Check recent workflow runs for failures
2. Analyze code changes
3. Provide feedback on potential issues
```

### Example 3: Documentation Search

Integrate external documentation sources:

```aw wrap
---
on:
  issues:
    types: [labeled]
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [default]
mcp-servers:
  microsoftdocs:
    url: "https://learn.microsoft.com/api/mcp"
    allowed: ["*"]
safe-outputs:
  add-comment:
---

# Documentation Helper

When an issue is labeled "docs-needed", search Microsoft documentation
for relevant resources and add a helpful comment.
```

### Example 4: Multi-Service Integration

Combine multiple MCP servers for complex workflows:

```aw wrap
---
on:
  schedule:
    - cron: "0 9 * * 1"
permissions:
  contents: read
  issues: write
tools:
  github:
    toolsets: [default]
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/"
    headers:
      Authorization: "Bearer ${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
imports:
  - shared/mcp/notion.md
safe-outputs:
  create-issue:
    title-prefix: "[weekly] "
---

# Weekly Research Agent

Every Monday:
1. Search the web for industry trends using Tavily
2. Check Notion for team priorities
3. Create a summary issue with findings
```

### Example 5: Security Scanning with Discussions

Create detailed security reports using discussions:

```aw wrap
---
on:
  schedule:
    - cron: "0 0 * * 0"
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

Weekly security analysis:
1. Review code scanning alerts
2. Check for new vulnerabilities
3. Create a discussion with findings and remediation suggestions
```

## Debugging MCP Configurations

Use CLI tools to inspect and troubleshoot MCP configurations.

### Inspect MCP Servers

View all configured MCP servers:

```bash
gh aw mcp inspect my-workflow
```

Get detailed server information:

```bash
gh aw mcp inspect my-workflow --server github --verbose
```

### List Available Tools

Discover tools from a specific MCP server:

```bash
gh aw mcp list-tools github my-workflow
```

View tool details:

```bash
gh aw mcp inspect my-workflow --server github --tool list_issues
```

### Validate Configuration

Check for configuration issues:

```bash
gh aw compile my-workflow --validate
```

Run with strict mode for production readiness:

```bash
gh aw compile my-workflow --strict
```

## Troubleshooting

Common issues and solutions for MCP configurations.

### Tool Not Found

**Symptom:** Workflow fails with "tool not found" error.

**Solutions:**
1. Check toolset coverage: Run `gh aw mcp inspect my-workflow` to see available tools.
2. Add missing toolset: Include the appropriate toolset that contains the needed tool.
3. For custom MCP servers: Verify the tool name in `allowed:` matches exactly.

### Authentication Errors

**Symptom:** MCP server returns 401 or authentication error.

**Solutions:**
1. Verify the secret exists: Check repository secrets in Settings.
2. Check token permissions: Ensure the token has required scopes.
3. For remote mode: Set `GH_AW_GITHUB_TOKEN` secret with a PAT.

### Connection Failures

**Symptom:** MCP server fails to connect or times out.

**Solutions:**
1. Verify URL syntax for HTTP servers.
2. Check network configuration for containerized servers.
3. For Docker-based servers: Ensure the container image exists.

### Configuration Validation Errors

**Symptom:** Compilation fails with validation errors.

**Solutions:**
1. Check YAML indentation and syntax.
2. Verify `toolsets:` uses array format: `[default]` not `default`.
3. For custom servers: Ensure `allowed:` is an array.

## Next Steps

Continue learning with these resources:

- [Using MCPs](/gh-aw/guides/mcps/) — Complete MCP configuration reference
- [Tools Reference](/gh-aw/reference/tools/) — All available tools and options
- [Security Guide](/gh-aw/guides/security/) — MCP security best practices
- [CLI Commands](/gh-aw/setup/cli/) — Full CLI documentation including `mcp` commands
- [Imports](/gh-aw/reference/imports/) — Modularize configurations with shared MCP files

Explore shared MCP configurations in `.github/workflows/shared/mcp/` for ready-to-use integrations with popular services.
