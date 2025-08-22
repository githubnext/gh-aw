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
- `output`: Output processing configuration for automatic issue creation and comment posting

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

## Output Processing (`output:`)

Configure automatic output processing from AI agent results:

```yaml
output:
  allowed-domains:                    # Optional: domains allowed in agent output URIs
    - github.com                      # Default GitHub domains are always included
    - api.github.com                  # Additional trusted domains can be specified
    - trusted-domain.com              # URIs from unlisted domains are replaced with "(redacted)"
  issue:
    title-prefix: "[ai] "           # Optional: prefix for issue titles
    labels: [automation, ai-agent]  # Optional: labels to attach to issues
  issue_comment: {}                 # Create comments on issues/PRs from agent output
  pull-request:
    title-prefix: "[ai] "           # Optional: prefix for PR titles
    labels: [automation, ai-agent]  # Optional: labels to attach to PRs
    draft: true                     # Optional: create as draft PR (defaults to true)
  labels:
    allowed: [triage, bug, enhancement] # Mandatory: allowed labels for addition
    max-count: 3                        # Optional: maximum number of labels to add (default: 3)
```

### Security and Sanitization

All agent output is automatically sanitized for security before being processed:

- **XML Character Escaping**: Special characters (`<`, `>`, `&`, `"`, `'`) are escaped to prevent injection attacks
- **URI Protocol Filtering**: Only HTTPS URIs are allowed; other protocols (HTTP, FTP, file://, javascript:, etc.) are replaced with "(redacted)"
- **Domain Allowlisting**: HTTPS URIs are checked against the `allowed-domains` list. Unlisted domains are replaced with "(redacted)"
- **Default Allowed Domains**: When `allowed-domains` is not specified, safe GitHub domains are used by default:
  - `github.com`
  - `github.io`
  - `githubusercontent.com`
  - `githubassets.com`
  - `github.dev`
  - `codespaces.new`
- **Length and Line Limits**: Content is truncated if it exceeds safety limits (0.5MB or 65,000 lines)
- **Control Character Removal**: Non-printable characters and ANSI escape sequences are stripped

### Issue Creation (`output.issue`)

**Behavior:**
- When `output.issue` is configured, the compiler automatically generates a separate `create_issue` job
- This job runs after the main AI agent job completes
- The agent's output content flows from the main job to the issue creation job via job output variables
- The issue creation job parses the output content, using the first non-empty line as the title and the remainder as the body
- **Important**: With output processing, the main job **does not** need `issues: write` permission since the write operation is performed in the separate job

**Generated Job Properties:**
- **Job Name**: `create_issue`
- **Dependencies**: Runs after the main agent job (`needs: [main-job-name]`)
- **Permissions**: Only the issue creation job has `issues: write` permission
- **Timeout**: 10-minute timeout to prevent hanging
- **Environment Variables**: Configuration passed via `GITHUB_AW_ISSUE_TITLE_PREFIX` and `GITHUB_AW_ISSUE_LABELS`
- **Outputs**: Returns `issue_number` and `issue_url` for downstream jobs

### Issue Comment Creation (`output.issue_comment`)

**Behavior:**
- When `output.issue_comment` is configured, the compiler automatically generates a separate `create_issue_comment` job
- This job runs after the main AI agent job completes and **only** if the workflow is triggered by an issue or pull request event
- The agent's output content flows from the main job to the comment creation job via job output variables
- The comment creation job posts the entire agent output as a comment on the triggering issue or pull request
- **Conditional Execution**: The job automatically skips if not running in an issue or pull request context

**Generated Job Properties:**
- **Job Name**: `create_issue_comment`
- **Dependencies**: Runs after the main agent job (`needs: [main-job-name]`)
- **Conditional**: Only runs when `github.event.issue.number || github.event.pull_request.number` is present
- **Permissions**: Only the comment creation job has `issues: write` and `pull-requests: write` permissions
- **Timeout**: 10-minute timeout to prevent hanging
- **Outputs**: Returns `comment_id` and `comment_url` for downstream jobs

**Example workflow using issue creation:**
```yaml
---
on: push
permissions:
  contents: read      # Main job only needs minimal permissions
  actions: read
engine: claude
output:
  issue:
    title-prefix: "[analysis] "
    labels: [automation, code-review]
---

# Code Analysis Agent

Analyze the latest commit and provide insights.
Write your analysis to ${{ env.GITHUB_AW_OUTPUT }} at the end.
```

**Example workflow using comment creation:**
```yaml
---
on:
  issues:
    types: [opened, labeled]
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read      # Main job only needs minimal permissions
  actions: read
engine: claude
output:
  issue_comment: {}
---

# Issue/PR Analysis Agent

Analyze the issue or pull request and provide feedback.
Write your analysis to ${{ env.GITHUB_AW_OUTPUT }} at the end.
```

This automatically creates GitHub issues or comments from the agent's analysis without requiring write permissions on the main job.

### Pull Request Creation (`output.pull-request`)

**Behavior:**
- When `output.pull-request` is configured, the compiler automatically generates a separate `create_output_pull_request` job
- This job runs after the main AI agent job completes
- The agent's output content flows from the main job to the pull request creation job via job output variables
- The job creates a new branch, applies git patches from the agent's output, and creates a pull request
- **Important**: With output processing, the main job **does not** need `contents: write` permission since the write operation is performed in the separate job

**Generated Job Properties:**
- **Job Name**: `create_output_pull_request`
- **Dependencies**: Runs after the main agent job (`needs: [main-job-name]`)
- **Permissions**: Only the pull request creation job has `contents: write` and `pull-requests: write` permissions
- **Timeout**: 10-minute timeout to prevent hanging
- **Environment Variables**: Configuration passed via `GITHUB_AW_PR_TITLE_PREFIX`, `GITHUB_AW_PR_LABELS`, `GITHUB_AW_PR_DRAFT`, `GITHUB_AW_WORKFLOW_ID`, and `GITHUB_AW_BASE_BRANCH`
- **Branch Creation**: Uses cryptographic random hex for secure branch naming (`{workflowId}/{randomHex}`)
- **Git Operations**: Creates branch using git CLI, applies patches, commits changes, and pushes to GitHub
- **Outputs**: Returns `pr_number` and `pr_url` for downstream jobs

**Configuration:**
```yaml
output:
  pull-request:
    title-prefix: "[ai] "           # Optional: prefix for PR titles
    labels: [automation, ai-agent]  # Optional: labels to attach to PRs
    draft: true                     # Optional: create as draft PR (defaults to true)
```

**Example workflow using pull request creation:**
```yaml
---
on: push
permissions:
  actions: read       # Main job only needs minimal permissions
engine: claude
output:
  pull-request:
    title-prefix: "[bot] "
    labels: [automation, ai-generated]
---

# Code Improvement Agent

Analyze the latest commit and suggest improvements.
Generate patches and write them to /tmp/aw.patch.
Write a summary to ${{ env.GITHUB_AW_OUTPUT }} with title and description.
```

**Required Patch Format:**
The agent must create git patches in `/tmp/aw.patch` for the changes to be applied. The pull request creation job validates patch existence and content before proceeding.

### Label Addition (`output.labels`)

**Behavior:**
- When `output.labels` is configured, the compiler automatically generates a separate `add_labels` job
- This job runs after the main AI agent job completes
- The agent's output content flows from the main job to the label addition job via job output variables
- The job parses labels from the agent output (one per line), validates them against the allowed list, and adds them to the current issue or pull request
- **Important**: Only **label addition** is supported; label removal is strictly prohibited and will cause the job to fail
- **Security**: The `allowed` list is mandatory and enforced at runtime - only labels from this list can be added

**Generated Job Properties:**
- **Job Name**: `add_labels`
- **Dependencies**: Runs after the main agent job (`needs: [main-job-name]`)
- **Permissions**: Only the label addition job has `issues: write` and `pull-requests: write` permissions
- **Timeout**: 10-minute timeout to prevent hanging
- **Conditional Execution**: Only runs when `github.event.issue.number` or `github.event.pull_request.number` is available
- **Environment Variables**: Configuration passed via `GITHUB_AW_LABELS_ALLOWED`
- **Outputs**: Returns `labels_added` as a newline-separated list of labels that were successfully added

**Configuration:**
```yaml
output:
  labels:
    allowed: [triage, bug, enhancement]  # Mandatory: list of allowed labels (must be non-empty)
    max-count: 3                         # Optional: maximum number of labels to add (default: 3)
```

**Agent Output Format:**
The agent should write labels to add, one per line, to the `${{ env.GITHUB_AW_OUTPUT }}` file:
```
triage
bug
needs-review
```

**Safety Features:**
- Empty lines in agent output are ignored
- Lines starting with `-` are rejected (no removal operations allowed)
- Duplicate labels are automatically removed
- All requested labels must be in the `allowed` list or the job fails with a clear error message
- Label count is limited by `max-count` setting (default: 3) - exceeding this limit causes job failure
- Only GitHub's `issues.addLabels` API endpoint is used (no removal endpoints)

**Example workflow using label addition:**
```yaml
---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read       # Main job only needs minimal permissions
engine: claude
output:
  labels:
    allowed: [triage, bug, enhancement, documentation, needs-review]
---

# Issue Labeling Agent

Analyze the issue content and add appropriate labels.
Write the labels you want to add (one per line) to ${{ env.GITHUB_AW_OUTPUT }}.
Only use labels from the allowed list: triage, bug, enhancement, documentation, needs-review.
```

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

#### Enhanced Concurrency Policies

GitHub Agentic Workflows automatically generates enhanced concurrency policies based on workflow trigger types to provide better isolation and resource management. Different workflow types receive different concurrency groups and cancellation behavior:

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

**Examples:**

```yaml
# Issue workflow - no cancellation, isolated by issue number
on:
  issues:
    types: [opened, edited]
# Generates: group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number }}"

# PR workflow - with cancellation, isolated by PR number  
on:
  pull_request:
    types: [opened, synchronize]
# Generates: group: "gh-aw-${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}"
#           cancel-in-progress: true

# Mixed workflow - handles both issues and PRs with cancellation
on:
  issues:
    types: [opened, edited]
  pull_request:
    types: [opened, synchronize]
# Generates: group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}"
#           cancel-in-progress: true
```

If you need custom concurrency behavior, you can override the automatic generation by specifying your own `concurrency` section in the frontmatter.

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
  contents: read      # Main job permissions (no issues: write needed)
  actions: read

engine:
  id: claude
  version: beta
  model: claude-3-5-sonnet-20241022

tools:
  github:
    allowed: [get_issue, add_issue_comment]

output:
  issue:
    title-prefix: "[analysis] "
    labels: [automation, ai-analysis]

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

Write your analysis to ${{ env.GITHUB_AW_OUTPUT }} for automatic issue creation.
```

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [MCPs](mcps.md) - Model Context Protocol setup and configuration
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
