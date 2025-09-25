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

## Global Flags

All `gh aw` commands support the following global flags:

- **`--verbose` / `-v`**: Enable verbose output showing detailed information about operations, including debugging details, step-by-step execution, and additional context
- **`--help` / `-h`**: Show help information for the command

These flags can be used with any command to get more detailed output or help information.

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

# Add multiple workflows at once
gh aw add samples/ci-doctor.md samples/daily-perf-improver.md -r githubnext/agentics

# Add workflow with custom name
gh aw add samples/weekly-research.md -r githubnext/agentics --name my-custom-research

# Add workflow and create pull request for review
gh aw add samples/issue-triage.md -r githubnext/agentics --pr

# Overwrite existing workflow files
gh aw add samples/weekly-research.md --force

# Create multiple numbered copies of a workflow
gh aw add samples/weekly-research.md --number 3

# Override AI engine for the added workflow
gh aw add samples/weekly-research.md --engine codex

# Add workflow from local repository (shortcut for install + add)
gh aw add samples/weekly-research.md -r githubnext/agentics
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

# Compile specific workflows by name or path
gh aw compile weekly-research
gh aw compile weekly-research daily-plan
gh aw compile workflow.md

# Compile with detailed output for debugging
gh aw compile --verbose

# Compile with schema validation to catch errors early
gh aw compile --validate

# Validate without generating lock files (dry-run)
gh aw compile --no-emit

# Override the AI engine for specific compilation
gh aw compile --engine codex

# Generate GitHub Copilot instructions file alongside workflows
gh aw compile --instructions

# Compile all workflows and remove orphaned .lock.yml files
gh aw compile --purge

# Compile from custom workflows directory
gh aw compile --workflows-dir custom/workflows
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

# Run workflow with enable-if-needed flag (enable if disabled, run, restore state)
gh aw run weekly-research --enable-if-needed

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

# Disable all agentic workflows to prevent execution and cancel in-progress runs
gh aw disable

# Disable specific workflows matching a pattern  
gh aw disable WorkflowPrefix
gh aw disable path/to/workflow.lock.yml
```

**Status Information Provided:**
The `status` command shows comprehensive information about your agentic workflows:
- Workflow names and their corresponding GitHub Actions workflow files
- Current enabled/disabled state of each workflow
- Last execution status and timestamp
- Compilation status (whether .md and .lock.yml files are in sync)
- Error information for workflows that failed to compile or execute

**Enable/Disable Behavior:**
- **`enable`**: Activates workflows in GitHub Actions for automatic execution based on their triggers
- **`disable`**: Stops workflows from executing automatically and cancels any currently running workflow instances
- Both commands support pattern matching to operate on multiple workflows at once

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

# Filter by relative time periods using delta syntax
gh aw logs --start-date -1w          # Last week's runs
gh aw logs --end-date -1d            # Up to yesterday
gh aw logs --start-date -1mo         # Last month's runs

# Filter by AI engine type
gh aw logs --engine claude           # Only Claude workflows
gh aw logs --engine codex            # Only Codex workflows

# Filter by branch name
gh aw logs --branch main             # Only runs from main branch
gh aw logs --branch feature-xyz      # Only runs from feature branch

# Filter by run ID range
gh aw logs --after-run-id 1000       # Runs after ID 1000
gh aw logs --before-run-id 2000      # Runs before ID 2000
gh aw logs --after-run-id 1000 --before-run-id 2000  # Runs in range

# Exclude staged workflow runs
gh aw logs --no-staged               # Filter out staged runs

# Generate tool usage analysis
gh aw logs --tool-graph              # Generate Mermaid tool sequence graph

# Analyze recent performance with verbose output
gh aw logs weekly-research -c 5 --verbose
```

**Metrics Included:**
- Execution duration from GitHub API timestamps (CreatedAt, StartedAt, UpdatedAt)  
- AI model token consumption and associated costs
- Success/failure rates and error categorization
- Workflow run frequency and scheduling patterns
- Resource usage and performance trends

## 🔍 MCP Server Management

The `mcp` command provides comprehensive tools for discovering, listing, and inspecting Model Context Protocol (MCP) servers configured in your workflows.

> **📘 Complete MCP Guide**: For comprehensive MCP setup, configuration examples, and troubleshooting, see the [MCPs](/gh-aw/guides/mcps/).

### MCP Server Discovery

**Basic Listing:**
```bash
# List all workflows that contain MCP server configurations
gh aw mcp list

# List with detailed MCP server information (shows server types, commands, allowed tools)
gh aw mcp list --verbose

# List MCP servers in a specific workflow only
gh aw mcp list workflow-name

# List MCP servers in a specific workflow with detailed configuration
gh aw mcp list workflow-name --verbose
```

**Output Details:**
- **Basic mode**: Shows workflow names and MCP server counts
- **Verbose mode**: Includes server names, types (stdio/http/docker), commands/URLs, arguments, allowed tools, and environment variables
- **Single workflow**: Displays detailed table of all MCP servers in that workflow

### MCP Server Inspection

**Deep Analysis and Connection Testing:**
```bash
# List all workflows that have MCP server configurations (same as mcp list)
gh aw mcp inspect

# Inspect and test connections to all MCP servers in a specific workflow
gh aw mcp inspect workflow-name

# Filter inspection to a specific MCP server by name
gh aw mcp inspect workflow-name --server server-name

# Show detailed information about a specific tool (requires --server flag)
gh aw mcp inspect workflow-name --server server-name --tool tool-name

# Enable verbose output with detailed connection logs and debugging info
gh aw mcp inspect workflow-name --verbose

# Launch the official @modelcontextprotocol/inspector web interface for interactive debugging
gh aw mcp inspect workflow-name --inspector
```

**Inspection Features:**
- **Connection Testing**: Attempts to start and connect to each MCP server
- **Capability Discovery**: Lists available tools, resources, and prompts from connected servers
- **Tool Details**: With `--tool` flag, shows parameter schemas, descriptions, and metadata
- **Protocol Support**: Works with stdio, Docker container, and HTTP MCP servers
- **Web Inspector**: Launches browser-based MCP debugging interface for interactive exploration
- **Permission Analysis**: Shows which tools are allowed vs. available

### MCP Server Management

**Adding MCP Servers from Registry:**
```bash
# Show available MCP servers from GitHub's MCP registry
gh aw mcp add

# Add an MCP server to a workflow from the registry
gh aw mcp add weekly-research makenotion/notion-mcp-server

# Add MCP server with specific transport preference (stdio/docker)
gh aw mcp add weekly-research makenotion/notion-mcp-server --transport stdio

# Add MCP server with custom tool ID for the workflow
gh aw mcp add weekly-research makenotion/notion-mcp-server --tool-id my-notion

# Use custom MCP registry instead of default GitHub registry
gh aw mcp add weekly-research server-name --registry https://custom.registry.com/v1
```

The `mcp add` command:
- Searches the MCP registry for servers by name (fuzzy matching)
- Automatically selects the best available transport (stdio preferred over docker)
- Adds the MCP server configuration to the workflow's tools section
- Automatically compiles the workflow to generate the `.lock.yml` file
- Prevents adding duplicate servers to the same workflow

**Serving gh-aw as MCP Server:**
```bash
# Launch MCP server exposing all gh-aw CLI functionality
gh aw mcp serve

# Serve with detailed logging and connection information
gh aw mcp serve --verbose

# Serve with only specific tools enabled (comma-separated list)
gh aw mcp serve --allowed-tools compile,logs,run,status
```

**Available CLI-to-MCP Tools:**
When using `gh aw mcp serve`, the following gh-aw CLI commands become available as MCP tools:
- `compile` - Compile markdown workflow files to YAML
- `logs` - Download and analyze agentic workflow logs
- `mcp_inspect` - Inspect MCP servers and tools in workflows
- `mcp_list` - List MCP server configurations
- `mcp_add` - Add MCP tools to workflows from registry
- `run` - Execute workflows on GitHub Actions
- `enable` - Enable workflow execution
- `disable` - Disable workflow execution
- `status` - Show workflow status

### MCP Server Development

The `mcp serve` command launches an MCP server that exposes gh-aw CLI functionality as MCP tools, making them available to AI assistants and other MCP clients:

**Available MCP Tools:**
- `compile` - Compile markdown workflow files to YAML
- `logs` - Download and analyze agentic workflow logs  
- `mcp_inspect` - Inspect MCP servers and tools
- `mcp_list` - List MCP server configurations
- `mcp_add` - Add MCP tools to workflows
- `run` - Execute workflows on GitHub Actions
- `enable` - Enable workflow execution
- `disable` - Disable workflow execution  
- `status` - Show workflow status

**Key Features:**
- **`mcp list`**: Quick overview of MCP servers across workflows with structured table output
- **`mcp inspect`**: Deep inspection with server connection testing and tool capability analysis
- **`mcp add`**: Registry-based MCP server addition with automatic workflow compilation
- **`mcp serve`**: Expose gh-aw functionality as MCP tools for AI assistants
- Server discovery and connection testing
- Tool and capability inspection
- Detailed tool information with `--tool` flag
- Permission analysis
- Multi-protocol support (stdio, Docker, HTTP)
- Web inspector integration
- Registry integration with GitHub's MCP registry (https://api.mcp.github.com/v0)

For detailed MCP debugging and troubleshooting guides, see [MCP Debugging](/gh-aw/guides/mcps/#debugging-and-troubleshooting).

## 👀 Watch Mode for Development
The `--watch` flag provides automatic recompilation during workflow development, monitoring for file changes in real-time. See [Authoring in VS Code](/gh-aw/tools/vscode/).

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

- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and file organization
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Configuration options for workflows
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure output processing including issue updates
- [Tools Configuration](/gh-aw/reference/tools/) - GitHub and MCP server configuration
- [Include Directives](/gh-aw/reference/include-directives/) - Modularizing workflows with includes
