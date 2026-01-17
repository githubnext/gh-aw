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

The frontmatter combines standard GitHub Actions properties (`on`, `permissions`, `run-name`, `runs-on`, `timeout-minutes`, `concurrency`, `env`, `environment`, `container`, `services`, `if`, `steps`, `cache`) with GitHub Agentic Workflows-specific elements (`description`, `source`, `github-token`, `imports`, `engine`, `strict`, `roles`, `features`, `safe-inputs`, `safe-outputs`, `network`, `tools`).

Tool configurations (such as `bash`, `edit`, `github`, `web-fetch`, `web-search`, `playwright`, `cache-memory`, and custom [Model Context Protocol](/gh-aw/reference/glossary/#mcp-model-context-protocol) (MCP) [servers](/gh-aw/reference/glossary/#mcp-server)) are specified under the `tools:` key. Custom inline tools can be defined with the [`safe-inputs:`](/gh-aw/reference/safe-inputs/) (custom tools defined inline) key. See [Tools](/gh-aw/reference/tools/) and [Safe Inputs](/gh-aw/reference/safe-inputs/) for complete documentation.

### Trigger Events (`on:`)

The `on:` section uses standard GitHub Actions syntax to define workflow triggers, with additional fields for security and approval controls:

- Standard GitHub Actions triggers (push, pull_request, issues, schedule, etc.)
- `reaction:` - Add emoji reactions to triggering items
- `stop-after:` - Automatically disable triggers after a deadline
- `manual-approval:` - Require manual approval using environment protection rules
- `forks:` - Configure fork filtering for pull_request triggers

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

### GitHub Token (`github-token:`)

Configures the default GitHub token for engine authentication, checkout steps, and safe-output operations.

```yaml wrap
github-token: ${{ secrets.CUSTOM_PAT }}
```

> [!CAUTION]
> Secret Expression Required
> Must use GitHub Actions secret expressions (e.g., `${{ secrets.CUSTOM_PAT }}`). Plaintext tokens are rejected. Valid: `${{ secrets.GITHUB_TOKEN }}`, `${{ secrets.CUSTOM_PAT }}`. Invalid: plaintext tokens, environment variables.

**Token precedence** (highest to lowest):
1. Individual safe-output `github-token` (e.g., `create-issue.github-token`)
2. Safe-outputs global `github-token`
3. Top-level `github-token`
4. Default: `${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}`

See the [Security Guide](/gh-aw/guides/security/#authorization-and-token-management) for complete documentation.

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

### Repository Access Roles (`roles:`)

Controls who can trigger agentic workflows based on repository permission level. Defaults to `[admin, maintainer, write]`.

```yaml wrap
roles: [admin, maintainer, write]  # Default
roles: all                         # Allow any user (⚠️ use with caution)
```

Available roles: `admin`, `maintainer`, `write`, `read`, `all`. Workflows with unsafe triggers (`push`, `issues`, `pull_request`) automatically enforce permission checks. Failed checks cancel the workflow with a warning.

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
4. Requires network config for custom MCP servers with containers
5. Enforces GitHub Actions pinned to commit SHAs
6. Refuses deprecated frontmatter fields

**Configuration:**
- **Frontmatter**: `strict: true/false` (per-workflow)
- **CLI flag**: `gh aw compile --strict` (all workflows, overrides frontmatter)

See [CLI Commands](/gh-aw/setup/cli/#compile) and [Security Guide](/gh-aw/guides/security/#strict-mode-validation) for details.

### Feature Flags (`features:`)

Enable experimental or optional features as boolean key-value pairs.

```yaml wrap
features:
  my-experimental-feature: true
```

> [!NOTE]
> Firewall Configuration
> The `features.firewall` field has been removed. The agent sandbox is now mandatory and defaults to AWF (Agent Workflow Firewall). See [Sandbox Configuration](/gh-aw/reference/sandbox/) for details.

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
timeout-minutes: 30                  # Defaults to 20 minutes (timeout_minutes deprecated)
```

**Note**: The `timeout_minutes` field is deprecated. Use `timeout-minutes` instead to follow GitHub Actions naming convention.

### Workflow Concurrency Control (`concurrency:`)

Automatically generates concurrency policies for the agent job. See [Concurrency Control](/gh-aw/reference/concurrency/).

## Environment Variables (`env:`)

Standard GitHub Actions `env:` syntax for workflow-level environment variables:

```yaml wrap
env:
  CUSTOM_VAR: "value"
  SECRET_VAR: ${{ secrets.MY_SECRET }}
```

Environment variables can be defined at multiple scopes (workflow, job, step, engine, safe-outputs, etc.) with clear precedence rules. See [Environment Variables](/gh-aw/reference/environment-variables/) for complete documentation on all 13 env scopes and precedence order.

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

## Main Workflows vs Included Files

Understanding which frontmatter properties are available in main workflows versus included (shared) files is crucial for avoiding validation errors and structuring your workflows correctly.

### Property Comparison Table

This table shows all available frontmatter properties and their availability in main workflows versus included files:

| Property | Main Workflow | Included File | Notes |
|----------|---------------|---------------|-------|
| `on` | ✅ | ❌ | **Required** in main workflows; triggers workflow execution |
| `command` | ✅ | ❌ | Command-line trigger configuration |
| `if` | ✅ | ❌ | Conditional execution expression |
| `github-token` | ✅ | ❌ | Security: Token configuration only in main |
| `roles` | ✅ | ❌ | Security: Access control only in main |
| `strict` | ✅ | ❌ | Security: Validation mode only in main |
| `run-name` | ✅ | ❌ | Runtime display name |
| `runs-on` | ✅ | ❌ | Runner specification |
| `timeout-minutes` | ✅ | ❌ | Execution timeout |
| `timeout_minutes` | ✅ | ❌ | Deprecated; use `timeout-minutes` |
| `concurrency` | ✅ | ❌ | Concurrency control |
| `container` | ✅ | ❌ | Container configuration |
| `environment` | ✅ | ❌ | Deployment environment |
| `jobs` | ✅ | ❌ | Custom job definitions |
| `post-steps` | ✅ | ❌ | Steps after agentic execution |
| `cache` | ✅ | ❌ | Cache configuration |
| `bots` | ✅ | ❌ | Bot actor configuration |
| `tracker-id` | ✅ | ❌ | Asset tracking identifier |
| `labels` | ✅ | ❌ | Workflow categorization labels |
| `source` | ✅ | ❌ | Workflow origin tracking |
| `name` | ✅ | ❌ | Workflow name |
| `engine` | ✅ | ⚠️ | Main: full config; Included: limited (no `command` field) |
| `env` | ✅ | ❌ | Workflow-level environment variables |
| `features` | ✅ | ❌ | Feature flag configuration |
| `imports` | ✅ | ❌ | Import declarations |
| `sandbox` | ✅ | ❌ | Sandbox configuration |
| `description` | ✅ | ✅ | Workflow/file description |
| `metadata` | ✅ | ✅ | Custom key-value metadata |
| `tools` | ✅ | ✅ | Tool configurations (bash, github, etc.) |
| `mcp-servers` | ✅ | ⚠️ | Main: full config; Included: limited (no container config) |
| `network` | ✅ | ✅ | Network permissions |
| `permissions` | ✅ | ✅ | GitHub Actions permissions (validated, not merged) |
| `safe-inputs` | ✅ | ✅ | Custom tool definitions |
| `safe-outputs` | ✅ | ✅ | Safe output configurations |
| `services` | ✅ | ✅ | Docker service containers |
| `steps` | ✅ | ✅ | Custom workflow steps |
| `runtimes` | ✅ | ✅ | Runtime version overrides (node, python, etc.) |
| `secret-masking` | ✅ | ✅ | Secret masking configuration |
| `inputs` | ❌ | ✅ | Input parameter declarations (for shared workflows) |
| `applyTo` | ❌ | ✅ | Glob patterns for custom agent targeting |

### Key Differences

#### Triggers and Execution Control
- **Main workflows** are entry points that respond to events (`on`), commands (`command`), or conditions (`if`)
- **Included files** cannot define triggers—they're imported and used by main workflows
- The presence of an `on` field distinguishes a main workflow from a shared component

#### Security and Authorization
- **Token management** (`github-token`): Only main workflows configure tokens
- **Access control** (`roles`): Only main workflows specify who can trigger execution
- **Validation mode** (`strict`): Only main workflows control security validation level
- This prevents included files from weakening security policies

#### Configuration Scope
- **Full engine configuration**: Main workflows use `engine.command`; included files have limited engine config
- **MCP servers**: Main workflows can configure containers; included files have restricted MCP config
- **Runtime settings**: Main workflows control runners, timeouts, concurrency, and environments
- **Custom jobs**: Only main workflows can define custom GitHub Actions jobs

#### Shared Component Features
- **Input parameters** (`inputs`): Only included files declare reusable inputs
- **Targeting patterns** (`applyTo`): Only included files (custom agents) use glob patterns to target specific code areas
- **Tool configurations**: Both can define tools, but included files enable reusable tool setups

### Common Pitfalls

#### ❌ Forgetting Required `on` Field in Main Workflows

```yaml
---
# ERROR: Missing required 'on' field
engine: copilot
tools:
  bash: {}
---
```

**Fix**: Add a trigger to make this a valid main workflow:

```yaml
---
on:
  issues:
    types: [opened]
engine: copilot
tools:
  bash: {}
---
```

Or remove the `on` field entirely if you intend this to be a shared component for import.

#### ❌ Using `engine.command` in Included Files

```yaml
# shared/tools.md - INCORRECT
---
engine:
  provider: copilot
  command: custom-copilot-agent  # ERROR: Not allowed in included files
tools:
  bash: {}
---
```

**Fix**: Move engine command configuration to the main workflow:

```yaml
# shared/tools.md - CORRECT
---
tools:
  bash: {}
---

# main.md
---
on: issues
engine:
  provider: copilot
  command: custom-copilot-agent  # Only in main workflow
imports:
  - shared/tools.md
---
```

#### ❌ Expecting Full MCP Container Config in Included Files

```yaml
# shared/mcp.md - INCORRECT
---
mcp-servers:
  custom:
    container:
      image: custom-mcp:latest  # ERROR: Container config restricted
---
```

**Fix**: Define MCP containers in the main workflow:

```yaml
# main.md - CORRECT
---
on: issues
mcp-servers:
  custom:
    container:
      image: custom-mcp:latest
imports:
  - shared/mcp.md  # Can provide other MCP settings
---
```

#### ❌ Using `inputs` Field in Main Workflows

```yaml
# main.md - INCORRECT
---
on: issues
inputs:  # ERROR: inputs only valid in included files
  max_items:
    type: number
    default: 10
---
```

**Fix**: `inputs` are only for shared workflows that will be imported. If you need dynamic configuration, use environment variables or GitHub Actions inputs via `workflow_dispatch`:

```yaml
# shared/component.md - CORRECT
---
inputs:
  max_items:
    type: number
    default: 10
---

# main.md
---
on:
  workflow_dispatch:
    inputs:
      item_count:
        type: number
        default: 10
imports:
  - path: shared/component.md
    inputs:
      max_items: ${{ github.event.inputs.item_count }}
---
```

### Design Philosophy

The separation between main workflows and included files follows these principles:

1. **Security boundary**: Critical security settings (`github-token`, `roles`, `strict`) are confined to main workflows to prevent imported files from weakening security posture
2. **Execution control**: Only main workflows define when and how workflows execute (`on`, `command`, `if`)
3. **Reusability**: Included files focus on portable, reusable configurations (tools, permissions, steps) that can be shared across workflows
4. **Composition**: Main workflows orchestrate execution by importing and combining shared components

### Cross-References

- [Imports](/gh-aw/reference/imports/) - Detailed guide to importing and merging frontmatter configurations
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Understanding workflow composition
- [Security Guide](/gh-aw/guides/security/) - Security policies and token management
- [Tools](/gh-aw/reference/tools/) - Configuring tools in main and included files

## Related Documentation

See also: [Trigger Events](/gh-aw/reference/triggers/), [AI Engines](/gh-aw/reference/engines/), [CLI Commands](/gh-aw/setup/cli/), [Workflow Structure](/gh-aw/reference/workflow-structure/), [Network Permissions](/gh-aw/reference/network/), [Command Triggers](/gh-aw/reference/command-triggers/), [MCPs](/gh-aw/guides/mcps/), [Tools](/gh-aw/reference/tools/), [Imports](/gh-aw/reference/imports/)
