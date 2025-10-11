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
gh aw add githubnext/agentics/ci-doctor

# Add workflow with custom name
gh aw add githubnext/agentics/ci-doctor --name my-custom-doctor

# Add workflow and create pull request for review
gh aw add githubnext/agentics/issue-triage --pr

# Overwrite existing workflow files
gh aw add githubnext/agentics/ci-doctor --force

# Create multiple numbered copies of a workflow
gh aw add githubnext/agentics/ci-doctor --number 3

# Override AI engine for the added workflow
gh aw add githubnext/agentics/ci-doctor --engine copilot
```

**Workflow Removal:**
```bash
# Remove a workflow and its compiled version
gh aw remove WorkflowName

# Remove workflow but keep shared include files
gh aw remove WorkflowName --keep-orphans
```

**Workflow Updates:**

The `update` command allows you to update workflows that were added from external repositories. It uses the `source` field in the workflow frontmatter to determine the source repository and intelligently updates to the latest version.

```bash
# Update all workflows that have a source field
gh aw update

# Update specific workflow by name
gh aw update ci-doctor

# Update multiple workflows
gh aw update ci-doctor issue-triage

# Allow major version updates (when updating tagged releases)
gh aw update ci-doctor --major

# Force update even if no changes detected
gh aw update --force

# Update with verbose output to see detailed resolution steps
gh aw update --verbose

# Override AI engine for the updated workflow compilation
gh aw update ci-doctor --engine copilot
```

**Update Logic:**

The update command intelligently determines how to update based on the current ref in the source field:

- **Semantic Version Tags** (e.g., `v1.2.3`):
  - Fetches the latest compatible release from the repository
  - By default, only updates within the same major version
  - Use `--major` flag to allow major version updates
  - Example: `v1.0.0` → `v1.2.5` (same major), or `v2.0.0` with `--major`

- **Branch References** (e.g., `main`, `develop`):
  - Fetches the latest commit SHA from that specific branch
  - Keeps the branch name in the source field but updates content
  - Example: `main` → latest commit on `main` branch

- **No Reference or Other**:
  - Fetches the latest commit from the repository's default branch
  - Automatically determines the default branch (usually `main` or `master`)

The update process:
1. Parses the source field to extract repository, path, and current ref
2. Resolves the latest compatible version/commit based on the ref type
3. Downloads the base version (original from source) and new version from GitHub
4. Performs a 3-way merge using `git merge-file` to intelligently combine changes:
   - Preserves both local modifications and upstream improvements when possible
   - Detects conflicts when both versions modify the same content
   - Uses diff3-style conflict markers for manual resolution when needed
5. Automatically recompiles the updated workflow (skips compilation if conflicts exist)

**Source Field Format:**

The source field in workflow frontmatter follows this format:
```yaml
source: "owner/repo/path/to/workflow.md@ref"
```

Examples:
- `githubnext/agentics/workflows/ci-doctor.md@v1.0.0` (tag)
- `githubnext/agentics/workflows/ci-doctor.md@main` (branch)
- `githubnext/agentics/workflows/ci-doctor.md` (no ref, uses default branch)

**Merge Behavior and Conflict Resolution:**

The update command uses a 3-way merge algorithm (via `git merge-file`) to intelligently combine changes:

- **Clean Merge**: When local and upstream changes don't overlap, both are automatically preserved
  - Example: Local adds markdown section, upstream adds frontmatter field → both included
  
- **Conflicts**: When both versions modify the same content, conflict markers are added:
  ```yaml
  <<<<<<< current (local changes)
  permissions:
    issues: write
  ||||||| base (original)
  =======
  permissions:
    pull-requests: write
  >>>>>>> new (upstream)
  ```
  
  To resolve conflicts:
  1. Review the conflict markers in the updated workflow file
  2. Manually edit to keep desired changes from both sides
  3. Remove conflict markers (`<<<<<<<`, `|||||||`, `=======`, `>>>>>>>`)
  4. Run `gh aw compile` to recompile the resolved workflow

- **Conflict Notification**: When conflicts occur, the update command displays a warning:
  ```
  ⚠ Updated ci-doctor.md from v1.0.0 to v1.1.0 with CONFLICTS - please review and resolve manually
  ```

## 🔧 Workflow Recompilation

The `compile` command transforms natural language workflow markdown files into executable GitHub Actions YAML files. This is the core functionality that converts your agentic workflow descriptions into automated GitHub workflows.

**Core Compilation:**
```bash
# Compile all workflows in .github/workflows/
gh aw compile

# Compile specific workflows by name or path
gh aw compile ci-doctor
gh aw compile ci-doctor daily-plan
gh aw compile workflow.md

# Compile with detailed output for debugging
gh aw compile --verbose

# Compile with schema validation to catch errors early
gh aw compile --validate

# Validate without generating lock files (dry-run)
gh aw compile --no-emit

# Enable strict mode validation for enhanced security checks
gh aw compile --strict

# Override the AI engine for specific compilation
gh aw compile --engine copilot

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

**Strict Mode Validation:**

The `--strict` flag enables enhanced validation for production workflows, enforcing security and reliability constraints:

```bash
# Compile with strict mode validation
gh aw compile --strict

# Combine strict mode with other flags
gh aw compile --strict --verbose
gh aw compile --strict --no-emit  # Validate without generating files
```

Strict mode enforces the following requirements:
- **Timeout Required**: Workflows must specify `timeout_minutes`
- **Write Permissions Blocked**: Prevents `contents:write`, `issues:write`, `pull-requests:write`
- **Network Configuration Required**: Must explicitly configure network access
- **No Network Wildcards**: Cannot use wildcard `*` in allowed domains
- **MCP Network Configuration**: Custom MCP servers with containers must have network configuration

Workflows can also enable strict mode declaratively using `strict: true` in their frontmatter. The CLI flag takes precedence over frontmatter settings.

## ⚙️ Workflow Operations on GitHub Actions

These commands control the execution and state of your compiled agentic workflows within GitHub Actions.

### Workflow Execution

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

### Trial Mode Execution

Trial mode creates a temporary private repository, installs the specified workflow(s), and runs them in a safe environment that captures outputs without affecting the target repository.

```bash
# Test a workflow from a source repository
gh aw trial githubnext/agentics/weekly-research

# Test a local workflow file
gh aw trial ./my-local-workflow.md

# Test multiple workflows for comparison
gh aw trial githubnext/agentics/daily-plan githubnext/agentics/weekly-research

# Specify target repository context (defaults to current repo)
gh aw trial githubnext/agentics/ci-doctor --logical-repo myorg/myrepo

# Use current repository as the trial host (instead of creating new one)
gh aw trial ./workflow.md --host-repo .

# Clean up trial repository after completion
gh aw trial githubnext/agentics/workflow --delete-host-repo

# Skip confirmation prompts
gh aw trial githubnext/agentics/workflow --yes

# Set custom timeout (default: 30 minutes)
gh aw trial githubnext/agentics/workflow --timeout 60
```

**Issue-Triggered Workflow Testing:**

For workflows that are triggered by issues, you can provide context to simulate the trigger:

```bash
# Test with GitHub issue URL
gh aw trial ./issue-workflow.md --trigger-context https://github.com/owner/repo/issues/123

# Test with issue reference
gh aw trial githubnext/agentics/issue-triage --trigger-context "#456"

# Test with plain issue number
gh aw trial githubnext/agentics/issue-handler --trigger-context "789"
```

**Trial Mode Features:**

Trial mode is particularly useful for:
- **Testing third-party workflows** before installation
- **Local workflow development** - test workflows you're building locally
- **Validating workflow behavior** against your repository structure  
- **Safe experimentation** with workflow modifications
- **Compliance and security reviews** of agentic automation
- **Comparing multiple workflows** side-by-side with identical inputs
- **Issue workflow testing** with realistic trigger context

**Output and Results:**

Trial results are automatically saved in multiple formats:
- **Console output**: Safe outputs displayed immediately 
- **Local files**: Results saved to `trials/` directory with timestamps
- **Trial repository**: Results committed to the trial repo for inspection
- **Artifact downloads**: All workflow artifacts are captured and analyzed

For multiple workflow trials, both individual and combined result files are generated for easy comparison.

> [!TIP]
> Trial mode automatically uses the `GH_AW_GITHUB_TOKEN` environment variable if set, allowing you to override authentication for testing purposes. See the [Security Guide](/gh-aw/guides/security/#authorization-and-token-management) for token management best practices.

### Workflow State Management

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

### Log Analysis and Monitoring

The `logs` command provides comprehensive analysis of workflow execution history, including performance metrics, cost tracking, and error analysis.

**Basic Log Retrieval:**
```bash
# Download logs for all agentic workflows
gh aw logs

# Download logs for a specific workflow
gh aw logs ci-doctor

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
gh aw logs --engine copilot          # Only Copilot workflows

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

# Parse agent logs with JavaScript parser
gh aw logs --parse                   # Run JS parser and write log.md
gh aw logs ci-doctor --parse         # Parse specific workflow logs

# Analyze recent performance with verbose output
gh aw logs ci-doctor -c 5 --verbose
```

**Metrics Included:**
- Execution duration from GitHub API timestamps (CreatedAt, StartedAt, UpdatedAt)  
- AI model token consumption and associated costs
- Success/failure rates and error categorization
- Workflow run frequency and scheduling patterns
- Resource usage and performance trends

**Log Parsing:**

The `--parse` flag runs the engine-specific JavaScript log parser on downloaded agent logs and generates a formatted markdown summary:

```bash
# Parse logs for all downloaded runs
gh aw logs --parse

# Parse logs for specific workflow
gh aw logs ci-doctor --parse --verbose
```

When `--parse` is used:
- Locates agent logs in each downloaded run directory:
  - First checks for files in the `agent_output` artifact directory
  - Falls back to `agent-stdio.log` artifact if `agent_output` doesn't exist
- Automatically selects the appropriate parser based on the engine (Claude, Codex, Copilot)
- Generates a `log.md` file in each run folder with formatted markdown output
- The parser extracts tool calls, reasoning, and other structured information from raw logs
- Uses a minimal Node.js environment that mocks the `@actions/core` API for parser execution

**Output Format:**

The generated `log.md` file contains:
- Tool usage section with formatted command execution details
- Reasoning and thinking sections showing the agent's decision-making process
- Status indicators (✅ success, ⚠️ warnings, ❌ errors) for tool executions
- Structured markdown suitable for review and analysis

Each engine's parser formats the output differently based on the log structure:
- **Claude**: Extracts tool_use and tool_result blocks, interleaved with reasoning text
- **Codex**: Parses thinking sections and command execution logs
- **Copilot**: Formats conversation flow and tool interactions

### Single Run Audit

The `audit` command investigates a single GitHub Actions workflow run and generates a concise markdown report suitable for AI agent consumption. It provides focused, detailed analysis of individual runs with smart permission handling.

**Basic Usage:**
```bash
# Audit a run by numeric ID
gh aw audit 12345678

# Audit using GitHub Actions run URL
gh aw audit https://github.com/owner/repo/actions/runs/12345678

# Audit using GitHub Actions job URL (automatically extracts run ID)
gh aw audit https://github.com/owner/repo/actions/runs/12345678/job/98765432

# Audit with custom output directory
gh aw audit 12345678 -o ./audit-reports

# Audit with verbose output for debugging
gh aw audit 12345678 -v
```

**Smart Permission Handling:**

The `audit` command intelligently handles permission/authentication errors, making it ideal for AI agents working in restricted environments:

1. **Checks local cache first**: Looks for cached artifacts in `logs/run-{id}` before attempting downloads
2. **Detects permission errors**: Automatically identifies GitHub API authentication failures
3. **Provides helpful instructions**: When permissions fail and no cache exists, provides MCP server usage instructions
4. **Processes cached data**: If cache exists but API access fails, automatically uses cached artifacts

**Example workflow for restricted environments:**

```bash
# Step 1: Try to audit (may fail with permission error)
gh aw audit 18167668416

# Output: Instructions to download artifacts using GitHub MCP server
# Use the github-mcp-server tool 'download_workflow_run_artifacts' with:
#   - run_id: 18167668416
#   - output_directory: logs/run-18167668416

# Step 2: Use MCP server to download artifacts (AI agents can learn this)
# (Use GitHub MCP server tool as instructed)

# Step 3: Run audit again to process cached artifacts
gh aw audit 18167668416
# Now processes cached data and generates report
```

**Report Sections:**

The audit command generates a structured markdown report with:

- **Overview**: Run ID, workflow name, status, duration, event type, branch, URL
- **Metrics**: Token usage, estimated cost, turns, errors, warnings
- **MCP Tool Usage**: Table showing tool calls, output sizes, and durations
- **MCP Server Failures**: Lists any servers that failed to initialize
- **Missing Tools**: Reports tools the agent attempted but weren't available
- **Available Artifacts**: Lists downloaded artifacts (aw_info.json, safe_output.jsonl, aw.patch, etc.)

**Benefits:**
- Works in restricted environments without direct GitHub API access
- Teaches AI agents how to download artifacts using MCP tools
- Graceful degradation with limited metadata
- Reusable cache saves time and API calls
- Concise format optimized for AI agent parsing

### MCP Server Management

The `mcp` command provides comprehensive tools for discovering, listing, and inspecting Model Context Protocol (MCP) servers configured in your workflows.

> **📘 Complete MCP Guide**: For comprehensive MCP setup, configuration examples, and troubleshooting, see the [MCPs](/gh-aw/guides/mcps/).
> 
> **🔧 MCP Server**: To run gh-aw as an MCP server exposing CLI tools, see the [MCP Server Guide](/gh-aw/tools/mcp-server/).

### MCP Server Discovery

```bash
# List all workflows that contain MCP server configurations
gh aw mcp list

# List all workflows with detailed MCP server information
gh aw mcp list --verbose

# List MCP servers in a specific workflow
gh aw mcp list workflow-name

# List MCP servers in a specific workflow with detailed configuration
gh aw mcp list workflow-name --verbose
```

### MCP Server Inspection

```bash
# List all workflows that contain MCP server configurations
gh aw mcp inspect

# Inspect all MCP servers in a specific workflow
gh aw mcp inspect workflow-name

# Filter inspection to specific servers by name
gh aw mcp inspect workflow-name --server server-name

# Show detailed information about a specific tool (requires --server)
gh aw mcp inspect workflow-name --server server-name --tool tool-name

# Enable verbose output with connection details
gh aw mcp inspect workflow-name --verbose

# Launch the official @modelcontextprotocol/inspector web interface
gh aw mcp inspect workflow-name --inspector
```

### MCP Tool Listing

```bash
# Find workflows containing a specific MCP server
gh aw mcp list-tools github

# List tools available from a specific MCP server in a workflow
gh aw mcp list-tools github ci-doctor

# List tools with detailed descriptions and allowance status
gh aw mcp list-tools safe-outputs issue-triage --verbose

# List tools from different MCP server types
gh aw mcp list-tools playwright test-workflow
gh aw mcp list-tools custom-server my-workflow
```

### MCP Server Management

The MCP commands help you discover, add, and manage MCP servers from the GitHub MCP registry.

#### Adding MCP Servers from Registry

```bash
# List available MCP servers from the GitHub MCP registry
gh aw mcp add

# Add an MCP server to a workflow from the registry
gh aw mcp add ci-doctor makenotion/notion-mcp-server

# Add MCP server with specific transport preference
gh aw mcp add ci-doctor makenotion/notion-mcp-server --transport stdio

# Add MCP server with custom tool ID
gh aw mcp add ci-doctor makenotion/notion-mcp-server --tool-id my-notion

# Use custom MCP registry
gh aw mcp add ci-doctor server-name --registry https://custom.registry.com/v1
```

**Key Features:**
- **Automatic Configuration**: Uses the modern `mcp-servers:` format in your workflow frontmatter
- **Registry Integration**: Connects to GitHub's MCP registry at `https://api.mcp.github.com/v0` by default
- **Automatic Compilation**: Compiles the workflow after adding the MCP server
- **Transport Selection**: Supports stdio, HTTP, and Docker transports
- **Custom Registries**: Can connect to private or custom MCP registries

**Key Features:**
- **`mcp list`**: Quick overview of MCP servers across workflows with structured table output
- **`mcp inspect`**: Deep inspection with server connection testing and tool capability analysis
- **`mcp list-tools`**: Focused tool listing for specific MCP servers with workflow discovery
- **`mcp add`**: Registry-based MCP server addition with automatic workflow compilation

- Server discovery and connection testing
- Tool and capability inspection
- Focused tool listing for specific MCP servers
- Detailed tool information with `--tool` flag
- Permission analysis
- Multi-protocol support (stdio, Docker, HTTP)
- Web inspector integration
- Registry integration with GitHub's MCP registry (https://api.mcp.github.com/v0)

For detailed MCP debugging and troubleshooting guides, see [MCP Debugging](/gh-aw/guides/mcps/#debugging-and-troubleshooting).

### MCP Server for gh aw

The `mcp-server` command runs gh-aw as a Model Context Protocol (MCP) server, exposing CLI commands as tools that can be called by AI agents and other MCP clients. This enables secure, isolated access to gh-aw functionality.

> **📘 Complete MCP Server Guide**: For comprehensive setup, security architecture, workflow integration examples, and troubleshooting, see the [MCP Server Guide](/gh-aw/tools/mcp-server/).

**Starting the Server:**

```bash
# Start with stdio transport (default) - for local CLI usage
gh aw mcp-server

# Start with HTTP/SSE transport - for workflow integration
gh aw mcp-server --port 3000

# Use custom gh-aw binary path
gh aw mcp-server --cmd ./gh-aw --port 3000
```

**Available Tools:**

The MCP server exposes four core commands as tools:

- `status` - Show workflow file status with pattern filtering
- `compile` - Compile markdown workflows to YAML (validation always enabled)
- `logs` - Download and analyze workflow logs (output forced to `/tmp/gh-aw/aw-mcp/logs`)
- `audit` - Investigate workflow runs (output forced to `/tmp/gh-aw/aw-mcp/logs`)

**Configuration Options:**

- `--port` - Port number for HTTP/SSE transport (uses stdio if not specified)
- `--cmd` - Path to gh-aw command to use (defaults to `gh aw`)

**Workflow Integration:**

Use the shared configuration for easy integration in agentic workflows:

```aw
---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
imports:
  - shared/mcp/gh-aw.md
---

# Workflow content with access to gh-aw MCP tools
```

**Security:**

The MCP server uses a subprocess wrapper architecture where each tool invocation spawns a `gh aw` CLI subprocess. This ensures GitHub tokens and secrets remain isolated from the MCP server process, preventing credential leakage to agentic workflows.

For complete documentation including examples, security details, and troubleshooting, see the [MCP Server Guide](/gh-aw/tools/mcp-server/).

## 👀 Watch Mode for Development
The `--watch` flag provides automatic recompilation during workflow development, monitoring for file changes in real-time. See [Authoring in VS Code](/gh-aw/tools/vscode/).

```bash
# Watch all workflow files in .github/workflows/ for changes
gh aw compile --watch

# Watch with verbose output for detailed compilation feedback
gh aw compile --watch --verbose
```

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and file organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options for workflows
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure output processing including issue updates
- [Tools](/gh-aw/reference/tools/) - GitHub and MCP server configuration
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
