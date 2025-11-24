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
| **Close Issue** | `close-issue:` | Close issues with comment | 1 | ✅ |
| **Add Comment** | `add-comment:` | Post comments on issues, PRs, or discussions | 1 | ✅ |
| **Update Issue** | `update-issue:` | Update issue status, title, or body | 1 | ✅ |
| **Update Project** | `update-project:` | Manage GitHub Projects boards and campaign labels | 10 | ❌ |
| **Add Labels** | `add-labels:` | Add labels to issues or PRs | 3 | ✅ |
| **Add Reviewer** | `add-reviewer:` | Add reviewers to pull requests | 3 | ✅ |
| **Assign Milestone** | `assign-milestone:` | Assign issues to milestones | 1 | ✅ |
| **Create PR** | `create-pull-request:` | Create pull requests with code changes | 1 | ✅ |
| **Close PR** | `close-pull-request:` | Close pull requests without merging | 10 | ✅ |
| **PR Review Comments** | `create-pull-request-review-comment:` | Create review comments on code lines | 1 | ✅ |
| **Create Discussion** | `create-discussion:` | Create GitHub discussions | 1 | ✅ |
| **Close Discussion** | `close-discussion:` | Close discussions with comment and resolution | 1 | ✅ |
| **Create Agent Task** | `create-agent-task:` | Create Copilot agent tasks | 1 | ✅ |
| **Assign to Agent** | `assign-to-agent:` | Assign Copilot agents to issues | 1 | ✅ |
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
    title-prefix: "[ai] "            # prefix for titles
    labels: [automation, agentic]    # labels to attach
    assignees: [user1, copilot]      # assignees (use 'copilot' for bot)
    max: 5                           # max issues (default: 1)
    target-repo: "owner/repo"        # cross-repository
```

### Close Issue (`close-issue:`)

Closes GitHub issues with an optional comment and state reason. Filters by labels and title prefix control which issues can be closed.

```yaml wrap
safe-outputs:
  close-issue:
    target: "triggering"              # "triggering" (default), "*", or number
    required-labels: [automated]      # only close with any of these labels
    required-title-prefix: "[bot]"    # only close matching prefix
    max: 20                           # max closures (default: 1)
    target-repo: "owner/repo"         # cross-repository
```

**Target**: `"triggering"` (requires issue event), `"*"` (any issue), or number (specific issue).

**State Reasons**: `completed`, `not_planned`, `reopened` (default: `completed`).

### Comment Creation (`add-comment:`)

Posts comments on issues, PRs, or discussions. Defaults to triggering item; configure `target` for specific items or `"*"` for any.

```yaml wrap
safe-outputs:
  add-comment:
    max: 3                    # max comments (default: 1)
    target: "*"               # "triggering" (default), "*", or number
    discussion: true          # target discussions
    target-repo: "owner/repo" # cross-repository
```

When combined with `create-issue`, `create-discussion`, or `create-pull-request`, comments automatically include a "Related Items" section.

### Add Labels (`add-labels:`)

Adds labels to issues or PRs. Specify `allowed` to restrict to specific labels.

```yaml wrap
safe-outputs:
  add-labels:
    allowed: [bug, enhancement]  # restrict to specific labels
    max: 3                       # max labels (default: 3)
    target: "*"                  # "triggering" (default), "*", or number
    target-repo: "owner/repo"    # cross-repository
```

### Add Reviewer (`add-reviewer:`)

Adds reviewers to pull requests. Specify `reviewers` to restrict to specific GitHub usernames.

```yaml wrap
safe-outputs:
  add-reviewer:
    reviewers: [user1, copilot]  # restrict to specific reviewers
    max: 3                       # max reviewers (default: 3)
    target: "*"                  # "triggering" (default), "*", or number
    target-repo: "owner/repo"    # cross-repository
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

Use `reviewers: copilot` to assign the Copilot PR reviewer bot. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` or legacy `GH_AW_COPILOT_TOKEN` / `GH_AW_GITHUB_TOKEN`.

### Assign Milestone (`assign-milestone:`)

Assigns issues to milestones. Specify `allowed` to restrict to specific milestone titles.

```yaml wrap
safe-outputs:
  assign-milestone:
    allowed: [v1.0, v2.0]    # restrict to specific milestone titles
    max: 1                   # max assignments (default: 1)
    target-repo: "owner/repo" # cross-repository
```

### Issue Updates (`update-issue:`)

Updates issue status, title, or body. Only explicitly enabled fields can be updated. Status must be "open" or "closed".

```yaml wrap
safe-outputs:
  update-issue:
    status:                   # enable status updates
    title:                    # enable title updates
    body:                     # enable body updates
    max: 3                    # max updates (default: 1)
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
```

### Project Board Updates (`update-project:`)

Manages GitHub Projects boards. Generated job runs with `projects: write` permissions, links the board to the repository, and maintains campaign metadata.

```yaml wrap
safe-outputs:
  update-project:
    max: 20                         # max project operations (default: 10)
    github-token: ${{ secrets.PROJECTS_PAT }} # token override with projects:write
```

Agent output must include a `project` identifier (name, number, or URL) and can supply `content_number`, `content_type`, `fields`, and `campaign_id`. The job adds the issue or PR to the board, updates custom fields, applies `campaign:<id>` labels, and exposes `project-id`, `project-number`, `project-url`, `campaign-id`, and `item-id` outputs. Cross-repository targeting not supported.

### Pull Request Creation (`create-pull-request:`)

Creates pull requests with code changes. Falls back to creating an issue if PR creation fails (e.g., organization settings block it).

```yaml wrap
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "         # prefix for titles
    labels: [automation]          # labels to attach
    reviewers: [user1, copilot]   # reviewers (use 'copilot' for bot)
    draft: true                   # create as draft (default: true)
    if-no-changes: "warn"         # "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # cross-repository
```

:::note
PR creation may fail if "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings → Actions → General → Workflow permissions. When fallback occurs, an issue is created with branch link and error details.
:::

### Close Pull Request (`close-pull-request:`)

Closes pull requests without merging, with an optional comment. Filters by labels and title prefix control which PRs can be closed.

```yaml wrap
safe-outputs:
  close-pull-request:
    target: "triggering"              # "triggering" (default), "*", or number
    required-labels: [automated, stale] # only close with any of these labels
    required-title-prefix: "[bot]"    # only close matching prefix
    max: 10                           # max closures (default: 1)
    target-repo: "owner/repo"         # cross-repository
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

Useful for automated cleanup of stale bot PRs or closing PRs that don't meet criteria. The comment explains why the PR was closed and includes workflow attribution.

### PR Review Comments (`create-pull-request-review-comment:`)

Creates review comments on specific code lines in PRs. Supports single-line and multi-line comments.

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
    max: 3                    # max comments (default: 1)
    side: "RIGHT"             # "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
```

### Code Scanning Alerts (`create-code-scanning-alert:`)

Creates security advisories in SARIF format and submits to GitHub Code Scanning. Supports severity: error, warning, info, note.

```yaml wrap
safe-outputs:
  create-code-scanning-alert:
    max: 50  # max findings (default: unlimited)
```

### Push to PR Branch (`push-to-pull-request-branch:`)

Pushes changes to a PR's branch. Validates via `title-prefix` and `labels` to ensure only approved PRs receive changes.

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                 # "triggering" (default), "*", or number
    title-prefix: "[bot] "      # require title prefix
    labels: [automated]         # require all labels
    if-no-changes: "warn"       # "warn" (default), "error", or "ignore"
```

When `create-pull-request` or `push-to-pull-request-branch` are enabled, file editing tools (Edit, Write, NotebookEdit) and git commands are added.

### Release Updates (`update-release:`)

Updates GitHub release descriptions: replace (complete replacement), append (add to end), or prepend (add to start).

```yaml wrap
safe-outputs:
  update-release:
    max: 1                       # max releases (default: 1, max: 10)
    target-repo: "owner/repo"    # cross-repository
    github-token: ${{ secrets.CUSTOM_TOKEN }}  # custom token
```

Agent output format: `{"type": "update_release", "tag": "v1.0.0", "operation": "replace", "body": "..."}`. The `tag` field is optional for release events (inferred from context). Workflow needs read access; only the generated job receives write permissions.

### No-Op Logging (`noop:`)

Enabled by default. Allows agents to produce completion messages when no actions are needed, preventing silent workflow completion.

```yaml wrap
safe-outputs:
  create-issue:     # noop enabled automatically
  noop: false       # explicitly disable
```

Agent output: `{"type": "noop", "message": "Analysis complete - no issues found"}`. Messages appear in the workflow conclusion comment or step summary.

### Missing Tool Reporting (`missing-tool:`)

Enabled by default. Automatically detects and reports tools lacking permissions or unavailable functionality.

```yaml wrap
safe-outputs:
  create-issue:           # missing-tool enabled automatically
  missing-tool: false     # explicitly disable
```

### Discussion Creation (`create-discussion:`)

Creates GitHub discussions. The `category` accepts a slug, name, or ID (defaults to first available category if omitted).

```yaml wrap
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "     # prefix for titles
    category: "general"       # category slug, name, or ID
    max: 3                    # max discussions (default: 1)
    target-repo: "owner/repo" # cross-repository
```

### Close Discussion (`close-discussion:`)

Closes GitHub discussions with optional comment and resolution reason. Filters by category, labels, and title prefix control which discussions can be closed.

```yaml wrap
safe-outputs:
  close-discussion:
    target: "triggering"         # "triggering" (default), "*", or number
    required-category: "Ideas"   # only close in category
    required-labels: [resolved]  # only close with labels
    required-title-prefix: "[ai]" # only close matching prefix
    max: 1                       # max closures (default: 1)
    target-repo: "owner/repo"    # cross-repository
```

**Target**: `"triggering"` (requires discussion event), `"*"` (any discussion), or number (specific discussion).

**Resolution Reasons**: `RESOLVED`, `DUPLICATE`, `OUTDATED`, `ANSWERED`.

### Agent Task Creation (`create-agent-task:`)

Creates GitHub Copilot agent tasks. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` (recommended) or legacy `GH_AW_COPILOT_TOKEN` / `GH_AW_GITHUB_TOKEN`. The default `GITHUB_TOKEN` lacks agent task permissions.

```yaml wrap
safe-outputs:
  create-agent-task:
    base: main                # base branch (defaults to current)
    target-repo: "owner/repo" # cross-repository
```

### Assign to Agent (`assign-to-agent:`)

Assigns GitHub Copilot coding agents to issues. Requires **all four workflow permissions** (`actions: write`, `contents: write`, `issues: write`, `pull-requests: write`) plus a token with sufficient scope.

```yaml wrap
safe-outputs:
  assign-to-agent:
    default-agent: "copilot"  # default: "copilot"
    max: 1                    # max assignments (default: 1)
    target-repo: "owner/repo" # cross-repository
```

**Permission Requirements:**

Assigning agents requires **all** of these permissions in the workflow:
```yaml
permissions:
  actions: write
  contents: write
  issues: write
  pull-requests: write
```

**Token Requirements:**

The default `GITHUB_TOKEN` **lacks permissions** to assign agents. You must use one of the following token methods:

1. **GitHub App installation token** (recommended):
   ```yaml wrap
   safe-outputs:
     app:
       app-id: ${{ vars.APP_ID }}
       private-key: ${{ secrets.APP_PRIVATE_KEY }}
     assign-to-agent:
   ```

2. **Classic PAT with `repo` scope**:
   ```yaml wrap
   safe-outputs:
     github-token: ${{ secrets.CLASSIC_PAT }}
     assign-to-agent:
   ```

3. **Fine-grained PAT** with Write access for Issues, Pull requests, Contents, and Actions.

:::caution
Fine-grained and classic PATs may fail with "Resource not accessible" errors depending on organization settings. GitHub App tokens provide the most reliable access.
:::

**Agent Output Format:**
```json
{
  "type": "assign_to_agent",
  "issue_number": 123,
  "agent": "copilot"
}
```

**Supported Agents:**
- `copilot` - GitHub Copilot coding agent (`copilot-swe-agent`)
- `claude` - Claude coding agent (`claude-swe-agent`)  
- `codex` - Codex coding agent (`codex-swe-agent`)

**Repository Settings:**

Ensure these settings are configured:
1. **Actions permissions**: Settings → Actions → General → Workflow permissions → "Read and write permissions"
2. **Copilot enabled**: Verify Copilot is enabled for your repository
3. **Organization settings**: Check if your org restricts bot assignments

**Cross-Repository:**

Requires `target-repo` and a token (PAT or GitHub App) with access to the target repository:
```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  assign-to-agent:
    target-repo: "org/other-repo"
```

Reference: [GitHub Copilot agent documentation](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/)

## Cross-Repository Operations

Many safe outputs support `target-repo` for cross-repository operations. Requires a PAT (via `github-token` or `GH_AW_GITHUB_TOKEN`) with access to target repositories. The default `GITHUB_TOKEN` only has permissions for the current repository.

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
```

Use specific repository names (no wildcards).

## Automatically Added Tools

When `create-pull-request` or `push-to-pull-request-branch` are configured, file editing tools (Edit, MultiEdit, Write, NotebookEdit) and git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

## Security and Sanitization

Agent output is automatically sanitized: XML escaped, HTTPS URIs only, domain allowlist (defaults to GitHub), content truncated at 0.5MB or 65k lines, control characters stripped.

```yaml wrap
safe-outputs:
  allowed-domains:        # additional trusted domains
    - api.github.com      # GitHub domains always included
```

## Global Configuration Options

### Custom GitHub Token (`github-token:`)

Token precedence: `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN` (default). Override globally or per safe output:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}  # global
  create-issue:
  create-pull-request:
    github-token: ${{ secrets.PR_PAT }}    # per-output
```

### GitHub App Token (`app:`)

Use GitHub App installation tokens instead of PATs for enhanced security. Tokens are minted on-demand at job start and auto-revoked at job end, even on failure.

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}                 # required
    private-key: ${{ secrets.APP_PRIVATE_KEY }} # required
    owner: "my-org"                             # installation owner
    repositories: ["repo1", "repo2"]            # scope to repositories
  create-issue:
```

**Benefits**: On-demand minting, automatic revocation, fine-grained permissions, better attribution, clear audit trail.

**Repository Scoping**: Not specified (current repo only), empty with `owner` (all repos in installation), or specified (listed repos only).

**Import Support**: App config can be imported from shared workflows. Local config takes precedence.

:::tip
Use GitHub App tokens for org-wide automation. Better security and audit capabilities than PATs.
:::

### Maximum Patch Size (`max-patch-size:`)

Limits git patch size for PR operations (1-10,240 KB, default: 1024 KB):

```yaml wrap
safe-outputs:
  max-patch-size: 512  # max patch size in KB
  create-pull-request:
```

## Assigning to Copilot

Use `assignees: copilot` or `reviewers: copilot` to assign to the Copilot bot. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` (recommended) or legacy `GH_AW_COPILOT_TOKEN` / `GH_AW_GITHUB_TOKEN`. The default `GITHUB_TOKEN` lacks bot assignment permissions.

```yaml wrap
safe-outputs:
  create-issue:
    assignees: copilot
  create-pull-request:
    reviewers: copilot
```

## Custom Runner Image

Specify a custom runner image for safe output jobs (default: `ubuntu-slim`):

```yaml wrap
safe-outputs:
  runs-on: ubuntu-22.04
  create-issue:
```

## Threat Detection

Automatically enabled. Analyzes agent output for prompt injection, secret leaks, and malicious patches before applying safe outputs. See [Threat Detection Guide](/gh-aw/guides/threat-detection/) for details.

```yaml wrap
safe-outputs:
  create-pull-request:
  threat-detection: true              # explicit enable (default)
```

## Campaign Workflows

Combine `create-issue` with `update-project` to launch coordinated initiatives. The project job returns a campaign identifier, applies `campaign:<id>` labels, and keeps boards synchronized. See [Campaign Workflows](/gh-aw/guides/campaigns/).

## Related Documentation

- [Threat Detection Guide](/gh-aw/guides/threat-detection/) - Complete threat detection documentation and examples
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
