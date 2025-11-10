---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
sidebar:
  order: 200
---

The `gh aw` CLI extension enables developers to create, manage, and execute AI-powered workflows directly from the command line. It transforms natural language markdown files into GitHub Actions.

## Quick Start

Get started in seconds:

```bash wrap
gh aw init                                  # Initialize repository
gh aw add githubnext/agentics/ci-doctor    # Add a workflow
gh aw trial ci-doctor                       # Test safely
```

**Common Tasks:**

| Task | Command |
|------|---------|
| Add workflow | `gh aw add githubnext/agentics/ci-doctor` |
| Create custom workflow | `gh aw new my-workflow` |
| Compile to YAML | `gh aw compile` |
| Test safely | `gh aw trial ./workflow.md` |
| Run immediately | `gh aw run workflow-name` |
| Check status | `gh aw status` |
| View logs | `gh aw logs workflow-name` |
| Debug run | `gh aw audit 12345678` |

## Installation

Install the GitHub CLI extension:

```bash wrap
gh extension install githubnext/gh-aw
```

**Alternative:** If authentication fails, use the standalone installer:

```bash wrap
curl -O https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh
chmod +x install-gh-aw.sh
./install-gh-aw.sh
```

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

**What it does:**
- Configures `.gitattributes` to mark `.lock.yml` files as generated
- Adds Copilot instructions for better AI assistance
- Sets up custom agent configuration

**With `--mcp` flag:**
- Creates GitHub Actions workflow for MCP server setup
- Configures `.vscode/mcp.json` for VS Code integration
- Enables gh-aw MCP tools in Copilot Agent

#### `add`

Add workflows from The Agentics collection or other repositories.

```bash wrap
gh aw add githubnext/agentics/ci-doctor           # Add single workflow
gh aw add "githubnext/agentics/ci-*"             # Add multiple with wildcards
gh aw add ci-doctor --dir shared --number 3      # Organize and create copies
gh aw add ci-doctor --pr                         # Create PR instead of direct commit
```

**Options:**
- `--dir`: Organize workflows in subdirectories
- `--number`: Create multiple numbered copies
- `--pr`: Create pull request instead of committing directly
- `--no-gitattributes`: Skip `.gitattributes` update

#### `new`

Create a new custom workflow from scratch.

```bash wrap
gh aw new my-custom-workflow
```

Creates a markdown workflow file in `.github/workflows/` with template frontmatter and automatically opens it for editing.

### Building

#### `compile`

Compile markdown workflows to GitHub Actions YAML.

```bash wrap
gh aw compile                              # Compile all workflows
gh aw compile my-workflow                  # Compile specific workflow
gh aw compile --watch                      # Auto-recompile on changes
gh aw compile --validate --strict          # Schema + strict mode validation
gh aw compile --zizmor                     # Security scan (warnings)
gh aw compile --strict --zizmor            # Security scan (fails on findings)
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --purge                      # Remove orphaned .lock.yml files
```

**Key Options:**

| Option | Description |
|--------|-------------|
| `--validate` | Schema validation and container checks |
| `--strict` | Requires timeouts, explicit network config, blocks write permissions |
| `--zizmor` | Security scanning with [zizmor](https://github.com/woodruffw/zizmor) |
| `--dependabot` | Generate npm/pip/Go manifests and update dependabot.yml |
| `--watch` | Auto-recompile on file changes |
| `--purge` | Remove orphaned `.lock.yml` files |

### Testing

#### `trial`

Test workflows safely in temporary private repositories.

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test remote workflow
gh aw trial ./workflow.md                          # Test local workflow
gh aw trial ./workflow.md --use-local-secrets      # Test with local API keys
gh aw trial ./workflow.md --logical-repo owner/repo # Act as different repo
gh aw trial ./issue-workflow.md --trigger-context "#123" # With issue context
```

**Key Options:**

| Option | Description |
|--------|-------------|
| `--engine` | Override AI engine for testing |
| `--auto-merge-prs` | Automatically merge created PRs |
| `--repeat N` | Repeat execution N times |
| `--delete-host-repo-after` | Delete trial repository after execution |
| `--use-local-secrets` | Use local API keys instead of repository secrets |
| `--logical-repo owner/repo` | Access issues/PRs from specific repository |
| `--clone-repo owner/repo` | Use different codebase for testing |
| `--trigger-context "#123"` | Provide issue/PR context |

Creates temporary private repository, executes workflow in isolated environment, and saves results to `trials/` directory.

#### `run`

Execute workflows immediately in GitHub Actions.

```bash wrap
gh aw run WorkflowName                      # Run workflow
gh aw run workflow1 workflow2               # Run multiple workflows
gh aw run workflow --repeat 3               # Repeat execution 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
```

**Options:**
- `--repeat N`: Execute the workflow N times
- `--use-local-secrets`: Temporarily push AI engine secrets from environment variables, then clean up

:::note[Codespaces]
From GitHub Codespaces, grant `actions: write` and `workflows: write` permissions. See [Managing repository access](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces).
:::

### Monitoring

#### `status`

Show status of all workflows in the repository.

```bash wrap
gh aw status                                # Show all workflow status
```

Lists all agentic workflows with their current state, enabled/disabled status, schedules, and configurations.

#### `logs`

Download and analyze workflow execution logs.

```bash wrap
gh aw logs                                 # Download logs for all workflows
gh aw logs workflow-name                   # Download logs for specific workflow
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --parse --json                 # Generate markdown + JSON output
```

**Key Options:**

| Option | Description | Example |
|--------|-------------|---------|
| `-c, --count N` | Limit number of runs | `-c 10` |
| `--start-date` | Filter runs from date | `--start-date -1w` |
| `--end-date` | Filter runs until date | `--end-date -1d` |
| `--parse` | Generate `log.md` and `firewall.md` | `--parse` |
| `--json` | Output structured metrics | `--json` |

Downloads workflow execution logs, analyzes tool usage and network patterns, and caches results for faster subsequent runs (~10-100x speedup).

#### `audit`

Investigate and analyze specific workflow runs.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By URL
gh aw audit 12345678 --parse                              # Parse logs to markdown
```

Provides detailed analysis including overview, execution metrics, tool usage patterns, MCP server failures, firewall analysis, and artifact information. Accepts run IDs or URLs from any repository and GitHub instance.

### Management

#### `enable`

Enable workflows for execution.

```bash wrap
gh aw enable                                # Enable all workflows
gh aw enable prefix                         # Enable workflows matching prefix
gh aw enable ci-*                          # Enable workflows with pattern
```

Enables workflows for automatic and manual execution with pattern matching support for bulk operations.

#### `disable`

Disable workflows and cancel running jobs.

```bash wrap
gh aw disable                               # Disable all workflows
gh aw disable prefix                        # Disable workflows matching prefix
gh aw disable ci-*                         # Disable workflows with pattern
```

Disables workflows to prevent execution and cancels any currently running workflow jobs.

#### `remove`

Remove workflows from the repository.

```bash wrap
gh aw remove WorkflowName
```

Removes both `.md` and `.lock.yml` files and updates repository configuration.

#### `update`

Update workflows to their latest versions.

```bash wrap
gh aw update                              # Update all workflows with source field
gh aw update ci-doctor                    # Update specific workflow
gh aw update ci-doctor --major --force    # Allow major version updates
```

**What it does:**
- Updates workflows based on `source` field format `owner/repo/path@ref`
- Performs 3-way merge preserving local changes
- Semantic version tags update within the same major version

**Options:**
- `--major`: Allow major version updates (breaking changes)
- `--force`: Force update even with conflicts

**Conflict handling:** When conflicts occur, diff3 markers are added and recompilation is skipped until resolved.

### Advanced

#### `mcp`

Manage MCP (Model Context Protocol) servers.

```bash wrap
gh aw mcp list                             # List all MCP servers
gh aw mcp list workflow-name               # List servers for specific workflow
gh aw mcp inspect workflow-name            # Inspect and test servers
gh aw mcp add                              # Add servers from registry
```

Lists MCP servers configured in workflows, inspects server configuration, tests connectivity, and adds new servers from the registry. See **[MCPs Guide](/gh-aw/guides/mcps/)** for complete documentation.

#### `pr transfer`

Transfer pull requests between repositories.

```bash wrap
gh aw pr transfer https://github.com/source/repo/pull/234
gh aw pr transfer <pr-url> --repo target-owner/target-repo
```

Transfers PR from source repository to target repository, preserving code changes, title, and description.

**Options:**
- `--repo target-owner/target-repo`: Specify target repository (defaults to current repository)

#### `mcp-server`

Start MCP server exposing CLI commands as tools.

```bash wrap
gh aw mcp-server              # stdio transport (local)
gh aw mcp-server --port 3000  # HTTP/SSE transport (workflows)
```

Exposes CLI commands (`status`, `compile`, `logs`, `audit`) as MCP tools, enabling AI assistants to interact with gh-aw programmatically. Supports both local (stdio) and remote (HTTP) transports.

**Options:**
- `--port N`: Start HTTP server on specified port (defaults to stdio transport)

See **[MCP Server Guide](/gh-aw/setup/mcp-server/)** for integration details.

## Examples

### Basic Workflow Lifecycle

```bash wrap
gh aw init                                  # Initialize repository
gh aw add githubnext/agentics/ci-doctor    # Add a workflow
gh aw compile                               # Compile to GitHub Actions
gh aw trial ci-doctor                       # Test safely
```

### Compile with Security Scanning

```bash wrap
gh aw compile --verbose                     # Detailed output
gh aw compile --strict --zizmor             # Strict mode + security scan
```

### Analyze Workflow Runs

```bash wrap
gh aw logs ci-doctor -c 5 --parse --json   # Download, parse, and export
gh aw audit 12345678 --parse                # Deep dive into specific run
```

## Debug Logging

Enable detailed debugging output for troubleshooting:

```bash wrap
DEBUG=* gh aw compile                # All debug logs
DEBUG=cli:* gh aw compile            # CLI operations only
DEBUG=cli:*,workflow:* gh aw compile # Multiple packages
DEBUG=*,-tests gh aw compile         # All except tests
```

**Features:**
- Shows namespace, message, and time diff (e.g., `+50ms`)
- Zero overhead when disabled
- Supports pattern matching with wildcards

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
- [Workflow Status](/gh-aw/status/) - Live workflow dashboard
