---
title: Frontmatter Options
description: Complete guide to all available frontmatter configuration options for GitHub Agentic Workflows, including triggers, permissions, AI engines, and workflow settings.
sidebar:
  order: 2
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
- `if`: Conditional execution of the workflow
- `steps`: Custom steps for the job
- `cache`: Cache configuration for workflow dependencies

**Properties specific to GitHub Agentic Workflows:**
- `engine`: AI engine configuration (claude/copilot/codex) with optional max-turns setting
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
- "Safe" triggers like `workflow_dispatch`, `schedule`, and `workflow_run` skip permission checks by default
- When permission checks fail, the workflow is automatically cancelled with a warning message
- Users without sufficient permissions will see the workflow start but then immediately stop

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
- **`id`** (required): Engine identifier (`claude`, `copilot`, `codex`)
- **`version`** (optional): Action version (`beta`, `stable`)
- **`model`** (optional): Specific LLM model to use
- **`max-turns`** (optional): Maximum number of chat iterations per run (cost-control option)
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
    file = "/tmp/custom.log"
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
file = "/tmp/custom.log"
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
timeout_minutes: 30                  # Defaults to 15 minutes
```

## Concurrency Control (`concurrency:`)

GitHub Agentic Workflows automatically generates enhanced concurrency policies based on workflow trigger types to provide better isolation and resource management. For example, most workflows produce this:

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

Different workflow types receive different concurrency groups and cancellation behavior:

| Trigger Type | Concurrency Group | Cancellation | Description |
|--------------|-------------------|--------------|-------------|
| `issues` | `gh-aw-${{ github.workflow }}-${{ github.event.issue.number }}` | ❌ | Issue workflows include issue number for isolation |
| `pull_request` | `gh-aw-${{ github.workflow }}-${{ github.event.pull_request.number \|\| github.ref }}` | ✅ | PR workflows include PR number with cancellation |
| `discussion` | `gh-aw-${{ github.workflow }}-${{ github.event.discussion.number }}` | ❌ | Discussion workflows include discussion number |
| Mixed issue/PR | `gh-aw-${{ github.workflow }}-${{ github.event.issue.number \|\| github.event.pull_request.number }}` | ✅ | Mixed workflows handle both contexts with cancellation |
| Alias workflows | `gh-aw-${{ github.workflow }}-${{ github.event.issue.number \|\| github.event.pull_request.number }}` | ❌ | Alias workflows handle both contexts without cancellation |
| Other triggers | `gh-aw-${{ github.workflow }}` | ❌ | Default behavior for schedule, push, etc. |

**Benefits:**
- **Better Isolation**: Workflows operating on different issues/PRs can run concurrently
- **Conflict Prevention**: No interference between unrelated workflow executions  
- **Resource Management**: Pull request workflows can cancel previous runs when updated
- **Predictable Behavior**: Consistent concurrency rules based on trigger type

If you need custom concurrency behavior, you can override the automatic generation by specifying your own `concurrency` section in the frontmatter.

## Environment Variables (`env:`)

GitHub Actions standard `env:` syntax:

```yaml
env:
  CUSTOM_VAR: "value"
  SECRET_VAR: ${{ secrets.MY_SECRET }}
```

## Conditional Execution (`if:`)

Standard GitHub Actions `if:` syntax:

```yaml
if: github.event_name == 'push'
```

## Custom Steps (`steps:`)

Add custom steps before the agentic execution step using GitHub Actions standard `steps:` syntax:

```yaml
steps:
  - name: Custom setup
    run: echo "Custom step before agentic execution"
  - uses: actions/setup-node@v4
    with:
      node-version: '18'
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
