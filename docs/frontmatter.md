# ⚙️ Frontmatter Options for GitHub Actions

This guide covers all available frontmatter configuration options for agentic workflows.

## Overview

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

**Agentic-Specific Properties:**
- `engine`: AI engine configuration (claude/codex)
- `tools`: Available tools and MCP servers for the AI engine  
- `stop-time`: Deadline when workflow should stop running (absolute or relative time)
- `max-turns`: Maximum number of chat iterations per run
- `alias`: Alias name for the workflow
- `ai-reaction`: Emoji reaction to add/remove on triggering GitHub item
- `cache`: Cache configuration for workflow dependencies

## Trigger Events (`on:`)

Standard GitHub Actions `on:` trigger section:

```yaml
on:
  issues:
    types: [opened]
```

**Default behavior** (if no `on:` specified):
```yaml
on:
  # Semi-active agent - triggers frequently and on repository activity
  schedule:
    - cron: "0/10 * * * *"
  issues:
    types: [opened, edited, closed]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, closed]
  push:
    branches:
      - main
  workflow_dispatch:
```

## Special `alias:` Trigger

GitHub Agentic Workflows add the convenience `alias:` trigger to create workflows that respond to `@mentions` in issues and comments.

```yaml
on:
  alias:
    name: my-bot  # Optional: defaults to filename without .md extension
```

This automatically creates:
- Issue and PR triggers (`opened`, `edited`, `reopened`)
- Comment triggers (`created`, `edited`)
- Conditional execution matching `@alias-name` mentions

You can combine `alias:` with other events like `workflow_dispatch` or `schedule`:

```yaml
on:
  alias:
    name: my-bot
  workflow_dispatch:
  schedule:
    - cron: "0 9 * * 1"
```

**Note**: You cannot combine `alias` with `issues`, `issue_comment`, or `pull_request` as they would conflict.

**Note**: Using this feature results in the addition of `.github/actions/check-team-member/action.yml` file to the repository when the workflow is compiled. This file is used to check if the user triggering the workflow has appropriate permissions to operate in the repository.

#### Example alias workflow

```markdown
---
on:
  alias:
    name: summarize-issue
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
---

# Issue Summarizer

When someone mentions @summarize-issue in an issue or comment, 
analyze and provide a helpful summary.

The current context text is: "${{ needs.task.outputs.text }}"
```

## Context Text (`needs.task.outputs.text`)

All workflows have access to a special computed `needs.task.outputs.text` value that provides context based on the triggering event:

```markdown
# Analyze this content: "${{ needs.task.outputs.text }}"
```

**How `text` is computed:**
- **Issues**: `title + "\n\n" + body`
- **Pull Requests**: `title + "\n\n" + body`  
- **Issue Comments**: `comment.body`
- **PR Review Comments**: `comment.body`
- **PR Reviews**: `review.body`
- **Other events**: Empty string

**Note**: Using this feature results in the addition of ".github/actions/compute-text/action.yml" file to the repository when the workflow is compiled.

## Permissions (`permissions:`)

Standard GitHub Actions permissions syntax. See [GitHub Actions permissions documentation](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions).

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

## AI Engine (`engine:`)

Specifies which AI engine to use. Defaults to `claude`.

```yaml
engine: claude  # Default: Claude Code
engine: codex   # Experimental: OpenAI Codex CLI with MCP support
```

**Engine Override**:
You can override the engine specified in frontmatter using CLI flags:
```bash
gh aw add weekly-research --engine codex
gh aw compile --engine claude
```

### Simple String Format

```yaml
engine: claude  # or codex
```

### Extended Object Format

```yaml
engine:
  id: claude                        # Required: engine identifier
  version: beta                     # Optional: version of the action
  model: claude-3-5-sonnet-20241022 # Optional: specific LLM model
```

**Fields:**
- **`id`** (required): Engine identifier (`claude`, `codex`)
- **`version`** (optional): Action version (`beta`, `stable`)
- **`model`** (optional): Specific LLM model to use

**Model Defaults:**
- **Claude**: Uses the default model from the claude-code-base-action (typically latest Claude model)
- **Codex**: Defaults to `o4-mini` when no model is specified

## Cost Control Options

### Maximum Turns (`max-turns:`)

Limit the number of chat iterations within a single agentic run:

```yaml
max-turns: 5
```

**Behavior:**
1. Passes the limit to the AI engine (e.g., Claude Code action)
2. Engine stops iterating when the turn limit is reached
3. Helps prevent runaway chat loops and control costs
4. Only applies to engines that support turn limiting (currently Claude)

### Stop Time (`stop-time:`)

Automatically disable workflow after a deadline:

**Relative time delta (calculated from compilation time):**
```yaml
stop-time: "+25h"      # 25 hours from now
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

## Visual Feedback (`ai-reaction:`)

Emoji reaction added/removed on triggering GitHub items:

```yaml
ai-reaction: "eyes"
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

**Note**: Using this feature results in the addition of ".github/actions/reaction/action.yml" file to the repository when the workflow is compiled.

## Cache Configuration (`cache:`)

Cache configuration using GitHub Actions `actions/cache` syntax:

### Single Cache
```yaml
cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules
  restore-keys: |
    node-modules-
```

### Multiple Caches
```yaml
cache:
  - key: node-modules-${{ hashFiles('package-lock.json') }}
    path: node_modules
    restore-keys: |
      node-modules-
  - key: build-cache-${{ github.sha }}
    path: 
      - dist
      - .cache
    restore-keys:
      - build-cache-
    fail-on-cache-miss: false
```

**Supported Parameters:**
- `key:` - Cache key (required)
- `path:` - Files/directories to cache (required, string or array)
- `restore-keys:` - Fallback keys (string or array)
- `upload-chunk-size:` - Chunk size for large files (integer)
- `fail-on-cache-miss:` - Fail if cache not found (boolean)
- `lookup-only:` - Only check cache existence (boolean)

## Standard GitHub Actions Properties

### Run Configuration

```yaml
run-name: "Custom workflow run name"  # Defaults to workflow name
runs-on: ubuntu-latest               # Defaults to ubuntu-latest
timeout_minutes: 30                  # Defaults to 15 minutes
```

### Concurrency Control

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

Defaults to single instance per workflow.

### Environment Variables

```yaml
env:
  CUSTOM_VAR: "value"
  SECRET_VAR: ${{ secrets.MY_SECRET }}
```

### Conditional Execution

```yaml
if: github.event_name == 'push'
```

### Custom Steps

```yaml
steps:
  - name: Custom setup
    run: echo "Custom step before agentic execution"
  - uses: actions/setup-node@v4
    with:
      node-version: '18'
```

## Complete Example

```yaml
---
name: Comprehensive Issue Handler
on:
  issues:
    types: [opened, labeled]
  alias:
    name: issue-bot

permissions:
  issues: write
  contents: read

engine:
  id: claude
  version: beta
  model: claude-3-5-sonnet-20241022

tools:
  github:
    allowed: [get_issue, add_issue_comment, update_issue]

cache:
  key: deps-${{ hashFiles('**/package-lock.json') }}
  path: node_modules

stop-time: "2025-12-31 23:59:59"
ai-reaction: "rocket"

run-name: "Issue Handler - #${{ github.event.issue.number }}"
timeout_minutes: 10

env:
  LOG_LEVEL: info

steps:
  - name: Setup environment
    run: echo "Preparing issue analysis..."

if: github.event.issue.state == 'open'
---

# Comprehensive Issue Handler

Analyze and respond to issues with full context awareness.
Current issue text: "${{ needs.task.outputs.text }}"
```

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [MCPs](mcps.md) - Model Context Protocol setup and configuration
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
