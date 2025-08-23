# 🛠️ Workflow Management Commands

This guide covers all available commands for managing agentic workflows with the GitHub CLI extension.

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
```

**Development Features:**
```bash
# Watch for changes and automatically recompile (ideal for development)
gh aw compile --watch

# Watch with verbose output for detailed compilation feedback
gh aw compile --watch --verbose

# Generate auto-compile workflow for automatic CI/CD integration
gh aw compile --auto-compile

# Combined development setup: watch + auto-compile + verbose
gh aw compile --watch --auto-compile --verbose
```

**Compilation Process:**
- Parses markdown frontmatter for workflow configuration
- Converts natural language instructions to GitHub Actions steps
- Validates YAML syntax and GitHub Actions schema (with `--validate`)
- Generates `.lock.yml` files ready for GitHub Actions execution
- Integrates with AI engines (Claude, Codex) for instruction processing

**Creation Features:**
- **Template Generation**: `new` creates comprehensive markdown with all configuration options
- **Sample Integration**: `add` pulls proven workflows from community repositories  
- **Pull Request Workflow**: Automatic PR creation for team review processes
- **Flexible Output**: Control where workflows are created in your repository
- **Include Management**: Smart handling of shared workflow components

## ⚙️ Workflow Operations on GitHub Actions

These commands control the execution and state of your compiled agentic workflows within GitHub Actions.

**Workflow Execution:**
```bash
# Run a workflow immediately in GitHub Actions
gh aw run WorkflowName

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

**Operational Features:**
- **Immediate Execution**: `run` triggers workflows outside their normal schedule
- **Bulk Operations**: Enable/disable multiple workflows with pattern matching
- **Status Monitoring**: View which workflows are active, disabled, or have errors
- **Pattern Matching**: Use prefixes or file paths to target specific workflows
- **GitHub Actions Integration**: Direct integration with GitHub's workflow execution engine

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

**Log Analysis Features:**
- **Automated Download**: Retrieves logs and artifacts from GitHub Actions API
- **Performance Metrics**: Extracts execution duration, token usage, and timing data
- **Cost Analysis**: Calculates AI model usage costs when available in logs
- **Error Pattern Detection**: Identifies common failure modes and error patterns
- **Aggregated Reporting**: Provides summary statistics across multiple workflow runs
- **Flexible Export**: Multiple output formats for integration with analysis tools

**Metrics Included:**
- Execution duration from GitHub API timestamps (CreatedAt, StartedAt, UpdatedAt)  
- AI model token consumption and associated costs
- Success/failure rates and error categorization
- Workflow run frequency and scheduling patterns
- Resource usage and performance trends

## 🔍 MCP Server Inspection

The `mcp-inspect` command allows you to analyze and troubleshoot Model Context Protocol (MCP) servers configured in your workflows.

> **📘 Complete MCP Guide**: For comprehensive MCP setup, configuration examples, and troubleshooting, see the [MCPs](mcps.md).

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

For detailed MCP debugging and troubleshooting guides, see [MCP Debugging](mcps.md#debugging-and-troubleshooting).

## 🔄 Auto-Compile Workflow Management

The `--auto-compile` flag enables automatic compilation of agentic workflows when markdown files change.

```bash
# Generate auto-compile workflow that triggers on markdown file changes
gh aw compile --auto-compile
```

Auto-compile workflow features:
- Triggers when .github/workflows/*.md files are modified
- Automatically compiles markdown files to .lock.yml files
- Commits and pushes the compiled workflow files
- Uses locally built gh-aw extension for development workflows

## 👀 Watch Mode for Development
The `--watch` flag provides automatic recompilation during workflow development, monitoring for file changes in real-time. See [Authoring in Visual Studio Code](./vscode.md).

```bash
# Watch all workflow files in .github/workflows/ for changes
gh aw compile --watch

# Watch with verbose output for detailed compilation feedback
gh aw compile --watch --verbose

# Watch with auto-compile workflow generation
gh aw compile --watch --auto-compile --verbose
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

- [Workflow Structure](workflow-structure.md) - Directory layout and file organization
- [Frontmatter Options](frontmatter.md) - Configuration options for workflows
- [Tools Configuration](tools.md) - GitHub and MCP server configuration
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
