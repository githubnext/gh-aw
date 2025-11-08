---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
sidebar:
  order: 200
---

This page lists available commands for managing agentic workflows with the GitHub CLI extension `gh aw`.

## Installation

```bash wrap
gh extension install githubnext/gh-aw
```

The CLI supports GitHub Enterprise Server through the `GITHUB_SERVER_URL` or `GH_HOST` environment variables. When set, commands like `gh aw add` will use the specified GitHub instance for cloning and accessing workflows.

## Quick Start

```bash wrap
# Get started
gh aw init                                   # Initialize repository
gh aw add githubnext/agentics/ci-doctor      # Add and compile a workflow
gh aw trial ./my-workflow.md                 # Test workflow safely
gh aw run daily-perf                         # Execute workflow
gh aw logs daily-perf                        # View execution logs
gh aw status                                 # Check workflow status
```

## Global Flags

- **`--verbose` / `-v`**: Enable verbose output with debugging details
- **`--help` / `-h`**: Show help information

## Getting Help

Use `gh aw --help`, `gh aw help [command]`, or `gh aw help all` for comprehensive command documentation.

## Workflow Creation and Management

### Repository Initialization

```bash wrap
gh aw init       # Configure .gitattributes, Copilot instructions, custom agent
gh aw init --mcp # Also setup MCP server integration for Copilot Agent
```

The `--mcp` flag creates GitHub Actions workflow for MCP setup, configures `.vscode/mcp.json`, and enables gh-aw MCP tools (`status`, `compile`, `logs`, `audit`) in Copilot Agent.

### Workflow Management

```bash wrap
# Create, add, and remove workflows
gh aw new my-custom-workflow
gh aw add githubnext/agentics/ci-doctor
gh aw add "githubnext/agentics/ci-*"             # Add multiple with wildcards
gh aw add ci-doctor --dir shared --number 3      # Organize and create copies
gh aw remove WorkflowName
```

The `add` command automatically updates `.gitattributes` to mark `.lock.yml` files as generated (use `--no-gitattributes` to disable). Use `--dir` to organize workflows in subdirectories under `.github/workflows/`. When workflows aren't found, available options are displayed in a formatted table.

### Workflow Updates

```bash wrap
gh aw update                              # Update all workflows with source field
gh aw update ci-doctor --major --force    # Allow major version updates
```

Updates use `source` field format `owner/repo/path@ref`. Semantic version tags update within the same major version (use `--major` for major updates). Performs 3-way merge preserving local changes; conflicts add diff3 markers and skip recompilation until resolved.

## Compilation

```bash wrap
gh aw compile                              # Compile all workflows
gh aw compile --watch                      # Auto-recompile on changes
gh aw compile --validate --strict          # Schema validation + strict mode
gh aw compile --zizmor                     # Security scan (warnings)
gh aw compile --strict --zizmor            # Security scan (fails on findings)
gh aw compile --dependabot                 # Generate dependency manifests
```

**Flags:**
- `--validate`: Schema validation and container checks (disabled by default)
- `--strict`: Requires timeouts, explicit network config, blocks write permissions
- `--zizmor`: Runs [zizmor](https://github.com/woodruffw/zizmor) security scanner for template injection and permission risks
- `--dependabot`: Generates npm/pip/Go manifests and updates `.github/dependabot.yml`
- `--watch`: Auto-recompile on file changes (see [VS Code setup](/gh-aw/setup/vscode/))

Compilation fails if workflows use `create-discussion` or `create-issue` but the repository lacks those features.

## Workflow Operations

### Execution

```bash wrap
gh aw run WorkflowName                      # Run workflow
gh aw run workflow1 workflow2 --repeat 3    # Run multiple, repeat 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
```

:::note[Codespaces]
From GitHub Codespaces, grant `actions: write` and `workflows: write` permissions. See [Managing repository access](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces).
:::

The `--use-local-secrets` flag temporarily pushes AI engine secrets from environment variables to the repository, then cleans them up. Only pushes secrets needed by the workflow's engine.

### State Management

```bash wrap
gh aw status                                # Show all workflow status
gh aw enable [prefix]                       # Enable workflows (supports patterns)
gh aw disable [prefix]                      # Disable and cancel workflows
```

### Log Analysis

```bash wrap
gh aw logs [workflow-name]                # Download and analyze logs
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --parse --json                 # Generate markdown + JSON output
```

Downloaded runs are cached (~10-100x faster on subsequent runs). Use `--parse` to generate `log.md` and `firewall.md` with tool calls and network patterns. Use `--json` for structured metrics output.

### Run Audit

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By URL
gh aw audit 12345678 --parse                              # Parse logs to markdown
```

Accepts run IDs or URLs from any repository and GitHub instance. Reports include overview, metrics, tool usage, MCP failures, firewall analysis, and artifacts.

### MCP Servers

```bash wrap
gh aw mcp list [workflow]                 # List MCP servers
gh aw mcp inspect [workflow]              # Inspect and test servers
gh aw mcp add                             # Add servers from registry
```

See [MCPs Guide](/gh-aw/guides/mcps/) and [MCP Server Guide](/gh-aw/setup/mcp-server/) for details.

## Repository Utilities

### Trial Mode

Test workflows in a temporary private repository:

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test workflow
gh aw trial ./workflow.md --use-local-secrets       # Test with local API keys
gh aw trial ./workflow.md --logical-repo owner/repo # Act as different repo
gh aw trial ./issue-workflow.md --trigger-context "#123" # With issue context
```

Common flags: `--engine`, `--auto-merge-prs`, `--repeat N`, `--delete-host-repo-after`

Trial results save to `trials/` directory. Use `--logical-repo` to access issues/PRs from a specific repository, or `--clone-repo` to use a different codebase.

### PR Transfer

```bash wrap
gh aw pr transfer https://github.com/source/repo/pull/234
gh aw pr transfer <pr-url> --repo target-owner/target-repo
```

Transfers PR between repositories, preserving code changes, title, and description.

### gh-aw MCP Server

```bash wrap
gh aw mcp-server              # stdio transport (local)
gh aw mcp-server --port 3000  # HTTP/SSE transport (workflows)
```

Exposes CLI commands as MCP tools. See [MCP Server Guide](/gh-aw/setup/mcp-server/).

## Debug Logging

```bash wrap
DEBUG=* gh aw compile                # All debug logs
DEBUG=cli:* gh aw compile            # CLI operations only
DEBUG=cli:*,workflow:* gh aw compile # Multiple packages
```

Debug logs show namespace, message, and time diff (e.g., `+50ms`). Zero overhead when disabled. Use `--verbose` for user-facing details.

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and file organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options for workflows
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure output processing including issue updates
- [Tools](/gh-aw/reference/tools/) - GitHub and MCP server configuration
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
