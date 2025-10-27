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
gh aw run daily-perf                        # Execute workflow
gh aw run daily-perf daily-plan             # Execute multiple workflows
gh aw run daily-perf --repeat 10            # Execute workflow 10 times
gh aw logs daily-perf                             # View execution logs
gh aw audit 12345678                             # Audit a specific run
gh aw pr transfer https://github.com/owner/repo/pull/123  # Transfer PR between repositories
```

## Global Flags

- **`--verbose` / `-v`**: Enable verbose output with debugging details
- **`--help` / `-h`**: Show help information

## Debug Logging

Enable detailed debug logs using the `DEBUG` environment variable with pattern matching:

```bash
# Enable all debug logs
DEBUG=* gh aw compile

# Enable specific package logs (e.g., CLI operations)
DEBUG=cli:* gh aw compile

# Enable compiler logs
DEBUG=workflow:* gh aw compile

# Enable multiple packages
DEBUG=cli:*,workflow:* gh aw compile

# Exclude specific patterns
DEBUG=*,-workflow:test gh aw compile

# Disable colors (auto-disabled when piping)
DEBUG_COLORS=0 DEBUG=* gh aw compile
```

Debug logs show:
- **Namespace**: Category of the log (e.g., `cli:compile_command`, `workflow:compiler`)
- **Message**: Debug information
- **Time diff**: Elapsed time since last log (e.g., `+50ms`, `+2.5s`)
- **Colors**: Automatic color coding for each namespace (when in terminal)

**When to use:**
- Use `DEBUG` for internal diagnostic information and performance insights
- Use `--verbose` for user-facing operational details
- Debug logs are zero-overhead when disabled (no performance impact)

## 📝 Workflow Creation and Management

The `add` and `new` commands help you create and manage agentic workflows, from templates and samples to completely custom workflows.

### Repository Initialization

The `init` command prepares your repository for agentic workflows by configuring `.gitattributes` and creating GitHub Copilot custom instructions:

```bash
gh aw init
```

After initialization, start a chat with an AI agent and use the following prompt to create a new workflow:

```
activate @.github/prompts/create-agentic-workflow.prompt.md
```

Alternatively, add pre-built workflows from the catalog using `gh aw add <workflow-name>`.

### Workflow Management

```bash
# Create new workflows
gh aw new my-custom-workflow
gh aw new issue-handler --force

# Add workflows from samples repository
gh aw add githubnext/agentics/ci-doctor
gh aw add githubnext/agentics/ci-doctor --name my-custom-doctor --pr --engine copilot
gh aw add githubnext/agentics/ci-doctor --number 3  # Create 3 copies
gh aw add githubnext/agentics/ci-doctor --append "Extra content"  # Append custom content

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
gh aw compile --validate --no-emit         # Validate schema and containers without generating files
gh aw compile --strict --engine copilot    # Strict mode with custom engine
gh aw compile --purge                      # Remove orphaned .lock.yml files

# Development features
gh aw compile --watch --verbose            # Auto-recompile on changes
gh aw compile --workflows-dir custom/      # Custom workflows directory

# Dependency management
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --dependabot --force         # Force overwrite existing files
```

**Validation:**

The `--validate` flag enables GitHub Actions workflow schema validation and container image validation. By default, validation is disabled for faster compilation. Enable it when you need to verify workflow correctness or validate that container images exist and are accessible.

**Strict Mode:**

Enables enhanced security validation requiring timeouts, explicit network configuration, and blocking write permissions. Use `--strict` flag or `strict: true` in frontmatter.

**Repository Feature Validation:**

The compile command validates that workflows using `create-discussion`, `create-issue`, or `add-comment` with discussions are compatible with the target repository. Compilation fails if:

- Workflows use `create-discussion` but the repository doesn't have discussions enabled
- Workflows use `create-issue` but the repository doesn't have issues enabled

Enable discussions or issues in repository settings, or remove the incompatible safe-outputs from workflows.

**Dependency Manifest Generation:**

The `--dependabot` flag scans workflows for package dependencies and generates manifest files for automated security updates:

- **npm**: Creates `package.json` and `package-lock.json` for packages used with `npx` (requires npm in PATH)
- **pip**: Creates `requirements.txt` for Python packages installed via `pip install` or `pip3 install`
- **Go**: Creates `go.mod` for Go packages installed via `go install` or `go get`

The command creates or updates `.github/dependabot.yml` to enable Dependabot monitoring for all detected ecosystems. Existing manifests are merged intelligently to preserve manual entries. Use `--force` to overwrite the Dependabot configuration file if needed.

```bash
# Scan workflows and generate manifests for detected dependencies
gh aw compile --dependabot

# Force overwrite of existing dependabot.yml configuration
gh aw compile --dependabot --force
```

:::note
The `--dependabot` flag cannot be used with specific workflow files or custom `--workflows-dir`. It processes all workflows in `.github/workflows/`.
:::

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
gh aw trial ./workflow.md --logical-repo myorg/myrepo --host-repo myorg/host-repo # Act as if in a different logical repo. Uses PAT to see issues/PRs
gh aw trial ./workflow.md --clone-repo myorg/myrepo --host-repo myorg/host-repo # Copy the code of the clone repo for into host repo. Agentic will see the codebase of clone repo but not the issues/PRs.
gh aw trial ./workflow.md --append "Extra content"  # Append custom content to workflow

# Test issue-triggered workflows with context
gh aw trial ./issue-workflow.md --trigger-context https://github.com/owner/repo/issues/123
gh aw trial githubnext/agentics/issue-triage --trigger-context "#456"

Other flags:
 --engine ENGINE               # Override engine (default: from frontmatter)
 --auto-merge-prs            # Auto-merge PRs created during trial
 --repeat N                  # Repeat N times
 --force-delete-host-repo-before  # Force delete existing host repo BEFORE start
 --delete-host-repo-after         # Delete host repo AFTER trial
 --append TEXT                # Append extra content to workflow files
```

Trial results are saved to `trials/` directory and captured in the trial repository for inspection. Set `GH_AW_GITHUB_TOKEN` to override authentication. See the [Security Guide](/gh-aw/guides/security/#authorization-and-token-management).

When using `gh aw trial --logical-repo`, the agentic workflow operates as if it is running in the specified logical repository, allowing it to read issues, pull requests, and other repository data from that context. This is useful for testing workflows that interact with repository data without needing to run them in the actual target repository. In this mode, to recompile workflows in the trial repository, use `gh aw compile --trial --logical-repo owner/repo`.

When using `gh aw trial --clone-repo`, the agentic workflow uses the codebase from the specified clone repository while still interacting with issues and pull requests from the host repository. This allows for testing how the workflow would behave with a different codebase while maintaining access to the relevant repository data.

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
gh aw logs --timeout 60                   # Limit execution to 60 seconds
```

**Timeout Option:**

The `--timeout` flag limits log download execution time (in seconds). When the timeout is reached, the command processes already-downloaded runs and returns partial results. Set to `0` for no timeout (default). The MCP server uses a 50-second default timeout automatically.

**Caching and Performance:**

Downloaded runs are cached with a `run_summary.json` file in each run folder. Subsequent `logs` commands reuse cached data for faster reprocessing (~10-100x faster), unless the CLI version changes.

Metrics include execution duration, token consumption, costs, success/failure rates, and resource usage trends.

**Log Parsing and JSON Output:**

- `--parse`: Generates `log.md` and `firewall.md` files with tool calls, reasoning, execution details, and network access patterns extracted by engine-specific parsers
- `--json`: Outputs structured JSON with summary metrics, runs, tool usage, missing tools, MCP failures, access logs, and firewall analysis

### Single Run Audit

Generate concise markdown reports for individual workflow runs with smart permission handling:

```bash
gh aw audit 12345678                                            # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123      # By URL from any repo
gh aw audit https://github.example.com/org/repo/runs/456        # GitHub Enterprise URL
gh aw audit 12345678 -o ./audit-reports -v
gh aw audit 12345678 --parse                                    # Parse logs to markdown
```

**URL Support:**

The audit command accepts workflow run URLs from any repository and GitHub instance:
- Standard URLs: `https://github.com/owner/repo/actions/runs/12345`
- Workflow run URLs: `https://github.com/owner/repo/runs/12345`
- Job URLs: `https://github.com/owner/repo/actions/runs/12345/job/98765`
- GitHub Enterprise: `https://github.example.com/org/repo/actions/runs/99999`

**Options:**

- `--parse`: Generates detailed `log.md` and `firewall.md` files with tool calls, reasoning, and network access patterns extracted by engine-specific parsers

The audit command checks local cache first (`logs/run-{id}`), then attempts download. On permission errors, it provides MCP server instructions for artifact downloads. Reports include overview, metrics, tool usage, MCP failures, firewall analysis, and available artifacts.

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

## 🔄 Repository Utilities

### Pull Request Transfer

Transfer pull requests between repositories, preserving code changes, title, and description:

```bash
# Transfer PR to current repository
gh aw pr transfer https://github.com/source-owner/source-repo/pull/234

# Transfer PR to specific repository
gh aw pr transfer https://github.com/source-owner/source-repo/pull/234 --repo target-owner/target-repo

# Verbose output for debugging
gh aw pr transfer https://github.com/source-owner/source-repo/pull/234 --verbose
```

**How it works:**

1. **Fetches PR details** - Title, body/description, author, and code changes
2. **Creates patch** - Uses `gh pr diff` to generate a unified patch file
3. **Applies changes** - Creates new branch in target repository with squashed commit
4. **Creates PR** - New pull request with original title, description, and attribution

**Requirements:**

- GitHub CLI (`gh`) must be authenticated
- Write access to target repository (for creating branches and PRs)
- Read access to source repository (for fetching PR details)

The transferred PR includes attribution showing the original PR URL and author in the description.

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
