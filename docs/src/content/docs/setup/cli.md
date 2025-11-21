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

When adding a workflow, the command displays the workflow description (extracted from the frontmatter `description` field) to provide context about the workflow's purpose.

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
gh aw compile --validate --json            # Validation with JSON output
gh aw compile --zizmor                     # Security scan (warnings)
gh aw compile --strict --zizmor            # Security scan (fails on findings)
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --purge                      # Remove orphaned .lock.yml files
```

Remote imports are automatically cached in `.github/aw/imports/` for offline compilation. First compilation downloads imports; subsequent compilations use cached files, eliminating network calls.

**Key Options:**

| Option | Description |
|--------|-------------|
| `--validate` | Schema validation and container checks |
| `--strict` | Enable strict mode validation for all workflows |
| `--zizmor` | Security scanning with [zizmor](https://github.com/woodruffw/zizmor) |
| `--dependabot` | Generate npm/pip/Go manifests and update dependabot.yml |
| `--json` | Output validation results in machine-readable JSON format |
| `--watch` | Auto-recompile on file changes |
| `--purge` | Remove orphaned `.lock.yml` files |

**Strict Mode (`--strict`):**

The `--strict` flag enables enhanced security validation for all workflows during compilation. This flag is recommended for production workflows that require strict security compliance.

When enabled, strict mode enforces:

1. **No Write Permissions**: Blocks `contents:write`, `issues:write`, and `pull-requests:write` permissions. Use [safe-outputs](/gh-aw/reference/safe-outputs/) instead.

2. **Explicit Network Configuration**: Requires explicit `network` configuration. No implicit defaults allowed.

3. **No Network Wildcards**: Refuses wildcard `*` in `network.allowed` domains. Use explicit domains or ecosystem identifiers.

4. **MCP Server Network**: Requires network configuration for custom MCP servers with containers.

5. **Action Pinning**: Enforces GitHub Actions to be pinned to specific commit SHAs.

6. **No Deprecated Fields**: Refuses deprecated frontmatter fields.

**Precedence:** The `--strict` CLI flag applies to all workflows being compiled and takes precedence over individual workflow `strict` frontmatter fields. Workflows cannot opt-out of strict mode when the CLI flag is set.

**Example:**
```bash wrap
# Enable strict mode for all workflows
gh aw compile --strict

# Strict mode with security scanning (fails on findings)
gh aw compile --strict --zizmor

# Validate schema and enforce strict mode
gh aw compile --validate --strict
```

See [Strict Mode reference](/gh-aw/reference/frontmatter/#strict-mode-strict) for frontmatter configuration and [Security Guide](/gh-aw/guides/security/#strict-mode-validation) for best practices.

### Testing

#### `trial`

Test workflows safely in temporary private repositories or run directly in a specified repository.

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test remote workflow
gh aw trial ./workflow.md                          # Test local workflow
gh aw trial ./workflow.md --use-local-secrets      # Test with local API keys
gh aw trial ./workflow.md --logical-repo owner/repo # Act as different repo
gh aw trial ./issue-workflow.md --trigger-context "#123" # With issue context
gh aw trial ./workflow.md --repo owner/repo        # Run directly in repository
```

When trialing a workflow, the command displays the workflow description (extracted from the frontmatter `description` field) to provide context about what the workflow does.

**Key Options:**

| Option | Description |
|--------|-------------|
| `-e, --engine` | Override AI engine for testing |
| `--auto-merge-prs` | Automatically merge created PRs |
| `--repeat N` | Repeat execution N times |
| `--delete-host-repo-after` | Delete trial repository after execution |
| `--use-local-secrets` | Use local API keys instead of repository secrets |
| `--logical-repo owner/repo` | Access issues/PRs from specific repository |
| `--clone-repo owner/repo` | Use different codebase for testing |
| `--trigger-context "#123"` | Provide issue/PR context |
| `--repo owner/repo` | Install and run workflow directly in specified repository |

**Trial Modes:**
- **Default mode**: Creates temporary private repository for safe testing
- **Direct repository mode** (`--repo`): Installs workflow in specified repository and executes immediately without waiting for completion

Results are saved to `trials/` directory.

#### `run`

Execute workflows immediately in GitHub Actions.

```bash wrap
gh aw run WorkflowName                      # Run workflow
gh aw run workflow1 workflow2               # Run multiple workflows
gh aw run workflow --repeat 3               # Repeat execution 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
```

After triggering a workflow, the command displays the workflow URL and suggests using `gh aw audit` to analyze the run.

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
gh aw logs workflow-name --repo owner/repo # Download logs from specific repository
```

**Key Options:**

| Option | Description | Example |
|--------|-------------|---------|
| `-c, --count N` | Limit number of runs | `-c 10` |
| `-e, --engine` | Filter by AI engine | `-e copilot` |
| `--start-date` | Filter runs from date | `--start-date -1w` |
| `--end-date` | Filter runs until date | `--end-date -1d` |
| `--parse` | Generate `log.md` and `firewall.md` | `--parse` |
| `--json` | Output structured metrics | `--json` |
| `--repo owner/repo` | Download logs from specific repository | `--repo owner/repo` |

Downloads workflow execution logs, analyzes tool usage and network patterns, and caches results for faster subsequent runs (~10-100x speedup). The overview table includes columns for errors, warnings, missing tools, and noop messages.

#### `audit`

Investigate and analyze specific workflow runs.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By URL
gh aw audit 12345678 --parse                              # Parse logs to markdown
```

Provides detailed analysis including overview, execution metrics, tool usage patterns, MCP server failures, firewall analysis, noop messages, and artifact information. Accepts run IDs or URLs from any repository and GitHub instance. JSON output includes parsed noop messages similar to missing-tool reports.

**GitHub Copilot Agent Detection:** Automatically detects GitHub Copilot agent runs and uses specialized log parsing to extract agent-specific metrics including turns, tool calls, errors, and token usage. Detection is based on workflow path (`copilot-swe-agent`) and agent-specific log patterns.

### Management

#### `enable`

Enable workflows for execution.

```bash wrap
gh aw enable                                # Enable all workflows
gh aw enable prefix                         # Enable workflows matching prefix
gh aw enable ci-*                          # Enable workflows with pattern
gh aw enable workflow-name --repo owner/repo # Enable in specific repository
```

Enables workflows for automatic and manual execution with pattern matching support for bulk operations.

**Options:**
- `--repo owner/repo`: Enable workflows in a specific repository (defaults to current repository)

#### `disable`

Disable workflows and cancel running jobs.

```bash wrap
gh aw disable                               # Disable all workflows
gh aw disable prefix                        # Disable workflows matching prefix
gh aw disable ci-*                         # Disable workflows with pattern
gh aw disable workflow-name --repo owner/repo # Disable in specific repository
```

Disables workflows to prevent execution and cancels any currently running workflow jobs.

**Options:**
- `--repo owner/repo`: Disable workflows in a specific repository (defaults to current repository)

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
gh aw update ci-doctor --merge            # Update with 3-way merge (preserve changes)
gh aw update ci-doctor --major --force    # Allow major version updates
gh aw update --dir custom/workflows       # Update workflows in custom directory
```

**What it does:**
- Updates workflows based on `source` field format `owner/repo/path@ref`
- Replaces local file with upstream version (default behavior)
- Semantic version tags update within the same major version
- Falls back to git commands when GitHub API authentication fails

**Options:**
- `--dir`: Specify workflow directory (defaults to `.github/workflows`)
- `--merge`: Perform 3-way merge to preserve local changes (creates conflict markers if needed)
- `--major`: Allow major version updates (breaking changes)
- `--force`: Force update even with conflicts

**Update Modes:**

| Mode | Behavior |
|------|----------|
| Default | Replaces local file with latest upstream version (no conflicts) |
| `--merge` | 3-way merge preserving local changes (may create conflict markers) |

**Authentication:** The update command works with public repositories without requiring GitHub authentication. When GitHub API calls fail due to missing or insufficient tokens, the command automatically falls back to git commands for downloading workflow content.

**Conflict handling:** When using `--merge` and conflicts occur, diff3 markers are added and recompilation is skipped until resolved.

### Advanced

#### `mcp`

Manage MCP (Model Context Protocol) servers.

```bash wrap
gh aw mcp list                             # List all MCP servers
gh aw mcp list workflow-name               # List servers for specific workflow
gh aw mcp list-tools <mcp-server>          # List tools for specific MCP server
gh aw mcp list-tools <mcp-server> workflow # List tools in specific workflow
gh aw mcp inspect workflow-name            # Inspect and test servers
gh aw mcp add                              # Add servers from registry
```

Lists MCP servers configured in workflows, inspects server configuration, tests connectivity, and adds new servers from the registry. See **[MCPs Guide](/gh-aw/guides/mcps/)** for complete documentation.

#### `pr`

Pull request management utilities.

**Subcommands:**

##### `pr transfer`

Transfer a pull request to another repository.

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

### Utility Commands

#### `version`

Show version information for the gh-aw CLI.

```bash wrap
gh aw version
```

Displays the current version of gh-aw and product information. Equivalent to using the `--version` flag.

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
- [Labs](/gh-aw/labs/) - Experimental workflows
