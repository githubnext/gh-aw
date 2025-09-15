---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
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
gh aw add samples/weekly-research.md -r githubnext/agentics  # Add workflow and compile to GitHub Actions
gh aw compile                                                # Recompile to GitHub Actions
gh aw status                                                 # Check status
gh aw run weekly-research                                    # Execute workflow
gh aw run weekly-research daily-plan                        # Execute multiple workflows
gh aw run weekly-research --repeat 3600                     # Execute workflow every hour
gh aw logs weekly-research                                   # View execution logs
```

## 📝 Workflow Creation and Management  

The `add` and `new` commands help you create and manage agentic workflows, from templates and samples to completely custom workflows.

**Creating New Workflows:**
```bash
# Create a new workflow with comprehensive template
gh aw new my-custom-workflow

# Create a new workflow, overwriting if it exists
gh aw new issue-handler --force
```

**Adding Workflows from Samples:**
```bash
# Add a workflow from the official samples repository
gh aw add samples/weekly-research.md -r githubnext/agentics

# Add workflow and create pull request for review
gh aw add samples/issue-triage.md -r githubnext/agentics --pr

# Add workflow to a specific directory
gh aw add samples/daily-standup.md -r githubnext/agentics --output .github/workflows/
```

**Workflow Removal:**
```bash
# Remove a workflow and its compiled version
gh aw remove WorkflowName

# Remove workflow but keep shared include files
gh aw remove WorkflowName --keep-orphans
```

## 🔧 Workflow Recompilation

The `compile` command transforms natural language workflow markdown files into executable GitHub Actions YAML files. This is the core functionality that converts your agentic workflow descriptions into automated GitHub workflows.

**Core Compilation:**
```bash
# Compile all workflows in .github/workflows/
gh aw compile

# Compile with detailed output for debugging
gh aw compile --verbose

# Compile with schema validation to catch errors early
gh aw compile --validate

# Override the AI engine for specific compilation
gh aw compile --engine codex

# Generate GitHub Copilot instructions file alongside workflows
gh aw compile --instructions

# Compile all workflows and remove orphaned .lock.yml files
gh aw compile --purge
```

**Development Features:**
```bash
# Watch for changes and automatically recompile (ideal for development)
gh aw compile --watch

# Watch with verbose output for detailed compilation feedback
gh aw compile --watch --verbose

# Clean up orphaned lock files from deleted workflows
gh aw compile --purge --verbose
```

### Orphaned File Cleanup

When markdown workflow files (`.md`) are deleted, their corresponding compiled workflow files (`.lock.yml`) remain behind. The `--purge` flag automatically removes these orphaned files:

```bash
# Remove orphaned .lock.yml files during compilation
gh aw compile --purge

# With verbose output to see which files are removed
gh aw compile --purge --verbose
```

## ⚙️ Workflow Operations on GitHub Actions

These commands control the execution and state of your compiled agentic workflows within GitHub Actions.

**Workflow Execution:**
```bash
# Run a single workflow immediately in GitHub Actions
gh aw run WorkflowName

# Run multiple workflows immediately in GitHub Actions
gh aw run WorkflowName1 WorkflowName2 WorkflowName3

# Run workflows and repeat every 3 minutes
gh aw run WorkflowName --repeat 180

# Run workflow with specific input parameters (if supported)
gh aw run weekly-research --input priority=high
```

**Workflow State Management:**
```bash
# Show status of all agentic workflows
gh aw status

# Show status of workflows matching a pattern
gh aw status WorkflowPrefix
gh aw status path/to/workflow.lock.yml

# Enable all agentic workflows for automatic execution
gh aw enable

# Enable specific workflows matching a pattern
gh aw enable WorkflowPrefix
gh aw enable path/to/workflow.lock.yml

# Disable all agentic workflows to prevent execution
gh aw disable

# Disable specific workflows matching a pattern  
gh aw disable WorkflowPrefix
gh aw disable path/to/workflow.lock.yml
```

## 📊 Log Analysis and Monitoring

The `logs` command provides comprehensive analysis of workflow execution history, including performance metrics, cost tracking, and error analysis.

**Basic Log Retrieval:**
```bash
# Download logs for all agentic workflows
gh aw logs

# Download logs for a specific workflow
gh aw logs weekly-research

# Download logs to custom directory for organization
gh aw logs -o ./workflow-analysis
```

**Advanced Filtering and Analysis:**
```bash
# Limit number of runs and filter by date range
gh aw logs -c 10 --start-date 2024-01-01 --end-date 2024-01-31

# Analyze recent performance with verbose output
gh aw logs weekly-research -c 5 --verbose

# Export logs for external analysis tools
gh aw logs --format json -o ./exports/
```

**Metrics Included:**
- Execution duration from GitHub API timestamps (CreatedAt, StartedAt, UpdatedAt)  
- AI model token consumption and associated costs
- Success/failure rates and error categorization
- Workflow run frequency and scheduling patterns
- Resource usage and performance trends

## 🔍 MCP Server Inspection

The `mcp-inspect` command allows you to analyze and troubleshoot Model Context Protocol (MCP) servers configured in your workflows.

> **📘 Complete MCP Guide**: For comprehensive MCP setup, configuration examples, and troubleshooting, see the [MCPs](../guides/mcps/).

```bash
# List all workflows that contain MCP server configurations
gh aw mcp-inspect

# Inspect all MCP servers in a specific workflow
gh aw mcp-inspect workflow-name

# Filter inspection to specific servers by name
gh aw mcp-inspect workflow-name --server server-name

# Show detailed information about a specific tool (requires --server)
gh aw mcp-inspect workflow-name --server server-name --tool tool-name

# Enable verbose output with connection details
gh aw mcp-inspect workflow-name --verbose

# Launch the official @modelcontextprotocol/inspector web interface
gh aw mcp-inspect workflow-name --inspector
```

**Key Features:**
- Server discovery and connection testing
- Tool and capability inspection
- Detailed tool information with `--tool` flag
- Permission analysis
- Multi-protocol support (stdio, Docker, HTTP)
- Web inspector integration

For detailed MCP debugging and troubleshooting guides, see [MCP Debugging](../guides/mcps/#debugging-and-troubleshooting).

## 👀 Watch Mode for Development
The `--watch` flag provides automatic recompilation during workflow development, monitoring for file changes in real-time. See [Authoring in VS Code](../tools/vscode/).

```bash
# Watch all workflow files in .github/workflows/ for changes
gh aw compile --watch

# Watch with verbose output for detailed compilation feedback
gh aw compile --watch --verbose
```

## 📦 Package Management

```bash
# Install workflow packages globally (default)
gh aw install org/repo

# Install packages locally in current project
gh aw install org/repo --local

# Install a specific version, branch, or commit
gh aw install org/repo@v1.0.0
gh aw install org/repo@main --local
gh aw install org/repo@commit-sha

# Uninstall a workflow package globally
gh aw uninstall org/repo

# Uninstall a workflow package locally
gh aw uninstall org/repo --local

# List all installed packages (global and local)
gh aw list --packages

# List only local packages
gh aw list --packages --local

# Uninstall a workflow package globally
gh aw uninstall org/repo

# Uninstall a workflow package locally
gh aw uninstall org/repo --local

# Show version information
gh aw version
```

**Package Management Features:**

- **Install from GitHub**: Download workflow packages from any GitHub repository's `workflows/` directory
- **Version Control**: Specify exact versions, branches, or commits using `@version` syntax
- **Global Storage**: Global packages are stored in `~/.aw/packages/org/repo/` directory structure
- **Local Storage**: Local packages are stored in `.aw/packages/org/repo/` directory structure
- **Flexible Installation**: Choose between global (shared across projects) or local (project-specific) installations

**Package Installation Requirements:**

- GitHub CLI (`gh`) to be installed and authenticated with access to the target repository
- Network access to download from GitHub repositories
- Target repository must have a `workflows/` directory containing `.md` files

**Package Removal:**
```bash
# Uninstall workflow packages globally (default)
gh aw uninstall org/repo

# Uninstall packages locally from current project
gh aw uninstall org/repo --local
```

## Related Documentation

- [Workflow Structure](../reference/workflow-structure/) - Directory layout and file organization
- [Frontmatter Options](../reference/frontmatter/) - Configuration options for workflows
- [Safe Outputs](../reference/safe-outputs/) - Secure output processing including issue updates
- [Tools Configuration](../reference/tools/) - GitHub and MCP server configuration
- [Include Directives](../reference/include-directives/) - Modularizing workflows with includes
