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

## Quick Start

```bash wrap
# Show version and help
gh aw version
gh aw --help

# Basic workflow lifecycle
gh aw init                                       # Initialize repository (first-time setup)
gh aw add githubnext/agentics/ci-doctor    # Add workflow and compile to GitHub Actions
gh aw compile                                    # Recompile to GitHub Actions
gh aw trial githubnext/agentics/ci-doctor  # Test workflow safely before adding
gh aw trial ./my-workflow.md --use-local-secrets  # Test local workflow with local API keys
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

```bash wrap
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

```bash wrap
gh aw init
gh aw init --mcp  # Configure GitHub Copilot Agent MCP integration
```

Prepares the repository by configuring `.gitattributes`, creating Copilot instructions at `.github/instructions/github-agentic-workflows.instructions.md`, and adding the `create-agentic-workflow` custom agent at `.github/agents/`.

With `--mcp`: Creates `.github/workflows/copilot-setup-steps.yml`, `.vscode/mcp.json`, and enables gh-aw MCP server tools (`status`, `compile`, `logs`, `audit`) in Copilot Agent.

After initialization, use `/agent` in Copilot and select `create-agentic-workflow`, or add pre-built workflows with `gh aw add <workflow-name>`.

### Workflow Management

```bash wrap
# Create new workflows
gh aw new my-custom-workflow
gh aw new issue-handler --force

# Add workflows from samples repository
gh aw add githubnext/agentics/ci-doctor
gh aw add githubnext/agentics/ci-doctor --name my-custom-doctor --pr --engine copilot
gh aw add githubnext/agentics/ci-doctor --number 3  # Create 3 copies
gh aw add githubnext/agentics/ci-doctor --append "Extra content"  # Append custom content
gh aw add githubnext/agentics/ci-doctor --no-gitattributes  # Skip .gitattributes update

# Remove workflows
gh aw remove WorkflowName
gh aw remove WorkflowName --keep-orphans  # Keep shared includes
```

**Automatic .gitattributes Configuration:** The `add` command automatically updates `.gitattributes` to mark `.lock.yml` files as generated. Use `--no-gitattributes` to disable.

**Workflow Updates:**

```bash wrap
gh aw update                              # Update all workflows with source field
gh aw update ci-doctor issue-triage      # Update specific workflows
gh aw update ci-doctor --major --force   # Allow major version updates
gh aw update --verbose --engine copilot  # Override engine
```

Updates use the `source` field format `owner/repo/path@ref`. Semantic version tags update within the same major version (use `--major` for major updates). Branch references fetch the latest commit. Performs 3-way merge with `git merge-file`, preserving local changes. Conflicts add diff3-style markers and skip recompilation until resolved.

## 🔧 Workflow Recompilation

Transforms markdown workflow files into executable GitHub Actions YAML files:

```bash wrap
# Core compilation
gh aw compile                              # Compile all workflows
gh aw compile ci-doctor daily-plan         # Compile specific workflows
gh aw compile --validate --no-emit         # Validate schema and containers without generating files
gh aw compile --strict --engine copilot    # Strict mode with custom engine
gh aw compile --purge                      # Remove orphaned .lock.yml files

# Development features
gh aw compile --watch --verbose            # Auto-recompile on changes
gh aw compile --workflows-dir custom/      # Custom workflows directory

# Security scanning
gh aw compile --zizmor                     # Run security scanner on compiled workflows
gh aw compile --strict --zizmor            # Strict mode with security scanning (fails on findings)

# Dependency management
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --dependabot --force         # Force overwrite existing files
```

**Validation:** The `--validate` flag enables schema validation and container image checks. Disabled by default for faster compilation.

**Security Scanning:** The `--zizmor` flag runs the [zizmor](https://github.com/woodruffw/zizmor) security scanner on compiled workflows to identify template injection, dangerous permissions, and other security risks. Findings are warnings by default; use `--strict` to treat them as errors.

**Strict Mode:** Requires timeouts, explicit network configuration, and blocks write permissions. With `--zizmor`, treats security findings as compilation errors requiring zero warnings before deployment. Enable via `--strict` flag or `strict: true` in frontmatter.

**Repository Feature Validation:** Compilation fails if workflows use `create-discussion` or `create-issue` but the target repository lacks those features. Enable them in repository settings or remove incompatible safe-outputs.

**Dependency Manifest Generation:** The `--dependabot` flag scans workflows and generates manifests for npm (`package.json`), pip (`requirements.txt`), and Go (`go.mod`). Creates or updates `.github/dependabot.yml` with intelligent merging. Use `--force` to overwrite configuration. Cannot be used with specific workflows or custom `--workflows-dir`.

## ⚙️ Workflow Operations on GitHub Actions

These commands control the execution and state of your compiled agentic workflows within GitHub Actions.

### Workflow Execution

```bash wrap
gh aw run WorkflowName                      # Run single workflow
gh aw run WorkflowName1 WorkflowName2       # Run multiple workflows
gh aw run WorkflowName --repeat 3           # Run 3 times total
gh aw run workflow --use-local-secrets      # Use local API keys for execution
gh aw run weekly-research --enable-if-needed --input priority=high
```

:::note[Running from GitHub Codespaces]
When using `gh aw run` from a GitHub Codespace, you need to update the codespace's repository permissions to include `actions: write` and `workflows: write`. This allows the codespace to trigger workflow runs.

To update permissions, go to your codespace settings and manage repository access. See [Managing repository access for your codespaces](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces) for detailed instructions.
:::

### Trial Mode

Test workflows safely in a temporary private repository without affecting your target repository:

```bash wrap
gh aw trial githubnext/agentics/ci-doctor  # Test from source repo
gh aw trial ./my-local-workflow.md         # Test local file
gh aw trial workflow1 workflow2            # Compare multiple workflows
gh aw trial ./workflow.md --use-local-secrets  # Use local API keys for trial
gh aw trial ./workflow.md --logical-repo myorg/myrepo --host-repo myorg/host-repo # Act as if in a different logical repo. Uses PAT to see issues/PRs
gh aw trial ./workflow.md --clone-repo myorg/myrepo --host-repo myorg/host-repo # Copy the code of the clone repo for into host repo. Agentic will see the codebase of clone repo but not the issues/PRs.
gh aw trial ./workflow.md --append "Extra content"  # Append custom content to workflow

# Test issue-triggered workflows with context
gh aw trial ./issue-workflow.md --trigger-context https://github.com/owner/repo/issues/123
gh aw trial githubnext/agentics/issue-triage --trigger-context "#456"

Other flags:
 --engine ENGINE               # Override engine (default: from frontmatter)
 --auto-merge-prs            # Auto-merge PRs created during trial
 --use-local-secrets         # Use local environment API keys (pushes/cleans up secrets)
 --repeat N                  # Repeat N times
 --force-delete-host-repo-before  # Force delete existing host repo BEFORE start
 --delete-host-repo-after         # Delete host repo AFTER trial
 --append TEXT                # Append extra content to workflow files
```

Trial results are saved to `trials/` directory and the trial repository. Set `GH_AW_GITHUB_TOKEN` to override authentication. See the [Security Guide](/gh-aw/guides/security/#authorization-and-token-management).

`--logical-repo` makes workflows operate as if running in the specified repository, accessing its issues and PRs. Use `gh aw compile --trial --logical-repo owner/repo` to recompile. `--clone-repo` uses the codebase from the clone repository while interacting with the host repository's issues and PRs.

### Using Local API Keys

```bash wrap
gh aw run my-workflow --use-local-secrets       # Use local API keys for run
gh aw trial ./workflow.md --use-local-secrets   # Use local API keys for trial
```

The `--use-local-secrets` flag temporarily pushes AI engine secrets from environment variables (`CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `COPILOT_GITHUB_TOKEN`) to the repository before execution, then automatically cleans them up. Only pushes secrets needed by the workflow's engine. Useful for testing without permanently storing secrets in the repository. Use with caution in shared environments.

### Workflow State Management

```bash wrap
gh aw status [WorkflowPrefix]               # Show workflow status
gh aw enable [WorkflowPrefix]               # Enable workflows
gh aw disable [WorkflowPrefix]              # Disable and cancel workflows
```

Status shows workflow names, enabled/disabled state, execution status, and compilation status. Enable/disable commands support pattern matching.

### Log Analysis and Monitoring

Download and analyze workflow execution history with performance metrics, cost tracking, and error analysis:

```bash wrap
gh aw logs [workflow-name] -o ./analysis  # Download logs
gh aw logs -c 10 --start-date -1w --end-date -1d
gh aw logs --engine claude --branch main
gh aw logs --after-run-id 1000 --before-run-id 2000
gh aw logs --no-staged --tool-graph       # Exclude staged runs, generate Mermaid graph
gh aw logs --parse --verbose --json       # Parse logs to markdown, output JSON
gh aw logs --timeout 60                   # Limit execution to 60 seconds
```

**Timeout:** `--timeout` limits execution time in seconds. Set to `0` for no timeout (default). MCP server uses 50s timeout automatically.

**Caching:** Downloaded runs are cached with `run_summary.json`. Subsequent `logs` commands reuse cached data (~10-100x faster) unless CLI version changes.

**Output:** `--parse` generates `log.md` and `firewall.md` with tool calls, reasoning, and network patterns. `--json` outputs structured metrics, runs, tool usage, MCP failures, and firewall analysis.

### Single Run Audit

Generate concise markdown reports for individual workflow runs with smart permission handling:

```bash wrap
gh aw audit 12345678                                            # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123      # By URL from any repo
gh aw audit https://github.example.com/org/repo/runs/456        # GitHub Enterprise URL
gh aw audit 12345678 -o ./audit-reports -v
gh aw audit 12345678 --parse                                    # Parse logs to markdown
```

Accepts workflow run URLs from any repository and GitHub instance (standard, workflow run, job URLs, and GitHub Enterprise). The `--parse` flag generates detailed `log.md` and `firewall.md` files. Checks local cache first, then attempts download. Reports include overview, metrics, tool usage, MCP failures, firewall analysis, and artifacts.

### MCP Server Management

Discover, list, inspect, and add Model Context Protocol (MCP) servers. See [MCPs Guide](/gh-aw/guides/mcps/) and [MCP Server Guide](/gh-aw/tools/mcp-server/).

```bash wrap
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

```bash wrap
# Transfer PR to current repository
gh aw pr transfer https://github.com/source-owner/source-repo/pull/234

# Transfer PR to specific repository
gh aw pr transfer https://github.com/source-owner/source-repo/pull/234 --repo target-owner/target-repo

# Verbose output for debugging
gh aw pr transfer https://github.com/source-owner/source-repo/pull/234 --verbose
```

Fetches PR details, creates a patch with `gh pr diff`, applies changes to a new branch in the target repository, and creates a PR with attribution. Requires authenticated `gh` CLI, write access to target repository, and read access to source repository.

### MCP Server for gh aw

Run gh-aw as an MCP server exposing CLI commands (`status`, `compile`, `logs`, `audit`) as tools for AI agents:

```bash wrap
gh aw mcp-server                    # stdio transport (local CLI)
gh aw mcp-server --port 3000        # HTTP/SSE transport (workflows)
gh aw mcp-server --cmd ./gh-aw --port 3000
```

Uses subprocess architecture for token isolation. Import with `shared/mcp/gh-aw.md` in workflows. See [MCP Server Guide](/gh-aw/tools/mcp-server/).

## 👀 Watch Mode for Development

Auto-recompile workflows on file changes. See [Authoring in VS Code](/gh-aw/tools/vscode/).

```bash wrap
gh aw compile --watch [--verbose]
```

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to adding, updating, and importing workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and file organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options for workflows
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure output processing including issue updates
- [Tools](/gh-aw/reference/tools/) - GitHub and MCP server configuration
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
