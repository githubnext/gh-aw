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
| **`gh aw add (workflow)`** | Add workflows from The Agentics collection or other repositories | [→ Documentation](#add) |
| **`gh aw status`** | Check current state of all workflows | [→ Documentation](#status) |
| **`gh aw compile`** | Convert markdown to GitHub Actions YAML | [→ Documentation](#compile) |
| **`gh aw run (workflow)`** | Execute workflows immediately in GitHub Actions | [→ Documentation](#run) |

**Complete command reference below** ↓

## Common Workflows

### After creating a workflow
```bash wrap
gh aw compile my-workflow      # Validate and compile
gh aw run my-workflow           # Test execution
gh aw logs my-workflow          # Review logs
```

### Debugging
```bash wrap
gh aw status                    # Check workflow state
gh aw logs my-workflow          # Recent execution logs
gh aw audit <run-id>            # Deep analysis (run-id from Actions tab URL)
```

### Fixing issues
```bash wrap
gh aw secrets bootstrap --engine copilot   # Check tokens
gh aw compile my-workflow --validate       # Validate syntax
gh aw fix my-workflow --write              # Auto-fix deprecated fields
```

## Installation

Install the GitHub CLI extension:

```bash wrap
gh extension install githubnext/gh-aw              # Latest version
gh extension install githubnext/gh-aw@v0.1.0       # Pin to specific version
```

Pin to specific versions for production environments, team consistency, or to avoid breaking changes. Check your version with `gh aw version`.

### Standalone Installer

If extension installation fails (common in Codespaces or with auth issues):

```bash wrap
# Install latest
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash

# Install specific version
curl -sL https://raw.githubusercontent.com/githubnext/gh-aw/main/install-gh-aw.sh | bash -s v0.1.0
```

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

Initialize repository for agentic workflows—configures `.gitattributes`, Copilot instructions, prompt files, and MCP server integration (use `--no-mcp` to skip MCP).

```bash wrap
gh aw init         # Full setup with MCP
gh aw init --no-mcp # Skip MCP integration
```

#### `add`

Add workflows from The Agentics collection or other repositories.

```bash wrap
gh aw add githubnext/agentics/ci-doctor           # Add single workflow
gh aw add "githubnext/agentics/ci-*"             # Add multiple with wildcards
gh aw add ci-doctor --dir shared --number 3      # Organize in subdirectory with copies
gh aw add ci-doctor --pr                         # Create pull request
```

**Options:** `--dir`, `--number`, `--pr`, `--no-gitattributes`

#### `new`

Create a new workflow with template configuration and auto-open for editing.

```bash wrap
gh aw new                      # Interactive mode
gh aw new my-workflow          # Create template
gh aw new my-workflow --force  # Overwrite existing
```

#### `secrets`

Manage GitHub Actions secrets and tokens.

##### `secrets set`

Create or update secrets from stdin, flags, or environment variables.

```bash wrap
gh aw secrets set MY_SECRET                                    # Prompt for value
gh aw secrets set MY_SECRET --value "secret123"                # From flag
gh aw secrets set MY_SECRET --value-from-env MY_TOKEN          # From env var
```

**Options:** `--owner`, `--repo`, `--value`, `--value-from-env`, `--api-url`

##### `secrets bootstrap`

Check token configuration and print setup instructions for missing secrets.

```bash wrap
gh aw secrets bootstrap --engine copilot   # Check Copilot tokens
gh aw secrets bootstrap --engine claude    # Check Claude tokens
```

**Options:** `--engine` (copilot/claude/codex), `--owner`, `--repo`

See [GitHub Tokens reference](/gh-aw/reference/tokens/) for details.

### Building

#### `fix`

Automatically fix deprecated fields using codemods (dry-run by default).

```bash wrap
gh aw fix                              # Check all (dry-run)
gh aw fix --write                      # Fix all workflows
gh aw fix my-workflow --write          # Fix specific workflow
gh aw fix --list-codemods              # List available fixes
```

**Options:** `--write`, `--list-codemods`

Applies migrations like `timeout_minutes` → `timeout-minutes`, `network.firewall` → `sandbox.agent`, and `on.command` → `on.slash_command`.

#### `compile`

Compile Markdown workflows to GitHub Actions YAML. Remote imports cached in `.github/aw/imports/`.

```bash wrap
gh aw compile                              # Compile all workflows
gh aw compile my-workflow                  # Compile specific workflow
gh aw compile --watch                      # Auto-recompile on changes
gh aw compile --validate --strict          # Schema + strict validation
gh aw compile --fix                        # Auto-fix then compile
gh aw compile --zizmor                     # Security scan
gh aw compile --dependabot                 # Generate dependency manifests
gh aw compile --purge                      # Remove orphaned .lock.yml files
```

**Options:** `--validate`, `--strict`, `--fix`, `--zizmor`, `--dependabot`, `--json`, `--watch`, `--purge`

**Strict Mode:** Enforces production-ready security (no write permissions, explicit network config, no wildcards, pinned Actions, no deprecated fields). CLI flag overrides per-workflow settings. See [Strict Mode reference](/gh-aw/reference/frontmatter/#strict-mode-strict) and [Security Guide](/gh-aw/guides/security/#strict-mode-validation).

**Agentic Campaigns:** Validates `*.campaign.md` specs and synthesizes coordinator workflows when specs include tracker labels, workflows, memory paths, metrics, or governance. See [`campaign` command](#campaign).

### Testing

#### `trial`

Test workflows safely in temporary private repositories or run directly in a target repository.

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test remote workflow
gh aw trial ./workflow.md                          # Test local workflow
gh aw trial ./workflow.md --use-local-secrets      # Use local API keys
gh aw trial ./workflow.md --repo owner/repo        # Run directly in repository
```

**Options:** `-e`, `--auto-merge-prs`, `--repeat`, `--delete-host-repo-after`, `--use-local-secrets`, `--logical-repo`, `--clone-repo`, `--trigger-context`, `--repo`

Results saved to `trials/` directory.

#### `run`

Execute workflows immediately in GitHub Actions.

```bash wrap
gh aw run workflow                          # Run workflow
gh aw run workflow1 workflow2               # Run multiple
gh aw run workflow --repeat 3               # Repeat 3 times
gh aw run workflow --use-local-secrets      # Use local API keys
```

**Options:** `--repeat`, `--use-local-secrets`

:::note[Codespaces Permissions]
Requires `workflows:write` permission. In Codespaces, either configure custom permissions in `devcontainer.json` or run `unset GH_TOKEN && gh auth login`. See [Managing repository access](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces).
:::

### Monitoring

#### `status`

Show workflow status, state, schedules, and configurations.

```bash wrap
gh aw status                                # All workflows
gh aw status --ref main                     # With latest run info
gh aw status --label automation             # Filter by label
gh aw status --repo owner/other-repo        # Different repository
```

**Options:** `--ref`, `--label`, `--json`, `--repo`

#### `logs`

Download and analyze workflow logs with caching (~10-100x speedup on repeated access).

```bash wrap
gh aw logs                                 # All workflows
gh aw logs workflow                        # Specific workflow
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --ref main --parse --json      # Branch-filtered with output
gh aw logs --campaign                      # Campaign workflows only
```

**Options:** `-c`, `-e`, `--campaign`, `--start-date`, `--end-date`, `--ref`, `--parse`, `--json`, `--repo`

#### `audit`

Deep analysis of specific workflow runs—metrics, tool usage, MCP failures, firewall activity, and artifacts. Accepts run IDs or URLs. Auto-detects Copilot agent runs for specialized parsing.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By URL
gh aw audit 12345678 --parse                              # Generate markdown
```

### Agentic campaigns

#### `campaign`

Manage agentic campaign specs in `.github/workflows/*.campaign.md`. Supports safe scaling with cursor-based checkpointing and governance controls.

```bash wrap
gh aw campaign                         # List campaigns
gh aw campaign security                # Filter by substring
gh aw campaign status                  # Live status
gh aw campaign new my-campaign-id      # Scaffold new spec
gh aw campaign validate                # Validate specs
```

**Subcommands:** `list`, `status`, `new`, `validate`

See [Agentic campaigns guide](/gh-aw/guides/campaigns/) for full spec documentation and [Getting Started guide](/gh-aw/guides/campaigns/getting-started/) for the issue form approach.

### Management

#### `enable`

Enable workflows with pattern matching for bulk operations.

```bash wrap
gh aw enable                                # All workflows
gh aw enable ci-*                          # Pattern match
gh aw enable workflow --repo owner/repo     # Specific repository
```

#### `disable`

Disable workflows and cancel running jobs.

```bash wrap
gh aw disable                               # All workflows
gh aw disable ci-*                         # Pattern match
```

#### `remove`

Remove workflow `.md` and `.lock.yml` files from repository.

```bash wrap
gh aw remove my-workflow
```

#### `update`

Update workflows based on `source` field. Default replaces with upstream version; `--merge` preserves local changes via 3-way merge.

```bash wrap
gh aw update                              # All with source field
gh aw update ci-doctor --merge            # Preserve local changes
gh aw update ci-doctor --major --force    # Allow major version updates
```

**Options:** `--dir`, `--merge`, `--major`, `--force`

### Advanced

#### `mcp`

Manage MCP (Model Context Protocol) servers—list, inspect, test, and add servers.

```bash wrap
gh aw mcp list                             # List all servers
gh aw mcp list workflow                    # Servers for specific workflow
gh aw mcp list-tools <mcp-server>          # Tools for server
gh aw mcp inspect workflow                 # Inspect and test (auto-detects safe-inputs)
gh aw mcp add                              # Add MCP tool to workflow
```

See [MCPs Guide](/gh-aw/guides/mcps/) for details.

#### `pr`

##### `pr transfer`

Transfer pull request to another repository, preserving changes and metadata.

```bash wrap
gh aw pr transfer <pr-url>
gh aw pr transfer <pr-url> --repo target-owner/target-repo
```

#### `mcp-server`

Run MCP server exposing gh-aw commands as tools. Spawns subprocess calls to isolate tokens/secrets.

```bash wrap
gh aw mcp-server              # stdio transport
gh aw mcp-server --port 8080  # HTTP server with SSE
```

**Available Tools:** status, compile, logs, audit, mcp-inspect, add, update

See [MCP Server Guide](/gh-aw/setup/mcp-server/).

### Utility Commands

#### `version`

Display gh-aw version and product information.

```bash wrap
gh aw version
```

## Shell Completions

Enable tab completion for bash, zsh, fish, and PowerShell.

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

Completes workflow names, engine names (copilot/claude/codex), and directory paths.

## Debug Logging

Enable detailed debugging with `DEBUG` environment variable. Zero overhead when disabled.

```bash wrap
DEBUG=* gh aw compile                # All logs
DEBUG=cli:* gh aw compile            # CLI only
DEBUG=*,-tests gh aw compile         # Exclude tests
```

Use `--verbose` flag for user-facing details.

## Smart Features

### Fuzzy Workflow Name Matching

Auto-suggests similar workflow names using Levenshtein distance (up to 3 suggestions, edit distance ≤ 3).

```bash wrap
gh aw compile audti-workflows
# Suggestions: Did you mean: audit-workflows?
```

Works across all commands accepting workflow names.

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
