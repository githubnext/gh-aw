---
title: Safe Outputs
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 800
---

The `safe-outputs:` element of your workflow's frontmatter declares that your agentic workflow should conclude with optional automated actions based on the agentic workflow's output. This enables your workflow to write content that is then automatically processed to create GitHub issues, comments, pull requests, or add labels—all without giving the agentic portion of the workflow any write permissions.

**How It Works:**
1. The agentic part of your workflow runs with minimal read-only permissions. It is given additional prompting to write its output to the special known files
2. The compiler automatically generates additional jobs that read this output and perform the requested actions
3. Only these generated jobs receive the necessary write permissions

For example:

```yaml wrap
safe-outputs:
  create-issue:
```

This declares that the workflow should create at most one new issue.

## Available Safe Output Types

| Output Type | Key | Description | Max | Cross-Repo |
|-------------|-----|-------------|-----|-----------|
| **Create Issue** | `create-issue:` | Create GitHub issues | 1 | ✅ |
| **Add Comment** | `add-comment:` | Post comments on issues, PRs, or discussions | 1 | ✅ |
| **Update Issue** | `update-issue:` | Update issue status, title, or body | 1 | ✅ |
| **Update Project** | `update-project:` | Manage GitHub Projects boards and campaign labels | 10 | ❌ |
| **Add Labels** | `add-labels:` | Add labels to issues or PRs | 3 | ✅ |
| **Create PR** | `create-pull-request:` | Create pull requests with code changes | 1 | ✅ |
| **PR Review Comments** | `create-pull-request-review-comment:` | Create review comments on code lines | 1 | ✅ |
| **Create Discussion** | `create-discussion:` | Create GitHub discussions | 1 | ✅ |
| **Create Agent Task** | `create-agent-task:` | Create Copilot agent tasks | 1 | ✅ |
| **Push to PR Branch** | `push-to-pull-request-branch:` | Push changes to PR branch | 1 | ❌ |
| **Update Release** | `update-release:` | Update GitHub release descriptions | 1 | ✅ |
| **Code Scanning Alerts** | `create-code-scanning-alert:` | Generate SARIF security advisories | unlimited | ❌ |
| **No-Op** | `noop:` | Log completion message for transparency (auto-enabled) | 1 | ❌ |
| **Missing Tool** | `missing-tool:` | Report missing tools (auto-enabled) | unlimited | ❌ |

Custom safe output types: [Custom Safe Output Jobs](/gh-aw/guides/custom-safe-outputs/).

### Custom Safe Output Jobs (`jobs:`)

The `jobs:` field creates custom post-processing jobs that execute after the main workflow. Custom jobs are registered as MCP tools for the agent to call.

```yaml wrap
safe-outputs:
  jobs:
    deploy-app:
      description: "Deploy application"
      runs-on: ubuntu-latest
      output: "Deployment completed!"
      permissions:
        contents: write
      inputs:
        environment:
          description: "Target environment"
          required: true
          type: choice
          options: ["staging", "production"]
      steps:
        - name: Deploy
          run: echo "Deploying to ${{ inputs.environment }}"
```

Jobs support standard GitHub Actions properties (`runs-on`, `permissions`, `env`, `if`, `timeout-minutes`) and automatically access agent output via `$GH_AW_AGENT_OUTPUT`. See [Custom Safe Output Jobs](/gh-aw/guides/custom-safe-outputs/).

### Issue Creation (`create-issue:`)

Creates GitHub issues based on workflow output.

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[ai] "            # Optional: prefix for titles
    labels: [automation, agentic]    # Optional: labels to attach
    assignees: [user1, copilot]      # Optional: assignees (use 'copilot' for bot)
    max: 5                           # Optional: max issues (default: 1)
    target-repo: "owner/repo"        # Optional: cross-repository
```

:::caution
Bot assignments (including `copilot`) require a PAT. Store as `COPILOT_GITHUB_TOKEN` (recommended) or legacy `GH_AW_COPILOT_TOKEN` / `GH_AW_GITHUB_TOKEN` in secrets. The default `GITHUB_TOKEN` lacks bot assignment permissions.
:::

### Comment Creation (`add-comment:`)

Posts comments on issues, PRs, or discussions. Defaults to triggering item; configure with `target` for specific items or `"*"` for any.

```yaml wrap
safe-outputs:
  add-comment:
    max: 3                    # Optional: max comments (default: 1)
    target: "*"               # Optional: "triggering" (default), "*", or number
    discussion: true          # Optional: target discussions
    target-repo: "owner/repo" # Optional: cross-repository
```

When combined with `create-issue`, `create-discussion`, or `create-pull-request`, comments automatically include a "Related Items" section with links.

### Add Labels (`add-labels:`)

Adds labels to issues or PRs. If `allowed` is specified, only those labels are permitted.

```yaml wrap
safe-outputs:
  add-labels:
    allowed: [bug, enhancement]  # Optional: restrict to specific labels
    max: 3                       # Optional: max labels (default: 3)
    target: "*"                  # Optional: "triggering" (default), "*", or number
    target-repo: "owner/repo"    # Optional: cross-repository
```

### Issue Updates (`update-issue:`)

Updates issue status, title, or body. Only explicitly enabled fields can be updated. Status must be "open" or "closed".

```yaml wrap
safe-outputs:
  update-issue:
    status:                   # Optional: enable status updates
    title:                    # Optional: enable title updates
    body:                     # Optional: enable body updates
    max: 3                    # Optional: max updates (default: 1)
    target: "*"               # Optional: "triggering" (default), "*", or number
    target-repo: "owner/repo" # Optional: cross-repository
```

### Project Board Updates (`update-project:`)

Manages GitHub Projects boards associated with the repository. The generated job runs with `projects: write` permissions, links the board to the repository, and maintains campaign metadata.

```yaml wrap
safe-outputs:
  update-project:
    max: 20                         # Optional: max project operations (default: 10)
    github-token: ${{ secrets.PROJECTS_PAT }} # Optional: token override with projects:write
```

Agent output for this safe output must include a `project` identifier (name, number, or project URL) and can supply `content_number`, `content_type`, `fields`, and `campaign_id` values. The job adds the referenced issue or pull request to the board, updates custom fields, applies a `campaign:<id>` label, and exposes `project-id`, `project-number`, `project-url`, `campaign-id`, and `item-id` outputs for downstream jobs. Cross-repository targeting is not supported.

### Pull Request Creation (`create-pull-request:`)

Creates pull requests with code changes. Falls back to creating an issue if PR creation fails (e.g., organization settings block it).

```yaml wrap
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "         # Optional: prefix for titles
    labels: [automation]          # Optional: labels to attach
    reviewers: [user1, copilot]   # Optional: reviewers (use 'copilot' for bot)
    draft: true                   # Optional: create as draft (default: true)
    if-no-changes: "warn"         # Optional: "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # Optional: cross-repository
```

:::caution
Bot reviewers (including `copilot`) require a PAT. Store as `COPILOT_GITHUB_TOKEN` (recommended) or legacy `GH_AW_COPILOT_TOKEN` / `GH_AW_GITHUB_TOKEN` in secrets.
:::

> [!NOTE]
> PR creation may fail if "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings → Actions → General → Workflow permissions. When fallback occurs, an issue is created with branch link and error details.

### PR Review Comments (`create-pull-request-review-comment:`)

Creates review comments on specific code lines in PRs. Supports single-line and multi-line comments.

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
    max: 3                    # Optional: max comments (default: 1)
    side: "RIGHT"             # Optional: "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"               # Optional: "triggering" (default), "*", or number
    target-repo: "owner/repo" # Optional: cross-repository
```

### Code Scanning Alerts (`create-code-scanning-alert:`)

Creates security advisories in SARIF format and submits to GitHub Code Scanning. Supports severity levels: error, warning, info, note.

```yaml wrap
safe-outputs:
  create-code-scanning-alert:
    max: 50  # Optional: max findings (default: unlimited)
```

### Push to PR Branch (`push-to-pull-request-branch:`)

Pushes additional changes to a PR's branch. Supports validation via `title-prefix` and `labels` to ensure only approved PRs receive changes.

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                 # Optional: "triggering" (default), "*", or number
    title-prefix: "[bot] "      # Optional: require title prefix
    labels: [automated]         # Optional: require all labels
    if-no-changes: "warn"       # Optional: "warn" (default), "error", or "ignore"
```

When `create-pull-request` or `push-to-pull-request-branch` are enabled, file editing tools (Edit, Write, NotebookEdit) and git commands are automatically added.

### Release Updates (`update-release:`)

Updates GitHub release descriptions with three operations: replace, append, or prepend content.

```yaml wrap
safe-outputs:
  update-release:
    max: 1                       # Optional: max releases (default: 1, max: 10)
    target-repo: "owner/repo"    # Optional: cross-repository
    github-token: ${{ secrets.CUSTOM_TOKEN }}  # Optional: custom token
```

**Operations:**
- **replace** - Completely replaces the release body
- **append** - Adds content to the end with separator and AI attribution
- **prepend** - Adds content to the start with AI attribution and separator

**Agent Output Format:**
```jsonl
{"type": "update_release", "tag": "v1.0.0", "operation": "replace", "body": "New content"}
{"type": "update_release", "tag": "v2.0.0", "operation": "append", "body": "Additional notes"}
{"type": "update_release", "operation": "prepend", "body": "Summary (tag inferred)"}
```

The `tag` field is optional when triggered by release events (automatically inferred from context). The workflow needs read access to releases; only the generated job receives write permissions.

### No-Op Logging (`noop:`)

Enabled by default with any safe-outputs configuration. Allows agents to produce human-visible completion messages when no actions are needed, ensuring workflows never complete silently.

```yaml wrap
safe-outputs:
  create-issue:     # noop enabled automatically
  noop: false       # Or explicitly disable
  # noop:
  #   max: 1        # Or configure max messages (default: 1)
```

**Agent Output Format:**
```jsonl
{"type": "noop", "message": "Analysis complete - no issues found"}
{"type": "noop", "message": "No changes needed, code follows best practices"}
```

Messages are displayed in the workflow conclusion comment (when reaction configured) or step summary. This provides transparency and prevents confusion from silent workflow completion.

### Missing Tool Reporting (`missing-tool:`)

Enabled by default with any safe-outputs configuration. Automatically detects and reports tools lacking permissions or unavailable functionality.

```yaml wrap
safe-outputs:
  create-issue:           # missing-tool enabled automatically
  missing-tool: false     # Or explicitly disable
  # missing-tool:
  #   max: 10             # Or configure max reports
```

### Discussion Creation (`create-discussion:`)

Creates GitHub discussions. The `category` accepts a slug, name, or ID. If omitted, uses the first available category.

```yaml wrap
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "     # Optional: prefix for titles
    category: "general"       # Optional: category slug, name, or ID
    max: 3                    # Optional: max discussions (default: 1)
    target-repo: "owner/repo" # Optional: cross-repository
```

### Agent Task Creation (`create-agent-task:`)

Creates GitHub Copilot agent tasks to delegate coding tasks. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` (recommended) or legacy `GH_AW_COPILOT_TOKEN` / `GH_AW_GITHUB_TOKEN`.

```yaml wrap
safe-outputs:
  create-agent-task:
    base: main                # Optional: base branch (defaults to current)
    target-repo: "owner/repo" # Optional: cross-repository
```

:::caution
Requires a PAT. The default `GITHUB_TOKEN` lacks agent task permissions.
:::

## Cross-Repository Operations

Many safe outputs support `target-repo` for cross-repository operations. Requires a PAT (via `github-token` field or `GH_AW_GITHUB_TOKEN`) with access to target repositories. The default `GITHUB_TOKEN` only has permissions for the current repository.

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
  add-comment:
    target-repo: "org/notifications-repo"
    target: "123"
```

Use specific repository names (no wildcards). Grant tokens minimum necessary permissions.

## Automatically Added Tools

When `create-pull-request` or `push-to-pull-request-branch` are configured, file editing tools (Edit, MultiEdit, Write, NotebookEdit) and git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

## Security and Sanitization

Agent output is automatically sanitized: XML characters escaped, only HTTPS URIs allowed, domains checked against allowlist (defaults to GitHub domains), content truncated if exceeding 0.5MB or 65k lines, control characters stripped.

```yaml wrap
safe-outputs:
  allowed-domains:        # Optional: additional trusted domains
    - api.github.com      # GitHub domains always included
```

## Global Configuration Options

### Custom GitHub Token (`github-token:`)

Token precedence: `GH_AW_GITHUB_TOKEN` (override) → `GITHUB_TOKEN` (default). Override globally or per safe output:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}  # Global override
  create-issue:
  create-pull-request:
    github-token: ${{ secrets.PR_PAT }}    # Per-output override
```

### Maximum Patch Size (`max-patch-size:`)

Limits git patch size for PR operations (range: 1-10,240 KB, default: 1024 KB):

```yaml wrap
safe-outputs:
  max-patch-size: 512  # Optional: max patch size in KB
  create-pull-request:
```

## Assigning to Copilot

Use `assignees: copilot` in `create-issue` or `reviewers: copilot` in `create-pull-request` to assign to the Copilot bot. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` (recommended) or legacy `GH_AW_COPILOT_TOKEN` / `GH_AW_GITHUB_TOKEN`. The default `GITHUB_TOKEN` lacks bot assignment permissions.

```yaml wrap
safe-outputs:
  create-issue:
    assignees: copilot
  create-pull-request:
    reviewers: copilot
```

## Custom Runner Image

Use `runs-on` to specify a custom runner image for safe output jobs (default: `ubuntu-slim`):

```yaml wrap
safe-outputs:
  runs-on: ubuntu-22.04
  create-issue:
```

## Threat Detection

Automatically enabled with safe outputs. Analyzes agent output for prompt injection, secret leaks, and malicious patches before applying safe outputs.

```yaml wrap
safe-outputs:
  create-pull-request:
  threat-detection: true              # Explicit enable (default)
  # Or with custom configuration:
  # threat-detection:
  #   enabled: true
  #   prompt: "Focus on SQL injection"
  #   steps:                          # Custom security scanning steps
  #     - name: Run TruffleHog
  #       uses: trufflesecurity/trufflehog@main
```

See [Threat Detection Guide](/gh-aw/guides/threat-detection/) for details.

## Campaign Workflows

Campaign workflows combine `create-issue` with `update-project` to launch coordinated initiatives. The project job returns a campaign identifier, applies `campaign:<id>` labels, and keeps project boards synchronized with generated issues and pull requests. Downstream worker workflows can reuse the same identifier to update board status. For end-to-end guidance, see [Campaign Workflows](/gh-aw/guides/campaigns/).

## Related Documentation

- [Threat Detection Guide](/gh-aw/guides/threat-detection/) - Complete threat detection documentation and examples
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
