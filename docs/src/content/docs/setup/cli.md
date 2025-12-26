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

## Common Workflows for Beginners

Learn how to use CLI commands together for common development tasks:

### After creating a new workflow

```bash wrap
gh aw compile my-workflow      # Check for syntax errors
gh aw run my-workflow           # Test it manually (if workflow_dispatch enabled)
gh aw logs my-workflow          # See what happened
```

**What each command does:**
- **`compile`**: Validates your markdown and generates the `.lock.yml` file—catches errors before running
- **`run`**: Triggers the workflow immediately in GitHub Actions (requires `workflow_dispatch` trigger)
- **`logs`**: Downloads and analyzes execution logs to see what the AI did

### When something goes wrong

```bash wrap
gh aw status                    # Check configuration and workflow state
gh aw logs my-workflow          # Review recent execution logs
gh aw audit (run-id)            # Analyze specific run in detail
```

**Finding the run-id:**
1. Go to your repository on GitHub.com
2. Click the **Actions** tab
3. Click on a workflow run from the list
4. The run-id is the number in the URL: `github.com/owner/repo/actions/runs/12345678` → run-id is `12345678`

**What each command reveals:**
- **`status`**: Shows if workflows are enabled, compiled, and their schedules
- **`logs`**: Reveals AI decisions, tool usage, network activity, and errors
- **`audit`**: Provides deep analysis of a specific run including execution metrics, failed tools, and artifacts

### Fixing common issues

```bash wrap
# Token or secret issues
gh aw secrets bootstrap --engine copilot   # Check token configuration

# Workflow not found or disabled
gh aw status                               # List all workflows
gh aw enable my-workflow                   # Enable if disabled

# Syntax errors in workflow markdown
gh aw compile my-workflow --validate       # Detailed validation output
gh aw fix my-workflow --write              # Auto-fix deprecated fields
```

## Installation

Install the GitHub CLI extension:

```bash wrap
gh extension install githubnext/gh-aw
```

### Pinning to a Specific Version

For production environments or to ensure reproducible builds, you can pin the installation to a specific version using the `@REF` syntax:

```bash wrap
# Pin to a specific release tag
gh extension install githubnext/gh-aw@v0.1.0

# Pin to a specific commit SHA
gh extension install githubnext/gh-aw@abc123def456
```

:::tip[When to pin versions]
Pinning is recommended when:
- Deploying to production environments
- Ensuring consistency across team members
- Testing specific versions before upgrading
- Avoiding unexpected breaking changes from automatic updates
:::

**Checking your current version:**

```bash wrap
gh aw version
```

**Upgrading to a new pinned version:**

```bash wrap
# Remove the current installation
gh extension remove gh-aw

# Install the new pinned version
gh extension install githubnext/gh-aw@v0.2.0
```

### Alternative: Standalone Installer

If the extension installation fails (common in Codespaces outside the githubnext organization or when authentication issues occur), use the standalone installer:

```bash wrap
# Install latest version
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash

# Install specific version
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash -s v0.1.0
```

After standalone installation, the binary is installed to `~/.local/share/gh/extensions/gh-aw/gh-aw` and can be used with `gh aw` commands just like the extension installation.

**Pinning with the standalone installer:**

To ensure reproducible installations, you can pin to a specific version by passing the version tag as an argument:

```bash wrap
# Download and run with a specific version
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash -s v0.1.0

# Or download the script first, then run with version argument
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh -o install-gh-aw.sh
chmod +x install-gh-aw.sh
./install-gh-aw.sh v0.1.0
```

The installer will download the specified version's pre-built binary for your platform (Linux, macOS, FreeBSD, or Windows) and install it to the standard extension directory.

### GitHub Enterprise Server Support

GitHub Agentic Workflows fully supports GitHub Enterprise Server deployments. Configure your enterprise instance using environment variables:

```bash wrap
# Set your enterprise hostname
export GH_HOST="github.enterprise.com"
# Or use GitHub Actions standard variable
export GITHUB_SERVER_URL="https://github.enterprise.com"

# Authenticate with your enterprise instance
gh auth login --hostname github.enterprise.com

# Use gh aw commands normally
gh aw status
gh aw logs workflow
```

When using the `--repo` flag, you can specify the enterprise host:

```bash wrap
gh aw logs workflow --repo github.enterprise.com/owner/repo
gh aw run workflow --repo github.enterprise.com/owner/repo
```

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
gh aw init         # Configure .gitattributes, Copilot instructions (MCP enabled by default)
gh aw init --no-mcp # Skip MCP server integration
```

Configures `.gitattributes` to mark `.lock.yml` files as generated, adds Copilot instructions for better AI assistance, sets up prompt files for workflow creation, and creates `.github/aw/logs/.gitignore` to prevent workflow logs from being committed. MCP server integration is enabled by default, creating GitHub Actions workflow for MCP server setup, configuring `.vscode/mcp.json` for VS Code integration, and enabling gh-aw MCP tools in Copilot Agent. Use `--no-mcp` to skip MCP server integration.

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

Create a new workflow Markdown file with example configuration.

```bash wrap
gh aw new                      # Interactive mode
gh aw new my-custom-workflow   # Create template file
gh aw new my-workflow.md       # Same as above (.md extension stripped)
gh aw new my-workflow --force  # Overwrite if exists
```

Creates a markdown workflow file in `.github/workflows/` with template frontmatter and automatically opens it for editing. The `.md` extension is optional and will be automatically normalized.

#### `secrets`

Manage GitHub Actions secrets and tokens for GitHub Agentic Workflows. Use this command to set secrets for workflows and check which recommended token secrets are configured for your repository.

**Subcommands:**

##### `secrets set`

Create or update a repository secret. The secret value can be provided via flag, environment variable, or stdin.

```bash wrap
gh aw secrets set MY_SECRET                                    # From stdin (prompts)
gh aw secrets set MY_SECRET --value "secret123"                # From flag
gh aw secrets set MY_SECRET --value-from-env MY_TOKEN          # From environment variable
gh aw secrets set MY_SECRET --owner myorg --repo myrepo        # For specific repository
```

**Options:** `--owner` (repository owner), `--repo` (repository name), `--value` (secret value), `--value-from-env` (environment variable to read from), `--api-url` (GitHub API base URL)

When `--owner` and `--repo` are not specified, the command operates on the current repository. Both flags must be provided together when targeting a specific repository.

##### `secrets bootstrap`

Check and suggest setup for gh-aw GitHub token secrets. This command is read-only and inspects repository secrets to identify missing tokens, then prints least-privilege setup instructions.

```bash wrap
gh aw secrets bootstrap                    # Check tokens for current repository
gh aw secrets bootstrap --engine copilot   # Check Copilot-specific tokens
gh aw secrets bootstrap --engine claude    # Check Claude-specific tokens
gh aw secrets bootstrap --engine codex     # Check Codex-specific tokens
gh aw secrets bootstrap --owner org --repo project # Check specific repository
```

**Options:** `--engine` (AI engine to check tokens for: copilot, claude, codex), `--owner` (repository owner, defaults to current), `--repo` (repository name, defaults to current)

The command checks for recommended secrets like `GH_AW_GITHUB_TOKEN`, `COPILOT_GITHUB_TOKEN`, `ANTHROPIC_API_KEY`, and `OPENAI_API_KEY` based on the specified engine. It provides setup instructions with suggested scopes for any missing tokens.

See [GitHub Tokens reference](/gh-aw/reference/tokens/) for detailed information about token precedence and security best practices.

### Building

#### `fix`

Automatically fix deprecated workflow fields using codemods. Applies migrations for field renames and API changes to keep workflows up-to-date with the latest schema.

```bash wrap
gh aw fix                              # Check all workflows (dry-run)
gh aw fix --write                      # Fix all workflows
gh aw fix my-workflow                  # Check specific workflow (dry-run)
gh aw fix my-workflow --write          # Fix specific workflow
gh aw fix --list-codemods              # List available codemods with versions
```

**Options:** `--write` (apply changes to files, defaults to dry-run), `--list-codemods` (show available fixes with version information)

**Available Codemods:**
- `timeout_minutes` → `timeout-minutes` (field rename, v0.1.0)
- `network.firewall` → `sandbox.agent` (migration with value mapping, v0.1.0)
- `on.command` → `on.slash_command` (trigger rename, v0.2.0)

The command runs in dry-run mode by default, showing what would be changed without modifying files. Use `--write` to apply the fixes. All changes preserve formatting, comments, and indentation.

**Integration:** The `fix` command is automatically run before compilation when using `gh aw compile --fix`, and is integrated into the build process via `make fix`.

#### `compile`

Compile Markdown workflows to GitHub Actions YAML. Remote imports are automatically cached in `.github/aw/imports/` for offline compilation.

```bash wrap
gh aw compile                              # Compile all workflows
gh aw compile my-workflow                  # Compile specific workflow
gh aw compile --watch                      # Auto-recompile on changes
gh aw compile --validate --strict          # Schema + strict mode validation
gh aw compile --validate --json            # Validation with JSON output
gh aw compile --fix                        # Run fix command before compilation
gh aw compile --zizmor                     # Security scan (warnings)
gh aw compile --strict --zizmor            # Security scan (fails on findings)
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --purge                      # Remove orphaned .lock.yml files
```

**Options:** `--validate` (schema validation and container checks), `--strict` (strict mode validation for all workflows), `--fix` (run `gh aw fix --write` before compiling), `--zizmor` (security scanning with [zizmor](https://github.com/woodruffw/zizmor)), `--dependabot` (generate npm/pip/Go manifests and update dependabot.yml), `--json` (machine-readable JSON output), `--watch` (auto-recompile on changes), `--purge` (remove orphaned `.lock.yml` files)

**Strict Mode (`--strict`):**

Enhanced security validation for production workflows. Enforces: (1) no write permissions - use [safe-outputs](/gh-aw/reference/safe-outputs/) instead, (2) explicit `network` configuration required, (3) no wildcard `*` in `network.allowed` domains, (4) network configuration required for custom MCP servers with containers, (5) GitHub Actions pinned to commit SHAs, (6) no deprecated frontmatter fields. The CLI flag applies to all workflows and takes precedence over individual workflow `strict` frontmatter fields.

**Example:**
```bash wrap
gh aw compile --strict                 # Enable strict mode for all workflows
gh aw compile --strict --zizmor        # Strict mode with security scanning
gh aw compile --validate --strict      # Validate schema and enforce strict mode
```

**Agentic campaign specs and generated workflows:** When agentic campaign spec files exist under `.github/workflows/*.campaign.md`, `gh aw compile` validates those specs (including referenced `workflows`) and fails if problems are found. By default, `compile` also synthesizes coordinator workflows for each valid spec that has meaningful details (e.g., `go-file-size-reduction.campaign.md` → `go-file-size-reduction.campaign.g.md` and `go-file-size-reduction.campaign.launcher.g.md`) and compiles each one to a corresponding `.lock.yml` file. Coordinator workflows are only generated when the agentic campaign spec includes tracker labels, workflows, memory paths, a metrics glob, or governance settings. See the [`campaign` command](#campaign) for management and inspection.

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
gh aw run workflow                          # Run workflow
gh aw run workflow1 workflow2               # Run multiple workflows
gh aw run workflow --repeat 3               # Repeat execution 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
```

**Options:** `--repeat N` (execute N times), `--use-local-secrets` (temporarily push AI engine secrets from environment variables, then clean up)

:::note[Codespaces Permissions]
The `gh aw run` command will fail in GitHub Codespaces by default because the Codespaces GitHub token does not have `workflows:write` permission.

**Solutions:**

1. **Configure custom permissions in devcontainer.json:**
   ```json
   {
     "customizations": {
       "codespaces": {
         "repositories": {
           "owner/repo": {
             "permissions": {
               "actions": "write",
               "workflows": "write"
             }
           }
         }
       }
     }
   }
   ```
   Learn more: [Managing repository access for your codespaces](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces)

2. **Clear GH_TOKEN and authenticate manually:**
   ```bash
   unset GH_TOKEN
   gh auth login
   ```
   This allows you to authenticate with a token that has the required permissions.
:::

### Monitoring

#### `status`

Show status of all workflows in the repository.

```bash wrap
gh aw status                                # Show all workflow status
gh aw status --ref main                     # Show status with latest run info for main branch
gh aw status --json --ref feature-branch    # JSON output with run status for specific branch
gh aw status --label automation             # Filter workflows by label
gh aw status --repo owner/other-repo        # Check status in different repository
```

Lists all agentic workflows with their current state, enabled/disabled status, schedules, labels, and configurations. When `--ref` is specified, includes the latest run status and conclusion for each workflow on that branch or tag.

**Options:** `--ref` (filter by branch or tag, shows latest run status and conclusion), `--label` (filter workflows by label, case-insensitive match), `--json` (output in JSON format), `--repo owner/repo` (check workflow status in specific repository, defaults to current)

#### `logs`

Download and analyze workflow execution logs. Downloads logs, analyzes tool usage and network patterns, and caches results for faster subsequent runs (~10-100x speedup). Overview table includes errors, warnings, missing tools, and noop messages.

```bash wrap
gh aw logs                                 # Download logs for all workflows
gh aw logs workflow                        # Download logs for specific workflow
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --ref main                      # Filter logs by branch or tag
gh aw logs --ref v1.0.0 --parse --json    # Generate markdown + JSON output for specific tag
gh aw logs --campaign                      # Filter to only campaign orchestrator workflows
gh aw logs workflow --repo owner/repo      # Download logs from specific repository
```

**Options:** `-c, --count N` (limit number of runs), `-e, --engine` (filter by AI engine like `-e copilot`), `--campaign` (filter to only campaign orchestrator workflows), `--start-date` (filter runs from date like `--start-date -1w`), `--end-date` (filter runs until date like `--end-date -1d`), `--ref` (filter by branch or tag like `--ref main` or `--ref v1.0.0`), `--parse` (generate `log.md` and `firewall.md`), `--json` (output structured metrics), `--repo owner/repo` (download logs from specific repository)

#### `audit`

Investigate and analyze specific workflow runs. Provides detailed analysis including overview, execution metrics, tool usage patterns, MCP server failures, firewall analysis, noop messages, and artifact information. Accepts run IDs or URLs from any repository and GitHub instance. JSON output includes parsed noop messages similar to missing-tool reports. Automatically detects GitHub Copilot agent runs and uses specialized log parsing to extract agent-specific metrics including turns, tool calls, errors, and token usage.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By URL
gh aw audit 12345678 --parse                              # Parse logs to markdown
```

### Agentic campaigns

#### `campaign`

Inspect first-class agentic campaign definitions declared in .github/workflows/*.campaign.md

For safe scaling and incremental discovery, campaign specs support:

- `cursor-glob`: durable cursor/checkpoint location in repo-memory.
- `governance.max-discovery-items-per-run`: maximum items processed during discovery.
- `governance.max-discovery-pages-per-run`: maximum pages fetched during discovery.

See the [Agentic campaigns guide](/gh-aw/guides/campaigns/) for the full spec shape and recommended defaults.

```bash wrap
gh aw campaign                         # List all agentic campaigns
gh aw campaign security                # Filter by ID or name substring
gh aw campaign --json                  # JSON output

gh aw campaign status                  # Live status for all agentic campaigns
gh aw campaign status incident         # Filter by ID or name substring
gh aw campaign status --json           # JSON status output

gh aw campaign new my-campaign-id      # Scaffold a new agentic campaign spec
gh aw campaign validate                # Validate agentic campaign specs (fails on problems)
gh aw campaign validate --no-strict    # Report problems without failing
```

**Alternative approach**: For a low-code/no-code method, use the "🚀 Start an Agentic Campaign" issue form in the GitHub UI. The form captures campaign intent with structured fields and can trigger an agent to scaffold the spec file automatically. See the [Getting Started guide](/gh-aw/guides/campaigns/getting-started/#start-an-agentic-campaign-with-github-issue-forms) for details.

### Management

#### `enable`

Enable workflows for execution with pattern matching support for bulk operations.

```bash wrap
gh aw enable                                # Enable all workflows
gh aw enable prefix                         # Enable workflows matching prefix
gh aw enable ci-*                          # Enable workflows with pattern
gh aw enable workflow --repo owner/repo     # Enable in specific repository
```

**Options:** `--repo owner/repo` (enable workflows in specific repository, defaults to current)

#### `disable`

Disable workflows to prevent execution and cancel any currently running workflow jobs.

```bash wrap
gh aw disable                               # Disable all workflows
gh aw disable prefix                        # Disable workflows matching prefix
gh aw disable ci-*                         # Disable workflows with pattern
gh aw disable workflow --repo owner/repo    # Disable in specific repository
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
gh aw mcp list workflow                    # List servers for specific workflow
gh aw mcp list-tools <mcp-server>          # List tools for specific MCP server
gh aw mcp list-tools <mcp-server> workflow # List tools in specific workflow
gh aw mcp inspect workflow                 # Inspect and test servers (auto-detects safe-inputs)
gh aw mcp add                              # Add servers from registry
```

The `mcp inspect` command automatically detects and inspects safe-inputs defined in workflows, including those imported from shared workflows. No additional flag is needed - safe-inputs are inspected alongside other MCP servers when present.

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

Run an MCP server exposing gh-aw commands as tools for integration with MCP-compatible applications.

```bash wrap
gh aw mcp-server              # Run with stdio transport
gh aw mcp-server --port 8080  # Run HTTP server on port 8080
gh aw mcp-server --cmd ./gh-aw # Use custom gh-aw binary
```

**Options:** `--port` (run HTTP server with SSE transport), `--cmd` (path to gh-aw binary)

**Available Tools:** status, compile, logs, audit, mcp-inspect, add, update

The server spawns subprocess calls for each tool invocation to ensure GitHub tokens and secrets are not shared with the MCP server process.

See **[MCP Server Guide](/gh-aw/setup/mcp-server/)** for integration details.

### Utility Commands

#### `version`

Show gh aw extension version information.

```bash wrap
gh aw version
```

Displays the current version of gh-aw and product information. Equivalent to using the `--version` flag.

## Shell Completions

gh-aw provides shell completion for bash, zsh, fish, and PowerShell. Completions enable tab completion for workflow names, engine names, and directory paths.

### Setting Up Completions

**Bash:**
```bash wrap
gh aw completion bash > /etc/bash_completion.d/gh-aw
# Or for local user installation:
gh aw completion bash > ~/.bash_completion.d/gh-aw
source ~/.bash_completion.d/gh-aw
```

**Zsh:**
```bash wrap
gh aw completion zsh > "${fpath[1]}/_gh-aw"
# Then restart your shell or run:
compinit
```

**Fish:**
```bash wrap
gh aw completion fish > ~/.config/fish/completions/gh-aw.fish
```

**PowerShell:**
```powershell
gh aw completion powershell | Out-String | Invoke-Expression
# For permanent setup, add to your PowerShell profile
```

### What Gets Completed

| Context | Completion Type |
|---------|-----------------|
| `gh aw compile <TAB>` | Workflow names from `.github/workflows/` |
| `gh aw run <TAB>` | Workflow names |
| `gh aw logs <TAB>` | Workflow names |
| `gh aw status <TAB>` | Workflow names |
| `gh aw enable/disable <TAB>` | Workflow names |
| `gh aw mcp inspect <TAB>` | Workflow names |
| `gh aw mcp list-tools <TAB>` | Common MCP server names |
| `gh aw mcp list-tools github <TAB>` | Workflow names |
| `--engine <TAB>` | Engine names (copilot, claude, codex, custom) |
| `--dir <TAB>` | Directory paths |
| `--output <TAB>` | Directory paths |

## Debug Logging

Enable detailed debugging output for troubleshooting. Shows namespace, message, and time diff (e.g., `+50ms`). Zero overhead when disabled. Supports pattern matching with wildcards.

```bash wrap
DEBUG=* gh aw compile                # All debug logs
DEBUG=cli:* gh aw compile            # CLI operations only
DEBUG=cli:*,workflow:* gh aw compile # Multiple packages
DEBUG=*,-tests gh aw compile         # All except tests
```

**Tip:** Use `--verbose` flag for user-facing details instead of DEBUG environment variable.

## Smart Features

### Fuzzy Workflow Name Matching

When a workflow name is not found, the CLI automatically suggests similar workflow names using fuzzy matching. This helps catch typos and provides helpful suggestions.

```bash wrap
gh aw compile audti-workflows
# ✗ workflow file not found: .github/workflows/audti-workflows.md
#
# Suggestions:
#   • Did you mean: audit-workflows?
#   • Run 'gh aw status' to see all available workflows
#   • Check for typos in the workflow name
```

Fuzzy matching works across all commands that accept workflow names:
- `gh aw compile <workflow>`
- `gh aw enable <workflow>`
- `gh aw disable <workflow>`
- `gh aw logs <workflow>`
- `gh aw mcp list <workflow>`
- `gh aw mcp add <workflow> <server>`
- `gh aw mcp list-tools <server> <workflow>`
- `gh aw mcp inspect <workflow>`

The feature uses Levenshtein distance to suggest up to 3 similar names within an edit distance of 3 characters.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `command not found: gh` | Install from [cli.github.com](https://cli.github.com/) |
| `extension not found: aw` | Run `gh extension install githubnext/gh-aw` |
| Compilation fails with YAML errors | Check indentation, colons, and array syntax in frontmatter |
| Workflow not found | Check typo suggestions or run `gh aw status` to list available workflows |
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
