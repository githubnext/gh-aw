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

- [**Create PR**](#pull-request-creation-create-pull-request) (`create-pull-request`) - Create pull requests with code changes (default max: 1, configurable)
- [**Update PR**](#pull-request-updates-update-pull-request) (`update-pull-request`) - Update PR title or body (max: 1)
- [**Close PR**](#close-pull-request-close-pull-request) (`close-pull-request`) - Close pull requests without merging (max: 10)
- [**PR Review Comments**](#pr-review-comments-create-pull-request-review-comment) (`create-pull-request-review-comment`) - Create review comments on code lines (max: 10)
- [**Reply to PR Review Comment**](#reply-to-pr-review-comment-reply-to-pull-request-review-comment) (`reply-to-pull-request-review-comment`) - Reply to existing review comments (max: 10)
- [**Resolve PR Review Thread**](#resolve-pr-review-thread-resolve-pull-request-review-thread) (`resolve-pull-request-review-thread`) - Resolve review threads after addressing feedback (max: 10)
- [**Push to PR Branch**](#push-to-pr-branch-push-to-pull-request-branch) (`push-to-pull-request-branch`) - Push changes to PR branch (default max: 1, configurable, same-repo only)

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
- [**Upload Assets**](#asset-uploads-upload-asset) (`upload-asset`) - Upload files to orphaned git branch (max: 10, same-repo only)

### Security & Agent Tasks

- [**Dispatch Workflow**](#workflow-dispatch-dispatch-workflow) (`dispatch-workflow`) - Trigger other workflows with inputs (max: 3, same-repo only)
- [**Code Scanning Alerts**](#code-scanning-alerts-create-code-scanning-alert) (`create-code-scanning-alert`) - Generate SARIF security advisories (max: unlimited, same-repo only)
- [**Autofix Code Scanning Alerts**](#autofix-code-scanning-alerts-autofix-code-scanning-alert) (`autofix-code-scanning-alert`) - Create automated fixes for code scanning alerts (max: 10, same-repo only)
- [**Create Agent Session**](#agent-session-creation-create-agent-session) (`create-agent-session`) - Create Copilot coding agent sessions (max: 1)

### System Types (Auto-Enabled)

- [**No-Op**](#no-op-logging-noop) (`noop`) - Log completion message for transparency (max: 1, same-repo only)
- [**Missing Tool**](#missing-tool-reporting-missing-tool) (`missing-tool`) - Report missing tools (max: unlimited, same-repo only)
- [**Missing Data**](#missing-data-reporting-missing-data) (`missing-data`) - Report missing data required to achieve goals (max: unlimited, same-repo only)

### Custom Safe Output Jobs (`jobs:`)

Create custom post-processing jobs registered as Model Context Protocol (MCP) tools. Support standard GitHub Actions properties and auto-access agent output via `$GH_AW_AGENT_OUTPUT`. See [Custom Safe Output Jobs](/gh-aw/reference/custom-safe-outputs/).

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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

> [!TIP]
> Use `footer: false` to omit the AI-generated footer while preserving workflow-id markers for searchability. See [Footer Control](/gh-aw/reference/footers/) for details.

#### Auto-Expiration

The `expires` field auto-closes issues after a time period. Supports integers (days), relative formats (`2h`, `7d`, `2w`, `1m`, `1y`), or `false` to disable expiration. Generates `agentics-maintenance.yml` workflow that runs at the minimum required frequency based on the shortest expiration time across all workflows:

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

In this example, if the workflow creates 5 issues, all will be automatically grouped under a parent issue, making it easy to track related work items together.

#### Temporary IDs for Issue References

Use temporary IDs (`aw_` + 3-8 alphanumeric chars) to reference parent issues before creation. References like `#aw_abc123` in bodies are replaced with actual numbers. The `parent` field creates sub-issue relationships.

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

#### Searching for Workflow-Created Items

All items created by workflows (issues, pull requests, discussions, and comments) include a hidden **workflow-id marker** in their body:

```html
<!-- gh-aw-workflow-id: WORKFLOW_NAME -->
```

You can use this marker to find all items created by a specific workflow on GitHub.com.

**Search Examples:**

Find all open issues created by the `daily-team-status` workflow:

```
repo:owner/repo is:issue is:open "gh-aw-workflow-id: daily-team-status" in:body
```

Find all pull requests created by the `security-audit` workflow:

```
repo:owner/repo is:pr "gh-aw-workflow-id: security-audit" in:body
```

Find all items (issues, PRs, discussions) from any workflow in your organization:

```
org:your-org "gh-aw-workflow-id:" in:body
```

Find comments from a specific workflow:

```
repo:owner/repo "gh-aw-workflow-id: bot-responder" in:comments
```

> [!TIP]
> **Search Tips for Workflow Markers**
>
> - Use quotes around the marker text to search for the exact phrase
> - Add `in:body` to search issue/PR descriptions, or `in:comments` for comments
> - Combine with other filters like `is:open`, `is:closed`, `created:>2024-01-01`
> - The workflow name in the marker is the workflow filename without the `.md` extension
> - Use GitHub's advanced search to refine results: [Advanced search documentation](https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests)

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

Posts comments on issues, PRs, or discussions. Defaults to triggering item; use `target: "*"` for any, or number for specific items. When combined with `create-issue`, `create-discussion`, or `create-pull-request`, includes "Related Items" section.

```yaml wrap
safe-outputs:
  add-comment:
    max: 3                       # max comments (default: 1)
    target: "*"                  # "triggering" (default), "*", or number
    discussion: true             # target discussions
    target-repo: "owner/repo"    # cross-repository
    hide-older-comments: true    # hide previous comments from same workflow
    allowed-reasons: [outdated]  # restrict hiding reasons (optional)
```

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

Use `reviewers: [copilot]` to assign the Copilot PR reviewer bot. This uses the same token resolution as Copilot agent assignment: `github-token:`, [`GH_AW_AGENT_TOKEN`](/gh-aw/reference/auth/#gh_aw_agent_token) (or [`GH_AW_GITHUB_TOKEN`](/gh-aw/reference/auth/#gh_aw_github_token), falling back to `GITHUB_TOKEN`).

### Assign Milestone (`assign-milestone:`)

Assigns issues to milestones. Specify `allowed` to restrict to specific milestone titles.

```yaml wrap
safe-outputs:
  assign-milestone:
    allowed: [v1.0, v2.0]    # restrict to specific milestone titles
    max: 1                   # max assignments (default: 1)
    target-repo: "owner/repo" # cross-repository
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

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

**Operation Types**:

- `append` (default): Adds content to the end with separator and attribution
- `prepend`: Adds content to the start with separator and attribution
- `replace`: Completely replaces existing body with new content and attribution

Title updates always replace the existing title. Disable fields by setting to `false`.

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

Creates new GitHub Projects V2 boards. Requires PAT or GitHub App token ([`GH_AW_PROJECT_GITHUB_TOKEN`](/gh-aw/reference/auth/#gh_aw_project_github_token))-default `GITHUB_TOKEN` lacks Projects v2 access. Supports optional view configuration to create custom project views at creation time.

```yaml wrap
safe-outputs:
  create-project:
    max: 1                              # max operations (default: 1)
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
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

Manages GitHub Projects boards. Requires PAT or GitHub App token ([`GH_AW_PROJECT_GITHUB_TOKEN`](/gh-aw/reference/auth/#gh_aw_project_github_token))-default `GITHUB_TOKEN` lacks Projects v2 access. Update-only by default; set `create_if_missing: true` to create boards (requires appropriate token permissions).

```yaml wrap
safe-outputs:
  update-project:
    project: "https://github.com/orgs/myorg/projects/42"  # required: target project URL
    max: 20                         # max operations (default: 10)
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    views:                          # optional: auto-create views
      - name: "Sprint Board"
        layout: board
        filter: "is:issue is:open"
      - name: "Task Tracker"
        layout: table
      - name: "Roadmap"
        layout: roadmap
```

**Configuration options:**

- `project` (required in configuration): Default project URL shown in examples. Note: Agent output messages **must** explicitly include the `project` field - the configured value is for documentation purposes only.
- `max`: Maximum number of operations per run (default: 10).
- `github-token`: Custom token with Projects permissions (required for Projects v2 access).
- `views`: Optional array of project views to create automatically.
- Exposes outputs: `project-id`, `project-number`, `project-url`, `item-id`.

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
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
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

**Layout types:**

- **`table`** - List view with customizable columns for detailed tracking
- **`board`** - Kanban-style cards grouped by status or custom field
- **`roadmap`** - Timeline visualization with date-based swimlanes

**Filter syntax examples:**

- `is:issue is:open` - Open issues only
- `is:pr` - Pull requests only  
- `is:issue is:pr` - Both issues and PRs
- `label:bug` - Items with bug label
- `assignee:@me` - Items assigned to viewer

Views are created automatically during workflow execution. The workflow must include at least one `update_project` operation to provide the target project URL.

### Project Status Updates (`create-project-status-update:`)

Creates status updates on GitHub Projects boards to communicate progress, findings, and trends. Status updates appear in the project's Updates tab and provide a historical record of execution. Requires PAT or GitHub App token ([`GH_AW_PROJECT_GITHUB_TOKEN`](/gh-aw/reference/auth/#gh_aw_project_github_token))-default `GITHUB_TOKEN` lacks Projects v2 access.

```yaml wrap
safe-outputs:
  create-project-status-update:
    project: "https://github.com/orgs/myorg/projects/73"  # required: target project URL
    max: 1                          # max updates per run (default: 1)
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

**Configuration options:**

- `project` (required in configuration): Default project URL shown in examples. Note: Agent output messages **must** explicitly include the `project` field - the configured value is for documentation purposes only.
- `max`: Maximum number of status updates per run (default: 1).
- `github-token`: Custom token with Projects permissions (required for Projects v2 access).
- Often used by scheduled workflows and orchestrator workflows to post run summaries.

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

#### Example Usage

```yaml
create-project-status-update:
  project: "https://github.com/orgs/myorg/projects/73"
  status: "ON_TRACK"
  start_date: "2026-01-06"
  target_date: "2026-01-31"
  body: |
  ## Run Summary

    **Discovered:** 25 items (15 issues, 10 PRs)
    **Processed:** 10 items added to project, 5 updated
    **Completion:** 60% (30/50 total tasks)

    ### Key Findings
    - Documentation coverage improved to 88%
    - 3 critical accessibility issues identified
    - Worker velocity: 1.2 items/day

    ### Trends
    - Velocity stable at 8-10 items/week
    - Blocked items decreased from 5 to 2
    - On track for end-of-month completion

    ### Next Steps
    - Continue processing remaining 15 items
    - Address 2 blocked items in next run
    - Target 95% documentation coverage by end of month
```

#### Status Indicators

- **`ON_TRACK`**: Progressing as planned, meeting expected targets
- **`AT_RISK`**: Potential issues identified (blocked items, slower velocity, dependencies)
- **`OFF_TRACK`**: Behind schedule, requires intervention or re-planning
- **`COMPLETE`**: Objectives met, no further work needed
- **`INACTIVE`**: Paused or not actively running

Exposes outputs: `status-update-id`, `project-id`, `status`.

### Pull Request Creation (`create-pull-request:`)

Creates PRs with code changes. By default, falls back to creating an issue if PR creation fails (e.g., org settings block it). Set `fallback-as-issue: false` to disable this fallback and avoid requiring `issues: write` permission. `expires` field (same-repo only) auto-closes after period: integers (days) or `2h`, `7d`, `2w`, `1m`, `1y` (hours < 24 treated as 1 day).

Multiple PRs per run are supported by setting `max` higher than 1. Each PR is created from its own branch with an independent patch, so concurrent calls do not conflict.

```yaml wrap
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "         # prefix for titles
    labels: [automation]          # labels to attach
    reviewers: [user1, copilot]   # reviewers (use 'copilot' for bot)
    draft: true                   # create as draft (default: true)
    max: 3                        # max PRs per run (default: 1)
    expires: 14                   # auto-close after 14 days (same-repo only)
    if-no-changes: "warn"         # "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # cross-repository
    base-branch: "vnext"          # target branch for PR (default: github.base_ref || github.ref_name)
    fallback-as-issue: false      # disable issue fallback (default: true)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

The `base-branch` field specifies which branch the pull request should target. This is particularly useful for cross-repository PRs where you need to target non-default branches (e.g., `vnext`, `release/v1.0`, `staging`). When not specified, defaults to `github.base_ref` (the PR's target branch) with a fallback to `github.ref_name` (the workflow's branch) for push events.

**Example use case:** A workflow in `org/engineering` that creates PRs in `org/docs` targeting the `vnext` branch for feature documentation:

```yaml wrap
safe-outputs:
  create-pull-request:
    target-repo: "org/docs"
    base-branch: "vnext"
    draft: true
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

> [!NOTE]
> PR creation may fail if "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings. By default (`fallback-as-issue: true`), fallback creates an issue with branch link and requires `issues: write` permission. Set `fallback-as-issue: false` to disable fallback and only require `contents: write` + `pull-requests: write`.

When `create-pull-request` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

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
    footer: "if-body"         # footer control: "always", "none", or "if-body"
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

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

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
    max: 10
  submit-pull-request-review:
    max: 1            # max reviews to submit (default: 1)
    target: "triggering"  # or "*", or e.g. ${{ github.event.inputs.pr_number }} when not in pull_request trigger
    footer: false     # omit AI-generated footer from review body (default: true)
```

### Resolve PR Review Thread (`resolve-pull-request-review-thread:`)

Resolves review threads on pull requests. Allows AI agents to mark review conversations as resolved after addressing the feedback. Uses the GitHub GraphQL API with the `resolveReviewThread` mutation.

Resolution is scoped to the triggering PR only — the handler validates that each thread belongs to the triggering pull request before resolving it.

```yaml wrap
safe-outputs:
  resolve-pull-request-review-thread:
    max: 10           # max threads to resolve (default: 10)
```

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

Pushes changes to a PR's branch. Validates via `title-prefix` and `labels` to ensure only approved PRs receive changes. Multiple pushes per run are supported by setting `max` higher than 1.

```yaml wrap
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                 # "triggering" (default), "*", or number
    title-prefix: "[bot] "      # require title prefix
    labels: [automated]         # require all labels
    max: 3                      # max pushes per run (default: 1)
    if-no-changes: "warn"       # "warn" (default), "error", or "ignore"
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

When `push-to-pull-request-branch` is configured, git commands (`checkout`, `branch`, `switch`, `add`, `rm`, `commit`, `merge`) are automatically enabled.

#### Fail-Fast on Code Push Failure

If `push-to-pull-request-branch` (or `create-pull-request`) fails, the safe-output pipeline cancels all remaining non-code-push outputs. Each cancelled output is marked with an explicit reason such as "Cancelled: code push operation failed". The failure details appear in the agent failure issue or comment generated by the conclusion job.

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

Uploads files (screenshots, charts, reports) to orphaned git branch with predictable URLs: `https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{filename}`. Agent registers files via `upload_asset` tool; separate job with `contents: write` commits them.

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

When `create-issue: true`, the agent creates or updates GitHub issues documenting missing data with:

- Detailed explanation of what data is needed and why
- Context about how the data would be used
- Possible alternatives if the data cannot be provided
- Encouragement message praising the agent's truthfulness

This rewards honest AI behavior and helps teams improve data accessibility for future agent runs.

### Discussion Creation (`create-discussion:`)

Creates discussions with optional `category` (slug, name, or ID; defaults to first available). `expires` field auto-closes after period (integers, `2h`, `7d`, `2w`, `1m`, `1y`, or `false` to disable; hours < 24 treated as 1 day) as "OUTDATED" with comment. Generates maintenance workflow with dynamic frequency based on shortest expiration time (see Auto-Expiration section above).

**Category Naming Standard**: Use lowercase, plural category names (e.g., `audits`, `general`, `reports`) for consistency and better searchability. GitHub Discussion category IDs (starting with `DIC_`) are also supported.

> [!WARNING]
> Only announcement-capable category succeeds; all non-announcement categories fail with integration-forbidden.

```yaml wrap
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "        # prefix for titles
    category: "announcements"    # category slug, name, or ID (use lowercase, prefer announcement-capable)
    expires: 3                   # auto-close after 3 days (or false to disable)
    max: 3                       # max discussions (default: 1)
    target-repo: "owner/repo"    # cross-repository
    fallback-to-issue: true      # fallback to issue creation on permission errors (default: true)
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

#### Fallback to Issue Creation

The `fallback-to-issue` field (default: `true`) automatically falls back to creating an issue when discussion creation fails due to permissions errors. This is useful in repositories where discussions are not enabled or where the workflow lacks the necessary permissions to create discussions.

When fallback is triggered:

- An issue is created instead of a discussion
- A note is added to the issue body indicating it was intended to be a discussion
- The issue includes all the same content as the intended discussion

To disable fallback behavior and fail if discussions cannot be created:

```yaml wrap
safe-outputs:
  create-discussion:
    fallback-to-issue: false
```

Common scenarios where fallback is useful:

- Repositories with discussions disabled
- Insufficient permissions (requires `discussions: write`)
- Organization policies restricting discussions
- Testing workflows across different repository configurations

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

**Shorthand Syntax:**

```yaml wrap
safe-outputs:
  dispatch-workflow: [worker-workflow, scanner-workflow]
```

#### Configuration

- **`workflows`** (required) - List of workflow names (without `.md` extension) that the agent is allowed to dispatch. Each workflow must exist in the same repository and support the `workflow_dispatch` trigger.
- **`max`** (optional) - Maximum number of workflow dispatches allowed (default: 1, maximum: 50). This prevents excessive workflow triggering.

#### Validation Rules

At compile time, the compiler validates:

1. **Workflow existence** - Each workflow in the `workflows` list must exist as either:
   - A markdown workflow file (`.md`)
   - A compiled lock file (`.lock.yml`)  
   - A standard GitHub Actions workflow (`.yml`)

2. **workflow_dispatch trigger** - Each workflow must include `workflow_dispatch` in its `on:` trigger section:

   ```yaml
   on: [push, workflow_dispatch]  # or
   on:
     push:
     workflow_dispatch:
       inputs:
         tracker_id:
           description: "Tracker identifier"
           required: true
   ```

3. **No self-reference** - A workflow cannot dispatch itself to prevent infinite loops.

4. **File resolution** - The compiler resolves the correct file extension (`.lock.yml` or `.yml`) at compile time and embeds it in the safe output configuration, ensuring the runtime handler dispatches the correct workflow file.

#### Defining Workflow Inputs

To enable the agent to provide inputs when dispatching workflows, define `workflow_dispatch` inputs in the target workflow:

**Target Workflow Example (`deploy-app.md`):**

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
      version:
        description: "Version to deploy"
        required: true
        type: string
      dry_run:
        description: "Perform dry run without actual deployment"
        required: false
        type: boolean
        default: false
---

# Deploy Application Workflow

Deploys the application to the specified environment...
```

#### Rate Limiting

To respect GitHub API rate limits, the handler automatically enforces a 5-second delay between consecutive workflow dispatches. The first dispatch has no delay.

**Security Considerations**

- **Same-repository only** - Cannot dispatch workflows in other repositories. This prevents cross-repository workflow triggering which could be a security risk.
- **Allowlist enforcement** - Only workflows explicitly listed in the `workflows` configuration can be dispatched. Requests for unlisted workflows are rejected.
- **Compile-time validation** - Workflows are validated at compile time to catch configuration errors early.

### Agent Session Creation (`create-agent-session:`)

Creates Copilot coding agent sessions. Requires [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) or [`GH_AW_GITHUB_TOKEN`](/gh-aw/reference/auth/#gh_aw_github_token) PAT-default `GITHUB_TOKEN` lacks permissions.

### Assign to Agent (`assign-to-agent:`)

Programmatically assigns GitHub Copilot coding agent to **existing** issues or pull requests through workflow automation. This safe output automates the [standard GitHub workflow for assigning issues to Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr#assigning-an-issue-to-copilot). Requires fine-grained PAT with actions, contents, issues, pull requests write access stored as [`GH_AW_AGENT_TOKEN`](/gh-aw/reference/auth/#gh_aw_agent_token), or GitHub App token. Supported agents: `copilot` (`copilot-swe-agent`).

Auto-resolves target from workflow context (issue/PR events) when `issue_number` or `pull_number` not explicitly provided. Restrict with `allowed` list. Target: `"triggering"` (default), `"*"` (any), or number.

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

**Target Issue or Pull Request:**

- `target: "triggering"` - Auto-resolves from `github.event.issue.number` or `github.event.pull_request.number`
- `target: "*"` - Requires explicit `issue_number` or `pull_number` in agent output
- `target: "123"` - Always uses issue/PR #123

**Cross-Repository PR Creation:**

The `pull-request-repo` parameter allows you to create pull requests in a different repository than where the issue lives. This is useful when:

- Issues are tracked in a central repository but code lives in separate repositories
- You want to separate issue tracking from code repositories

When `pull-request-repo` is configured, Copilot will create the pull request in the specified repository instead of the issue's repository. The issue repository is determined by `target-repo` or defaults to the workflow's repository.

The repository specified by `pull-request-repo` is automatically allowed - you don't need to list it in `allowed-pull-request-repos`. Use `allowed-pull-request-repos` to specify additional repositories where PRs can be created.

Use `base-branch` to specify which branch in the target repository the pull request should target. When omitted, the target repository's actual default branch is used automatically. Only relevant when `pull-request-repo` is configured.

**Assignee Filtering:**
When `allowed` list is configured, existing agent assignees not in the list are removed while regular user assignees are preserved.

Use `assign-to-agent` when you need to programmatically assign agents to **existing** issues or PRs through workflow automation. If you're creating new issues and want to assign an agent immediately, use `assignees: copilot` in your [`create-issue`](#issue-creation-create-issue) configuration instead.

### Assign to User (`assign-to-user:`)

Assigns users to issues. Restrict with `allowed` list. Target: `"triggering"` (issue event), `"*"` (any), or number. Supports single or multiple assignees.

```yaml wrap
safe-outputs:
  assign-to-user:
    allowed: [user1, user2]    # restrict to specific users
    max: 3                     # max assignments (default: 1)
    target: "*"                # "triggering" (default), "*", or number
    target-repo: "owner/repo"  # cross-repository
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
    github-token: ${{ secrets.SOME_CUSTOM_TOKEN }} # optional custom token for permissions
```

## Cross-Repository Operations

Most safe outputs support `target-repo`. This will generally require an authorization option (`github-token` or `app:`)-to take effect.

```yaml wrap
safe-outputs:
  create-issue:
    target-repo: "org/tracking-repo"
    github-token: ${{ secrets.CROSS_REPO_PAT }}
```

## Global Configuration Options

### Group Reports (`group-reports:`)

Controls whether failed workflow runs are grouped under a parent "[agentics] Failed runs" issue. This is opt-in and defaults to `false`.

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

### GitHub App Token (`app:`)

Use GitHub App tokens for enhanced security: on-demand token minting, automatic revocation, fine-grained permissions, and better attribution.

See [GitHub App for Safe Outputs](/gh-aw/reference/auth/#github-app-for-safe-outputs) for configuration details and security benefits.

### Text Sanitization (`allowed-domains:`, `allowed-github-references:`)

The text output by AI agents is automatically sanitized to prevent injection of malicious content and ensure safe rendering on GitHub. The auto-sanitization applied is: XML escaped, HTTPS only, domain allowlist (GitHub by default), 0.5MB/65k line limits, control char stripping.

You can configure sanitization options:

```yaml wrap
safe-outputs:
  allowed-domains: [api.github.com]  # GitHub domains always included
  allowed-github-references: []      # Escape all GitHub references
```

**Domain Filtering** (`allowed-domains`): Controls which domains are allowed in URLs. URLs from other domains are replaced with `(redacted)`.

**Reference Escaping** (`allowed-github-references`): Controls which GitHub repository references (`#123`, `owner/repo#456`) are allowed in workflow output. When configured, references to unlisted repositories are escaped with backticks to prevent GitHub from creating timeline items. This is particularly useful for [SideRepoOps](/gh-aw/patterns/side-repo-ops/) workflows to prevent automation from cluttering your main repository's timeline.

- `[]` - Escape all references (prevents all timeline items)
- `["repo"]` - Allow only the target repository's references
- `["repo", "owner/other-repo"]` - Allow specific repositories
- Not specified (default) - All references allowed

### Maximum Patch Size (`max-patch-size:`)

Limits git patch size for PR operations (1-10,240 KB, default: 1024 KB):

```yaml wrap
safe-outputs:
  max-patch-size: 512  # max patch size in KB
  create-pull-request:
```

### Custom Runner Image

Specify custom runner for safe output jobs (default: `ubuntu-slim`): `runs-on: ubuntu-22.04`

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

**Variables**: `{workflow_name}`, `{run_url}`, `{triggering_number}`, `{workflow_source}`, `{workflow_source_url}`, `{event_type}`, `{status}`, `{operation}`

## Related Documentation

- [Threat Detection Guide](/gh-aw/reference/threat-detection/) - Complete threat detection documentation and examples
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
