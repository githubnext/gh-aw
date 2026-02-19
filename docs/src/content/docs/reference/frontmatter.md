---
title: Frontmatter
description: Complete guide to all available frontmatter configuration options for GitHub Agentic Workflows, including triggers, permissions, AI engines, and workflow settings.
sidebar:
  order: 200
---

The [frontmatter](/gh-aw/reference/glossary/#frontmatter) (YAML configuration section between `---` markers) of GitHub Agentic Workflows includes the triggers, permissions, AI [engines](/gh-aw/reference/glossary/#engine) (which AI model/provider to use), and workflow settings. For example:

```yaml wrap
---
on:
  issues:
    types: [opened]

tools:
  edit:
  bash: ["gh issue comment"]
---
...markdown instructions...
```

## Frontmatter Elements

Below is a comprehensive reference to all available frontmatter fields for GitHub Agentic Workflows.

### Trigger Events (`on:`)

The `on:` section uses standard GitHub Actions syntax to define workflow triggers, with additional fields for security and approval controls:

- Standard GitHub Actions triggers (push, pull_request, issues, schedule, etc.)
- `reaction:` - Add emoji reactions to triggering items
- `stop-after:` - Automatically disable triggers after a deadline
- `manual-approval:` - Require manual approval using environment protection rules
- `forks:` - Configure fork filtering for pull_request triggers
- `skip-roles:` - Skip workflow execution for specific repository roles
- `skip-bots:` - Skip workflow execution for specific GitHub actors

See [Trigger Events](/gh-aw/reference/triggers/) for complete documentation.

### Description (`description:`)

Provides a human-readable description of the workflow rendered as a comment in the generated lock file.

```yaml wrap
description: "Workflow that analyzes pull requests and provides feedback"
```

### Source Tracking (`source:`)

Tracks workflow origin in format `owner/repo/path@ref`. Automatically populated when using `gh aw add` to install workflows from external repositories. Optional for manually created workflows.

```yaml wrap
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"
```

### Labels (`labels:`)

Optional array of strings for categorizing and organizing workflows. Labels are displayed in `gh aw status` command output and can be filtered using the `--label` flag.

```yaml wrap
labels: ["automation", "ci", "diagnostics"]
```

Labels help organize workflows by purpose, team, or functionality. They appear in status command table output as `[automation ci diagnostics]` and as a JSON array in `--json` mode. Filter workflows by label using `gh aw status --label automation`.

### Metadata (`metadata:`)

Optional key-value pairs for storing custom metadata compatible with the [GitHub Copilot custom agent spec](https://docs.github.com/en/copilot/reference/custom-agents-configuration).

```yaml wrap
metadata:
  author: John Doe
  version: 1.0.0
  category: automation
```

**Constraints:**

- Keys: 1-64 characters
- Values: Maximum 1024 characters
- Only string values are supported

Metadata provides a flexible way to add descriptive information to workflows without affecting execution.

### Plugins (`plugins:`)

:::caution[Experimental Feature]
Plugin support is experimental and may change in future releases. Using plugins will emit a compilation warning.
:::

Specifies plugins to install before workflow execution. Plugins are installed using engine-specific CLI commands (`copilot plugin install`, `claude plugin install`, `codex plugin install`).

**Array format** (simple):

```yaml wrap
plugins:
  - github/test-plugin
  - acme/custom-tools
```

**Object format** (with custom token):

```yaml wrap
plugins:
  repos:
    - github/test-plugin
    - acme/custom-tools
  github-token: ${{ secrets.CUSTOM_PLUGIN_TOKEN }}
```

**Token precedence** for plugin installation (highest to lowest):

1. Custom `plugins.github-token` from object format
2. `${{ secrets.GH_AW_PLUGINS_TOKEN }}`
3. [`${{ secrets.GH_AW_GITHUB_TOKEN }}`](/gh-aw/reference/auth/#gh_aw_github_token)
4. `${{ secrets.GITHUB_TOKEN }}` (default)

Each plugin repository must be specified in `org/repo` format. The compiler generates installation steps that run after the engine CLI is installed but before workflow execution begins.

### Runtimes (`runtimes:`)

Override default runtime versions for languages and tools used in workflows. The compiler automatically detects runtime requirements from tool configurations and workflow steps, then installs the specified versions.

**Format**: Object with runtime name as key and configuration as value

**Fields per runtime**:

- `version`: Runtime version string (required)
- `action-repo`: Custom GitHub Actions setup action (optional, overrides default)
- `action-version`: Version of the setup action (optional, overrides default)

**Supported runtimes**:

| Runtime | Default Version | Default Setup Action |
|---------|----------------|---------------------|
| `node` | 24 | `actions/setup-node@v6` |
| `python` | 3.12 | `actions/setup-python@v5` |
| `go` | 1.25 | `actions/setup-go@v5` |
| `uv` | latest | `astral-sh/setup-uv@v5` |
| `bun` | 1.1 | `oven-sh/setup-bun@v2` |
| `deno` | 2.x | `denoland/setup-deno@v2` |
| `ruby` | 3.3 | `ruby/setup-ruby@v1` |
| `java` | 21 | `actions/setup-java@v4` |
| `dotnet` | 8.0 | `actions/setup-dotnet@v4` |
| `elixir` | 1.17 | `erlef/setup-beam@v1` |
| `haskell` | 9.10 | `haskell-actions/setup@v2` |

**Examples**:

Override Node.js version:

```yaml wrap
runtimes:
  node:
    version: "22"
```

Use specific Python version with custom setup action:

```yaml wrap
runtimes:
  python:
    version: "3.12"
    action-repo: "actions/setup-python"
    action-version: "v5"
```

Multiple runtime overrides:

```yaml wrap
runtimes:
  node:
    version: "20"
  python:
    version: "3.11"
  go:
    version: "1.22"
```

**Default Behavior**: If not specified, workflows use default runtime versions as defined in the system. The compiler automatically detects which runtimes are needed based on tool configurations (e.g., `bash: ["node"]`, `bash: ["python"]`) and workflow steps.

**Use Cases**:

- Pin specific runtime versions for reproducibility
- Use preview/beta runtime versions for testing
- Use custom setup actions (forks, enterprise mirrors)
- Override system defaults for compatibility requirements

**Note**: Runtimes from imported shared workflows are automatically merged with your workflow's runtime configuration.

### Permissions (`permissions:`)

The `permissions:` section uses standard GitHub Actions permissions syntax to specify the permissions relevant to the agentic (natural language) part of the execution of the workflow. See [GitHub Actions permissions documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions).

```yaml wrap
# Specific permissions
permissions:
  issues: write
  contents: read
  pull-requests: write

# All permissions
permissions: write-all
permissions: read-all

# No permissions
permissions: {}
```

If you specify any permission, unspecified ones are set to `none`.

#### Permission Validation

The compiler validates workflows have sufficient permissions for their configured tools.

**Non-strict mode** (default): Emits warnings with suggestions to add missing permissions or reduce toolset requirements.

**Strict mode** (`gh aw compile --strict`): Treats under-provisioned permissions as compilation errors. Use for production workflows requiring enhanced security validation.

### Repository Access Roles (`on.roles:`)

Controls who can trigger agentic workflows based on repository permission level. Defaults to `[admin, maintainer, write]`.

```yaml wrap
on:
  issues:
    types: [opened]
  roles: [admin, maintainer, write]  # Default
```

```yaml wrap
on:
  workflow_dispatch:
  roles: all                         # Allow any user (⚠️ use with caution)
```

Available roles: `admin`, `maintainer`, `write`, `read`, `all`. Workflows with unsafe triggers (`push`, `issues`, `pull_request`) automatically enforce permission checks. Failed checks cancel the workflow with a warning.

> [!TIP]
> Run `gh aw fix workflow.md --write` to automatically migrate top-level `roles:` to `on.roles:` using the built-in codemod.

### Bot Filtering (`on.bots:`)

Configure which GitHub bot accounts can trigger workflows. Useful for allowing specific automation bots while maintaining security controls.

```yaml wrap
on:
  issues:
    types: [opened]
  bots:
    - "dependabot[bot]"
    - "renovate[bot]"
    - "agentic-workflows-dev[bot]"
```

**Behavior**:

- When specified, only the listed bot accounts can trigger the workflow
- The bot must be active (installed) on the repository to trigger the workflow
- Combine with `on.roles:` for comprehensive access control
- Applies to all workflow triggers (`pull_request`, `issues`, etc.)
- When `on.roles: all` is set, bot filtering is not enforced

**Common bot names**:

- `dependabot[bot]` - GitHub Dependabot for dependency updates
- `renovate[bot]` - Renovate bot for automated dependency management
- `github-actions[bot]` - GitHub Actions bot
- `agentic-workflows-dev[bot]` - Development bot for testing workflows

> [!TIP]
> Run `gh aw fix workflow.md --write` to automatically migrate top-level `bots:` to `on.bots:` using the built-in codemod.

### Skip Roles (`on.skip-roles`)

Skip workflow execution for users with specific repository permission levels. Useful for exempting team members from automated checks that should only apply to external contributors.

```yaml wrap
on:
  issues:
    types: [opened]
  skip-roles: [admin, maintainer, write]
```

**Available roles**: `admin`, `maintainer`, `write`, `read`

**Behavior**:

- Workflow is cancelled during pre-activation when triggered by users with listed roles
- Check runs before agent execution to avoid unnecessary compute costs
- Merged as union when importing workflows (all skip-roles from imported workflows are combined)
- Useful for AI moderation workflows that should only check external user content

**Example use case**: An AI content moderation workflow that checks issues for policy violations but exempts trusted team members with write access or higher.

### Skip Bots (`on.skip-bots`)

Skip workflow execution when triggered by specific GitHub actors (users or bots). Complements `skip-roles` by filtering based on actor identity rather than permission level.

```yaml wrap
on:
  issues:
    types: [opened]
  skip-bots: [github-actions, copilot, dependabot]
```

**Bot name matching**: Automatic flexible matching handles bot names with or without the `[bot]` suffix. For example, specifying `github-actions` matches both `github-actions` and `github-actions[bot]` actors automatically.

**Behavior**:

- Workflow is cancelled during pre-activation when `github.actor` matches any listed actor
- Check runs before agent execution to avoid unnecessary compute costs
- Merged as union when importing workflows (all skip-bots from imported workflows are combined)
- Accepts both user accounts and bot accounts

**String or array format**:

```yaml wrap
# Single bot
skip-bots: github-actions

# Multiple bots
skip-bots: [github-actions, copilot, renovate]
```

**Example use cases**:

- Skip AI workflows when triggered by automation bots to avoid bot-to-bot interactions
- Prevent workflow loops where one workflow's output triggers another
- Exempt specific known bots from content checks or policy enforcement

### Strict Mode (`strict:`)

Enables enhanced security validation for production workflows. **Enabled by default**.

```yaml wrap
strict: true   # Enable (default)
strict: false  # Disable for development/testing
```

**Enforcement areas:**

1. Refuses write permissions (`contents:write`, `issues:write`, `pull-requests:write`) - use [safe-outputs](/gh-aw/reference/safe-outputs/) instead
2. Requires explicit [network configuration](/gh-aw/reference/network/)
3. Refuses wildcard `*` in `network.allowed` domains
4. Requires ecosystem identifiers (e.g., `python`, `node`) instead of individual ecosystem domains (e.g., `pypi.org`, `npmjs.org`) for all engines
5. Requires network config for custom MCP servers with containers
6. Enforces GitHub Actions pinned to commit SHAs
7. Refuses deprecated frontmatter fields

When strict mode rejects individual ecosystem domains, helpful error messages suggest the appropriate ecosystem identifier (e.g., "Did you mean: 'pypi.org' belongs to ecosystem 'python'?").

**Configuration:**

- **Frontmatter**: `strict: true/false` (per-workflow)
- **CLI flag**: `gh aw compile --strict` (all workflows, overrides frontmatter)

See [Network Permissions - Strict Mode Validation](/gh-aw/reference/network/#strict-mode-validation) for details on network validation and [CLI Commands](/gh-aw/setup/cli/#compile) for compilation options.

### Feature Flags (`features:`)

Enable experimental or optional features as key-value pairs.

```yaml wrap
features:
  my-experimental-feature: true
  action-mode: "script"
```

#### Action Mode (`features.action-mode`)

Controls how the workflow compiler generates custom action references in compiled workflows. Can be set to `"dev"`, `"release"`, or `"script"`.

```yaml wrap
features:
  action-mode: "script"
```

**Available modes:**

- **`dev`** (default): References custom actions using local paths (e.g., `uses: ./actions/setup`). Best for development and testing workflows in the gh-aw repository.

- **`release`**: References custom actions using SHA-pinned remote paths (e.g., `uses: github/gh-aw/actions/setup@sha`). Used for production workflows with version pinning.

- **`script`**: Generates direct shell script calls instead of using GitHub Actions `uses:` syntax. The compiler:
  1. Checks out the `github/gh-aw` repository's `actions` folder to `/tmp/gh-aw/actions-source`
  2. Runs the setup script directly: `bash /tmp/gh-aw/actions-source/actions/setup/setup.sh`
  3. Uses shallow clone (`depth: 1`) for efficiency

**When to use script mode:**

- Testing custom action scripts during development
- Debugging action installation issues
- Environments where local action references are not available
- Advanced debugging scenarios requiring direct script execution

**Example:**

```yaml wrap
---
name: Debug Workflow
on: workflow_dispatch
features:
  action-mode: "script"
permissions:
  contents: read
---

Debug workflow using script mode for custom actions.
```

**Note:** The `action-mode` can also be overridden via the CLI flag `--action-mode` or the environment variable `GH_AW_ACTION_MODE`. The precedence is: CLI flag > feature flag > environment variable > auto-detection.

### AI Engine (`engine:`)

Specifies which AI engine interprets the markdown section. See [AI Engines](/gh-aw/reference/engines/) for details.

```yaml wrap
engine: copilot
```

### Network Permissions (`network:`)

Controls network access using ecosystem identifiers and domain allowlists. See [Network Permissions](/gh-aw/reference/network/) for full documentation.

```yaml wrap
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - "api.example.com"    # Custom domain
```

### Safe Inputs (`safe-inputs:`)

Enables defining custom MCP tools inline using JavaScript or shell scripts. See [Safe Inputs](/gh-aw/reference/safe-inputs/) for complete documentation on creating custom tools with controlled secret access.

### Safe Outputs (`safe-outputs:`)

Enables automatic issue creation, comment posting, and other safe outputs. See [Safe Outputs Processing](/gh-aw/reference/safe-outputs/).

### Run Configuration (`run-name:`, `runs-on:`, `timeout-minutes:`)

Standard GitHub Actions properties:

```yaml wrap
run-name: "Custom workflow run name"  # Defaults to workflow name
runs-on: ubuntu-latest               # Defaults to ubuntu-latest (main job only)
timeout-minutes: 30                  # Defaults to 20 minutes
```

### Workflow Concurrency Control (`concurrency:`)

Automatically generates concurrency policies for the agent job. See [Concurrency Control](/gh-aw/reference/concurrency/).

## Environment Variables (`env:`)

Standard GitHub Actions `env:` syntax for workflow-level environment variables:

```yaml wrap
env:
  CUSTOM_VAR: "value"
```

Environment variables can be defined at multiple scopes (workflow, job, step, engine, safe-outputs, etc.) with clear precedence rules. See [Environment Variables](/gh-aw/reference/environment-variables/) for complete documentation on all 13 env scopes and precedence order.

> [!WARNING]
> Do not use `${{ secrets.* }}` expressions in the workflow-level `env:` section. Environment variables defined here are passed directly to the agent container, which means secret values would be visible to the AI model. In strict mode, this is a compilation error. In non-strict mode, it emits a warning.
>
> Use engine-specific secret configuration instead of the `env:` section to pass secrets securely.

## Secrets (`secrets:`)

Defines secret values passed to workflow execution. Secrets are typically used to provide sensitive configuration to MCP servers or workflow components. Values must be GitHub Actions expressions that reference secrets (e.g., `${{ secrets.API_KEY }}`).

```yaml wrap
secrets:
  API_TOKEN: ${{ secrets.API_TOKEN }}
  DATABASE_URL: ${{ secrets.DB_URL }}
```

Secrets can also include descriptions for documentation:

```yaml wrap
secrets:
  API_TOKEN:
    value: ${{ secrets.API_TOKEN }}
    description: "API token for external service"
  DATABASE_URL:
    value: ${{ secrets.DB_URL }}
    description: "Production database connection string"
```

**Security best practices:**

- Always use GitHub Actions secret expressions (`${{ secrets.NAME }}`)
- Never commit plaintext secrets to workflow files
- Use environment-specific secrets when possible (via `environment:` field)
- Limit secret access to only the components that need them

**Note:** For passing secrets to reusable workflows, use the `jobs.<job_id>.secrets` field instead. The top-level `secrets:` field is for workflow-level secret configuration.

## Environment Protection (`environment:`)

Specifies the environment for deployment protection rules and environment-specific secrets. Standard GitHub Actions syntax.

```yaml wrap
environment: production
```

See [GitHub Actions environment docs](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment).

## Container Configuration (`container:`)

Specifies a container to run job steps in.

```yaml wrap
container: node:18
```

See [GitHub Actions container docs](https://docs.github.com/en/actions/how-tos/write-workflows/choose-where-workflows-run/run-jobs-in-a-container).

## Service Containers (`services:`)

Defines service containers that run alongside your job (databases, caches, etc.).

```yaml wrap
services:
  postgres:
    image: postgres:13
    env:
      POSTGRES_PASSWORD: postgres
    ports:
      - 5432:5432
```

See [GitHub Actions service docs](https://docs.github.com/en/actions/using-containerized-services).

## Conditional Execution (`if:`)

Standard GitHub Actions `if:` syntax:

```yaml wrap
if: github.event_name == 'push'
```

## Custom Steps (`steps:`)

Add custom steps before agentic execution. If unspecified, a default checkout step is added automatically.

```yaml wrap
steps:
  - name: Install dependencies
    run: npm ci
```

Use custom steps to precompute data, filter triggers, or prepare context for AI agents. See [Deterministic & Agentic Patterns](/gh-aw/guides/deterministic-agentic-patterns/) for combining computation with AI reasoning.

Custom steps run outside the firewall sandbox. These steps execute with standard GitHub Actions security.

## Post-Execution Steps (`post-steps:`)

Add custom steps after agentic execution. Run after AI engine completes regardless of success/failure (unless conditional expressions are used).

```yaml wrap
post-steps:
  - name: Upload Results
    if: always()
    uses: actions/upload-artifact@v4
    with:
      name: workflow-results
      path: /tmp/gh-aw/
      retention-days: 7
```

Useful for artifact uploads, summaries, cleanup, or triggering downstream workflows.

Post-execution steps run OUTSIDE the firewall sandbox. These steps execute with standard GitHub Actions security.

## Custom Jobs (`jobs:`)

Define custom jobs that run before agentic execution. Supports complete GitHub Actions step specification.

```yaml wrap
jobs:
  super_linter:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - name: Run Super-Linter
        uses: super-linter/super-linter@v7
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The agentic execution job waits for all custom jobs to complete. Custom jobs can share data through artifacts or job outputs. See [Deterministic & Agentic Patterns](/gh-aw/guides/deterministic-agentic-patterns/) for multi-job workflows.

Custom jobs run outside the firewall sandbox. These jobs execute with standard GitHub Actions security.

### Job Outputs

Custom jobs can expose outputs accessible in the agentic execution prompt via `${{ needs.job-name.outputs.output-name }}`:

```yaml wrap
jobs:
  release:
    outputs:
      release_id: ${{ steps.get_release.outputs.release_id }}
      version: ${{ steps.get_release.outputs.version }}
    steps:
      - id: get_release
        run: echo "version=${{ github.event.release.tag_name }}" >> $GITHUB_OUTPUT
---

Generate highlights for release ${{ needs.release.outputs.version }}.
```

Job outputs must be string values.

## Cache Configuration (`cache:`)

Cache configuration using standard GitHub Actions `actions/cache` syntax:

Single cache:

```yaml wrap
cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules
  restore-keys: |
    node-modules-
```

## Related Documentation

See also: [Trigger Events](/gh-aw/reference/triggers/), [AI Engines](/gh-aw/reference/engines/), [CLI Commands](/gh-aw/setup/cli/), [Workflow Structure](/gh-aw/reference/workflow-structure/), [Network Permissions](/gh-aw/reference/network/), [Command Triggers](/gh-aw/reference/command-triggers/), [MCPs](/gh-aw/guides/mcps/), [Tools](/gh-aw/reference/tools/), [Imports](/gh-aw/reference/imports/)
