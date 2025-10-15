---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
sidebar:
  order: 200
---

This page lists available commands for managing agentic workflows with the GitHub CLI extension `gh aw`.

## Installation

```bash
gh extension install githubnext/gh-aw
```

## Quick Start

```bash
# Show version and help
gh aw version
gh aw --help

# Basic workflow lifecycle
gh aw init                                       # Initialize repository (first-time setup)
gh aw add githubnext/agentics/ci-doctor    # Add workflow and compile to GitHub Actions
gh aw compile                                    # Recompile to GitHub Actions
gh aw trial githubnext/agentics/ci-doctor  # Test workflow safely before adding
gh aw trial ./my-workflow.md                # Test local workflow during development
gh aw update                                     # Update all workflows with source field
gh aw status                                     # Check status
gh aw run ci-doctor                        # Execute workflow
gh aw run ci-doctor daily-plan             # Execute multiple workflows
gh aw run ci-doctor --repeat 3600          # Execute workflow every hour
gh aw logs ci-doctor                             # View execution logs
gh aw audit 12345678                             # Audit a specific run
```

## Global Flags

- **`--verbose` / `-v`**: Enable verbose output with debugging details
- **`--help` / `-h`**: Show help information

## 📝 Workflow Creation and Management  

The `add` and `new` commands help you create and manage agentic workflows, from templates and samples to completely custom workflows.

The `init` command prepares your repository for agentic workflows by configuring `.gitattributes` and creating GitHub Copilot custom instructions.

```bash
# Create new workflows
gh aw new my-custom-workflow
gh aw new issue-handler --force

# Add workflows from samples repository
gh aw add githubnext/agentics/ci-doctor
gh aw add githubnext/agentics/ci-doctor --name my-custom-doctor --pr --engine copilot
gh aw add githubnext/agentics/ci-doctor --number 3  # Create 3 copies

# Remove workflows
gh aw remove WorkflowName
gh aw remove WorkflowName --keep-orphans  # Keep shared includes
```

**Workflow Updates:**

Update workflows from external repositories using the `source` field in frontmatter:

```bash
gh aw update                              # Update all workflows with source field
gh aw update ci-doctor issue-triage      # Update specific workflows
gh aw update ci-doctor --major --force   # Allow major version updates
gh aw update --verbose --engine copilot  # Override engine
```

**Update Logic:**

Updates based on the `source` field format: `owner/repo/path/to/workflow.md@ref`

- **Semantic Version Tags** (`v1.2.3`): Updates within same major version (use `--major` for major updates)
- **Branch References** (`main`): Fetches latest commit from specified branch
- **No Reference**: Uses repository's default branch

The update performs a 3-way merge using `git merge-file`, preserving local modifications while applying upstream changes. When conflicts occur, diff3-style markers are added for manual resolution, and recompilation is skipped until resolved.

## 🔧 Workflow Recompilation

Transforms markdown workflow files into executable GitHub Actions YAML files:

```bash
# Core compilation
gh aw compile                              # Compile all workflows
gh aw compile ci-doctor daily-plan         # Compile specific workflows
gh aw compile --validate --no-emit         # Validate without generating files
gh aw compile --strict --engine copilot    # Strict mode with custom engine
gh aw compile --purge                      # Remove orphaned .lock.yml files

# Development features
gh aw compile --watch --verbose            # Auto-recompile on changes
gh aw compile --workflows-dir custom/      # Custom workflows directory
```

**Strict Mode:**

Enables enhanced security validation requiring timeouts, explicit network configuration, and blocking write permissions. Use `--strict` flag or `strict: true` in frontmatter.

## ⚙️ Workflow Operations on GitHub Actions

These commands control the execution and state of your compiled agentic workflows within GitHub Actions.

### Workflow Execution

```bash
gh aw run WorkflowName                      # Run single workflow
gh aw run WorkflowName1 WorkflowName2       # Run multiple workflows
gh aw run WorkflowName --repeat 3           # Run 3 times total
gh aw run weekly-research --enable-if-needed --input priority=high
```

### Trial Mode

Test workflows safely in a temporary private repository without affecting your target repository:

```bash
gh aw trial githubnext/agentics/ci-doctor  # Test from source repo
gh aw trial ./my-local-workflow.md         # Test local file
gh aw trial workflow1 workflow2            # Compare multiple workflows
gh aw trial ./workflow.md --logical-repo myorg/myrepo --timeout 60
gh aw trial ./workflow.md --host-repo . --delete-host-repo --yes

# Test issue-triggered workflows with context
gh aw trial ./issue-workflow.md --trigger-context https://github.com/owner/repo/issues/123
gh aw trial githubnext/agentics/issue-triage --trigger-context "#456"
```

Trial results are saved to `trials/` directory and captured in the trial repository for inspection. Set `GH_AW_GITHUB_TOKEN` to override authentication. See the [Security Guide](/gh-aw/guides/security/#authorization-and-token-management).

### Workflow State Management

```bash
gh aw status [WorkflowPrefix]               # Show workflow status
gh aw enable [WorkflowPrefix]               # Enable workflows
gh aw disable [WorkflowPrefix]              # Disable and cancel workflows
```

Status shows workflow names, enabled/disabled state, execution status, and compilation status. Enable/disable commands support pattern matching.

### Log Analysis and Monitoring

Download and analyze workflow execution history with performance metrics, cost tracking, and error analysis:

```bash
gh aw logs [workflow-name] -o ./analysis  # Download logs
gh aw logs -c 10 --start-date -1w --end-date -1d
gh aw logs --engine claude --branch main
gh aw logs --after-run-id 1000 --before-run-id 2000
gh aw logs --no-staged --tool-graph       # Exclude staged runs, generate Mermaid graph
gh aw logs --parse --verbose --json       # Parse logs to markdown, output JSON
```

Metrics include execution duration, token consumption, costs, success/failure rates, and resource usage trends.

**Log Parsing and JSON Output:**

- `--parse`: Generates `log.md` files with tool calls, reasoning, and execution details extracted by engine-specific parsers
- `--json`: Outputs structured JSON with summary metrics, runs, tool usage, missing tools, MCP failures, and access logs

### Single Run Audit

Generate concise markdown reports for individual workflow runs with smart permission handling:

```bash
gh aw audit 12345678                                            # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123      # By URL
gh aw audit 12345678 -o ./audit-reports -v
```

The audit command checks local cache first (`logs/run-{id}`), then attempts download. On permission errors, it provides MCP server instructions for artifact downloads. Reports include overview, metrics, tool usage, MCP failures, and available artifacts.

### MCP Server Management

Discover, list, inspect, and add Model Context Protocol (MCP) servers. See [MCPs Guide](/gh-aw/guides/mcps/) and [MCP Server Guide](/gh-aw/tools/mcp-server/).

```bash
# Discovery and listing
gh aw mcp list [workflow-name] [--verbose]
gh aw mcp list-tools github [ci-doctor] [--verbose]

# Inspection and testing
gh aw mcp inspect [workflow-name] [--server server-name] [--tool tool-name]
gh aw mcp inspect workflow-name --inspector  # Launch web inspector

# Add from registry
gh aw mcp add                                # List available servers
gh aw mcp add ci-doctor makenotion/notion-mcp-server --transport stdio --tool-id my-notion
gh aw mcp add ci-doctor server-name --registry https://custom.registry.com/v1
```

Features include server connection testing, tool capability analysis, multi-protocol support (stdio, Docker, HTTP), and automatic compilation.

### MCP Server for gh aw

Run gh-aw as an MCP server exposing CLI commands (`status`, `compile`, `logs`, `audit`) as tools for AI agents:

```bash
gh aw mcp-server                    # stdio transport (local CLI)
gh aw mcp-server --port 3000        # HTTP/SSE transport (workflows)
gh aw mcp-server --cmd ./gh-aw --port 3000
```

Uses subprocess architecture for token isolation. Import with `shared/mcp/gh-aw.md` in workflows. See [MCP Server Guide](/gh-aw/tools/mcp-server/).

## 👀 Watch Mode for Development

Auto-recompile workflows on file changes. See [Authoring in VS Code](/gh-aw/tools/vscode/).

```bash
gh aw compile --watch [--verbose]
```

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and file organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options for workflows
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure output processing including issue updates
- [Tools](/gh-aw/reference/tools/) - GitHub and MCP server configuration
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
