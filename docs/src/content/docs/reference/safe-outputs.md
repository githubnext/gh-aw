---
title: Safe Outputs
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 800
---

The [`safe-outputs:`](/gh-aw/reference/glossary/#safe-outputs) (validated GitHub operations) element of your workflow's [frontmatter](/gh-aw/reference/glossary/#frontmatter) declares that your agentic workflow should conclude with optional automated actions based on the agentic workflow's output. This enables your workflow to write content that is then automatically processed to create GitHub issues, comments, pull requests, or add labels - all without giving the agentic portion of the workflow any write permissions.

Safe outputs enforce security through separation: agents run read-only and request actions via structured output, while separate permission-controlled jobs execute those requests. This provides least privilege, defense against prompt injection, auditability, and controlled limits per operation.

Example:

```yaml wrap
safe-outputs:
  create-issue:
```

The agent requests issue creation; a separate job with `issues: write` creates it.

## Available Safe Output Types

### Issues & Discussions

- [**Create Issue**](#issue-creation-create-issue) (`create-issue`) - Create GitHub issues (max: 1)
- [**Update Issue**](#issue-updates-update-issue) (`update-issue`) - Update issue status, title, or body (max: 1)
- [**Close Issue**](#close-issue-close-issue) (`close-issue`) - Close issues with comment (max: 1)
- [**Link Sub-Issue**](#link-sub-issue-link-sub-issue) (`link-sub-issue`) - Link issues as sub-issues (max: 1)
- [**Create Discussion**](#discussion-creation-create-discussion) (`create-discussion`) - Create GitHub discussions (max: 1)
- [**Update Discussion**](#discussion-updates-update-discussion) (`update-discussion`) - Update discussion title, body, or labels (max: 1)
- [**Close Discussion**](#close-discussion-close-discussion) (`close-discussion`) - Close discussions with comment and resolution (max: 1)

### Pull Requests

- [**Create PR**](/gh-aw/reference/safe-outputs-pull-requests/#pull-request-creation-create-pull-request) (`create-pull-request`) - Create pull requests with code changes (default max: 1, configurable)
- [**Update PR**](#pull-request-updates-update-pull-request) (`update-pull-request`) - Update PR title or body (max: 1)
- [**Close PR**](#close-pull-request-close-pull-request) (`close-pull-request`) - Close pull requests without merging (max: 10)
- [**PR Review Comments**](#pr-review-comments-create-pull-request-review-comment) (`create-pull-request-review-comment`) - Create review comments on code lines (max: 10)
- [**Reply to PR Review Comment**](#reply-to-pr-review-comment-reply-to-pull-request-review-comment) (`reply-to-pull-request-review-comment`) - Reply to existing review comments (max: 10)
- [**Resolve PR Review Thread**](#resolve-pr-review-thread-resolve-pull-request-review-thread) (`resolve-pull-request-review-thread`) - Resolve review threads after addressing feedback (max: 10)
- [**Push to PR Branch**](/gh-aw/reference/safe-outputs-pull-requests/#push-to-pr-branch-push-to-pull-request-branch) (`push-to-pull-request-branch`) - Push changes to PR branch (default max: 1, configurable, same-repo only)

### Labels, Assignments & Reviews

- [**Add Comment**](#comment-creation-add-comment) (`add-comment`) - Post comments on issues, PRs, or discussions (max: 1)
- [**Hide Comment**](#hide-comment-hide-comment) (`hide-comment`) - Hide comments on issues, PRs, or discussions (max: 5)
- [**Add Labels**](#add-labels-add-labels) (`add-labels`) - Add labels to issues or PRs (max: 3)
- [**Remove Labels**](#remove-labels-remove-labels) (`remove-labels`) - Remove labels from issues or PRs (max: 3)
- [**Add Reviewer**](#add-reviewer-add-reviewer) (`add-reviewer`) - Add reviewers to pull requests (max: 3)
- [**Assign Milestone**](#assign-milestone-assign-milestone) (`assign-milestone`) - Assign issues to milestones (max: 1)
- [**Assign to Agent**](#assign-to-agent-assign-to-agent) (`assign-to-agent`) - Assign Copilot coding agent to issues or PRs (max: 1)
- [**Assign to User**](#assign-to-user-assign-to-user) (`assign-to-user`) - Assign users to issues (max: 1)
- [**Unassign from User**](#unassign-from-user-unassign-from-user) (`unassign-from-user`) - Remove user assignments from issues or PRs (max: 1)

### Projects, Releases & Assets

- [**Create Project**](#project-creation-create-project) (`create-project`) - Create new GitHub Projects boards (max: 1, cross-repo)
- [**Update Project**](#project-board-updates-update-project) (`update-project`) - Manage GitHub Projects boards (max: 10, same-repo only)
- [**Create Project Status Update**](#project-status-updates-create-project-status-update) (`create-project-status-update`) - Create project status updates
- [**Update Release**](#release-updates-update-release) (`update-release`) - Update GitHub release descriptions (max: 1)
- [**Upload Assets**](#asset-uploads-upload-asset) (`upload-asset`) - Upload files to orphaned git branch (max: 10, same-repo only). **Prefer `upload-artifact` with `skip-archive` instead.**

### Security & Agent Tasks

- [**Dispatch Workflow**](#workflow-dispatch-dispatch-workflow) (`dispatch-workflow`) - Trigger other workflows with inputs (max: 3, same-repo only)
- [**Call Workflow**](#workflow-call-call-workflow) (`call-workflow`) - Call reusable workflows via compile-time fan-out (max: 1, same-repo only)
- [**Dispatch Repository Event**](#repository-dispatch-dispatch_repository) (`dispatch_repository`) - Trigger `repository_dispatch` events in external repositories, experimental (cross-repo)
- [**Code Scanning Alerts**](#code-scanning-alerts-create-code-scanning-alert) (`create-code-scanning-alert`) - Generate SARIF security advisories (max: unlimited, same-repo only)
- [**Autofix Code Scanning Alerts**](#autofix-code-scanning-alerts-autofix-code-scanning-alert) (`autofix-code-scanning-alert`) - Create automated fixes for code scanning alerts (max: 10, same-repo only)
- [**Create Agent Session**](#agent-session-creation-create-agent-session) (`create-agent-session`) - Create Copilot coding agent sessions (max: 1)

### System Types (Auto-Enabled)

- [**No-Op**](#no-op-logging-noop) (`noop`) - Log completion message for transparency (max: 1, same-repo only)
- [**Missing Tool**](#missing-tool-reporting-missing-tool) (`missing-tool`) - Report missing tools (max: unlimited, same-repo only)
- [**Missing Data**](#missing-data-reporting-missing-data) (`missing-data`) - Report missing data required to achieve goals (max: unlimited, same-repo only)

### Custom Safe Output Jobs (`jobs:`)

Create custom post-processing jobs registered as Model Context Protocol (MCP) tools. Support standard GitHub Actions properties and auto-access agent output via `$GH_AW_AGENT_OUTPUT`. See [Custom Safe Output Jobs](/gh-aw/reference/custom-safe-outputs/).

### GitHub Action Wrappers (`actions:`)

Mount any public GitHub Action as a once-callable MCP tool. The compiler pins the action reference to a SHA at compile time and derives the tool's input schema from the action's `action.yml`. See [GitHub Action Wrappers](/gh-aw/reference/custom-safe-outputs/#github-action-wrappers-safe-outputsactions).

### Issue Creation (`create-issue:`)

Creates GitHub issues based on workflow output.

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[ai] "            # prefix for titles
    labels: [automation, agentic]    # labels to attach
    assignees: [user1, copilot]      # assignees (use 'copilot' for bot)
    max: 5                           # max issues (default: 1)
    expires: 7                       # auto-close after 7 days (or false to disable)
    group: true                      # group as sub-issues under parent
    close-older-issues: true         # close previous issues from same workflow
    target-repo: "owner/repo"        # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for comprehensive documentation on `target-repo`, `allowed-repos`, and cross-repository authentication.

> [!TIP]
> Use `footer: false` to omit the AI-generated footer while preserving workflow-id markers for searchability. See [Footer Control](/gh-aw/reference/footers/) for details.

#### Auto-Expiration

The `expires` field auto-closes issues after a time period. Supports day-string format (`7d`, `2w`, `1m`, `1y`, `2h`) or `false` to disable expiration. Integer values (e.g., `expires: 7`) are also accepted as shorthand for days and can be migrated to string format with `gh aw fix --write`. Generates `agentics-maintenance.yml` workflow that runs at the minimum required frequency based on the shortest expiration time across all workflows:

- 1 day or less → every 2 hours
- 2 days → every 6 hours
- 3-4 days → every 12 hours
- 5+ days → daily

Hours less than 24 are treated as 1 day minimum for expiration calculation.

To explicitly disable expiration (useful when create-issue has a default expiration), use `expires: false`:

#### Issue Grouping

The `group` field (default: `false`) automatically organizes multiple issues as sub-issues under a parent issue. When enabled:

- Parent issues are automatically created and managed using the workflow ID as the group identifier
- Child issues are linked to the parent using GitHub's sub-issue relationships
- Maximum of 64 sub-issues per parent issue
- Parent issues include metadata tracking all sub-issues

This is useful for workflows that create multiple related issues, such as planning workflows that break down epics into tasks, or batch processing workflows that create issues for individual items.

**Example:**

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[plan] "
    labels: [plan, ai-generated]
    max: 5
    group: true
```

#### Auto-Close Older Issues

The `close-older-issues` field (default: `false`) automatically closes previous open issues from the same workflow when a new issue is created. This is useful for workflows that generate recurring reports or status updates, ensuring only the latest issue remains open.

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[weekly-report] "
    labels: [report, automation]
    close-older-issues: true
```

When enabled:

- Searches for open issues containing the same workflow-id marker in their body
- Closes found issues as "not planned" with a comment linking to the new issue
- Maximum 10 older issues will be closed
- Only runs if the new issue creation succeeds

#### Group By Day

The `group-by-day` field (default: `false`) groups multiple same-day workflow runs into a single issue. When enabled, the handler searches for an existing open issue created **today (UTC)** with the same workflow-id marker (or `close-older-key` if set). If found, the new content is posted as a **comment** on that existing issue instead of creating a new one.

```yaml wrap
safe-outputs:
  create-issue:
    title-prefix: "[Contribution Check Report]"
    labels: [report]
    close-older-issues: true
    group-by-day: true
```

This is useful for scheduled workflows (e.g. every 4 hours) that produce recurring daily reports: all runs on the same day contribute to one issue, eliminating duplicate open/closed issues. The max-count slot is not consumed when posting as a comment; on failure of the pre-check, normal issue creation proceeds as a fallback.

#### Searching for Workflow-Created Items

All items created by workflows (issues, pull requests, discussions, and comments) include a hidden **workflow-id marker** in their body:

```html
<!-- gh-aw-workflow-id: WORKFLOW_NAME -->
```

You can use this marker to find all items created by a specific workflow on GitHub.com.

**Search Examples:**

```
# Open issues from a specific workflow
repo:owner/repo is:issue is:open "gh-aw-workflow-id: daily-team-status" in:body

# All items from any workflow in an org
org:your-org "gh-aw-workflow-id:" in:body

# Comments from a specific workflow
repo:owner/repo "gh-aw-workflow-id: bot-responder" in:comments
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
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    state-reason: "duplicate"         # completed (default), not_planned, duplicate
```

**Target**: `"triggering"` (requires issue event), `"*"` (any issue), or number (specific issue).

**State Reasons**: `completed`, `not_planned`, `duplicate` (default: `completed`). Can also be set per-item in agent output.

### Comment Creation (`add-comment:`)

Posts comments on issues, PRs, or discussions. Defaults to triggering item; use `target: "*"` for any, or number for specific items. When combined with `create-issue`, `create-discussion`, or `create-pull-request`, includes "Related Items" section.

```yaml wrap
safe-outputs:
  add-comment:
    max: 3                       # max comments (default: 1)
    target: "*"                  # "triggering" (default), "*", or number
    discussions: false           # exclude discussions:write permission (default: true)
    target-repo: "owner/repo"    # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    hide-older-comments: true    # hide previous comments from same workflow
    allowed-reasons: [outdated]  # restrict hiding reasons (optional)
    footer: false                # omit AI-generated footer (default: true)
```

> [!TIP]
> Use `footer: false` to suppress the "Generated by..." attribution line in posted comments. See [Footer Control](/gh-aw/reference/footers/) for global and per-handler options.

The author of the parent issue, PR, or discussion receiving the comment is automatically preserved as an allowed mention. This means `@username` references to the issue/PR/discussion author are not neutralized when the workflow posts a reply.

#### Hide Older Comments

Set `hide-older-comments: true` to minimize previous comments from the same workflow (identified by `GITHUB_WORKFLOW`) before posting new ones. Useful for status updates. Allowed reasons: `spam`, `abuse`, `off_topic`, `outdated` (default), `resolved`.

#### Append-Only Status Comments

By default, gh-aw posts an activation comment when a workflow starts, then updates that same comment with the final status.

If you prefer an append-only timeline (never editing existing comments), set:

```yaml wrap
safe-outputs:
  messages:
    append-only-comments: true
```

When enabled, the workflow completion notifier creates a new comment instead of editing the activation comment.

### Hide Comment (`hide-comment:`)

Collapses comments in GitHub UI with reason. Requires GraphQL node IDs (e.g., `IC_kwDOABCD123456`), not REST numeric IDs. Reasons: `spam`, `abuse`, `off_topic`, `outdated`, `resolved`.

```yaml wrap
safe-outputs:
  hide-comment:
    max: 5                    # max comments (default: 5)
    target-repo: "owner/repo" # cross-repository
```

### Add Labels (`add-labels:`)

Adds labels to issues or PRs. Specify `allowed` to restrict to specific labels, or `blocked` to deny specific label patterns regardless of the allow list.

```yaml wrap
safe-outputs:
  add-labels:
    allowed: [bug, enhancement]  # restrict to specific labels
    blocked: ["~*", "*[bot]"]   # deny labels matching these glob patterns
    max: 3                       # max labels (default: 3)
    target: "*"                  # "triggering" (default), "*", or number
    target-repo: "owner/repo"    # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
```

#### Blocked Label Patterns

The `blocked` field accepts glob patterns that are evaluated before the `allowed` list. Any label matching a blocked pattern is rejected, even if it also appears in the allowed list. This provides infrastructure-level protection against prompt injection attacks in repositories with many labels where maintaining an exhaustive allowlist is impractical.

Common patterns:

| Pattern | Effect |
|---------|--------|
| `~*` | Denies all labels starting with `~` (often used as workflow triggers) |
| `*[bot]` | Denies all labels ending with `[bot]` (administrative bot labels) |
| `stale` | Denies the exact `stale` label |

```yaml wrap
safe-outputs:
  add-labels:
    blocked: ["~*", "*[bot]"]    # Blocked patterns evaluated first
    allowed: [bug, enhancement]  # Allowed list applied after blocked check
    max: 5
```

### Remove Labels (`remove-labels:`)

Removes labels from issues or PRs. Specify `allowed` to restrict which labels can be removed, or `blocked` to prevent removal of specific label patterns. If a label is not present on the item, it will be silently skipped.

```yaml wrap
safe-outputs:
  remove-labels:
    allowed: [automated, stale]  # restrict to specific labels (optional)
    blocked: ["~*"]              # deny removal of labels matching these glob patterns
    max: 3                       # max operations (default: 3)
    target: "*"                  # "triggering" (default), "*", or number
    target-repo: "owner/repo"    # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
```

**Target**: `"triggering"` (requires issue/PR event), `"*"` (any issue/PR), or number (specific issue/PR).

When `allowed` is omitted or set to `null`, any labels can be removed. Use `allowed` to restrict removal to specific labels only, providing control over which labels agents can manipulate. The `blocked` field takes precedence over `allowed`.

**Example use case**: Label lifecycle management where agents add temporary labels during triage and remove them once processed.

```yaml wrap
safe-outputs:
  add-labels:
    allowed: [needs-triage, automation]
  remove-labels:
    allowed: [needs-triage]  # agents can remove triage label after processing
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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

Use `reviewers: [copilot]` to assign the Copilot PR reviewer bot. See [Assign to Agent](/gh-aw/reference/assign-to-copilot/).

### Assign Milestone (`assign-milestone:`)

Assigns issues to milestones. Specify `allowed` to restrict to specific milestone titles. Agents can provide a milestone by title (`milestone_title`) instead of by number (`milestone_number`), and the handler resolves the number internally.

```yaml wrap
safe-outputs:
  assign-milestone:
    allowed: [v1.0, v2.0]    # restrict to specific milestone titles
    auto_create: true         # auto-create milestones in the allowed list if they don't exist
    max: 1                   # max assignments (default: 1)
    target-repo: "owner/repo" # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

When `auto_create: true` is set, any milestone from the `allowed` list that does not yet exist in the repository is created automatically before the assignment. Without `auto_create`, the handler returns a clear error listing the available milestones and suggesting `auto_create: true`.

### Issue Updates (`update-issue:`)

Updates issue status, title, or body. Only explicitly enabled fields can be updated. Status must be "open" or "closed". The `operation` field controls how body updates are applied: `append` (default), `prepend`, `replace`, or `replace-island`. Use `title-prefix` to restrict updates to issues whose titles start with a specific prefix.

```yaml wrap
safe-outputs:
  update-issue:
    status:                   # enable status updates
    title:                    # enable title updates
    body:                     # enable body updates
    title-prefix: "[bot] "    # only update issues with this title prefix
    max: 3                    # max updates (default: 1)
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires issue event), `"*"` (any issue), or number (specific issue).

When using `target: "*"`, the agent must provide `issue_number` or `item_number` in the output to identify which issue to update.

**Title Prefix**: When `title-prefix` is set, the update is rejected if the target issue's current title does not start with the specified prefix. This ensures agents can only modify issues that have been explicitly tagged for automated updates.

**Operation Types** (for body updates):

- `append` (default): Adds content to the end with separator and attribution
- `prepend`: Adds content to the start with separator and attribution
- `replace`: Completely replaces existing body with new content and attribution
- `replace-island`: Updates a specific section marked with HTML comments

Agent output format: `{"type": "update_issue", "issue_number": 123, "operation": "append", "body": "..."}`. The `operation` field is optional (defaults to `append`).

### Pull Request Updates (`update-pull-request:`)

Updates PR title or body. Both fields are enabled by default. The `operation` field controls how body updates are applied: `append` (default), `prepend`, or `replace`.

```yaml wrap
safe-outputs:
  update-pull-request:
    title: true               # enable title updates (default: true)
    body: true                # enable body updates (default: true)
    footer: false             # omit AI-generated footer from body updates (default: true)
    max: 1                    # max updates (default: 1)
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires PR event), `"*"` (any PR), or number (specific PR).

When using `target: "*"`, the agent must provide `pull_request_number` in the output to identify which pull request to update.

**Operation Types**: Same as `update-issue` above (`append`, `prepend`, `replace`). Title updates always replace the existing title. Disable fields by setting to `false`.

### Link Sub-Issue (`link-sub-issue:`)

Links issues as sub-issues using GitHub's parent-child issue relationships. Supports filtering by labels and title prefixes for both parent and sub issues.

```yaml wrap
safe-outputs:
  link-sub-issue:
    parent-required-labels: [epic]        # parent must have these labels
    parent-title-prefix: "[Epic]"         # parent must match prefix
    sub-required-labels: [task]           # sub must have these labels
    sub-title-prefix: "[Task]"            # sub must match prefix
    max: 1                                # max links (default: 1)
    target-repo: "owner/repo"             # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

Agent output includes `parent_issue_number` and `sub_issue_number`. Validation ensures both issues exist and meet label/prefix requirements before linking.

### Project Creation (`create-project:`)

Creates new GitHub Projects V2 boards. Requires a write-capable PAT or GitHub App token ([project token authentication](/gh-aw/patterns/project-ops/#project-token-authentication)); default `GITHUB_TOKEN` lacks Projects v2 access. Supports optional view configuration to create custom project views at creation time.

Use separate tokens as shown in ProjectOps examples:
- `GH_AW_READ_PROJECT_TOKEN` for `tools.github` reads
- `GH_AW_WRITE_PROJECT_TOKEN` for `safe-outputs` project writes

```yaml wrap
safe-outputs:
  create-project:
    max: 1                              # max operations (default: 1)
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
    target-owner: "myorg"               # default target owner (optional)
    title-prefix: "Project"             # default title prefix (optional)
    views:                              # optional: auto-create views
      - name: "Sprint Board"
        layout: board
        filter: "is:issue is:open"
      - name: "Task Tracker"
        layout: table
```

When `views` are configured, they are created automatically after project creation. GitHub's default "View 1" will remain, and configured views are created as additional views.

The `target-owner` field is an optional default. When configured, the agent can omit the owner field in tool calls, and the default will be used. The agent can still override by providing an explicit owner value.

**Without default** (agent must provide owner):

```javascript
create_project({
  title: "Project: Security Q1 2025",
  owner: "myorg",
  owner_type: "org",  // "org" or "user" (default: "org")
  item_url: "https://github.com/myorg/repo/issues/123"  // Optional issue to add
});
```

**With default configured** (agent only needs title):

```javascript
create_project({
  title: "Project: Security Q1 2025"
  // owner uses configured default
  // owner_type defaults to "org"
  // Can still override: owner: "...", owner_type: "user"
});
```

Optionally include `item_url` (GitHub issue URL) to add the issue as the first project item. Exposes outputs: `project-id`, `project-number`, `project-title`, `project-url`, `item-id` (if item added).

> [!IMPORTANT]
> **Token Requirements**: The default `GITHUB_TOKEN` **cannot** create projects. You **must** configure a PAT with Projects permissions:
>
> - **Classic PAT**: `project` scope (user projects) or `project` + `repo` scope (org projects)
> - **Fine-grained PAT**: Organization permissions → Projects: Read & Write

> [!NOTE]
> You can configure views directly during project creation using the `views` field (see above), or later using `update-project` to add custom fields and additional views. For pattern guidance, see [Projects & Monitoring](/gh-aw/patterns/monitoring/).

### Project Board Updates (`update-project:`)

Manages GitHub Projects boards. Requires a write-capable PAT or GitHub App token ([project token authentication](/gh-aw/patterns/project-ops/#project-token-authentication)); default `GITHUB_TOKEN` lacks Projects v2 access. Update-only by default; set `create_if_missing: true` to create boards (requires appropriate token permissions).

```yaml wrap
safe-outputs:
  update-project:
    project: "https://github.com/orgs/myorg/projects/42"  # required: target project URL
    max: 20                         # max operations (default: 10)
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
    target-repo: "org/default-repo"         # optional: default repo for target_repo resolution
    allowed-repos: ["org/repo-a", "org/repo-b"]  # optional: additional repos for cross-repo items
    views:                          # optional: auto-create views
      - name: "Sprint Board"
        layout: board
        filter: "is:issue is:open"
      - name: "Task Tracker"
        layout: table
      - name: "Roadmap"
        layout: roadmap
```

Agent output messages **must** explicitly include the `project` field — the configured value is for documentation purposes only. Exposes outputs: `project-id`, `project-number`, `project-url`, `item-id`.

#### Cross-Repository Content Resolution

For **organization-level projects** that aggregate issues from multiple repositories, use `target_repo` in the agent output to specify which repo contains the issue or PR:

```yaml wrap
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
    allowed-repos: ["org/docs", "org/backend", "org/frontend"]
```

The agent can then specify `target_repo` alongside `content_number`:

```json
{
  "type": "update_project",
  "project": "https://github.com/orgs/myorg/projects/42",
  "content_type": "issue",
  "content_number": 123,
  "target_repo": "org/docs",
  "fields": { "Status": "In Progress" }
}
```

Without `target_repo`, the workflow's host repository is used to resolve `content_number`.

#### Supported Field Types

GitHub Projects V2 supports various custom field types. The following field types are automatically detected and handled:

- **`TEXT`** - Text fields (default)
- **`DATE`** - Date fields (format: `YYYY-MM-DD`)
- **`NUMBER`** - Numeric fields (story points, estimates, etc.)
- **`ITERATION`** - Sprint/iteration fields (matched by iteration title)
- **`SINGLE_SELECT`** - Dropdown/select fields (creates missing options automatically)

**Example field usage:**

```yaml
fields:
  status: "In Progress"          # SINGLE_SELECT field
  start_date: "2026-01-04"       # DATE field
  story_points: 8                # NUMBER field
  sprint: "Sprint 42"            # ITERATION field (by title)
  priority: "High"               # SINGLE_SELECT field
```

> [!NOTE]
> Field names are case-insensitive and automatically normalized (e.g., `story_points` matches `Story Points`).

#### Creating Project Views

Project views can be created automatically by declaring them in the `views` array. Views are created when the workflow runs, after processing update_project items from the agent.

**View configuration:**

```yaml
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
    views:
      - name: "Sprint Board"        # required: view name
        layout: board               # required: table, board, or roadmap
        filter: "is:issue is:open"  # optional: filter query
      - name: "Task Tracker"
        layout: table
        filter: "is:issue is:pr"
      - name: "Roadmap"
        layout: roadmap
```

**View properties:**

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | View name (e.g., "Sprint Board", "Task Tracker") |
| `layout` | string | Yes | View layout: `table`, `board`, or `roadmap` |
| `filter` | string | No | Filter query (e.g., `is:issue is:open`, `label:bug`) |
| `visible-fields` | array | No | Field IDs to display (table/board only, not roadmap) |

**Layout types:** `table` (list), `board` (Kanban), `roadmap` (timeline). The `filter` field accepts standard GitHub search syntax (e.g., `is:issue is:open`, `label:bug`).

Views are created automatically during workflow execution. The workflow must include at least one `update_project` operation to provide the target project URL.

### Project Status Updates (`create-project-status-update:`)

Creates status updates on GitHub Projects boards to communicate progress, findings, and trends. Status updates appear in the project's Updates tab and provide a historical record of execution. Requires a write-capable PAT or GitHub App token ([project token authentication](/gh-aw/patterns/project-ops/#project-token-authentication)); default `GITHUB_TOKEN` lacks Projects v2 access.

```yaml wrap
safe-outputs:
  create-project-status-update:
    project: "https://github.com/orgs/myorg/projects/73"  # required: target project URL
    max: 1                          # max updates per run (default: 1)
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
```

Agent output messages **must** explicitly include the `project` field. Often used by scheduled and orchestrator workflows to post run summaries.

#### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `project` | URL | Full GitHub project URL (e.g., `https://github.com/orgs/myorg/projects/73`). **Required** in every agent output message. |
| `body` | Markdown | Status update content with summary, findings, and next steps |

#### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `status` | Enum | `ON_TRACK` | Status indicator: `ON_TRACK`, `AT_RISK`, `OFF_TRACK`, `COMPLETE`, `INACTIVE` |
| `start_date` | Date | Today | Run start date (format: `YYYY-MM-DD`) |
| `target_date` | Date | Today | Projected completion or milestone date (format: `YYYY-MM-DD`) |

**Status values:** `ON_TRACK` (on schedule), `AT_RISK` (potential issues), `OFF_TRACK` (behind schedule), `COMPLETE` (finished), `INACTIVE` (paused).

Exposes outputs: `status-update-id`, `project-id`, `status`.

### Pull Request Creation (`create-pull-request:`)

Creates PRs with code changes. Includes configurable [Protected Files](/gh-aw/reference/safe-outputs-pull-requests/#protected-files) against supply chain attacks.

See the full reference: [Safe Outputs (Pull Requests) — create-pull-request](/gh-aw/reference/safe-outputs-pull-requests/#pull-request-creation-create-pull-request)

```yaml wrap
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "
    labels: [automation]
    reviewers: [user1, copilot]
    assignees: [user1]            # assignees for fallback issues created when PR creation cannot proceed (including protected-files fallback)
    protected-files: fallback-to-issue  # create review issue if protected files modified, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.
```

### Close Pull Request (`close-pull-request:`)

Closes PRs without merging with optional comment. Filter by labels and title prefix. Target: `"triggering"` (PR event), `"*"` (any), or number.

```yaml wrap
safe-outputs:
  close-pull-request:
    target: "triggering"              # "triggering" (default), "*", or number
    required-labels: [automated, stale] # only close with these labels
    required-title-prefix: "[bot]"    # only close matching prefix
    max: 10                           # max closures (default: 1)
    target-repo: "owner/repo"         # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

### PR Review Comments (`create-pull-request-review-comment:`)

Creates review comments on specific code lines in PRs. Supports single-line and multi-line comments. Comments are buffered and submitted as a single PR review (see `submit-pull-request-review` below).

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
    max: 3                    # max comments (default: 10)
    side: "RIGHT"             # "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    footer: "if-body"         # footer control: "always", "none", or "if-body"
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

When `target: "*"` is configured, the agent must supply `pull_request_number` in each `create_pull_request_review_comment` tool call to identify which PR to comment on — omitting it will cause the comment to fail. For cross-repository scenarios, the agent can also supply `repo` (in `owner/repo` format) to route the comment to a PR in a different repository; the value must match `target-repo` or appear in `allowed-repos`.

### Reply to PR Review Comment (`reply-to-pull-request-review-comment:`)

Replies to existing review comments on pull requests. Use this to respond to reviewer feedback, answer questions, or acknowledge comments. The `comment_id` must be the numeric ID of an existing review comment.

```yaml wrap
safe-outputs:
  reply-to-pull-request-review-comment:
    max: 10                              # max replies (default: 10)
    target: "triggering"                 # "triggering" (default), "*", or number
    target-repo: "owner/repo"            # cross-repository
    allowed-repos: ["org/other-repo"]    # additional allowed repositories
    footer: true                         # add AI-generated footer (default: true)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

The `footer` field controls whether AI-generated footers are added to PR review comments:

- `"always"` (default) - Always include footer on review comments
- `"none"` - Never include footer on review comments
- `"if-body"` - Only include footer when the review has a body text

With `footer: "if-body"`, approval reviews without body text appear clean without the AI-generated footer, while reviews with explanatory text still include the footer for attribution.

### Submit PR Review (`submit-pull-request-review:`)

Submits a consolidated pull request review with a status decision. All `create-pull-request-review-comment` outputs are automatically collected and included as inline comments in the review.

If the agent calls `submit_pull_request_review`, it can specify a review `body` and `event` (APPROVE, REQUEST_CHANGES, or COMMENT). Both fields are optional — `event` defaults to COMMENT when omitted, and `body` is only required for REQUEST_CHANGES. The agent can also submit a body-only review (e.g., APPROVE) without any inline comments.

If the agent does not call `submit_pull_request_review` at all, buffered comments are still submitted as a COMMENT review automatically.

When the workflow is not triggered by a pull request (e.g. `workflow_dispatch`), set `target` to the PR number (e.g. `${{ github.event.inputs.pr_number }}`) so the review can be submitted. Same semantics as [add-comment](#comment-creation-add-comment) `target`: `"triggering"` (default), `"*"` (use `pull_request_number` from the message), or an explicit number.

For cross-repository scenarios, use `target-repo` to specify the repository where the PR lives. This mirrors the behavior of `create-pull-request-review-comment` and `add-comment`.

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
    max: 10
  submit-pull-request-review:
    max: 1            # max reviews to submit (default: 1)
    target: "triggering"  # or "*", or e.g. ${{ github.event.inputs.pr_number }} when not in pull_request trigger
    target-repo: "owner/repo"  # cross-repository: submit review on PR in another repo
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    allowed-events: [COMMENT, REQUEST_CHANGES]  # restrict allowed review event types (default: all allowed)
    footer: false     # omit AI-generated footer from review body (default: true)
```

Use `allowed-events` to restrict which review event types the agent can submit. This provides infrastructure-level enforcement — for example, `allowed-events: [COMMENT, REQUEST_CHANGES]` prevents the agent from submitting APPROVE reviews regardless of what the agent attempts to output. If omitted, all event types (APPROVE, COMMENT, REQUEST_CHANGES) are allowed.

### Resolve PR Review Thread (`resolve-pull-request-review-thread:`)

Resolves review threads on pull requests. Allows AI agents to mark review conversations as resolved after addressing the feedback. Uses the GitHub GraphQL API with the `resolveReviewThread` mutation.

By default, resolution is scoped to the triggering PR. Use `target`, `target-repo`, and `allowed-repos` for cross-repository thread resolution.

```yaml wrap
safe-outputs:
  resolve-pull-request-review-thread:
    max: 10                              # max threads to resolve (default: 10)
    target: "triggering"                 # "triggering" (default), "*", or number
    target-repo: "owner/repo"            # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for documentation on `target-repo`, `allowed-repos`, and cross-repository authentication.

**Agent output format:**

```json
{"type": "resolve_pull_request_review_thread", "thread_id": "PRRT_kwDOABCD..."}
```

### Code Scanning Alerts (`create-code-scanning-alert:`)

Creates security advisories in SARIF format and submits to GitHub Code Scanning. Supports severity: error, warning, info, note.

```yaml wrap
safe-outputs:
  create-code-scanning-alert:
    max: 50  # max findings (default: unlimited)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

### Autofix Code Scanning Alerts (`autofix-code-scanning-alert:`)

Creates automated fixes for code scanning alerts. Agent outputs fix suggestions that are submitted to GitHub Code Scanning.

```yaml wrap
safe-outputs:
  autofix-code-scanning-alert:
    max: 10  # max autofixes (default: 10)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

### Push to PR Branch (`push-to-pull-request-branch:`)

Pushes changes to a PR's branch. Includes configurable [Protected Files](/gh-aw/reference/safe-outputs-pull-requests/#protected-files) against supply chain attacks.

See the full reference: [Safe Outputs (Pull Requests) — push-to-pull-request-branch](/gh-aw/reference/safe-outputs-pull-requests/#push-to-pr-branch-push-to-pull-request-branch)

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                 # "triggering" (default), "*", or number
    title-prefix: "[bot] "      # require title prefix
    labels: [automated]         # require all labels
    protected-files: fallback-to-issue  # create review issue if protected files modified
```

When `push-to-pull-request-branch` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

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

### Asset Uploads (`upload-asset:`)

:::caution[Prefer `upload-artifact` with `skip-archive`]
For uploading images, charts, and screenshots, prefer using `upload-artifact` with `skip-archive: true` instead (see the [shared upload-artifact configuration](https://github.com/github/gh-aw/blob/main/.github/workflows/shared/safe-output-upload-artifact.md)). It puts less pressure on the git storage system and automatically destroys the image once the artifact expires. Use `upload-asset` only when you need persistent, publicly addressable URLs that survive artifact expiration.
:::

Uploads files (screenshots, charts, reports) to orphaned git branch with predictable URLs: `https://github.com/{owner}/{repo}/blob/{branch}/{filename}?raw=true`. Agent registers files via `upload_asset` tool; separate job with `contents: write` commits them.

```yaml wrap
safe-outputs:
  upload-asset:
    branch: "assets/my-workflow"     # default: "assets/${{ github.workflow }}"
    max-size: 5120                   # KB (default: 10240 = 10MB)
    allowed-exts: [.png, .jpg, .svg] # default: [.png, .jpg, .jpeg]
    max: 20                          # default: 10
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Branch Requirements**: New branches require `assets/` prefix for security. Existing branches allow any name. Create custom branches manually:

```bash
git checkout --orphan my-custom-branch && git rm -rf . && git commit --allow-empty -m "Initialize" && git push origin my-custom-branch
```

**Security**: File path validation (workspace/`/tmp` only), extension allowlist, size limits, SHA-256 verification, orphaned branch isolation, minimal permissions.

**Outputs**: `published_count`, `branch_name`. **Limits**: Same-repo only, max 50MB/file, 100 assets/run.

### No-Op Logging (`noop:`)

:::danger[Required when no action is taken]
**`noop` MUST be called when no GitHub action is needed.** This is the #1 runtime failure mode for safe-output workflows. If the agent finishes without calling any safe-output tool, the workflow fails silently with no output. Always call `noop` when your analysis concludes that no action is required.
:::

Enabled by default. Allows agents to produce completion messages when no actions are needed, preventing silent workflow completion.

```yaml wrap
safe-outputs:
  create-issue:     # noop enabled automatically
  noop: false       # explicitly disable
```

**When to call `noop`**: Any time no GitHub action (issue, comment, PR, label, etc.) is needed — e.g., no issues found, no changes detected, or repository already in desired state. Do NOT call `noop` if any other safe-output action was taken.

Agent output: `{"noop": {"message": "No action needed: analysis complete - no issues found"}}`. Messages appear in the workflow conclusion comment or step summary.

**Always include explicit `noop` instructions in your workflow prompts:**

```markdown
If no action is needed, you MUST call the `noop` tool with a message explaining why:
{"noop": {"message": "No action needed: [brief explanation]"}}
```

### Missing Tool Reporting (`missing-tool:`)

Enabled by default. Automatically detects and reports tools lacking permissions or unavailable functionality.

```yaml wrap
safe-outputs:
  create-issue:           # missing-tool enabled automatically
  missing-tool: false     # explicitly disable
```

### Missing Data Reporting (`missing-data:`)

Enabled by default. Allows AI agents to report missing data required to achieve their goals, encouraging truthfulness over hallucination.

```yaml wrap
safe-outputs:
  missing-data:
    create-issue: true          # create GitHub issues for missing data
    title-prefix: "[data]"      # prefix for issue titles (default: "[missing data]")
    labels: [data, blocked]     # labels to attach to issues
    max: 10                     # max reports per run (default: unlimited)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

When `create-issue: true`, the agent creates or updates GitHub issues documenting what data is needed and why, possible alternatives, and context for how the data would be used. This rewards honest AI behavior and helps teams improve data accessibility for future agent runs.

### Discussion Creation (`create-discussion:`)

Creates discussions with optional `category` (slug, name, or ID; defaults to first available). `expires` field auto-closes after period (integers, `2h`, `7d`, `2w`, `1m`, `1y`, or `false` to disable; hours < 24 treated as 1 day) as "OUTDATED" with comment. Generates maintenance workflow with dynamic frequency based on shortest expiration time (see Auto-Expiration section above).

**Category Naming Standard**: Use lowercase, plural category names (e.g., `audits`, `general`, `reports`) for consistency and better searchability. GitHub Discussion category IDs (starting with `DIC_`) are also supported.

```yaml wrap
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "        # prefix for titles
    category: "announcements"    # category slug, name, or ID (use lowercase)
    expires: 3                   # auto-close after 3 days (or false to disable)
    max: 3                       # max discussions (default: 1)
    target-repo: "owner/repo"    # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    fallback-to-issue: true      # fallback to issue creation on permission errors (default: true)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

#### Fallback to Issue Creation

The `fallback-to-issue` field (default: `true`) automatically falls back to creating an issue when discussion creation fails (e.g., discussions disabled, insufficient `discussions: write` permissions, or org policy restrictions). The issue body notes it was intended to be a discussion. Set to `false` to fail instead of falling back.

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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Target**: `"triggering"` (requires discussion event), `"*"` (any discussion), or number (specific discussion).

**Resolution Reasons**: `RESOLVED`, `DUPLICATE`, `OUTDATED`, `ANSWERED`.

### Discussion Updates (`update-discussion:`)

Updates discussion title, body, or labels. Only explicitly enabled fields can be updated.

```yaml wrap
safe-outputs:
  update-discussion:
    title:                    # enable title updates
    body:                     # enable body updates
    labels:                   # enable label updates
    allowed-labels: [bug, idea] # restrict to specific labels
    max: 1                    # max updates (default: 1)
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

**Field Enablement**: Include `title:`, `body:`, or `labels:` keys to enable updates for those fields. Without these keys, the field cannot be updated. Setting `allowed-labels` implicitly enables label updates.

**Target**: `"triggering"` (requires discussion event), `"*"` (any discussion), or number (specific discussion).

When using `target: "*"`, the agent must provide `discussion_number` in the output to identify which discussion to update.

### Workflow Dispatch (`dispatch-workflow:`)

Triggers other workflows in the same repository using GitHub's `workflow_dispatch` event. This enables orchestration patterns, such as orchestrator workflows that coordinate multiple worker workflows.

> [!NOTE]
> When installing a workflow with `gh aw add`, workflows listed in `dispatch-workflow` are automatically fetched and added to the target repository alongside the main workflow.

**Shorthand Syntax:**

```yaml wrap
safe-outputs:
  dispatch-workflow: [worker-workflow, scanner-workflow]
```

#### Configuration

- **`workflows`** (required) - List of workflow names (without `.md` extension) that the agent is allowed to dispatch. Each workflow must exist in the same repository and support the `workflow_dispatch` trigger.
- **`max`** (optional) - Maximum number of workflow dispatches allowed (default: 1, maximum: 50). This prevents excessive workflow triggering.

#### Validation Rules

At compile time, the compiler validates that each workflow exists (`.md`, `.lock.yml`, or `.yml`), declares `workflow_dispatch` in its `on:` section, does not self-reference, and resolves the correct file extension.

#### Defining Workflow Inputs

Define `workflow_dispatch` inputs in the target workflow so the agent can provide values when dispatching:

```yaml wrap
---
on:
  workflow_dispatch:
    inputs:
      environment:
        description: "Target deployment environment"
        required: true
        type: choice
        options: [staging, production]
      dry_run:
        type: boolean
        default: false
---
```

#### Rate Limiting

To respect GitHub API rate limits, the handler automatically enforces a 5-second delay between consecutive workflow dispatches. The first dispatch has no delay.

**Security**: Same-repo only; only allowlisted workflows can be dispatched; compile-time validation catches errors early.

### Workflow Call (`call-workflow:`)

Calls reusable workflows (`workflow_call`) via compile-time fan-out—no GitHub API call at runtime. The compiler reads each worker's `workflow_call.inputs`, generates a typed MCP tool per worker, and emits a conditional `uses:` job for each. At runtime, only the worker whose name the agent selected runs.

Unlike `dispatch-workflow` (which fires a `workflow_dispatch` event and loses the original actor context), `call-workflow` preserves `github.actor` and billing attribution because the worker job is part of the same workflow run.

> [!NOTE]
> When installing a workflow with `gh aw add`, workflows listed in `call-workflow` are automatically fetched and added to the target repository alongside the main workflow.

**Shorthand Syntax:**

```yaml wrap
safe-outputs:
  call-workflow: [spring-boot-bugfix, frontend-dep-upgrade]
```

**Full Syntax:**

```yaml wrap
safe-outputs:
  call-workflow:
    workflows:
      - spring-boot-bugfix
      - frontend-dep-upgrade
    max: 1
```

#### Configuration

- **`workflows`** (required) - List of workflow names (without `.md` extension) that the agent is allowed to call. Each workflow must exist in the same repository and declare `workflow_call` as a trigger.
- **`max`** (optional) - Maximum number of times the agent may invoke the tool per run (default: 1, maximum: 50). Since a single `call_workflow_name` step output is produced, only the last selected worker executes regardless of `max`; in practice, leave this at 1.

#### Worker Inputs

All agent arguments are serialized into a `payload` JSON string passed via `call_workflow_payload`. Workers always receive this `payload` input. To use typed inputs directly (without parsing JSON), declare additional `workflow_call.inputs` beyond `payload` — the compiler auto-derives `fromJSON(...).<inputName>` forwarding for each, so workers can reference `${{ inputs.<name> }}` directly:

```yaml title="deploy.md (worker)"
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
      environment:
        description: Target environment
        type: choice
        options: [dev, staging, production]
        required: true
      dry_run:
        type: boolean
        required: false
```

Supported input types: `string`, `number`, `boolean`, `choice` (rendered as an enum).

#### Validation Rules

At compile time, the compiler validates:

1. **Workflow existence** - Each workflow must exist as a `.lock.yml`, `.yml`, or `.md` file.
2. **`workflow_call` trigger** - Each worker must declare `workflow_call` in its `on:` section.
3. **No self-reference** - A gateway cannot call itself.
4. **File resolution** - The compiler resolves the correct extension and embeds it in the generated job.

#### Comparing `call-workflow` and `dispatch-workflow`

| | `call-workflow` | `dispatch-workflow` |
|---|---|---|
| Mechanism | Compile-time `uses:` job | Runtime `workflow_dispatch` API |
| API calls | None | One per dispatch |
| `github.actor` | Preserved | Replaced by triggering actor |
| Billing | Attributed to triggering run | Attributed to dispatched run |
| Cross-repository | No | No |
| Worker trigger | `workflow_call` | `workflow_dispatch` |

Use `call-workflow` for deterministic fan-out where actor attribution and zero API overhead matter. Use `dispatch-workflow` when workers need to run asynchronously or outlive the parent run.

**Security**: Same-repo only; only allowlisted workflows can be called; compile-time validation catches misconfiguration early.

### Repository Dispatch (`dispatch_repository`)

> [!CAUTION]
> This is an experimental feature. Compiling a workflow with `dispatch_repository` emits a warning: `Using experimental feature: dispatch_repository`. The API may change in future releases.

Triggers [`repository_dispatch`](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#repository_dispatch) events in external repositories. Unlike `dispatch-workflow` (same-repo only), `dispatch_repository` is designed for cross-repository orchestration.

Each key under `dispatch_repository:` defines a named tool exposed to the agent:

```yaml wrap
safe-outputs:
  dispatch_repository:
    trigger_ci:
      description: Trigger CI in another repository
      workflow: ci.yml
      event_type: ci_trigger
      repository: ${{ inputs.target_repo }}   # GitHub Actions expressions supported
      inputs:
        environment:
          type: choice
          options: [staging, production]
          default: staging
      max: 1
    notify_service:
      workflow: notify.yml
      event_type: notify_event
      allowed_repositories:
        - org/service-repo
        - ${{ vars.DYNAMIC_REPO }}             # Expressions bypass slug format validation
      inputs:
        message:
          type: string
```

#### Configuration Fields (per tool)

- **`workflow`** (required) — Identifier forwarded in `client_payload.workflow` so the receiving workflow can route by job type.
- **`event_type`** (required) — The `event_type` sent with the `repository_dispatch` event.
- **`repository`** (required, unless `allowed_repositories` is set) — Static `owner/repo` slug or a GitHub Actions expression (`${{ ... }}`). Expressions are passed through without format validation.
- **`allowed_repositories`** (required, unless `repository` is set) — List of allowed `owner/repo` slugs or expressions. The agent selects the target from this list at runtime.
- **`inputs`** (optional) — Structured input schema forwarded in `client_payload`. Supports `type: string`, `type: choice` (with `options`), and `default` values.
- **`description`** (optional) — Human-readable description of the tool shown to the agent.
- **`max`** (optional) — Maximum number of dispatches allowed per run (default: 1).

#### Security

- **Cross-repo allowlist** — At runtime the handler validates the target repository against the configured `repository` or `allowed_repositories` before calling the API (SEC-005).
- **Staged mode** — Supports `staged: true` for preview without dispatching.

### Agent Session Creation (`create-agent-session:`)

Creates Copilot coding agent sessions from workflow output. Allows workflows to spawn new agent sessions for follow-up work.

```yaml wrap
safe-outputs:
  create-agent-session:
    base: "main"                 # base branch for agent session PR
    max: 1                       # max sessions (default: 1, maximum: 10)
    target-repo: "owner/repo"    # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

### Assign to Agent (`assign-to-agent:`)

Programmatically assigns GitHub Copilot coding agent to **existing** issues or pull requests through workflow automation. This safe output automates the [standard GitHub workflow for assigning issues to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot).

```yaml wrap
safe-outputs:
  assign-to-agent:
    name: "copilot"            # default agent (default: "copilot")
    model: "claude-opus-4.6"   # default AI model (default: "auto")
    custom-agent: "agent-id"   # default custom agent ID (optional)
    custom-instructions: "..."  # default custom instructions (optional)
    allowed: [copilot]         # restrict to specific agents (optional)
    max: 1                     # max assignments (default: 1)
    target: "triggering"       # "triggering" (default), "*", or number
    target-repo: "owner/repo"  # where the issue lives (cross-repository)
    pull-request-repo: "owner/repo"      # where the PR should be created (may differ from issue repo)
    allowed-pull-request-repos: [owner/repo1, owner/repo2]  # additional allowed PR repositories
    base-branch: "develop"     # target branch for PR (default: target repo's default branch)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

See **[Assign to Copilot](/gh-aw/reference/assign-to-copilot/)** for complete configuration options and authorization setup.

If you're creating new issues and want to assign an agent immediately, use `assignees: copilot` in your [`create-issue`](#issue-creation-create-issue) configuration instead.

### Assign to User (`assign-to-user:`)

Assigns users to issues. Restrict with `allowed` list. Target: `"triggering"` (issue event), `"*"` (any), or number. Supports single or multiple assignees.

```yaml wrap
safe-outputs:
  assign-to-user:
    allowed: [user1, user2]    # restrict to specific users
    max: 3                     # max assignments (default: 1)
    target: "*"                # "triggering" (default), "*", or number
    target-repo: "owner/repo"  # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    unassign-first: true       # unassign all current assignees before assigning (default: false)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

### Unassign from User (`unassign-from-user:`)

Removes user assignments from issues or pull requests. Restrict with `allowed` list to control which users can be unassigned. Target: `"triggering"` (issue/PR event), `"*"` (any), or number.

```yaml wrap
safe-outputs:
  unassign-from-user:
    allowed: [user1, user2]    # restrict to specific users
    max: 1                     # max unassignments (default: 1)
    target: "*"                # "triggering" (default), "*", or number
    target-repo: "owner/repo"  # cross-repository
    allowed-repos: ["org/repo1", "org/repo2"]  # additional allowed repositories
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

## Cross-Repository Operations

Most safe outputs support cross-repository operations:

- **`target-repo`**: Set a fixed target repository (`owner/repo` format), or use `"*"` as a wildcard to let the agent supply any repository at runtime.
- **`allowed-repos`**: Allow the agent to dynamically choose from an allowlist of repositories (supports glob patterns, e.g. `org/*`).

Using `target-repo: "*"` enables fully dynamic routing — the agent provides the `repo` field in each tool call. Note that `create-pull-request-review-comment`, `reply-to-pull-request-review-comment`, `submit-pull-request-review`, `create-agent-session`, and `manage-project-items` do not support the wildcard; use an explicit repository or `allowed-repos` for those types.

See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for comprehensive documentation.

## Global Configuration Options

### Workflow Call Outputs (`workflow_call`)

When a workflow uses `on: workflow_call` (or includes `workflow_call` in its triggers) and configures safe outputs, the compiler automatically injects `on.workflow_call.outputs` exposing the results of each configured safe output type. This makes gh-aw workflows composable building blocks in larger automation pipelines.

The following named outputs are exposed for each configured safe output type:

| Safe Output Type | Output Names |
|---|---|
| `create-issue` | `created_issue_number`, `created_issue_url` |
| `create-pull-request` | `created_pr_number`, `created_pr_url` |
| `add-comment` | `comment_id`, `comment_url` |
| `push-to-pull-request-branch` | `push_commit_sha`, `push_commit_url` |

These outputs are automatically available to calling workflows without any additional frontmatter configuration. User-declared `outputs` in the frontmatter are preserved and take precedence over the auto-injected values.

**Example — calling workflow using safe-output results:**

```yaml wrap
jobs:
  run-agent:
    uses: ./.github/workflows/my-agent.lock.yml
  follow-up:
    needs: run-agent
    steps:
      - run: echo "Created issue ${{ needs.run-agent.outputs.created_issue_number }}"
```

### Failure Issue Reporting (`report-failure-as-issue:`)

Controls whether workflow failures are reported as GitHub issues (default: `true`). Set to `false` to suppress automatic failure issue creation for a specific workflow.

```yaml wrap
safe-outputs:
  report-failure-as-issue: false
  create-issue:
```

This mirrors the `noop.report-as-issue` pattern. Use this to silence noisy failure reports for workflows where failures are expected or handled externally.

### Failure Issue Repository (`failure-issue-repo:`)

Redirects failure tracking issues to a different repository. Useful when the current repository has issues disabled (e.g. `github/docs-internal`).

```yaml wrap
safe-outputs:
  failure-issue-repo: github/docs-engineering
  create-issue:
```

The value must be in `owner/repo` format. The `GITHUB_TOKEN` used must have permission to create issues in the target repository. When not set, failure issues are created in the current repository.

### Group Reports (`group-reports:`)

Controls whether failed workflow runs are grouped under a parent "[aw] Failed runs" issue. This is opt-in and defaults to `false`.

```yaml wrap
safe-outputs:
  create-issue:
  group-reports: true   # Enable parent issue grouping for failed runs (default: false)
```

When enabled, individual failed run reports are linked as sub-issues under a shared parent issue, making it easier to track recurring failures across workflow runs. When disabled (the default), each failure is reported independently.

### Custom GitHub Token (`github-token:`)

Override for all safe outputs, or per safe output:

```yaml wrap
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}  # global
  create-issue:
  create-pull-request:
    github-token: ${{ secrets.PR_PAT }}    # per-output
```

### Using a GitHub App for Authentication (`github-app:`)

Use GitHub App tokens for enhanced security: on-demand token minting, automatic revocation, fine-grained permissions, and better attribution.

See [Using a GitHub App for Authentication](/gh-aw/reference/auth/#using-a-github-app-for-authentication).

### Environment Protection (`environment:`)

Specifies the deployment environment for all compiler-generated safe-output jobs (`safe_outputs`, `conclusion`, `pre_activation`, custom safe-jobs). This makes environment-scoped secrets accessible in those jobs — for example, GitHub App credentials stored as environment secrets.

The top-level `environment:` field is automatically propagated to all safe-output jobs. Use `safe-outputs.environment:` to override this independently:

```yaml wrap
safe-outputs:
  environment: dev   # overrides top-level environment for safe-output jobs only
  github-app:
    app-id: ${{ secrets.WORKFLOW_APP_ID }}
    private-key: ${{ secrets.WORKFLOW_APP_PRIVATE_KEY }}
```

Accepts a plain string or an object with `name` and optional `url`, consistent with the top-level `environment:` syntax.

### Text Sanitization (`allowed-domains:`, `allowed-github-references:`)

The text output by AI agents is automatically sanitized to prevent injection of malicious content and ensure safe rendering on GitHub. The auto-sanitization applied is: XML escaped, HTTPS only, domain allowlist (GitHub by default), 0.5MB/65k line limits, control char stripping.

You can configure sanitization options:

```yaml wrap
safe-outputs:
  allowed-domains: [api.github.com]  # GitHub domains always included
  allowed-github-references: []      # Escape all GitHub references
```

**Domain Filtering** (`allowed-domains`): Controls which domains are allowed in URLs. URLs from other domains are replaced with `(redacted)`. Accepts specific domain strings or [ecosystem identifiers](/gh-aw/reference/network/#ecosystem-identifiers):

```yaml wrap
safe-outputs:
  # Allow specific domains
  allowed-domains: [api.example.com, "*.storage.example.com"]

  # Use ecosystem identifiers
  allowed-domains: [default-safe-outputs]  # defaults + dev-tools + github + local

  # Mix identifiers and custom domains
  allowed-domains: [default-safe-outputs, api.example.com]
```

The `default-safe-outputs` compound ecosystem is the recommended baseline — it covers infrastructure certificates (`defaults`), GitHub domains (`github`), popular developer tooling (`dev-tools`), and loopback addresses (`local`).

**Reference Escaping** (`allowed-github-references`): Controls which GitHub repository references (`#123`, `owner/repo#456`) are allowed in workflow output. When configured, references to unlisted repositories are escaped with backticks to prevent GitHub from creating timeline items. This is particularly useful for [SideRepoOps](/gh-aw/patterns/side-repo-ops/) workflows to prevent automation from cluttering your main repository's timeline.

- `[]` - Escape all references (prevents all timeline items)
- `["repo"]` - Allow only the target repository's references
- `["repo", "owner/other-repo"]` - Allow specific repositories
- Not specified (default) - All references allowed

### Bot Mention Limit (`max-bot-mentions:`)

Agent output is automatically scanned for bot trigger phrases (e.g., `@copilot`, `@github-actions`) to prevent accidental automation triggering. By default, the first 10 occurrences are left unchanged and any excess are escaped with backticks. Entries already wrapped in backticks are skipped.

Use `max-bot-mentions` to adjust this threshold:

```yaml wrap
safe-outputs:
  max-bot-mentions: 3   # Allow 3 unescaped bot mentions per output
  create-issue:
```

Accepts a literal integer or a GitHub Actions expression string (e.g., `${{ inputs.max-mentions }}`). Set to `0` to escape all bot trigger phrases. Default: 10.

### Templatable Fields

`max`, `expires`, and `max-bot-mentions` accept GitHub Actions expression strings in addition to literal integers, allowing workflow inputs or repository variables to control limits at runtime:

```yaml wrap
safe-outputs:
  max-bot-mentions: ${{ inputs.max-mentions }}
  create-issue:
    max: ${{ inputs.max-issues }}
    expires: ${{ inputs.expires-days }}
  create-pull-request:
    max: ${{ inputs.max-prs }}
    draft: ${{ inputs.create-draft }}
```

Most boolean configuration fields also accept expression strings. Fields that influence permission computation (such as `create-pull-request.fallback-as-issue`) remain literal booleans.

### Maximum Patch Size (`max-patch-size:`)

Limits git patch size for PR operations (1-10,240 KB, default: 1024 KB):

```yaml wrap
safe-outputs:
  max-patch-size: 512  # max patch size in KB
  create-pull-request:
```

### Custom Runner Image

Specify a custom runner for safe output jobs (default: `ubuntu-slim`):

```aw
---
safe-outputs:
  runs-on: ubuntu-22.04
  create-issue: {}
---
```

`safe-outputs.runs-on` overrides `runs-on-slim:` for safe-output jobs specifically. To override the runner for all framework jobs at once, use the top-level [`runs-on-slim:`](/gh-aw/guides/self-hosted-runners/#configuring-the-framework-job-runner) field instead.

### Safe Outputs Job Concurrency (`concurrency-group:`)

Control concurrency for the compiled `safe_outputs` job. When set, the job uses this group with `cancel-in-progress: false` (queuing semantics — in-progress runs are never cancelled).

```yaml wrap
safe-outputs:
  concurrency-group: "safe-outputs-${{ github.repository }}"
  create-issue:
```

Supports GitHub Actions expressions. Use this to prevent concurrent safe output jobs from racing on shared resources (e.g., creating duplicate issues or conflicting PRs).

### Custom Messages (`messages:`)

Customize notifications using template variables and Markdown. Import from shared workflows (local overrides imported).

```yaml wrap
safe-outputs:
  messages:
    footer: "> 🤖 Generated by [{workflow_name}]({run_url})"
    append-only-comments: true
    run-started: "🚀 Processing {event_type}..."
    run-success: "✅ Completed successfully"
    run-failure: "❌ Encountered {status}"
  create-issue:
```

**Templates**: `footer`, `footer-install`, `staged-title`, `staged-description`, `run-started`, `run-success`, `run-failure`

**Options**: `append-only-comments` (default: `false`)

**Variables**: `{workflow_name}`, `{run_url}`, `{agentic_workflow_url}`, `{triggering_number}`, `{workflow_source}`, `{workflow_source_url}`, `{event_type}`, `{status}`, `{operation}`, `{effective_tokens}`, `{effective_tokens_formatted}`, `{effective_tokens_suffix}`

`{effective_tokens}` contains the raw total effective token count for the run (e.g. `1200`), and `{effective_tokens_formatted}` is the compact human-readable form (e.g. `1.2K`, `3M`). Both are only present when the effective token count is greater than zero. `{effective_tokens_suffix}` is a pre-formatted, always-safe suffix string (e.g. `" · ● 1.2K"` or `""`) that can be inserted directly into footer templates alongside `{history_link}`. The default footer automatically includes the formatted value — use these variables in custom footer templates to include token usage in your own format. See [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/) for details on how effective tokens are computed.

## Staged Mode

Staged mode lets you preview what safe outputs a workflow would create without actually creating anything. Every write operation is skipped; instead, a 🎭-labelled preview appears in the GitHub Actions step summary.

Enable it globally by adding `staged: true` to the `safe-outputs:` block:

```yaml wrap
safe-outputs:
  staged: true
  create-issue:
    title-prefix: "[ai] "
    labels: [automation]
```

You can also scope staged mode to a specific output type by adding `staged: true` directly to that type while leaving the global setting at `false`:

```yaml wrap
safe-outputs:
  create-pull-request:
    staged: true   # preview only
  add-comment:     # executes normally
```

To disable staged mode and start creating real resources, remove the `staged: true` setting or set it to `false`.

See [Staged Mode](/gh-aw/reference/staged-mode/) for the full guide, including the preview message format, per-type support table, custom message templates, and how to implement staged mode in [custom safe output jobs](/gh-aw/reference/custom-safe-outputs/#staged-mode-support).

## Replaying Safe Outputs

If the `safe_outputs` job fails or is skipped — for example, due to a transient API error, threat detection blocking the output, or a cancelled run — you can replay safe outputs from a previous run using the **Agentic Maintenance** workflow.

> [!NOTE]
> The Agentic Maintenance workflow (`agentics-maintenance.yml`) is generated automatically when any workflow uses the `expires` field in `create-issue`, `create-discussion`, or `create-pull-request` safe outputs.

To replay safe outputs:

1. Go to your repository's **Actions** tab.
2. Select the **Agentic Maintenance** workflow.
3. Click **Run workflow**.
4. Set **Optional maintenance operation** to `safe_outputs`.
5. Set **Run URL or run ID** to the URL or run ID of the previous workflow run:
   - Full URL: `https://github.com/OWNER/REPO/actions/runs/12345`
   - Run ID only: `12345`
6. Click **Run workflow**.

The `apply_safe_outputs` job downloads the `agent_output.json` artifact from the specified run and applies all safe outputs as if the original run had completed successfully. The job requires admin or maintainer permissions.

> [!TIP]
> Find the run URL by opening the failed or cancelled run in the **Actions** tab — the URL in your browser's address bar is the run URL.

## Related Documentation

- [Staged Mode](/gh-aw/reference/staged-mode/) - Preview safe output operations without making changes
- [Threat Detection Guide](/gh-aw/reference/threat-detection/) - Complete threat detection documentation and examples
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
