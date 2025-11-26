---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
sidebar:
  order: 200
---

The `gh aw` CLI extension enables developers to create, manage, and execute AI-powered workflows directly from the command line. It transforms natural language markdown files into GitHub Actions.

## 🚀 Most Common Commands

95% of users only need these 5 commands:

:::tip[New to gh-aw?]
Start here! These commands cover the essential workflow lifecycle from setup to monitoring.
:::

| Command | When to Use | Details |
|---------|-------------|---------|
| **`gh aw init`** | Set up your repository for agentic workflows | [→ Documentation](#init) |
| **`gh aw add (workflow)`** | Add workflows from The Agentics collection | [→ Documentation](#add) |
| **`gh aw status`** | Check current state of all workflows | [→ Documentation](#status) |
| **`gh aw compile`** | Convert markdown to GitHub Actions YAML | [→ Documentation](#compile) |
| **`gh aw run (workflow)`** | Execute workflows immediately in GitHub Actions | [→ Documentation](#run) |

**Complete command reference below** ↓



## Installation

Install the GitHub CLI extension:

```bash wrap
gh extension install githubnext/gh-aw
```

### Alternative: Standalone Installer

If the extension installation fails (common in Codespaces outside the githubnext organization or when authentication issues occur), use the standalone installer:

```bash wrap
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash
```

After standalone installation, the binary is available as `./gh-aw` in the current directory. To use it globally:

```bash wrap
sudo mv gh-aw /usr/local/bin/
```

:::note[Command differences]
When using the standalone binary, run commands as `./gh-aw` or `gh-aw` (if moved to PATH) instead of `gh aw`. For example:
- Extension: `gh aw compile`
- Standalone: `./gh-aw compile`
:::

**GitHub Enterprise Server:** Set `GITHUB_SERVER_URL` or `GH_HOST` environment variables to use your GitHub instance.

## Global Options

| Flag | Description |
|------|-------------|
| `-h`, `--help` | Show help information |
| `-v`, `--verbose` | Enable verbose output with debugging details |

**Help Commands:**
```bash wrap
gh aw --help           # Show all commands
gh aw help [command]   # Command-specific help
gh aw help all         # Comprehensive documentation
```

## Commands

Commands are organized by workflow lifecycle: creating, building, testing, monitoring, and managing workflows.

### Getting Workflows

#### `init`

Initialize your repository for agentic workflows.

```bash wrap
gh aw init       # Configure .gitattributes, Copilot instructions, custom agent
gh aw init --mcp # Also setup MCP server integration for Copilot Agent
```

Configures `.gitattributes` to mark `.lock.yml` files as generated, adds Copilot instructions for better AI assistance, and sets up custom agent configuration. The `--mcp` flag additionally creates GitHub Actions workflow for MCP server setup, configures `.vscode/mcp.json` for VS Code integration, and enables gh-aw MCP tools in Copilot Agent.

#### `add`

Add workflows from The Agentics collection or other repositories. Displays the workflow description (from frontmatter `description` field) to provide context.

```bash wrap
gh aw add githubnext/agentics/ci-doctor           # Add single workflow
gh aw add "githubnext/agentics/ci-*"             # Add multiple with wildcards
gh aw add ci-doctor --dir shared --number 3      # Organize and create copies
gh aw add ci-doctor --create-pull-request        # Create PR instead of direct commit
```

**Options:** `--dir` (organize in subdirectories), `--number` (create numbered copies), `--create-pull-request` or `--pr` (create pull request), `--no-gitattributes` (skip `.gitattributes` update)

#### `new`

Create a new custom workflow from scratch.

```bash wrap
gh aw new my-custom-workflow
```

Creates a markdown workflow file in `.github/workflows/` with template frontmatter and automatically opens it for editing.

### Building

#### `compile`

Compile markdown workflows to GitHub Actions YAML. Remote imports are automatically cached in `.github/aw/imports/` for offline compilation.

```bash wrap
gh aw compile                              # Compile all workflows
gh aw compile my-workflow                  # Compile specific workflow
gh aw compile --watch                      # Auto-recompile on changes
gh aw compile --validate --strict          # Schema + strict mode validation
gh aw compile --validate --json            # Validation with JSON output
gh aw compile --zizmor                     # Security scan (warnings)
gh aw compile --strict --zizmor            # Security scan (fails on findings)
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --purge                      # Remove orphaned .lock.yml files
```

**Options:** `--validate` (schema validation and container checks), `--strict` (strict mode validation for all workflows), `--zizmor` (security scanning with [zizmor](https://github.com/woodruffw/zizmor)), `--dependabot` (generate npm/pip/Go manifests and update dependabot.yml), `--json` (machine-readable JSON output), `--watch` (auto-recompile on changes), `--purge` (remove orphaned `.lock.yml` files)

**Strict Mode (`--strict`):**

Enhanced security validation for production workflows. Enforces: (1) no write permissions - use [safe-outputs](/gh-aw/reference/safe-outputs/) instead, (2) explicit `network` configuration required, (3) no wildcard `*` in `network.allowed` domains, (4) network configuration required for custom MCP servers with containers, (5) GitHub Actions pinned to commit SHAs, (6) no deprecated frontmatter fields. The CLI flag applies to all workflows and takes precedence over individual workflow `strict` frontmatter fields.

**Example:**
```bash wrap
gh aw compile --strict                 # Enable strict mode for all workflows
gh aw compile --strict --zizmor        # Strict mode with security scanning
gh aw compile --validate --strict      # Validate schema and enforce strict mode
```

See [Strict Mode reference](/gh-aw/reference/frontmatter/#strict-mode-strict) for frontmatter configuration and [Security Guide](/gh-aw/guides/security/#strict-mode-validation) for best practices.

### Testing

#### `trial`

Test workflows safely in temporary private repositories or run directly in a specified repository. Displays workflow description from frontmatter.

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test remote workflow
gh aw trial ./workflow.md                          # Test local workflow
gh aw trial ./workflow.md --use-local-secrets      # Test with local API keys
gh aw trial ./workflow.md --logical-repo owner/repo # Act as different repo
gh aw trial ./issue-workflow.md --trigger-context "#123" # With issue context
gh aw trial ./workflow.md --repo owner/repo        # Run directly in repository
```

**Options:** `-e, --engine` (override AI engine), `--auto-merge-prs` (auto-merge created PRs), `--repeat N` (repeat N times), `--delete-host-repo-after` (delete trial repository), `--use-local-secrets` (use local API keys), `--logical-repo owner/repo` (access issues/PRs from specific repository), `--clone-repo owner/repo` (use different codebase), `--trigger-context "#123"` (provide issue/PR context), `--repo owner/repo` (install and run directly without waiting)

**Trial Modes:** Default creates temporary private repository for safe testing. Direct repository mode (`--repo`) installs workflow in specified repository and executes immediately. Results saved to `trials/` directory.

#### `run`

Execute workflows immediately in GitHub Actions. After triggering, displays workflow URL and suggests using `gh aw audit` to analyze the run.

```bash wrap
gh aw run workflow-name                     # Run workflow
gh aw run workflow1 workflow2               # Run multiple workflows
gh aw run workflow --repeat 3               # Repeat execution 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
```

**Options:** `--repeat N` (execute N times), `--use-local-secrets` (temporarily push AI engine secrets from environment variables, then clean up)

:::note[Codespaces]
From GitHub Codespaces, grant `actions: write` and `workflows: write` permissions. See [Managing repository access](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces).
:::

### Monitoring

#### `status`

Show status of all workflows in the repository.

```bash wrap
gh aw status                                # Show all workflow status
gh aw status --ref main                     # Show status with latest run info for main branch
gh aw status --json --ref feature-branch    # JSON output with run status for specific branch
```

Lists all agentic workflows with their current state, enabled/disabled status, schedules, and configurations. When `--ref` is specified, includes the latest run status and conclusion for each workflow on that branch or tag.

**Options:** `--ref` (filter by branch or tag, shows latest run status and conclusion), `--json` (output in JSON format)

#### `logs`

Download and analyze workflow execution logs. Downloads logs, analyzes tool usage and network patterns, and caches results for faster subsequent runs (~10-100x speedup). Overview table includes errors, warnings, missing tools, and noop messages.

```bash wrap
gh aw logs                                 # Download logs for all workflows
gh aw logs workflow-name                   # Download logs for specific workflow
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --ref main                      # Filter logs by branch or tag
gh aw logs --ref v1.0.0 --parse --json    # Generate markdown + JSON output for specific tag
gh aw logs workflow-name --repo owner/repo # Download logs from specific repository
```

**Options:** `-c, --count N` (limit number of runs), `-e, --engine` (filter by AI engine like `-e copilot`), `--start-date` (filter runs from date like `--start-date -1w`), `--end-date` (filter runs until date like `--end-date -1d`), `--ref` (filter by branch or tag like `--ref main` or `--ref v1.0.0`), `--parse` (generate `log.md` and `firewall.md`), `--json` (output structured metrics), `--repo owner/repo` (download logs from specific repository)

#### `audit`

Investigate and analyze specific workflow runs. Provides detailed analysis including overview, execution metrics, tool usage patterns, MCP server failures, firewall analysis, noop messages, and artifact information. Accepts run IDs or URLs from any repository and GitHub instance. JSON output includes parsed noop messages similar to missing-tool reports. Automatically detects GitHub Copilot agent runs and uses specialized log parsing to extract agent-specific metrics including turns, tool calls, errors, and token usage.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By URL
gh aw audit 12345678 --parse                              # Parse logs to markdown
```

### Management

#### `enable`

Enable workflows for execution with pattern matching support for bulk operations.

```bash wrap
gh aw enable                                # Enable all workflows
gh aw enable prefix                         # Enable workflows matching prefix
gh aw enable ci-*                          # Enable workflows with pattern
gh aw enable workflow-name --repo owner/repo # Enable in specific repository
```

**Options:** `--repo owner/repo` (enable workflows in specific repository, defaults to current)

#### `disable`

Disable workflows to prevent execution and cancel any currently running workflow jobs.

```bash wrap
gh aw disable                               # Disable all workflows
gh aw disable prefix                        # Disable workflows matching prefix
gh aw disable ci-*                         # Disable workflows with pattern
gh aw disable workflow-name --repo owner/repo # Disable in specific repository
```

**Options:** `--repo owner/repo` (disable workflows in specific repository, defaults to current)

#### `remove`

Remove workflows from the repository.

```bash wrap
gh aw remove my-workflow
```

Removes both `.md` and `.lock.yml` files and updates repository configuration.

#### `update`

Update workflows to their latest versions. Updates workflows based on `source` field format `owner/repo/path@ref`. Replaces local file with upstream version (default) or performs 3-way merge to preserve local changes. Semantic version tags update within the same major version. Falls back to git commands when GitHub API authentication fails. Works with public repositories without requiring GitHub authentication.

```bash wrap
gh aw update                              # Update all workflows with source field
gh aw update ci-doctor                    # Update specific workflow
gh aw update ci-doctor --merge            # Update with 3-way merge (preserve changes)
gh aw update ci-doctor --major --force    # Allow major version updates
gh aw update --dir custom/workflows       # Update workflows in custom directory
```

**Options:** `--dir` (specify workflow directory, defaults to `.github/workflows`), `--merge` (3-way merge to preserve local changes, creates conflict markers if needed), `--major` (allow major version updates/breaking changes), `--force` (force update even with conflicts)

**Update Modes:** Default replaces local file with latest upstream version (no conflicts). `--merge` performs 3-way merge preserving local changes (may create conflict markers). When conflicts occur with `--merge`, diff3 markers are added and recompilation is skipped until resolved.

### Advanced

#### `mcp`

Manage MCP (Model Context Protocol) servers. Lists MCP servers configured in workflows, inspects server configuration, tests connectivity, and adds new servers from the registry.

```bash wrap
gh aw mcp list                             # List all MCP servers
gh aw mcp list workflow-name               # List servers for specific workflow
gh aw mcp list-tools <mcp-server>          # List tools for specific MCP server
gh aw mcp list-tools <mcp-server> workflow # List tools in specific workflow
gh aw mcp inspect workflow-name            # Inspect and test servers
gh aw mcp add                              # Add servers from registry
```

See **[MCPs Guide](/gh-aw/guides/mcps/)** for complete documentation.

#### `pr`

Pull request management utilities.

**Subcommands:**

##### `pr transfer`

Transfer a pull request to another repository, preserving code changes, title, and description.

```bash wrap
gh aw pr transfer https://github.com/source/repo/pull/234
gh aw pr transfer <pr-url> --repo target-owner/target-repo
```

**Options:** `--repo target-owner/target-repo` (specify target repository, defaults to current)

#### `mcp-server`

Start MCP server exposing CLI commands as tools (`status`, `compile`, `logs`, `audit`). Enables AI assistants to interact with gh-aw programmatically. Supports both local (stdio) and remote (HTTP) transports.

```bash wrap
gh aw mcp-server              # stdio transport (local)
gh aw mcp-server --port 3000  # HTTP/SSE transport (workflows)
```

**Options:** `--port N` (start HTTP server on specified port, defaults to stdio transport)

See **[MCP Server Guide](/gh-aw/setup/mcp-server/)** for integration details.

### Utility Commands

#### `version`

Show version information for the gh-aw CLI.

```bash wrap
gh aw version
```

Displays the current version of gh-aw and product information. Equivalent to using the `--version` flag.



## Debug Logging

Enable detailed debugging output for troubleshooting. Shows namespace, message, and time diff (e.g., `+50ms`). Zero overhead when disabled. Supports pattern matching with wildcards.

```bash wrap
DEBUG=* gh aw compile                # All debug logs
DEBUG=cli:* gh aw compile            # CLI operations only
DEBUG=cli:*,workflow:* gh aw compile # Multiple packages
DEBUG=*,-tests gh aw compile         # All except tests
```

**Tip:** Use `--verbose` flag for user-facing details instead of DEBUG environment variable.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `command not found: gh` | Install from [cli.github.com](https://cli.github.com/) |
| `extension not found: aw` | Run `gh extension install githubnext/gh-aw` |
| Compilation fails with YAML errors | Check indentation, colons, and array syntax in frontmatter |
| Workflow not found | Run `gh aw status` to list available workflows |
| Permission denied | Check file permissions or repository access |
| Trial creation fails | Check GitHub rate limits and authentication |

See [Common Issues](/gh-aw/troubleshooting/common-issues/) and [Error Reference](/gh-aw/troubleshooting/errors/) for detailed troubleshooting.

## Related Documentation

- [Quick Start](/gh-aw/setup/quick-start/) - Get your first workflow running
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options
- [Packaging & Distribution](/gh-aw/guides/packaging-imports/) - Adding and updating workflows
- [Security Guide](/gh-aw/guides/security/) - Security best practices
- [VS Code Setup](/gh-aw/setup/vscode/) - Editor integration and watch mode
- [MCP Server Guide](/gh-aw/setup/mcp-server/) - MCP server configuration
- [Labs](/gh-aw/labs/) - Experimental workflows
