# 🛠️ Workflow Management Commands

This guide covers all available commands for managing agentic workflows with the GitHub CLI extension.

## Basic Commands

```bash
# List all available natural language action workflows
gh aw list

# List installed workflow packages
gh aw list --packages

# Create a new workflow from template
gh aw new my-workflow

# Create a new workflow with force overwrite
gh aw new issue-handler --force

# Add a new workflow from a sample
gh aw add samples/weekly-research.md -r githubnext/agentics

# Compile existing workflows after modifying the markdown file
gh aw compile

# Compile workflows with verbose output
gh aw compile --verbose

# Compile workflows with schema validation
gh aw compile --validate

# Compile workflows with engine override
gh aw compile --engine codex

# Generate GitHub Copilot instructions file
gh aw compile --instructions

# Compile workflows and generate auto-compile workflow for automatic compilation
gh aw compile --auto-compile

# Compile workflows with both verbose output and auto-compile workflow generation
gh aw compile --verbose --auto-compile

# Watch for changes and automatically recompile workflows
gh aw compile --watch

# Watch with verbose output for development
gh aw compile --watch --verbose

# Watch with auto-compile workflow generation
gh aw compile --watch --auto-compile --verbose

# Remove a workflow
gh aw remove WorkflowName

# Remove a workflow without removing orphaned include files
gh aw remove WorkflowName --keep-orphans

# Run a workflow immediately
gh aw run WorkflowName

# Show status of all natural language action workflows
gh aw status

# Show status of workflows matching a pattern
gh aw status WorkflowPrefix
gh aw status path/to/workflow.lock.yml

# Disable all natural language action workflows
gh aw disable

# Disable workflows matching a pattern
gh aw disable WorkflowPrefix
gh aw disable path/to/workflow.lock.yml

# Enable all natural language action workflows
gh aw enable

# Enable workflows matching a pattern
gh aw enable WorkflowPrefix
gh aw enable path/to/workflow.lock.yml

# Download and analyze workflow logs
gh aw logs

# Download logs for specific workflow
gh aw logs weekly-research

# Download last 10 runs with date filtering
gh aw logs -c 10 --start-date 2024-01-01 --end-date 2024-01-31

# Download logs to custom directory
gh aw logs -o ./my-logs

# Inspect MCP servers in workflows
gh aw mcp-inspect

# Inspect MCP servers in a specific workflow
gh aw mcp-inspect weekly-research

# Inspect only specific MCP servers
gh aw mcp-inspect weekly-research --server repo-mind

# Verbose inspection with connection details
gh aw mcp-inspect weekly-research -v

# Launch the official MCP inspector tool
gh aw mcp-inspect weekly-research --inspector
```

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

# Auto-compile workflow features:
# - Triggers when .github/workflows/*.md files are modified
# - Automatically compiles markdown files to .lock.yml files
# - Commits and pushes the compiled workflow files
# - Uses locally built gh-aw extension for development workflows
```

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

## �️ Package Removal

```bash
# Uninstall workflow packages globally (default)
gh aw uninstall org/repo

# Uninstall packages locally from current project
gh aw uninstall org/repo --local
```

**Package Removal Features:**

- **Clean Uninstall**: Removes all downloaded workflow files and metadata
- **Global vs Local**: Supports both global (`~/.aw/packages/`) and local (`.aw/packages/`) removal
- **Safe Operation**: Only removes the specified package without affecting others
- **Validation**: Confirms package exists before attempting removal

## �📝 Creating New Workflows

The `gh aw new` command creates a new workflow markdown file with comprehensive template content and examples.

```bash
# Create a new workflow with example configuration
gh aw new my-custom-workflow

# Create a new workflow, overwriting if it exists
gh aw new issue-handler --force
```

**New Workflow Features:**

- **Template Generation**: Creates a comprehensive markdown file with commented examples
- **All Options Covered**: Includes examples of all trigger types, permissions, tools, and frontmatter options
- **Ready to Use**: Generated file serves as both documentation and working example
- **Customizable**: Easy to modify for specific use cases

**Generated Template Includes:**
- Complete frontmatter examples with all available options
- Trigger event configurations (issues, pull requests, schedule, etc.)
- Permissions and security settings
- AI processor configurations (Claude, Codex)
- Tools configuration (GitHub, MCP servers, etc.)
- Example workflow instructions

## 📊 Workflow Logs and Analysis

The `gh aw logs` command downloads and analyzes workflow execution logs with aggregated metrics and cost analysis.

```bash
# Download logs for all agentic workflows in the repository
gh aw logs

# Download logs for a specific workflow
gh aw logs weekly-research

# Limit the number of runs and filter by date
gh aw logs -c 10 --start-date 2024-01-01 --end-date 2024-01-31

# Download to custom directory
gh aw logs -o ./workflow-logs
```

**Workflow Logs Features:**

- **Automated Download**: Downloads logs and artifacts from GitHub Actions
- **Metrics Analysis**: Extracts token usage and cost information from log files
- **GitHub API Timing**: Uses GitHub API timestamps for accurate duration calculation
- **Aggregated Reporting**: Provides summary statistics across multiple runs
- **Flexible Filtering**: Filter by date range and limit number of runs
- **Cost Tracking**: Analyzes AI model usage costs when available
- **Custom Output**: Specify custom output directory for organized storage

**Log Analysis Includes:**
- Execution duration from GitHub API timestamps (CreatedAt, StartedAt, UpdatedAt)
- AI model token consumption and costs extracted from engine-specific logs
- Success/failure rates and error patterns
- Workflow run frequency and patterns
- Artifact and log file organization

## Related Documentation

- [Workflow Structure](workflow-structure.md) - Directory layout and file organization
- [Frontmatter Options](frontmatter.md) - Configuration options for workflows
- [Tools Configuration](tools.md) - GitHub and MCP server configuration
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
