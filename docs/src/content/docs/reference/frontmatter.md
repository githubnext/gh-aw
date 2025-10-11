---
title: Frontmatter Options
description: Complete guide to all available frontmatter configuration options for GitHub Agentic Workflows, including triggers, permissions, AI engines, and workflow settings.
sidebar:
  order: 200
---

The YAML frontmatter supports standard GitHub Actions properties plus additional agentic-specific options:

**Standard GitHub Actions Properties:**
- `on`: Trigger events for the workflow
- `permissions`: Required permissions for the workflow
- `run-name`: Name of the workflow run
- `runs-on`: Runner environment for the workflow
- `timeout_minutes`: Workflow timeout
- `concurrency`: Concurrency settings for the workflow
- `env`: Environment variables for the workflow
- `environment`: Environment that the job references (for protected environments and deployments)
- `container`: Container to run the job steps in
- `services`: Service containers for the job (databases, caches, etc.)
- `if`: Conditional execution of the workflow
- `steps`: Custom steps for the job
- `cache`: Cache configuration for workflow dependencies

**Properties specific to GitHub Agentic Workflows:**
- `description`: Human-readable description rendered as a comment in the lock file
- `source`: Source reference tracking where the workflow was added from (format: `owner/repo/path@ref`)
- `imports`: List of files to import (see [Packaging and Imports](/gh-aw/guides/packaging-imports/))
- `engine`: AI engine configuration (copilot/claude/codex) with optional max-turns setting
- `strict`: Enable strict mode validation (boolean, defaults to false)
- `roles`: Permission restrictions based on repository access levels
- `safe-outputs`: [Safe Output Processing](/gh-aw/reference/safe-outputs/)
- `network`: Network access control for AI engines
- `tools`: Available tools and MCP servers for the AI engine
- `cache-memory`: [Persistent memory configuration](/gh-aw/reference/cache-memory/)

## Trigger Events (`on:`)

The `on:` section uses standard GitHub Actions syntax to define workflow triggers. Here are some common examples:

```yaml
on:
  issues:
    types: [opened]
```

### Stop After Configuration (`stop-after:`)

You can add a `stop-after:` option within the `on:` section as a cost-control measure to automatically disable workflow triggering after a deadline:

```yaml
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+25h"  # 25 hours from compilation time
```

**Relative time delta (calculated from compilation time):**
```yaml
on:
  issues:
    types: [opened]
  stop-after: "+25h"      # 25 hours from now
```

**Supported absolute date formats:**
- Standard: `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD`
- US format: `MM/DD/YYYY HH:MM:SS`, `MM/DD/YYYY`  
- European: `DD/MM/YYYY HH:MM:SS`, `DD/MM/YYYY`
- Readable: `January 2, 2006`, `2 January 2006`, `Jan 2, 2006`
- Ordinals: `1st June 2025`, `June 1st 2025`, `23rd December 2025`
- ISO 8601: `2006-01-02T15:04:05Z`

**Supported delta units:**
- `d` - days
- `h` - hours
- `m` - minutes

Note that if you specify a relative time, it is calculated at the time of workflow compilation, not when the workflow runs. If you re-compile your workflow, e.g. after a change, the effective stop time will be reset.

### Reactions (`reaction:`)

You can add a `reaction:` option within the `on:` section to enable emoji reactions on the triggering GitHub item (issue, PR, comment, discussion) to provide visual feedback about the workflow status:

```yaml
on:
  issues:
    types: [opened]
  reaction: "eyes"
```

**Behavior:**
- **For `issues` and `pull_request` events**: Adds the emoji reaction AND creates a comment with a link to the workflow run
- **For comment events** (`issue_comment`, `pull_request_review_comment`): Adds the emoji reaction and edits the comment to include the workflow run link (command workflows only)

**Outputs:**
The `add_reaction` job exposes the following outputs for use by downstream jobs:
- `reaction_id`: The ID of the created reaction
- `comment_id`: The ID of the created comment (for `issues`/`pull_request` events)
- `comment_url`: The URL of the created comment (for `issues`/`pull_request` events)

**Available reactions:**
- `+1` (👍)
- `-1` (👎)
- `laugh` (😄)
- `confused` (😕)
- `heart` (❤️)
- `hooray` (🎉)
- `rocket` (🚀)
- `eyes` (👀)

### Command Triggers (`command:`)

An additional kind of trigger called `command:` is supported, see [Command Triggers](/gh-aw/reference/command-triggers/) for special `/my-bot` triggers and context text functionality.

> [!NOTE]
> Command workflows automatically enable the "eyes" (👀) reaction by default. This can be customized by explicitly specifying a different reaction in the `reaction:` field.

### Label Filtering (`names:`)

When using `labeled` or `unlabeled` event types for `issues` or `pull_request` triggers, you can filter to specific label names using the `names:` field:

```yaml
on:
  issues:
    types: [labeled, unlabeled]
    names: [bug, critical, security]
```

**How it works:**
- The `names:` field is removed from the final workflow YAML and commented out for documentation
- A conditional `if` expression is automatically generated to check if the label name matches
- The workflow only runs when one of the specified labels triggers the event

**Syntax options:**

```yaml
# Single label name
names: bug

# Multiple label names (array)
names: [bug, enhancement, feature]
```

**Example for pull requests:**

```yaml
on:
  pull_request:
    types: [labeled]
    names: ready-for-review
```

This filtering is especially useful for [LabelOps workflows](/gh-aw/guides/labelops/) where specific labels trigger different automation behaviors.

## Description (`description:`)

The `description:` field provides a human-readable description of the workflow that is rendered as a comment in the generated lock file. This helps document the purpose and functionality of the workflow.

```yaml
description: "Workflow that analyzes pull requests and provides feedback"
```

The description appears in the lock file header as a comment:

```yaml
# This file was automatically generated by gh-aw. DO NOT EDIT.
# To update this file, edit the corresponding .md file and run:
#   gh aw compile
# For more information: https://github.com/githubnext/gh-aw/blob/main/.github/instructions/github-agentic-workflows.instructions.md
#
# Workflow that analyzes pull requests and provides feedback

name: "PR Analyzer"
...
```

## Source Tracking (`source:`)

The `source:` field tracks the origin of workflows added using the `gh aw add` command. This field is automatically populated when installing workflows from external repositories and provides traceability for workflow provenance.

**Format:** `owner/repo/path@ref`

```yaml
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"
```

**Examples:**
- `githubnext/agentics/workflows/ci-doctor.md@v1.0.0` - Workflow from a specific version tag
- `githubnext/agentics/workflows/daily-plan.md@main` - Workflow from the main branch
- `githubnext/agentics/workflows/helper-bot.md` - Workflow without version specification

**Automatic Population:**

When you use the `gh aw add` command, the source field is automatically added to the workflow frontmatter:

```bash
# Command
gh aw add githubnext/agentics/ci-doctor@v1.0.0

# Generated frontmatter includes:
source: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0"
```

**Rendering in Lock Files:**

The source field is rendered as a comment in the lock file header, positioned after the description:

```yaml
# This file was automatically generated by gh-aw. DO NOT EDIT.
# To update this file, edit the corresponding .md file and run:
#   gh aw compile
# For more information: https://github.com/githubnext/gh-aw/blob/main/.github/instructions/github-agentic-workflows.instructions.md
#
# CI Doctor workflow - added from githubnext/agentics
#
# Source: githubnext/agentics/workflows/ci-doctor.md@v1.0.0

name: "CI Doctor"
...
```

**Benefits:**
- **Traceability**: Know exactly where a workflow came from and which version
- **Updates**: Easy identification of the source repository for checking updates
- **Documentation**: Automatic documentation of workflow provenance
- **Auditing**: Track workflow origins for security and compliance purposes

**Note:** The `source` field is optional. Workflows created manually or without using `gh aw add` don't need to include it.

## Permissions (`permissions:`)

The `permissions:` section uses standard GitHub Actions permissions syntax to specify the permissions relevant to the agentic (natural language) part of the execution of the workflow. See [GitHub Actions permissions documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions).

```yaml
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

## Repository Access Roles (`roles:`)

The `roles:` section controls who can trigger agentic workflows based on their repository permission level. By default, workflows are restricted to users with `admin` or `maintainer` permissions for security reasons.

```yaml
# Default behavior (admin or maintainer required)
roles: [admin, maintainer]

# Allow additional permission levels
roles: [admin, maintainer, write]

# Allow any authenticated user (use with caution)
roles: all

# Single role as a string
roles: admin
```

**Available repository roles:**
- **`admin`**: Full administrative access to the repository
- **`maintainer`**: Can manage the repository and its settings (renamed from `maintain` in GitHub)  
- **`write`**: Can push to the repository and manage issues and pull requests
- **`read`**: Can read and clone the repository
- **`all`**: Disables permission checking entirely (⚠️ security consideration)

**Behavior:**
- Workflows with potentially unsafe triggers (like `push`, `issues`, `pull_request`) automatically include permission checks
- "Safe" triggers like `schedule` and `workflow_run` skip permission checks by default
- `workflow_dispatch` is treated as safe only when `write` is in the allowed roles (since workflow_dispatch can be triggered by users with write access)
- When permission checks fail, the workflow is automatically cancelled with a warning message
- Users without sufficient permissions will see the workflow start but then immediately stop

## Strict Mode (`strict:`)

The `strict:` field enables enhanced validation for production workflows, enforcing security and reliability constraints. When enabled, the compiler will reject workflows that don't meet strict mode requirements.

```yaml
# Enable strict mode for this workflow
strict: true

# Explicitly disable strict mode (default)
strict: false
```

**Strict Mode Requirements:**

When `strict: true`, the workflow must satisfy these requirements:

1. **Write Permissions Blocked**: Cannot use `contents: write`, `issues: write`, or `pull-requests: write` permissions (use `safe-outputs` instead for controlled GitHub API interactions)
2. **Network Configuration Required**: Must explicitly configure network access (cannot rely on default behavior)
3. **No Network Wildcards**: Cannot use wildcard `*` in `network.allowed` domains
4. **MCP Network Configuration**: Custom MCP servers with containers must have network configuration

**Example Strict Mode Workflow:**

```yaml
---
on: push
strict: true
permissions:
  contents: read
engine: claude
network:
  allowed:
    - "api.example.com"
    - "*.trusted.com"
---

# Strict Mode Workflow
This workflow follows all strict mode requirements.
```

**Enabling Strict Mode:**

Strict mode can be enabled in two ways:
- **Frontmatter**: Add `strict: true` to the workflow frontmatter (per-workflow control)
- **CLI flag**: Use `gh aw compile --strict` (applies to all workflows being compiled)

The CLI `--strict` flag takes precedence over frontmatter settings. If the CLI flag is used, workflows with `strict: false` will still be validated in strict mode.

**Use Cases:**
- Production workflows that require enhanced security validation
- Workflows with elevated permissions that need extra scrutiny
- Workflows that need to comply with security policies

## AI Engine (`engine:`)

The `engine:` section specifies which AI engine to use to interpret the markdown section of the workflow, and controls options about how this execution proceeds. Defaults to `copilot`.

```yaml
engine: copilot # Default: GitHub Copilot CLI with MCP support
engine: claude  # Anthropic Claude Code
engine: codex   # Experimental: OpenAI Codex CLI with MCP support
engine: custom  # Custom: Execute user-defined GitHub Actions steps
```

For complete engine documentation including advanced configuration options, see [AI Engines](/gh-aw/reference/engines/).

Extended format:

```yaml
engine:
  id: copilot                       # Required: engine identifier (copilot, claude, codex, custom)
  version: latest                   # Optional: version of the action
  model: gpt-5                      # Optional: specific LLM model (for copilot)
  max-turns: 5                      # Optional: maximum chat iterations per run (for claude)
  max-concurrency: 3                # Optional: max concurrent workflows across all workflows (default: 3)
  env:                              # Optional: custom environment variables
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
    DEBUG_MODE: "true"
  config: |                         # Optional: additional TOML configuration (codex only)
    [custom_section]
    key1 = "value1"
    key2 = "value2"
```

**Fields:**
- **`id`** (required): Engine identifier (`copilot`, `claude`, `codex`, `custom`)
- **`version`** (optional): Action version (`beta`, `stable`)
- **`model`** (optional): Specific LLM model to use
- **`max-turns`** (optional): Maximum number of chat iterations per run (cost-control option)
- **`max-concurrency`** (optional): Maximum number of concurrent workflows across all workflows (default: 3)
- **`env`** (optional): Custom environment variables to pass to the agentic engine as key-value pairs
- **`config`** (optional): Additional TOML configuration text appended to generated config.toml (codex engine only)

### Environment Variables and Secret Overrides

The `env` option supports overriding default secrets and environment variables used by engines:

**Basic Environment Variables:**
```yaml
engine:
  id: claude
  env:
    AWS_REGION: us-west-2
    CUSTOM_API_ENDPOINT: https://api.example.com
    DEBUG_MODE: "true"
```

**Secret Override Example:**

You can override default secrets used by engines. This is particularly useful for Codex workflows when you need to use a different OpenAI API key:

```yaml
engine:
  id: codex
  model: gpt-4
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY_CI }}
```

This configuration overrides the default `OPENAI_API_KEY` secret with your custom secret, allowing you to use organization-specific API keys without duplicating secrets.

### Codex Engine Custom Configuration

The Codex engine supports an additional `config` field that allows you to append custom TOML configuration to the generated `config.toml` file:

```yaml
engine:
  id: codex
  config: |
    [custom_section]
    key1 = "value1"
    key2 = "value2"
    
    [server_settings]
    timeout = 60
    retries = 3
    
    [logging]
    level = "debug"
    file = "/tmp/gh-aw/custom.log"
```

**Key Features:**
- **Optional**: The `config` field is completely optional and only applies to the codex engine
- **Raw TOML**: Accepts any valid TOML configuration text
- **Proper Formatting**: Automatically indented and formatted in the generated workflow
- **Appended**: Added after all standard MCP server configurations in the config.toml file

**Generated config.toml structure:**
```toml
[history]
persistence = "none"

[mcp_servers.github]
user_agent = "workflow-name"
command = "docker"
# ... standard MCP server config ...

# Custom configuration
[custom_section]
key1 = "value1"
key2 = "value2"

[server_settings]
timeout = 60
retries = 3

[logging]
level = "debug"
file = "/tmp/gh-aw/custom.log"
```

This feature enables advanced customization scenarios not covered by the standard engine configuration options.

### Turn Limiting

The `max-turns` option is configured within the engine configuration to limit the number of chat iterations within a single agentic run:

```yaml
engine:
  id: claude
  max-turns: 5
```

**Behavior:**
1. Passes the limit to the AI engine (e.g., Claude Code action)
2. Engine stops iterating when the turn limit is reached
3. Helps prevent runaway chat loops and control costs
4. Only applies to engines that support turn limiting (currently Claude)

### Agent Job Concurrency

The `concurrency` field in the engine configuration controls how many agentic workflow runs can run concurrently. It uses GitHub Actions concurrency syntax:

```yaml
engine:
  id: claude
  concurrency:
    group: "custom-group-${{ github.workflow }}"
    cancel-in-progress: true
```

**Default Behavior:** Single job per engine across all workflows (group: `gh-aw-{engine-id}`)

**How it works:**
- Creates a concurrency group for the agent job
- Default pattern: `gh-aw-{engine-id}` ensures one job per engine across all workflows
- Supports full GitHub Actions concurrency syntax (group + optional cancel-in-progress)
- Different engines (claude, copilot, codex) can run concurrently without interfering

**Simple string format:**
```yaml
engine:
  id: claude
  concurrency: "my-custom-group-${{ github.ref }}"
```

This is converted to:
```yaml
concurrency:
  group: "my-custom-group-${{ github.ref }}"
```

**Object format (full control):**
```yaml
engine:
  id: claude
  concurrency:
    group: "my-group-${{ github.workflow }}-${{ github.ref }}"
    cancel-in-progress: true
```

**Example configurations:**

```yaml
# Default: single job per engine across all workflows
engine:
  id: claude
  # No concurrency specified, uses gh-aw-claude
```

```yaml
# Allow multiple concurrent jobs with different workflow names
engine:
  id: claude
  concurrency:
    group: "gh-aw-claude-${{ github.workflow }}"
```

```yaml
# Per-branch concurrency
engine:
  id: copilot
  concurrency:
    group: "gh-aw-copilot-${{ github.ref }}"
    cancel-in-progress: true
```

**Generated concurrency in agent job:**
```yaml
jobs:
  agent:
    concurrency:
      group: "gh-aw-claude"  # Default pattern
```

The concurrency group applies **only** to the agent job (not the workflow level). This ensures concurrency control for AI execution while allowing activation jobs to run freely.

## Tools Configuration (`tools:`)

The `tools:` section specifies which tools and MCP (Model Context Protocol) servers are available to the AI engine. This enables integration with GitHub APIs, browser automation, and other external services.

```yaml
tools:
  github:
    allowed: [create_issue, update_issue]
  playwright:
    allowed_domains: ["github.com", "*.example.com"]
  edit:
  bash: ["echo", "ls", "git status"]
```

For complete tool configuration options, including GitHub tools, Playwright browser automation, custom MCP servers, and security considerations, see [Tools Configuration](/gh-aw/reference/tools/).

## Network Permissions (`network:`)

Control network access for AI engines using ecosystem identifiers and domain allowlists. See [Network Permissions](/gh-aw/reference/network/) for detailed configuration options, security model, and examples.

Quick example:
```yaml
engine:
  id: claude

network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - "api.example.com"    # Custom domain
```

## Safe Outputs Configuration (`safe-outputs:`)

See [Safe Outputs Processing](/gh-aw/reference/safe-outputs/) for automatic issue creation, comment posting and other safe outputs.

## Run Configuration (`run-name:`, `runs-on:`, `timeout_minutes:`)

Standard GitHub Actions properties:
```yaml
run-name: "Custom workflow run name"  # Defaults to workflow name
runs-on: ubuntu-latest               # Defaults to ubuntu-latest
timeout_minutes: 30                  # Defaults to 20 minutes
```

## Concurrency Control (`concurrency:`)

GitHub Agentic Workflows automatically generates concurrency policies for the agent job to control concurrent execution.

See [Concurrency Control](/gh-aw/reference/concurrency/) for complete documentation on agent concurrency configuration.

**Quick reference:**
- Configure via `engine.concurrency` field
- Default: Single job per engine across all workflows (group: `gh-aw-{engine-id}`)
- Applied at the agent job level only
- Different engines can run concurrently without interfering
- Supports full GitHub Actions concurrency syntax

**Example:**
```yaml
engine:
  id: claude
  concurrency:
    group: "custom-${{ github.workflow }}"
    cancel-in-progress: true
```

Generates agent job concurrency:
```yaml
jobs:
  agent:
    concurrency:
      group: "custom-${{ github.workflow }}"
      cancel-in-progress: true
```

You can also override workflow-level concurrency by specifying a custom `concurrency` section in the frontmatter (separate from engine concurrency).

## Environment Variables (`env:`)

GitHub Actions standard `env:` syntax:

```yaml
env:
  CUSTOM_VAR: "value"
  SECRET_VAR: ${{ secrets.MY_SECRET }}
```

## Environment Protection (`environment:`)

The `environment:` section specifies the environment that the job references, enabling deployment protection rules and environment-specific secrets and variables. This follows standard GitHub Actions syntax for job-level environment configuration.

**Simple environment name:**
```yaml
environment: production
```

For more information about environments, see [GitHub Action's environment documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment).

## Container Configuration (`container:`)

The `container:` section specifies a container to run the job steps in, useful for standardized execution environments or specific runtime requirements.

**Simple container image:**
```yaml
container: node:18
```

For more information about environments, see [GitHub Action's container documentation](https://docs.github.com/en/actions/how-tos/write-workflows/choose-where-workflows-run/run-jobs-in-a-container).

## Service Containers (`services:`)

The `services:` section defines service containers that run alongside your job, commonly used for databases, caches, or other dependencies during testing and deployment.

**Simple service:**
```yaml
services:
  postgres:
    image: postgres:13
    env:
      POSTGRES_PASSWORD: postgres
    ports:
      - 5432:5432
```

For more information about containers and services, see [GitHub Action's container documentation](https://docs.github.com/en/actions/using-containerized-services).

## Conditional Execution (`if:`)

Standard GitHub Actions `if:` syntax:

```yaml
if: github.event_name == 'push'
```

## Custom Steps (`steps:`)

Add custom steps before the agentic execution step using GitHub Actions standard `steps:` syntax:

```yaml
steps:
  - name: Install dependencies
    run: npm ci
```

If no custom steps are specified, a default step to checkout the repository is added automatically.

## Cache Configuration (`cache:`)

Cache configuration using standard GitHub Actions `actions/cache` syntax:

Single cache:
```yaml
cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules
  restore-keys: |
    node-modules-
```

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Complete guide to Claude, Copilot, Codex, and custom engines
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Network Permissions](/gh-aw/reference/network/) - Network access control configuration
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special @mention triggers and context text
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup and configuration
- [Tools Configuration](/gh-aw/reference/tools/) - GitHub and other tools setup
- [Include Directives](/gh-aw/reference/include-directives/) - Modularizing workflows with includes
