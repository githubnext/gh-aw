---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
sidebar:
  order: 200
---

## Overview

The `gh aw` CLI extension enables developers to create, manage, and execute AI-powered workflows directly from the command line. It transforms natural language markdown files into GitHub Actions, allowing you to:

- **Automate repository tasks** with AI-driven workflows that understand context and make decisions
- **Standardize workflow patterns** across teams with reusable, shareable configurations
- **Save time** by eliminating manual issue triage, code reviews, and documentation updates
- **Test safely** with isolated trial environments before deploying to production

**Who it's for:**
- Development teams automating repetitive tasks
- Repository maintainers managing issues, PRs, and discussions
- DevOps engineers standardizing CI/CD workflows
- Open source maintainers scaling community interactions

## Quick Start (Recommended)

> Get productive in one command.

```bash wrap
gh aw init
```

This initializes your repository with recommended defaults, configuring `.gitattributes`, Copilot instructions, and setting up the foundation for agentic workflows.

**Next Steps:**
- Add a workflow: `gh aw add githubnext/agentics/ci-doctor`
- Test it safely: `gh aw trial ./my-workflow.md`
- See **[Use Cases](#use-cases--applications)** for specific scenarios

## Use Cases / Applications

> Real workflows showing *when and why* to use the CLI.

| Goal / Scenario | Command to use |
|-----------------|----------------|
| Initialize a repository for agentic workflows | `gh aw init` |
| Add a pre-built workflow from The Agentics collection | `gh aw add githubnext/agentics/ci-doctor` |
| Create a custom workflow from scratch | `gh aw new my-custom-workflow` |
| Test a workflow safely before deploying | `gh aw trial ./my-workflow.md` |
| Compile workflows to GitHub Actions YAML | `gh aw compile` |
| Compile with security scanning | `gh aw compile --strict --zizmor` |
| Run a workflow immediately | `gh aw run daily-perf` |
| Check status of all workflows | `gh aw status` |
| View execution logs and analyze runs | `gh aw logs workflow-name` |
| Debug a specific workflow run | `gh aw audit 12345678` |
| Enable/disable workflows | `gh aw enable [pattern]` / `gh aw disable [pattern]` |
| Update workflows to latest versions | `gh aw update` |
| Inspect MCP server configuration | `gh aw mcp inspect [workflow]` |
| Transfer a PR between repositories | `gh aw pr transfer <pr-url>` |

### Example: Add and test a workflow

```bash wrap
gh aw add githubnext/agentics/ci-doctor
gh aw trial ci-doctor
```

**Output (expected)**

```
✔ Workflow added successfully to .github/workflows/ci-doctor.md
✔ Compiled to .github/workflows/ci-doctor.lock.yml
✔ Trial repository created: gh-aw-trial-abc123
✔ Workflow executed successfully
```

**Next:** See **[Commands](#commands)** for detailed options and **[Configuration](/gh-aw/reference/frontmatter/)** to customize behavior.

## Installation

```bash wrap
gh extension install githubnext/gh-aw
```

**Alternative Installation:**

If the above fails due to authentication issues, use the standalone installer:

```bash wrap
curl -O https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh
chmod +x install-gh-aw.sh
./install-gh-aw.sh
```

**GitHub Enterprise Server:** The CLI supports GitHub Enterprise Server through the `GITHUB_SERVER_URL` or `GH_HOST` environment variables. When set, commands like `gh aw add` will use the specified GitHub instance for cloning and accessing workflows.

## Usage

**Syntax**

```bash wrap
gh aw [command] [options] <input>
```

**Global Help**

```bash wrap
gh aw --help                    # Show all commands
gh aw help [command]            # Show command-specific help
gh aw help all                  # Comprehensive documentation
```

## Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-h`, `--help` | Show help information | — |
| `-v`, `--verbose` | Enable verbose output with debugging details | false |

## Commands

### Command Overview

| Command | Description |
|---------|-------------|
| `init` | Initialize repository for agentic workflows |
| `new` | Create a new workflow from scratch |
| `add` | Add workflows from The Agentics collection or other repositories |
| `remove` | Remove workflows from the repository |
| `update` | Update workflows to latest versions |
| `compile` | Compile markdown workflows to GitHub Actions YAML |
| `run` | Execute workflows immediately |
| `status` | Show status of all workflows |
| `enable` | Enable workflows |
| `disable` | Disable workflows and cancel running jobs |
| `logs` | Download and analyze workflow execution logs |
| `audit` | Investigate specific workflow runs |
| `trial` | Test workflows in temporary private repositories |
| `mcp` | Manage MCP (Model Context Protocol) servers |
| `pr transfer` | Transfer pull requests between repositories |
| `mcp-server` | Start MCP server exposing CLI as tools |

## Workflow Creation and Management

### `init`

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
- Enables gh-aw MCP tools (`status`, `compile`, `logs`, `audit`) in Copilot Agent

**Related:** See **[Quick Start](#quick-start-recommended)** and **[MCP Server Guide](/gh-aw/setup/mcp-server/)**.

### `new`

Create a new custom workflow from scratch.

```bash wrap
gh aw new my-custom-workflow
```

**What it does:**
- Creates a new markdown workflow file in `.github/workflows/`
- Provides a template with frontmatter and instructions
- Automatically opens the file for editing

**Related:** See **[Use Cases](#use-cases--applications)** and **[Workflow Structure](/gh-aw/reference/workflow-structure/)**.

---

### `add`

Add workflows from The Agentics collection or other repositories.

```bash wrap
gh aw add githubnext/agentics/ci-doctor           # Add single workflow
gh aw add "githubnext/agentics/ci-*"             # Add multiple with wildcards
gh aw add ci-doctor --dir shared --number 3      # Organize and create copies
gh aw add githubnext/agentics/ci-doctor --pr     # Create PR instead of direct commit
```

**What it does:**
- Downloads workflow from specified repository
- Automatically compiles to `.lock.yml`
- Updates `.gitattributes` (use `--no-gitattributes` to disable)

**Options:**
- `--dir`: Organize workflows in subdirectories under `.github/workflows/`
- `--number`: Create multiple numbered copies
- `--pr`: Create a pull request instead of committing directly

**Note:** When workflows aren't found, available options are displayed in a formatted table.

**Related:** See **[Packaging and Updating](/gh-aw/guides/packaging-imports/)** for complete guide.

---

### `remove`

Remove workflows from the repository.

```bash wrap
gh aw remove WorkflowName
```

**What it does:**
- Removes both `.md` and `.lock.yml` files
- Updates repository configuration

**Related:** See **[Workflow Management](#workflow-creation-and-management)**.

### `update`

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

**Related:** See **[Packaging and Updating](/gh-aw/guides/packaging-imports/)** for version management strategies.

### `compile`

Compile markdown workflows to GitHub Actions YAML.

```bash wrap
gh aw compile                              # Compile all workflows
gh aw compile my-workflow                  # Compile specific workflow
gh aw compile --watch                      # Auto-recompile on changes
gh aw compile --validate --strict          # Schema validation + strict mode
gh aw compile --zizmor                     # Security scan (warnings)
gh aw compile --strict --zizmor            # Security scan (fails on findings)
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --purge                      # Remove orphaned .lock.yml files
```

**What it does:**
- Transforms `.md` files into `.lock.yml` GitHub Actions workflows
- Validates frontmatter schema and configuration
- Checks for security issues and best practices

**Compilation Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `--validate` | Schema validation and container checks | false |
| `--strict` | Requires timeouts, explicit network config, blocks write permissions | false |
| `--zizmor` | Runs [zizmor](https://github.com/woodruffw/zizmor) security scanner | false |
| `--dependabot` | Generates npm/pip/Go manifests and updates `.github/dependabot.yml` | false |
| `--watch` | Auto-recompile on file changes | false |
| `--purge` | Remove orphaned `.lock.yml` files | false |

**Note:** Compilation fails if workflows use `create-discussion` or `create-issue` but the repository lacks those features.

**Related:** See **[VS Code setup](/gh-aw/setup/vscode/)** for watch mode integration and **[Security Guide](/gh-aw/guides/security/)** for security best practices.

## Workflow Operations

### `run`

Execute workflows immediately in GitHub Actions.

```bash wrap
gh aw run WorkflowName                      # Run workflow
gh aw run workflow1 workflow2               # Run multiple workflows
gh aw run workflow --repeat 3               # Repeat execution 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
```

**What it does:**
- Triggers workflow execution in GitHub Actions
- Waits for completion and reports status
- Supports multiple workflows and repeated executions

**Options:**
- `--repeat N`: Execute the workflow N times
- `--use-local-secrets`: Temporarily push AI engine secrets from environment variables, then clean up

:::note[Codespaces]
From GitHub Codespaces, grant `actions: write` and `workflows: write` permissions. See [Managing repository access](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces).
:::

**Related:** See **[Use Cases](#use-cases--applications)** for common execution patterns.

### `status`

Show status of all workflows in the repository.

```bash wrap
gh aw status                                # Show all workflow status
```

**What it does:**
- Lists all agentic workflows with their current state
- Shows enabled/disabled status, schedules, and configurations
- Provides overview of workflow health

**Related:** See **[Workflow Status](/gh-aw/status/)** page for live status dashboard.

---

### `enable`

Enable workflows for execution.

```bash wrap
gh aw enable                                # Enable all workflows
gh aw enable prefix                         # Enable workflows matching prefix
gh aw enable ci-*                          # Enable workflows with pattern
```

**What it does:**
- Enables workflows for automatic and manual execution
- Supports pattern matching for bulk operations

**Related:** See **[`disable`](#disable)** for the opposite operation.

---

### `disable`

Disable workflows and cancel running jobs.

```bash wrap
gh aw disable                               # Disable all workflows
gh aw disable prefix                        # Disable workflows matching prefix
gh aw disable ci-*                         # Disable workflows with pattern
```

**What it does:**
- Disables workflows to prevent execution
- Cancels any currently running workflow jobs
- Supports pattern matching for bulk operations

**Related:** See **[`enable`](#enable)** to re-enable workflows.

### `logs`

Download and analyze workflow execution logs.

```bash wrap
gh aw logs                                 # Download logs for all workflows
gh aw logs workflow-name                   # Download logs for specific workflow
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --parse --json                 # Generate markdown + JSON output
```

**What it does:**
- Downloads workflow execution logs from GitHub Actions
- Analyzes tool usage, timing, and network patterns
- Caches downloaded runs (~10-100x faster on subsequent runs)

**Log Options:**

| Option | Description | Example |
|--------|-------------|---------|
| `-c, --count N` | Limit number of runs | `-c 10` |
| `--start-date` | Filter runs from date | `--start-date -1w` |
| `--end-date` | Filter runs until date | `--end-date -1d` |
| `--parse` | Generate `log.md` and `firewall.md` | `--parse` |
| `--json` | Output structured metrics in JSON | `--json` |

**Related:** See **[`audit`](#audit)** for detailed investigation of specific runs.

### `audit`

Investigate and analyze specific workflow runs.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By URL
gh aw audit 12345678 --parse                              # Parse logs to markdown
```

**What it does:**
- Provides detailed analysis of a single workflow run
- Accepts run IDs or URLs from any repository and GitHub instance
- Generates comprehensive reports

**Report includes:**
- Overview and execution metrics
- Tool usage patterns
- MCP server failures
- Firewall analysis
- Artifact information

**Options:**
- `--parse`: Generate markdown report from logs

**Related:** See **[`logs`](#logs)** for bulk log analysis and **[Use Cases](#use-cases--applications)** for debugging scenarios.

### `mcp`

Manage MCP (Model Context Protocol) servers.

```bash wrap
gh aw mcp list                             # List all MCP servers
gh aw mcp list workflow-name               # List servers for specific workflow
gh aw mcp inspect workflow-name            # Inspect and test servers
gh aw mcp add                              # Add servers from registry
```

**What it does:**
- Lists MCP servers configured in workflows
- Inspects server configuration and tests connectivity
- Adds new MCP servers from the registry

**Related:** See **[MCPs Guide](/gh-aw/guides/mcps/)** and **[MCP Server Guide](/gh-aw/setup/mcp-server/)** for complete documentation.

## Repository Utilities

### `trial`

Test workflows safely in temporary private repositories.

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test remote workflow
gh aw trial ./workflow.md                          # Test local workflow
gh aw trial ./workflow.md --use-local-secrets       # Test with local API keys
gh aw trial ./workflow.md --logical-repo owner/repo # Act as different repo
gh aw trial ./issue-workflow.md --trigger-context "#123" # With issue context
```

**What it does:**
- Creates a temporary private repository for testing
- Executes workflow in isolated environment
- Saves results to `trials/` directory
- Cleans up after completion (optional)

**Trial Options:**

| Option | Description |
|--------|-------------|
| `--engine` | Override AI engine for testing |
| `--auto-merge-prs` | Automatically merge created PRs |
| `--repeat N` | Repeat execution N times |
| `--delete-host-repo-after` | Delete trial repository after execution |
| `--use-local-secrets` | Use local API keys instead of repository secrets |
| `--logical-repo owner/repo` | Access issues/PRs from specific repository |
| `--clone-repo owner/repo` | Use different codebase for testing |
| `--trigger-context "#123"` | Provide issue/PR context for testing |

**Related:** See **[Use Cases](#use-cases--applications)** for testing workflows before deployment.

### `pr transfer`

Transfer pull requests between repositories.

```bash wrap
gh aw pr transfer https://github.com/source/repo/pull/234
gh aw pr transfer <pr-url> --repo target-owner/target-repo
```

**What it does:**
- Transfers PR from source repository to target repository
- Preserves code changes, title, and description
- Maintains PR metadata where possible

**Options:**
- `--repo target-owner/target-repo`: Specify target repository (defaults to current repository)

**Related:** See **[Use Cases](#use-cases--applications)** for repository migration scenarios.

### `mcp-server`

Start MCP server exposing CLI commands as tools.

```bash wrap
gh aw mcp-server              # stdio transport (local)
gh aw mcp-server --port 3000  # HTTP/SSE transport (workflows)
```

**What it does:**
- Exposes CLI commands (`status`, `compile`, `logs`, `audit`) as MCP tools
- Enables AI assistants to interact with gh-aw programmatically
- Supports both local (stdio) and remote (HTTP) transports

**Options:**
- `--port N`: Start HTTP server on specified port (defaults to stdio transport)

**Related:** See **[MCP Server Guide](/gh-aw/setup/mcp-server/)** for integration details.

## Examples

### Basic workflow lifecycle

```bash wrap
# Initialize repository
gh aw init

# Add a workflow
gh aw add githubnext/agentics/ci-doctor

# Compile to GitHub Actions
gh aw compile

# Test safely
gh aw trial ci-doctor
```

**Output (expected)**

```
✔ Repository initialized
✔ Workflow added to .github/workflows/ci-doctor.md
✔ Compiled to ci-doctor.lock.yml
✔ Trial completed successfully
```

---

### Test with verbose logging

```bash wrap
gh aw compile --verbose
```

**Output (expected)**

```
Compiling workflows...
  ✔ ci-doctor.md → ci-doctor.lock.yml
  ✔ Schema validation passed
  ✔ Security checks passed
Compilation completed in 1.2s
```

---

### Debug with environment variables

```bash wrap
DEBUG=* gh aw compile
```

**Output (expected)**

```
cli:compile Compiling workflows... +0ms
workflow:compiler Processing ci-doctor.md +50ms
workflow:validator Validating frontmatter +100ms
cli:compile Compilation completed +1200ms
```

---

### Analyze workflow runs

```bash wrap
gh aw logs ci-doctor -c 5 --parse --json
```

**Output (expected)**

```
✔ Downloaded 5 runs
✔ Generated log.md with execution details
✔ Generated firewall.md with network analysis
✔ Exported metrics to runs.json
```

## Debug Logging

Enable detailed debugging output for troubleshooting.

```bash wrap
DEBUG=* gh aw compile                # All debug logs
DEBUG=cli:* gh aw compile            # CLI operations only
DEBUG=cli:*,workflow:* gh aw compile # Multiple packages
DEBUG=*,-tests gh aw compile         # All except tests
```

**Debug Features:**
- Shows namespace, message, and time diff (e.g., `+50ms`)
- Zero overhead when disabled
- Supports pattern matching with wildcards

**For user-facing details:** Use `--verbose` flag instead of DEBUG environment variable.

**Related:** See **[Troubleshooting](#troubleshooting)** for common debugging scenarios.

## Exit Codes

The CLI uses standard exit codes to indicate command status.

| Code | Meaning | Description |
|------|---------|-------------|
| 0 | Success | Command completed successfully |
| 1 | General error | Command failed with a recoverable error |
| 2 | Usage error | Invalid command usage or missing required arguments |

**Example usage in scripts:**

```bash wrap
if gh aw compile; then
  echo "Compilation successful"
else
  echo "Compilation failed with exit code $?"
  exit 1
fi
```

## Troubleshooting

Common issues and their solutions.

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| `command not found: gh` | GitHub CLI not installed | Install from [cli.github.com](https://cli.github.com/) |
| `extension not found: aw` | Extension not installed | Run `gh extension install githubnext/gh-aw` |
| Compilation fails with YAML errors | Frontmatter syntax errors | Check indentation, colons, and array syntax |
| Workflow not found | Incorrect workflow name | Run `gh aw status` to list available workflows |
| Permission denied on files | Write access needed | Check file permissions or repository access |
| Trial creation fails | Repository creation limits | Check GitHub rate limits and authentication |
| MCP server connection fails | Server not installed | Verify MCP server package availability |

**For detailed troubleshooting:** See [Common Issues](/gh-aw/troubleshooting/common-issues/) and [Error Reference](/gh-aw/troubleshooting/errors/).

## See Also

**Getting Started:**
- [Quick Start](/gh-aw/get-started/quick-start/) - Get your first workflow running
- [Concepts](/gh-aw/get-started/concepts/) - Understanding agentic workflows
- [About](/gh-aw/get-started/about/) - How natural language becomes GitHub Actions

**Workflow Management:**
- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Adding, updating, and importing workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and file organization
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options for workflows

**Advanced Topics:**
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure output processing including issue updates
- [Tools](/gh-aw/reference/tools/) - GitHub and MCP server configuration
- [Imports](/gh-aw/reference/imports/) - Modularizing workflows with includes
- [Security Guide](/gh-aw/guides/security/) - Security best practices

**Setup and Integration:**
- [VS Code Setup](/gh-aw/setup/vscode/) - Editor integration and watch mode
- [MCP Server Guide](/gh-aw/setup/mcp-server/) - MCP server configuration
- [Agentic Authoring](/gh-aw/setup/agentic-authoring/) - AI-assisted workflow creation

**Reference:**
- [Workflow Status](/gh-aw/status/) - Live workflow dashboard
- [Error Reference](/gh-aw/troubleshooting/errors/) - Detailed error messages
- [Common Issues](/gh-aw/troubleshooting/common-issues/) - Frequently encountered problems
