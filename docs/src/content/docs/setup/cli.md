---
title: CLI Commands
description: Complete guide to all available CLI commands for managing agentic workflows with the GitHub CLI extension, including installation, compilation, and execution.
sidebar:
  order: 200
---

The `gh aw` CLI extension enables developers to create, manage, and execute AI-powered workflows directly from the command line. It transforms natural language markdown files into GitHub Actions.

## Most Common Commands

| Command | Description |
|---------|-------------|
| [`gh aw init`](#init) | Set up your repository for agentic workflows |
| [`gh aw add-wizard`](#add-wizard) | Add workflows with interactive guided setup |
| [`gh aw add`](#add) | Add workflows from other repositories (non-interactive) |
| [`gh aw compile`](#compile) | Convert markdown to GitHub Actions YAML |
| [`gh aw list`](#list) | Quick listing of all workflows |
| [`gh aw run`](#run) | Execute workflows immediately in GitHub Actions |
| [`gh aw status`](#status) | Check current state of all workflows |

## Installation

Install the GitHub CLI extension:

```bash wrap
gh extension install github/gh-aw
```

### Pinning to a Specific Version

Pin to specific versions for production environments, team consistency, or avoiding breaking changes:

```bash wrap
gh extension install github/gh-aw@v0.1.0          # Pin to release tag
gh extension install github/gh-aw@abc123def456    # Pin to commit SHA
gh aw version                                         # Check current version

# Upgrade pinned version
gh extension remove gh-aw
gh extension install github/gh-aw@v0.2.0
```

### Alternative: Standalone Installer

Use the standalone installer if extension installation fails (common in Codespaces, restricted networks, or with auth issues):

```bash wrap
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash                # Latest
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash -s v0.1.0      # Pinned
```

Installs to `~/.local/share/gh/extensions/gh-aw/gh-aw`. Supports Linux, macOS, FreeBSD, Windows, and Android (Termux). Works behind corporate firewalls using direct release download URLs.

### GitHub Actions Setup Action

Install the CLI in GitHub Actions workflows using the `setup-cli` action with automatic checksum verification and platform detection:

``````yaml wrap
- name: Install gh-aw CLI
  uses: github/gh-aw/actions/setup-cli@main
  with:
    version: v0.37.18
``````

See the [setup-cli action README](https://github.com/github/gh-aw/blob/main/actions/setup-cli/README.md) for complete documentation.

### GitHub Enterprise Server Support

Configure for GitHub Enterprise Server deployments:

```bash wrap
export GH_HOST="github.enterprise.com"                           # Set hostname
gh auth login --hostname github.enterprise.com                   # Authenticate
gh aw logs workflow --repo github.enterprise.com/owner/repo      # Use with commands
```

For GHE Cloud with data residency (`*.ghe.com`), see the dedicated [Debugging GHE Cloud guide](/gh-aw/troubleshooting/debug-ghe/) for setup and troubleshooting steps.

Commands that support `--create-pull-request` (such as `gh aw add`, `gh aw add-wizard`, `gh aw init`, `gh aw update`, and `gh aw upgrade`) automatically detect the enterprise host from the git remote and route PR creation to the correct GHES instance. No extra flags are needed.

`gh aw audit` and `gh aw add-wizard` also auto-detect the GHES host from the git remote, so running them inside a GHES repository works without setting `GH_HOST` manually.

#### Configuring `gh` CLI on GHES

The compiled agent job automatically runs `configure_gh_for_ghe.sh` before the agent starts executing. The script detects the GitHub host from the `GITHUB_SERVER_URL` environment variable (set by GitHub Actions on GHES) and configures `gh` to authenticate against it. No configuration is required for the agent to use `gh` CLI commands on your GHES instance.

Custom workflow jobs (independent GitHub Actions jobs defined in workflow frontmatter) and the safe-outputs job automatically have `GH_HOST` derived from `GITHUB_SERVER_URL` at the start of each job. On github.com this is a no-op; on GHES/GHEC it ensures all `gh` CLI commands in the job target the correct instance without any manual setup.

For custom `steps:` that require additional authentication setup (for example, when running `gh` commands without a `GH_TOKEN` in scope), the helper script is available:

```yaml wrap
steps:
  - name: Configure gh for GHE
    run: source /opt/gh-aw/actions/configure_gh_for_ghe.sh

  - name: Fetch repository data
    env:
      GH_TOKEN: ${{ github.token }}
    run: |
      gh issue list --state open --limit 500 --json number,labels
      gh pr list --state open --limit 200 --json number,title
```

The script is installed to `/opt/gh-aw/actions/configure_gh_for_ghe.sh` by the setup action. When `GH_TOKEN` is already set in the environment, the script skips `gh auth login` and only exports `GH_HOST` — the token handles authentication.

> [!NOTE]
> Custom steps run outside the agent firewall sandbox and have access to standard GitHub Actions environment variables including `GITHUB_SERVER_URL`, `GITHUB_TOKEN`, and `GH_TOKEN`.

## Global Options

| Flag | Description |
|------|-------------|
| `-h`, `--help` | Show help (`gh aw help [command]` for command-specific help) |
| `-v`, `--verbose` | Enable verbose output with debugging details |

### The `--push` Flag

The `run` command supports `--push` to automatically commit and push changes before dispatching the workflow. It stages all changes, commits, and pushes to the remote. Requires a clean working directory.

For `init`, `update`, and `upgrade`, use `--create-pull-request` to create a pull request with the changes instead.

## Commands

Commands are organized by workflow lifecycle: creating, building, testing, monitoring, and managing workflows.

### Getting Workflows

#### `init`

Initialize repository for agentic workflows. Configures `.gitattributes`, creates the dispatcher agent file (`.github/agents/agentic-workflows.agent.md`), and logs `.gitignore`. Enables MCP server integration by default (use `--no-mcp` to skip). Without arguments, enters interactive mode for engine selection and secret configuration.

```bash wrap
gh aw init                              # Interactive mode: select engine and configure secrets
gh aw init --no-mcp                     # Skip MCP server integration
gh aw init --codespaces                 # Configure devcontainer for current repo
gh aw init --codespaces repo1,repo2     # Configure devcontainer for additional repos
gh aw init --completions                # Install shell completions
gh aw init --create-pull-request        # Initialize and open a pull request
```

**Options:** `--no-mcp`, `--codespaces`, `--completions`, `--create-pull-request`

#### `add-wizard`

Add a workflow with interactive guided setup. Checks requirements, adds the markdown file, and generates the compiled YAML. Prompts for missing API keys and secrets.

```bash wrap
gh aw add-wizard githubnext/agentics/ci-doctor           # Interactive setup
gh aw add-wizard https://github.com/org/repo/blob/main/workflows/my-workflow.md
gh aw add-wizard githubnext/agentics/ci-doctor --skip-secret  # Skip secret prompt
```

**Options:** `--skip-secret`, `--dir/-d`, `--engine/-e`, `--no-gitattributes`, `--no-stop-after`, `--stop-after`

#### `add`

Add workflows from The Agentics collection or other repositories to `.github/workflows`.

```bash wrap
gh aw add githubnext/agentics/ci-doctor           # Add single workflow
gh aw add "githubnext/agentics/ci-*"             # Add multiple with wildcards
gh aw add ci-doctor --dir shared                  # Organize in subdirectory
gh aw add ci-doctor --create-pull-request        # Create PR instead of commit
```

**Options:** `--dir/-d`, `--repo/-r`, `--create-pull-request`, `--no-gitattributes`, `--append`, `--disable-security-scanner`, `--engine/-e`, `--force/-f`, `--name/-n`, `--no-stop-after`, `--stop-after`

#### `new`

Create a workflow template in `.github/workflows/`. Opens for editing automatically.

```bash wrap
gh aw new                              # Interactive mode
gh aw new my-custom-workflow           # Create template (.md extension optional)
gh aw new my-workflow --force          # Overwrite if exists
gh aw new my-workflow --engine claude  # Inject engine into frontmatter
```

**Options:** `--force`, `--engine/-e`

When `--engine` is specified, the engine is injected into the generated frontmatter template:

```yaml wrap
---
permissions:
  contents: read
engine: claude
network: defaults
...
```

#### `secrets`

Manage GitHub Actions secrets and tokens.

##### `secrets set`

Create or update a repository secret (from stdin, flag, or environment variable).

```bash wrap
gh aw secrets set MY_SECRET                                    # From stdin (current repo)
gh aw secrets set MY_SECRET --repo myorg/myrepo                # Specify target repo
gh aw secrets set MY_SECRET --value "secret123"                # From flag
gh aw secrets set MY_SECRET --value-from-env MY_TOKEN          # From env var
```

**Options:** `--repo`, `--value`, `--value-from-env`, `--api-url`

##### `secrets bootstrap`

Analyze workflows to determine required secrets and interactively prompt for missing ones. Auto-detects engines in use and validates tokens before uploading to the repository.

```bash wrap
gh aw secrets bootstrap                                  # Analyze all workflows and prompt for missing secrets
gh aw secrets bootstrap --engine copilot                 # Check only Copilot secrets
gh aw secrets bootstrap --non-interactive                # Display missing secrets without prompting
```

**Options:** `--engine` (copilot, claude, codex), `--non-interactive`, `--repo`

See [Authentication](/gh-aw/reference/auth/) for details.

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

Notable codemods include `expires-integer-to-string`, which converts bare integer `expires` values (e.g., `expires: 7`) to the preferred day-string format (e.g., `expires: 7d`) in all `safe-outputs` blocks. Run `gh aw fix --list-codemods` to see all available codemods.

#### `compile`

Compile Markdown workflows to GitHub Actions YAML. Remote imports cached in `.github/aw/imports/`.

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

**Options:** `--validate`, `--strict`, `--fix`, `--zizmor`, `--dependabot`, `--json`, `--watch`, `--purge`, `--stats`

**Error Reporting:** Displays detailed error messages with file paths, line numbers, column positions, and contextual code snippets.

**Dependabot Integration (`--dependabot`):** Generates dependency manifests and `.github/dependabot.yml` by analyzing runtime tools across all workflows. See [Dependabot Support reference](/gh-aw/reference/dependabot/).

**Strict Mode (`--strict`):** Enforces security best practices: no write permissions (use [safe-outputs](/gh-aw/reference/safe-outputs/)), explicit `network` config, no wildcard domains, pinned Actions, no deprecated fields. See [Strict Mode reference](/gh-aw/reference/frontmatter/#strict-mode-strict).

**Shared Workflows:** Workflows without an `on` field are detected as shared components. Validated with relaxed schema and skip compilation. See [Imports reference](/gh-aw/reference/imports/).

#### `validate`

Validate agentic workflows by running the compiler with all linters enabled, without generating lock files. Equivalent to `gh aw compile --validate --no-emit --zizmor --actionlint --poutine`.

```bash wrap
gh aw validate                              # Validate all workflows
gh aw validate my-workflow                  # Validate specific workflow
gh aw validate my-workflow daily            # Validate multiple workflows
gh aw validate --json                       # Output results in JSON format
gh aw validate --strict                     # Enforce strict mode validation
gh aw validate --fail-fast                  # Stop at the first error
gh aw validate --dir custom/workflows       # Validate from custom directory
gh aw validate --engine copilot             # Override AI engine
```

**Options:** `--engine/-e`, `--dir/-d`, `--strict`, `--json/-j`, `--fail-fast`, `--stats`, `--no-check-update`

All linters (`zizmor`, `actionlint`, `poutine`), `--validate`, and `--no-emit` are always-on defaults and cannot be disabled. Accepts the same workflow ID format as `compile`.

### Testing

#### `trial`

Test workflows in temporary private repositories (default) or run directly in specified repository (`--host-repo`). Results saved to `trials/`.

```bash wrap
gh aw trial githubnext/agentics/ci-doctor          # Test remote workflow
gh aw trial ./workflow.md --logical-repo owner/repo # Act as different repo
gh aw trial ./workflow.md --host-repo owner/repo   # Run directly in repository
gh aw trial ./workflow.md --dry-run                # Preview without executing
```

**Options:** `-e/--engine`, `--repeat`, `--delete-host-repo-after`, `--logical-repo/-l`, `--clone-repo`, `--trigger-context`, `--host-repo`, `--dry-run`, `--append`, `--auto-merge-prs`, `--disable-security-scanner`, `--force-delete-host-repo-before`, `--timeout`, `--yes/-y`

**Secret Handling:** API keys required for the selected engine are automatically checked. If missing from the target repository, they are prompted for interactively and uploaded.

#### `run`

Execute workflows immediately in GitHub Actions. Displays workflow URL for tracking.

```bash wrap
gh aw run workflow                          # Run workflow
gh aw run workflow1 workflow2               # Run multiple workflows
gh aw run workflow --repeat 3               # Repeat 3 times
gh aw run workflow --push                   # Auto-commit, push, and dispatch workflow
gh aw run workflow --push --ref main        # Push to specific branch
gh aw run workflow --json                   # Output triggered workflow results as JSON
```

**Options:** `--repeat`, `--push` (see [--push flag](#the---push-flag)), `--ref`, `--enable-if-needed`, `--json/-j`, `--auto-merge-prs`, `--dry-run`, `--engine/-e`, `--raw-field/-F`, `--repo/-r`

When `--json` is set, a JSON array of triggered workflow results is written to stdout.

When `--push` is used, automatically recompiles outdated `.lock.yml` files, stages all transitive imports, and triggers workflow run after successful push. Without `--push`, warnings are displayed for missing or outdated lock files.

> [!NOTE]
> Codespaces Permissions
> Requires `workflows:write` permission. In Codespaces, either configure custom permissions in `devcontainer.json` ([docs](https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces)) or authenticate manually: `unset GH_TOKEN && gh auth login`

### Monitoring

#### `list`

List workflows with basic information (name, engine, compilation status) without checking GitHub Actions state.

```bash wrap
gh aw list                                  # List all workflows
gh aw list ci-                              # Filter by pattern (case-insensitive)
gh aw list --json                           # Output in JSON format
gh aw list --label automation               # Filter by label
gh aw list --dir custom/workflows           # List from a local custom directory
gh aw list --repo owner/repo --path .github/workflows  # List from a remote repository
```

**Options:** `--json`, `--label`, `--dir/-d`, `--path`, `--repo`

Two flags control the workflow directory location, with different purposes:
- `--dir` (`-d`): overrides the **local** workflow directory. Applies only when `--repo` is not set.
- `--path`: specifies the workflow directory path in a **remote** repository. Use together with `--repo`.

Fast enumeration without GitHub API queries. For detailed status including enabled/disabled state and run information, use `status` instead.

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

Download and analyze logs with tool usage, network patterns, errors, warnings. Results cached for 10-100x speedup on subsequent runs.

```bash wrap
gh aw logs workflow                        # Download logs for workflow
gh aw logs -c 10 --start-date -1w         # Filter by count and date
gh aw logs --ref main --parse --json      # With markdown/JSON output for branch
```

With `--json`, the output also includes deterministic lineage data under `.episodes[]` and `.edges[]`. Use these fields to group orchestrated runs into execution episodes instead of reconstructing relationships from `.runs[]` alone.

**Workflow name matching**: The logs command accepts both workflow IDs (kebab-case filename without `.md`, e.g., `ci-failure-doctor`) and display names (from frontmatter, e.g., `CI Failure Doctor`). Matching is case-insensitive for convenience:

```bash wrap
gh aw logs ci-failure-doctor               # Workflow ID
gh aw logs CI-FAILURE-DOCTOR               # Case-insensitive ID
gh aw logs "CI Failure Doctor"             # Display name
gh aw logs "ci failure doctor"             # Case-insensitive display name
```

**Options:** `-c`, `--count`, `-e`, `--engine`, `--start-date`, `--end-date`, `--ref`, `--parse`, `--json`, `--repo`, `--firewall`, `--no-firewall`, `--safe-output`, `--filtered-integrity`, `--after-run-id`, `--before-run-id`, `--no-staged`, `--tool-graph`, `--timeout`

#### `audit`

Analyze specific runs with a rich multi-section report. Accepts run IDs, workflow run URLs, job URLs, and step-level URLs. Auto-detects Copilot coding agent runs for specialized parsing. Job URLs automatically extract specific job logs; step URLs extract specific steps; without step, extracts first failing step.

```bash wrap
gh aw audit 12345678                                      # By run ID
gh aw audit https://github.com/owner/repo/actions/runs/123 # By workflow run URL
gh aw audit https://github.com/owner/repo/actions/runs/123/job/456 # By job URL (extracts first failing step)
gh aw audit https://github.com/owner/repo/actions/runs/123/job/456#step:7:1 # By step URL (extracts specific step)
gh aw audit 12345678 --parse                              # Parse logs to markdown
gh aw audit 12345678 --repo owner/repo                    # Specify repository for bare run ID
```

**Options:** `--parse`, `--json`, `--repo/-r`

The `--repo` flag accepts `owner/repo` format and is required when passing a bare numeric run ID without a full URL, allowing the command to locate the correct repository.

Logs are saved to `logs/run-{id}/` with filenames indicating the extraction level. Pre-agent failures (integrity filtering, missing secrets, binary install) surface the actual error in `failure_analysis.error_summary`. Invalid run IDs return a human-readable error.

**Report sections:**

| Section | Description |
|---------|-------------|
| **Overview** | Run status, duration, trigger event, repository |
| **Engine Configuration** | Engine ID, model, CLI version, firewall version, MCP servers configured |
| **Prompt Analysis** | Prompt size and source file |
| **Session & Agent Performance** | Wall time, turn count, average turn duration, tokens per minute, timeout detection, agent active ratio |
| **MCP Server Health** | Per-server request counts, error rates, average latency, health status, and slowest tool calls |
| **Safe Output Summary** | Total safe output items broken down by type (comments, PRs, issues, etc.) |
| **Metrics** | Tool usage, token consumption, cost |
| **MCP Failures** | Failed MCP tool calls with error details |
| **Firewall Analysis** | Network requests blocked or allowed by the firewall |
| **Jobs** | Status of each GitHub Actions job in the run |
| **Artifacts** | Downloaded artifacts and their contents |

#### `health`

Display workflow health metrics and success rates.

```bash wrap
gh aw health                       # Summary of all workflows (last 7 days)
gh aw health issue-monster         # Detailed metrics for specific workflow
gh aw health --days 30             # Summary for last 30 days
gh aw health --threshold 90        # Alert if below 90% success rate
gh aw health --json                # Output in JSON format
gh aw health issue-monster --days 90  # 90-day metrics for workflow
```

**Options:** `--days`, `--threshold`, `--repo`, `--json`

Shows success/failure rates, trend indicators (↑ improving, → stable, ↓ degrading), execution duration, token usage, costs, and alerts when success rate drops below threshold.

#### `checks`

Classify CI check state for a pull request and emit a normalized result.

```bash wrap
gh aw checks 42                    # Classify checks for PR #42
gh aw checks 42 --repo owner/repo  # Specify repository
gh aw checks 42 --json             # Output in JSON format
```

**Options:** `--repo/-r`, `--json/-j`

Maps PR check rollups to one of the following normalized states: `success`, `failed`, `pending`, `no_checks`, `policy_blocked`. JSON output includes two state fields: `state` (aggregate across all checks) and `required_state` (derived from required checks only, ignoring optional third-party statuses like deployment integrations).

### Management

#### `enable`

Enable one or more workflows by ID, or all workflows if no IDs provided.

```bash wrap
gh aw enable                                # Enable all workflows
gh aw enable ci-doctor                      # Enable specific workflow
gh aw enable ci-doctor daily                # Enable multiple workflows
gh aw enable ci-doctor --repo owner/repo    # Enable in specific repository
```

**Options:** `--repo`

#### `disable`

Disable one or more workflows and cancel any in-progress runs.

```bash wrap
gh aw disable                               # Disable all workflows
gh aw disable ci-doctor                     # Disable specific workflow
gh aw disable ci-doctor daily               # Disable multiple workflows
gh aw disable ci-doctor --repo owner/repo   # Disable in specific repository
```

**Options:** `--repo`

#### `remove`

Remove workflows (both `.md` and `.lock.yml`).

```bash wrap
gh aw remove my-workflow
```

#### `update`

Update workflows based on `source` field (`owner/repo/path@ref`). By default, performs a 3-way merge to preserve local changes; use `--no-merge` to override with upstream. Semantic versions update within same major version.

By default, `update` also force-updates all GitHub Actions referenced in your workflows (both in `actions-lock.json` and workflow files) to their latest major version. Use `--disable-release-bump` to restrict force-updates to core `actions/*` actions only.

If no workflows in the repository contain a `source` field, the command exits gracefully with an informational message rather than an error. This is expected behavior for repositories that have not yet added updatable workflows.

```bash wrap
gh aw update                              # Update all with source field
gh aw update ci-doctor                    # Update specific workflow (3-way merge)
gh aw update ci-doctor --no-merge         # Override local changes with upstream
gh aw update ci-doctor --major --force    # Allow major version updates
gh aw update --disable-release-bump       # Update workflows; only force-update core actions/*
gh aw update --create-pull-request        # Update and open a pull request
```

**Options:** `--dir`, `--no-merge`, `--major`, `--force`, `--engine`, `--no-stop-after`, `--stop-after`, `--disable-release-bump`, `--create-pull-request`, `--no-compile`

#### `upgrade`

Upgrade repository with latest agent files and apply codemods to all workflows.

```bash wrap
gh aw upgrade                              # Upgrade repository agent files and all workflows
gh aw upgrade --no-fix                     # Update agent files only (skip codemods, actions, and compilation)
gh aw upgrade --create-pull-request        # Upgrade and open a pull request
gh aw upgrade --audit                      # Run dependency health audit
gh aw upgrade --audit --json               # Dependency audit in JSON format
```

**Options:** `--dir/-d`, `--no-fix`, `--no-actions`, `--no-compile`, `--create-pull-request`, `--audit`, `--json/-j`

### Advanced

#### `mcp`

Manage MCP (Model Context Protocol) servers in workflows. `mcp inspect` auto-detects mcp-scripts.

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
gh aw mcp-server                      # stdio transport
gh aw mcp-server --port 8080          # HTTP server with SSE
gh aw mcp-server --validate-actor     # Enable actor validation
```

**Options:** `--port` (HTTP server port), `--cmd` (custom subprocess command), `--validate-actor` (enforce actor validation for logs and audit tools)

**Available Tools:** status, compile, logs, audit, mcp-inspect, add, update, fix

When `--validate-actor` is enabled, logs and audit tools require write+ repository access via GitHub API (permissions cached for 1 hour). See [MCP Server Guide](/gh-aw/reference/gh-aw-as-mcp-server/).

#### `domains`

List network domains configured in agentic workflows.

```bash wrap
gh aw domains                           # List all workflows with domain counts
gh aw domains weekly-research           # List domains for specific workflow
gh aw domains --json                    # Output summary in JSON format
gh aw domains weekly-research --json    # Output workflow domains in JSON format
```

**Options:** `--json/-j`

When no workflow is specified, lists all workflows with a summary of allowed and blocked domain counts. When a workflow is specified, lists all effective allowed and blocked domains including domains expanded from ecosystem identifiers (e.g. `node`, `python`, `github`) and engine defaults.

### Utility Commands

#### `version`

Show gh-aw version and product information.

```bash wrap
gh aw version
```

#### `completion`

Generate and manage shell completion scripts for tab completion.

```bash wrap
gh aw completion install              # Auto-detect and install
gh aw completion uninstall            # Remove completions
gh aw completion bash                 # Generate bash script
gh aw completion zsh                  # Generate zsh script
gh aw completion fish                 # Generate fish script
gh aw completion powershell           # Generate powershell script
```

**Subcommands:** `install`, `uninstall`, `bash`, `zsh`, `fish`, `powershell`. See [Shell Completions](#shell-completions).

#### `project`

Create and manage GitHub Projects V2 boards.

##### `project new`

Create a new GitHub Project V2 owned by a user or organization with optional repository linking.

```bash wrap
gh aw project new "My Project" --owner @me                      # Create user project
gh aw project new "Team Board" --owner myorg                    # Create org project
gh aw project new "Bugs" --owner myorg --link myorg/myrepo     # Create and link to repo
```

**Options:**
- `--owner` (required): Project owner - use `@me` for current user or specify organization name
- `--link`: Repository to link project to (format: `owner/repo`)

**Token Requirements:**

> [!IMPORTANT]
> The default `GITHUB_TOKEN` cannot create projects. Use a Personal Access Token (PAT) with Projects permissions:
>
> - **Classic PAT**: `project` scope (user projects) or `project` + `repo` (org projects)
> - **Fine-grained PAT**: Organization permissions → Projects: Read & Write
>
> Configure via `GH_AW_PROJECT_GITHUB_TOKEN` environment variable or `gh auth login`. See [Authentication](/gh-aw/reference/auth/).

#### `hash-frontmatter`

Compute a deterministic SHA-256 hash of workflow frontmatter for detecting configuration changes.

```bash wrap
gh aw hash-frontmatter my-workflow.md
gh aw hash-frontmatter .github/workflows/audit-workflows.md
```

Includes all frontmatter fields, imported workflow frontmatter (BFS traversal), template expressions containing `env.` or `vars.`, and version information (gh-aw, awf, agents).

## Shell Completions

Enable tab completion for workflow names, engines, and paths. After running `gh aw completion install`, restart your shell or source your configuration file.

### Manual Installation

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

## Debug Logging

Enable detailed debugging with namespace, message, and time diffs.

```bash wrap
DEBUG=* gh aw compile                # All logs
DEBUG=cli:* gh aw compile            # CLI only
DEBUG=*,-tests gh aw compile         # All except tests
```

Use `--verbose` flag for user-facing details.

## Smart Features

### Fuzzy Workflow Name Matching

Auto-suggests similar workflow names on typos using Levenshtein distance.

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
| `extension not found: aw` | Run `gh extension install github/gh-aw` |
| Compilation fails with YAML errors | Check indentation, colons, and array syntax in frontmatter |
| Workflow not found | Check typo suggestions or run `gh aw status` to list available workflows |
| Permission denied | Check file permissions or repository access |
| Trial creation fails | Check GitHub rate limits and authentication |

See [Common Issues](/gh-aw/troubleshooting/common-issues/) and [Error Reference](/gh-aw/troubleshooting/errors/) for detailed troubleshooting.

## Related Documentation

- [Quick Start](/gh-aw/setup/quick-start/) - Get your first workflow running
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options
- [Reusing Workflows](/gh-aw/guides/packaging-imports/) - Adding and updating workflows
- [Security Guide](/gh-aw/introduction/architecture/) - Security best practices
- [MCP Server Guide](/gh-aw/reference/gh-aw-as-mcp-server/) - MCP server configuration
- [Agent Factory](/gh-aw/agent-factory-status/) - Agent factory status
