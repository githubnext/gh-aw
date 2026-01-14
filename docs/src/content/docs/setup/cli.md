---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
sidebar:
  order: 200
---

The `gh aw` CLI extension enables developers to create, manage, and execute AI-powered workflows directly from the command line. It transforms natural language markdown files into GitHub Actions.

## 🚀 Most Common Commands

95% of users only need these 5 commands:

> [!TIP]
> New to gh-aw?
> Start here! These commands cover the essential workflow lifecycle from setup to monitoring.

| Command | When to Use | Details |
|---------|-------------|---------|
| **`gh aw init`** | Set up your repository for agentic workflows | [→ Documentation](#init) |
| **`gh aw add (workflow)`** | Add workflows from The Agentics collection or other repositories | [→ Documentation](#add) |
| **`gh aw status`** | Check current state of all workflows | [→ Documentation](#status) |
| **`gh aw compile`** | Convert markdown to GitHub Actions YAML | [→ Documentation](#compile) |
| **`gh aw run (workflow)`** | Execute workflows immediately in GitHub Actions | [→ Documentation](#run) |

**Complete command reference below** ↓

## Common Workflows for Beginners

### After creating a new workflow

```bash wrap
gh aw compile my-workflow           # Validate markdown and generate .lock.yml
gh aw run my-workflow                # Test it manually (requires workflow_dispatch)
gh aw run my-workflow --push         # Auto-commit, push, and run (all-in-one)
gh aw logs my-workflow               # Download and analyze execution logs
```

### Troubleshooting

```bash wrap
gh aw status                    # Check workflow state and configuration
gh aw logs my-workflow          # Review execution logs (AI decisions, tool usage, errors)
gh aw audit (run-id-or-url)     # Analyze specific run in detail

# Fix issues
gh aw secrets bootstrap --engine copilot   # Check token configuration
gh aw compile my-workflow --validate       # Detailed validation
gh aw fix my-workflow --write              # Auto-fix deprecated fields
```

The audit command accepts run IDs, workflow URLs, job URLs, or step URLs:
- Run ID from URL: `github.com/owner/repo/actions/runs/12345678` → `12345678`
- Or use the full URL: `https://github.com/owner/repo/actions/runs/12345678`
- Job URL: `https://github.com/owner/repo/actions/runs/123/job/456` (extracts first failing step)
- Step URL: `https://github.com/owner/repo/actions/runs/123/job/456#step:7:1` (extracts specific step)

## Installation

Install the GitHub CLI extension:

```bash wrap
gh extension install githubnext/gh-aw
```

### Pinning to a Specific Version

Pin to specific versions for production environments, team consistency, or avoiding breaking changes:

```bash wrap
gh extension install githubnext/gh-aw@v0.1.0          # Pin to release tag
gh extension install githubnext/gh-aw@abc123def456    # Pin to commit SHA
gh aw version                                         # Check current version

# Upgrade pinned version
gh extension remove gh-aw
gh extension install githubnext/gh-aw@v0.2.0
```

### Alternative: Standalone Installer

Use the standalone installer if extension installation fails (common in Codespaces or with auth issues):

```bash wrap
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash                # Latest
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash -s v0.1.0      # Pinned
```

Installs to `~/.local/share/gh/extensions/gh-aw/gh-aw` and works with all `gh aw` commands. Supports Linux, macOS, FreeBSD, and Windows.

### GitHub Enterprise Server Support

Configure for GitHub Enterprise Server deployments:

```bash wrap
export GH_HOST="github.enterprise.com"                           # Set hostname
gh auth login --hostname github.enterprise.com                   # Authenticate
gh aw logs workflow --repo github.enterprise.com/owner/repo      # Use with commands
```

## Global Options

| Flag | Description |
|------|-------------|
| `-h`, `--help` | Show help (`gh aw help [command]` for command-specific help) |
| `-v`, `--verbose` | Enable verbose output with debugging details |

## Commands

Commands are organized by workflow lifecycle: creating, building, testing, monitoring, and managing workflows.

### Getting Workflows

#### `init`

Initialize repository for agentic workflows. Configures `.gitattributes`, Copilot instructions, prompt files, and logs `.gitignore`. Enables MCP server integration by default (use `--no-mcp` to skip).

```bash wrap
gh aw init                              # With MCP integration (default)
gh aw init --no-mcp                     # Skip MCP server integration
gh aw init --tokens --engine copilot    # Check Copilot token configuration
gh aw init --codespaces                 # Configure devcontainer for current repo
gh aw init --codespaces repo1,repo2     # Configure devcontainer for additional repos
gh aw init --campaign                   # Enable campaign functionality
gh aw init --completions                # Install shell completions
```

**Options:** `--no-mcp`, `--tokens`, `--engine` (copilot, claude, codex), `--codespaces`, `--campaign`, `--completions`

#### `add`

Add workflows from The Agentics collection or other repositories to `.github/workflows`.

```bash wrap
gh aw add githubnext/agentics/ci-doctor           # Add single workflow
gh aw add "githubnext/agentics/ci-*"             # Add multiple with wildcards
gh aw add ci-doctor --dir shared --number 3      # Organize in subdirectories with copies
gh aw add ci-doctor --create-pull-request        # Create PR instead of commit
```

**Options:** `--dir`, `--number`, `--create-pull-request` (or `--pr`), `--no-gitattributes`

#### `new`

Create a workflow template in `.github/workflows/`. Opens for editing automatically.

```bash wrap
gh aw new                      # Interactive mode
gh aw new my-custom-workflow   # Create template (.md extension optional)
gh aw new my-workflow --force  # Overwrite if exists
```

#### `secrets`

Manage GitHub Actions secrets and tokens.

##### `secrets set`

Create or update a repository secret (from stdin, flag, or environment variable).

```bash wrap
gh aw secrets set MY_SECRET                                    # From stdin
gh aw secrets set MY_SECRET --value "secret123"                # From flag
gh aw secrets set MY_SECRET --value-from-env MY_TOKEN          # From env var
```

**Options:** `--owner`, `--repo`, `--value`, `--value-from-env`, `--api-url`

##### `secrets bootstrap`

Check token configuration and print setup instructions for missing secrets (read-only).

```bash wrap
gh aw secrets bootstrap --engine copilot   # Check Copilot tokens
gh aw secrets bootstrap --engine claude    # Check Claude tokens
```

**Options:** `--engine` (copilot, claude, codex), `--owner`, `--repo`

See [GitHub Tokens reference](/gh-aw/reference/tokens/) for details.

### Building

#### `fix`

Auto-fix deprecated workflow fields using codemods. Runs in dry-run mode by default; use `--write` to apply changes.

```bash wrap
gh aw fix                              # Check all workflows (dry-run)
gh aw fix --write                      # Fix all workflows
gh aw fix my-workflow --write          # Fix specific workflow
gh aw fix --list-codemods              # List available codemods
```

**Options:** `--write`, `--list-codemods`

Available codemods: `timeout_minutes` → `timeout-minutes`, `network.firewall` → `sandbox.agent`, `on.command` → `on.slash_command`

#### `compile`

Compile Markdown workflows to GitHub Actions YAML. Remote imports cached in `.github/aw/imports/`. Validates campaign specs and generates coordinator workflows when present.

```bash wrap
gh aw compile                              # Compile all workflows
gh aw compile my-workflow                  # Compile specific workflow
gh aw compile --watch                      # Auto-recompile on changes
gh aw compile --validate --strict          # Schema + strict mode validation
gh aw compile --fix                        # Run fix before compilation
gh aw compile --zizmor                     # Security scan (warnings)
gh aw compile --strict --zizmor            # Security scan (fails on findings)
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --purge                      # Remove orphaned .lock.yml files
```

**Options:** `--validate`, `--strict`, `--fix`, `--zizmor`, `--dependabot`, `--json`, `--watch`, `--purge`

**Strict Mode (`--strict`):** Enforces security best practices: no write permissions (use [safe-outputs](/gh-aw/reference/safe-outputs/)), explicit `network` config, no wildcard domains, pinned Actions, no deprecated fields. See [Strict Mode reference](/gh-aw/reference/frontmatter/#strict-mode-strict).

**Shared Workflows:** Workflows without an `on` field are automatically detected as shared workflow components intended for import by other workflows. These files are validated using a relaxed schema that permits optional markdown content and skip compilation with an informative message. To use a shared workflow, import it in another workflow's frontmatter or with markdown directives. See [Imports reference](/gh-aw/reference/imports/).

### Testing

#### `trial`

Test workflows in temporary private repositories (default) or run directly in specified repository (`--repo`). Results saved to `trials/`.

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test remote workflow
gh aw trial ./workflow.md --use-local-secrets      # Test with local API keys
gh aw trial ./workflow.md --logical-repo owner/repo # Act as different repo
gh aw trial ./workflow.md --repo owner/repo        # Run directly in repository
```

**Options:** `-e`, `--engine`, `--auto-merge-prs`, `--repeat`, `--delete-host-repo-after`, `--use-local-secrets`, `--logical-repo`, `--clone-repo`, `--trigger-context`, `--repo`

#### `run`

Execute workflows immediately in GitHub Actions. Displays workflow URL for tracking.

```bash wrap
gh aw run workflow                          # Run workflow
gh aw run workflow1 workflow2               # Run multiple workflows
gh aw run workflow --repeat 3               # Repeat 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
gh aw run workflow --push                   # Auto-commit, push, and dispatch workflow
gh aw run workflow --push --ref main        # Push to specific branch
```

**Options:** `--repeat`, `--use-local-secrets`, `--push`, `--ref`

##### `--push` Flag

The `--push` flag automatically handles workflow updates before execution:

1. **Auto-recompilation**: Detects when `.lock.yml` is outdated and recompiles workflow
2. **Transitive imports**: Collects and stages all imported files recursively
3. **Confirmation prompt**: Interactive confirmation dialog before commit
4. **Smart staging**: Stages workflow `.md` and `.lock.yml` files plus dependencies
5. **Automatic commit**: Creates commit with message "Updated agentic workflow"
6. **Branch validation**: Verifies specified branch with `--ref` exists before pushing
7. **Workflow dispatch**: Triggers workflow run after successful push

When `--push` is not used, warnings are displayed for missing or outdated lock files.

> [!NOTE]
> Codespaces Permissions
> Requires `workflows:write` permission. In Codespaces, either configure custom permissions in `devcontainer.json` ([docs](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces)) or authenticate manually: `unset GH_TOKEN && gh auth login`

### Monitoring

#### `status`

List workflows with state, enabled/disabled status, schedules, and labels. With `--ref`, includes latest run status.

```bash wrap
gh aw status                                # All workflows
gh aw status --ref main                     # With run info for main branch
gh aw status --label automation             # Filter by label
gh aw status --repo owner/other-repo        # Check different repository
```

**Options:** `--ref`, `--label`, `--json`, `--repo`

#### `logs`

Download and analyze logs with tool usage, network patterns, errors, warnings. Results cached for ~10-100x speedup on subsequent runs.

```bash wrap
gh aw logs workflow                        # Download logs for workflow
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --ref main --parse --json      # With markdown/JSON output for branch
gh aw logs --campaign                      # Campaign orchestrators only
```

**Options:** `-c`, `--count`, `-e`, `--engine`, `--campaign`, `--start-date`, `--end-date`, `--ref`, `--parse`, `--json`, `--repo`

#### `audit`

Analyze specific runs with overview, metrics, tool usage, MCP failures, firewall analysis, noops, and artifacts. Accepts run IDs, workflow run URLs, job URLs, and step-level URLs. Auto-detects Copilot agent runs for specialized parsing.

When provided with a job URL, automatically extracts logs for the specific job. When a step fragment is included, extracts only that step's output. If no step is specified, automatically identifies and extracts the first failing step.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By workflow run URL
gh aw audit https://github.com/owner/repo/actions/runs/123/job/456 # By job URL (extracts first failing step)
gh aw audit https://github.com/owner/repo/actions/runs/123/job/456#step:7:1 # By step URL (extracts specific step)
gh aw audit 12345678 --parse                              # Parse logs to markdown
```

Logs are saved to `logs/run-{id}/` with filenames indicating the extraction level (job logs, specific step, or first failing step).

### Agentic campaigns

#### `campaign`

Manage campaign definitions in `.github/workflows/*.campaign.md`. Supports cursor-based discovery, governance controls for safe scaling.

```bash wrap
gh aw campaign                         # List all campaigns
gh aw campaign security                # Filter by ID/name
gh aw campaign status                  # Live status
gh aw campaign new my-campaign-id      # Scaffold new spec
gh aw campaign validate                # Validate specs (fails on problems)
```

See [Agentic campaigns guide](/gh-aw/guides/campaigns/) for full spec and defaults. Alternative: create an issue with the `create-agentic-campaign` label to trigger automated campaign creation ([docs](/gh-aw/guides/campaigns/getting-started/#automated-campaign-creation)).

### Management

#### `enable` / `disable`

Enable or disable workflows with pattern matching support. Disable also cancels running jobs.

```bash wrap
gh aw enable                                # Enable all
gh aw enable ci-*                          # Enable with pattern
gh aw disable workflow --repo owner/repo    # Disable in specific repo
```

**Options:** `--repo`

#### `remove`

Remove workflows (both `.md` and `.lock.yml`).

```bash wrap
gh aw remove my-workflow
```

#### `update`

Update workflows based on `source` field (`owner/repo/path@ref`). Default replaces local file; `--merge` performs 3-way merge. Semantic versions update within same major version.

```bash wrap
gh aw update                              # Update all with source field
gh aw update ci-doctor --merge            # Update with 3-way merge
gh aw update ci-doctor --major --force    # Allow major version updates
```

**Options:** `--dir`, `--merge`, `--major`, `--force`

### Advanced

#### `mcp`

Manage MCP (Model Context Protocol) servers in workflows. `mcp inspect` auto-detects safe-inputs.

```bash wrap
gh aw mcp list workflow                    # List servers for workflow
gh aw mcp list-tools <mcp-server>          # List tools for server
gh aw mcp inspect workflow                 # Inspect and test servers
gh aw mcp add                              # Add MCP tool to workflow
```

See [MCPs Guide](/gh-aw/guides/mcps/).

#### `pr transfer`

Transfer pull request to another repository, preserving changes, title, and description.

```bash wrap
gh aw pr transfer <pr-url> --repo target-owner/target-repo
```

#### `mcp-server`

Run MCP server exposing gh-aw commands as tools. Spawns subprocesses to isolate GitHub tokens.

```bash wrap
gh aw mcp-server              # stdio transport
gh aw mcp-server --port 8080  # HTTP server with SSE
```

**Options:** `--port`, `--cmd`
**Available Tools:** status, compile, logs, audit, mcp-inspect, add, update

See [MCP Server Guide](/gh-aw/setup/mcp-server/).

### Utility Commands

#### `version`

Show gh-aw version and product information.

```bash wrap
gh aw version
```

## Shell Completions

Enable tab completion for workflow names, engines, and paths.

```bash wrap
# Bash
gh aw completion bash > ~/.bash_completion.d/gh-aw && source ~/.bash_completion.d/gh-aw

# Zsh
gh aw completion zsh > "${fpath[1]}/_gh-aw" && compinit

# Fish
gh aw completion fish > ~/.config/fish/completions/gh-aw.fish

# PowerShell
gh aw completion powershell | Out-String | Invoke-Expression
```

Completes workflow names, engine names (copilot, claude, codex), and directory paths.

## Debug Logging

Enable detailed debugging with namespace, message, and time diffs. Zero overhead when disabled.

```bash wrap
DEBUG=* gh aw compile                # All logs
DEBUG=cli:* gh aw compile            # CLI only
DEBUG=*,-tests gh aw compile         # All except tests
```

Use `--verbose` flag for user-facing details instead of DEBUG.

## Smart Features

### Fuzzy Workflow Name Matching

Auto-suggests similar workflow names on typos using Levenshtein distance (up to 3 suggestions, edit distance ≤ 3).

```bash wrap
gh aw compile audti-workflows
# ✗ workflow file not found
# Did you mean: audit-workflows?
```

Works with: compile, enable, disable, logs, mcp commands.

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
