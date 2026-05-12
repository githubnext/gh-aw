---
description: Complete safe-output operations reference for GitHub Agentic Workflows — all output types, options, and global configuration.
---

# Safe Outputs Reference

Safe outputs are the primary mechanism for write operations in agentic workflows. The `safe-outputs:` frontmatter field configures which GitHub API write operations the agent can trigger. Each operation runs in a separate secured job with appropriate permissions — the main agent job never needs write permissions.

> See also: [github-agentic-workflows.md](github-agentic-workflows.md) for the complete workflow format, [syntax.md](syntax.md) for all frontmatter fields.

- `create-issue:` - Safe GitHub issue creation (bugs, features)

  ```yaml
  safe-outputs:
    create-issue:
      title-prefix: "[ai] "           # Optional: prefix for issue titles
      labels: [automation, agentic]    # Optional: labels to attach to issues
      allowed-labels: [bug, task]     # Optional: restrict which labels the agent can set (any label allowed if omitted)
      allowed-fields: [Priority, Iteration]  # Optional: restrict which issue fields the agent may set via the `fields` runtime parameter (omit/empty = any field; ["*"] explicitly allows all)
      assignees: [user1, copilot]     # Optional: assignees (use 'copilot' for bot)
      max: 5                          # Optional: maximum number of issues (default: 1)
      expires: 7                      # Optional: auto-close after 7 days (supports: 2h, 7d, 2w, 1m, 1y, or false)
      group: true                     # Optional: group as sub-issues under a parent issue (default: false)
      group-by-day: true              # Optional: group same-day runs into one issue by posting as comments (default: false)
      close-older-issues: true        # Optional: close previous issues from same workflow (default: false)
      close-older-key: "my-key"       # Optional: explicit deduplication key for close-older matching (uses gh-aw-close-key marker)
      footer: false                   # Optional: omit AI-generated footer while preserving XML markers (default: true)
      target-repo: "owner/repo"       # Optional: cross-repository
      allowed-repos: [owner/other]    # Optional: additional repos agent can target (agent uses `repo` field in output)
  ```

  **Auto-Expiration**: The `expires` field auto-closes issues after a time period. Supports integers (days) or relative formats (2h, 7d, 2w, 1m, 1y). Generates `agentics-maintenance.yml` workflow that runs at minimum required frequency based on shortest expiration time: 1 day or less → every 2 hours, 2 days → every 6 hours, 3-4 days → every 12 hours, 5+ days → daily.
  **Deduplication for Scheduled Workflows**: When a `schedule:` trigger is combined with `create-issue`, use `skip-if-match:` in the `on:` block to prevent opening a duplicate issue on every run. Pair with `expires:` so stale issues are cleaned up automatically:

  ```yaml
  on:
    schedule: daily on weekdays
    skip-if-match: 'is:issue is:open in:title "[my-workflow] "'
  safe-outputs:
    create-issue:
      title-prefix: "[my-workflow] "
      expires: 7   # auto-close after 7 days
  ```

  Without `skip-if-match`, the workflow creates a new issue on every scheduled run even when an identical open issue already exists.

  **Temporary IDs and Sub-Issues:**
  When creating multiple issues, use `temporary_id` (format: `aw_` + 3-8 alphanumeric chars) to reference parent issues before creation. References like `#aw_abc123` in issue bodies are automatically replaced with actual issue numbers. Use the `parent` field to create sub-issue relationships:

  ```json
  {"type": "create_issue", "temporary_id": "aw_abc123", "title": "Parent", "body": "Parent issue"}
  {"type": "create_issue", "parent": "aw_abc123", "title": "Sub-task", "body": "References #aw_abc123"}
  ```

  **Setting Issue Fields on Creation**: Agents can include a `fields` array in the `create_issue` output to set custom field values immediately after creation. Each item is `{"name": <field-display-name>, "value": <string-or-number>}`. Use a number for numeric fields; string for single-select, iteration title, date `YYYY-MM-DD`, or text. Restrict allowed names with `allowed-fields:`.

  ```json
  {"type": "create_issue", "title": "Triage: flaky parser", "body": "...", "fields": [{"name": "Priority", "value": "High"}, {"name": "Story Points", "value": 3}]}
  ```

- `close-issue:` - Close issues with comment (use this to close issues, not update-issue)

  ```yaml
  safe-outputs:
    close-issue:
      target: "triggering"              # Optional: "triggering" (default), "*", or number
      required-labels: [automated]      # Optional: only close with any of these labels
      required-title-prefix: "[bot]"    # Optional: only close matching prefix
      max: 20                           # Optional: max closures (default: 1)
      state-reason: "not_planned"       # Optional: "completed" (default), "not_planned", "duplicate"
      target-repo: "owner/repo"         # Optional: cross-repository
      allowed-repos: [owner/other]      # Optional: additional repos agent can close issues in
  ```

- `create-discussion:` - Safe GitHub discussion creation (status, audits, reports, logs)

  ```yaml
  safe-outputs:
    create-discussion:
      title-prefix: "[ai] "           # Optional: prefix for discussion titles
      category: "General"             # Optional: discussion category name, slug, or ID (defaults to first category if not specified)
      labels: [status]                # Optional: labels to attach (used for matching when close-older-discussions is enabled)
      allowed-labels: [status, audit] # Optional: restrict which labels the agent can set (any label allowed if omitted)
      max: 3                          # Optional: maximum number of discussions (default: 1)
      close-older-discussions: true   # Optional: close older discussions with same prefix/labels (default: false)
      close-older-key: "my-key"       # Optional: explicit deduplication key for close-older matching
      expires: 7                      # Optional: auto-close after 7 days (supports: 2h, 7d, 2w, 1m, 1y, or false)
      fallback-to-issue: true         # Optional: create issue if discussion creation fails (default: true)
      footer: false                   # Optional: omit AI-generated footer while preserving XML markers (default: true)
      target-repo: "owner/repo"       # Optional: cross-repository
      allowed-repos: [owner/other]    # Optional: additional repos agent can target (agent uses `repo` field in output)
  ```

  The `category` field is optional and can be specified by name (e.g., "General"), slug (e.g., "general"), or ID (e.g., "DIC_kwDOGFsHUM4BsUn3"). If not specified, discussions will be created in the first available category. Category resolution tries ID first, then name, then slug.

  Set `close-older-discussions: true` to automatically close older discussions matching the same title prefix or labels. Up to 10 older discussions are closed as "OUTDATED" with a comment linking to the new discussion. Requires `title-prefix` or `labels` to identify matching discussions.

- `close-discussion:` - Close discussions with comment and resolution

  ```yaml
  safe-outputs:
    close-discussion:
      target: "triggering"              # Optional: "triggering" (default), "*", or number
      required-category: "Ideas"        # Optional: only close in category
      required-labels: [resolved]       # Optional: only close with labels
      required-title-prefix: "[ai]"     # Optional: only close matching prefix
      max: 1                            # Optional: max closures (default: 1)
      target-repo: "owner/repo"         # Optional: cross-repository
  ```

  Resolution reasons: `RESOLVED`, `DUPLICATE`, `OUTDATED`, `ANSWERED`.
- `add-comment:` - Safe comment creation on issues/PRs/discussions

  ```yaml
  safe-outputs:
    add-comment:
      max: 3                          # Optional: maximum number of comments (default: 1)
      target: "*"                     # Optional: target for comments (default: "triggering")
      hide-older-comments: true       # Optional: minimize previous comments from same workflow
      allowed-reasons: [outdated]     # Optional: restrict hiding reasons (default: outdated)
      discussions: true               # Optional: set false to exclude discussions:write permission (default: true)
      issues: true                    # Optional: set false to exclude issues:write permission (default: true)
      pull-requests: true             # Optional: set false to exclude pull-requests:write permission (default: true)
      footer: true                    # Optional: when false, omits visible footer but preserves XML markers (default: true)
      target-repo: "owner/repo"       # Optional: cross-repository
      allowed-repos: [owner/other]    # Optional: additional repos agent can target (agent uses `repo` field in output)
  ```

  **Hide Older Comments**: Set `hide-older-comments: true` to minimize previous comments from the same workflow before posting new ones. Useful for status updates. Allowed reasons: `spam`, `abuse`, `off_topic`, `outdated` (default), `resolved`.

  **Discussion Thread Replies**: Agents can include `reply_to_id` in their output to post a threaded reply within a GitHub Discussion (requires `discussions: true`):

  ```json
  {"type": "add_comment", "body": "Thread reply text", "reply_to_id": 12345}
  ```

- `create-pull-request:` - Safe pull request creation with git patches

  ```yaml
  safe-outputs:
    create-pull-request:
      title-prefix: "[ai] "           # Optional: prefix for PR titles
      labels: [automation, ai-agent]  # Optional: labels to attach to PRs
      reviewers: [user1, copilot]     # Optional: reviewers (use 'copilot' for bot)
      team-reviewers: [platform-team] # Optional: team slugs to assign as reviewers
      draft: true                     # Optional: create as draft PR (defaults to true)
      if-no-changes: "warn"           # Optional: "warn" (default), "error", or "ignore"
      allow-empty: false              # Optional: create PR with empty branch, no changes required (default: false)
      expires: 7                      # Optional: auto-close after 7 days (supports: 2h, 7d, 2w, 1m, 1y; min: 2h)
      auto-merge: false               # Optional: enable auto-merge when checks pass (default: false)
      base-branch: "vnext"            # Optional: base branch for PR (defaults to workflow's branch)
      preserve-branch-name: true      # Optional: skip random salt suffix on agent-specified branch names (default: false)
      recreate-ref: false             # Optional: force-recreate existing remote branch when preserve-branch-name is true (default: false)
      allow-workflows: false          # Optional: add workflows:write permission when allowed-files targets .github/workflows/ paths (default: false; requires github-app)
      assignees: [user1]              # Optional: assignees for fallback issues on PR creation failure
      fallback-labels: [needs-review] # Optional: labels for fallback issues (defaults to PR labels)
      fallback-as-issue: false        # Optional: when true (default), creates a fallback issue on PR creation failure; on permission errors, the issue includes a one-click link to create the PR via GitHub's compare URL
      auto-close-issue: false         # Optional: when true (default), adds "Fixes #N" closing keyword when triggered from an issue; set to false to prevent auto-closing the triggering issue on merge. Accepts a boolean or GitHub Actions expression.
      target-repo: "owner/repo"       # Optional: cross-repository
      github-token-for-extra-empty-commit: ${{ secrets.MY_CI_PAT }}  # Optional: PAT or "app" to trigger CI on created PRs
      allowed-files:                  # Optional: exclusive allowlist of glob patterns for eligible files
        - "src/**"
        - "docs/**"
      excluded-files:                 # Optional: glob patterns to strip from the patch entirely
        - "**/*.lock"
      protected-files: blocked        # Optional: "blocked" (default), "fallback-to-issue", or "allowed"
      allowed-base-branches:          # Optional: glob patterns for allowed base branch overrides per run
        - "release/*"
        - "main"
  ```

  **Dynamic Base Branch**: When `allowed-base-branches` is set, the agent can provide a `base` field in its output to override the default base branch for a single run — but only if the value matches one of the configured glob patterns. Without `allowed-base-branches`, only the static `base-branch:` is used. Accepts a literal array or a GitHub Actions expression resolving to a comma-separated list (e.g. `${{ inputs.allowed-base-branches }}`).

  **File Restrictions**: Use `allowed-files` as an **exclusive allowlist** — every file touched must match at least one pattern or the operation is refused. Use `excluded-files` to strip files (e.g. lock files) from the patch before any checks. The `protected-files` field controls handling of sensitive files (package manifests, CI configs, agent instruction files): `blocked` (default, hard-block), `fallback-to-issue` (push branch and create a review issue), or `allowed` (no restriction — use only when the workflow is explicitly designed to manage these files). Object form is also supported: `protected-files: { policy: fallback-to-issue, exclude: [AGENTS.md] }`.

  **Auto-Expiration**: The `expires` field auto-closes PRs after a time period. Supports integers (days) or relative formats (2h, 7d, 2w, 1m, 1y). Minimum duration: 2 hours. Only for same-repo PRs without target-repo. Generates `agentics-maintenance.yml` workflow.

  **Branch Name Preservation**: Set `preserve-branch-name: true` to skip the random salt suffix on agent-specified branch names. Useful when CI enforces branch naming conventions (e.g., Jira keys in uppercase). Invalid characters are still replaced for security; casing is always preserved. Set `recreate-ref: true` alongside this to force-recreate an existing remote branch (e.g., when a previous PR was already merged into the branch).

  **Workflow File Changes**: To modify files under `.github/workflows/`, set `allow-workflows: true`. This adds `workflows: write` to the token used for the PR — a permission that requires `safe-outputs.github-app` to be configured, since `GITHUB_TOKEN` cannot hold this permission.

  **CI Triggering**: By default, PRs created with `GITHUB_TOKEN` do not trigger CI workflow runs. To trigger CI, set `github-token-for-extra-empty-commit` to a PAT with `Contents: Read & Write` permission, or to `"app"` to use the configured GitHub App. Alternatively, set the magic secret `GH_AW_CI_TRIGGER_TOKEN` to a suitable PAT — this is automatically used without requiring explicit configuration in the workflow.

- `create-pull-request-review-comment:` - Safe PR review comment creation on code lines

  ```yaml
  safe-outputs:
    create-pull-request-review-comment:
      max: 3                          # Optional: maximum number of review comments (default: 10)
      side: "RIGHT"                   # Optional: side of diff ("LEFT" or "RIGHT", default: "RIGHT")
      target: "*"                     # Optional: "triggering" (default), "*", or number
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

- `submit-pull-request-review:` - Submit a PR review with status (APPROVE, REQUEST_CHANGES, COMMENT)

  ```yaml
  safe-outputs:
    submit-pull-request-review:
      max: 1                          # Optional: maximum number of reviews to submit (default: 1)
      footer: "if-body"               # Optional: footer control ("always", "none", "if-body", default: "always")
      allowed-events: [COMMENT, REQUEST_CHANGES]  # Optional: restrict allowed review event types; omit to allow all (APPROVE, COMMENT, REQUEST_CHANGES)
  ```

  **Footer Control**: The `footer` field controls when AI-generated footers appear in the PR review body. Values: `"always"` (default), `"none"`, `"if-body"` (only when body is non-empty). Boolean values supported: `true` → `"always"`, `false` → `"none"`. Useful for clean approval reviews — with `"if-body"`, approvals without explanatory text appear without a footer.

- `reply-to-pull-request-review-comment:` - Reply to existing review comments on PRs

  ```yaml
  safe-outputs:
    reply-to-pull-request-review-comment:
      max: 10                         # Optional: maximum number of replies (default: 10)
      target-repo: "owner/repo"       # Optional: cross-repository
      footer: "always"                # Optional: footer control ("always", "none", "if-body", default: "always")
  ```

  **Footer Control**: The `footer` field controls when AI-generated footers appear. Values: `"always"` (default), `"none"`, `"if-body"` (only when body is non-empty). Boolean values supported: `true` → `"always"`, `false` → `"none"`.

- `resolve-pull-request-review-thread:` - Resolve PR review threads after addressing feedback

  ```yaml
  safe-outputs:
    resolve-pull-request-review-thread:
      max: 10                         # Optional: maximum number of threads to resolve (default: 10)
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

  This safe-output type allows agents to programmatically resolve review comment threads after addressing feedback, improving PR review workflows.

- `update-issue:` - Update issue title, body, labels, assignees, or milestone (NOT for closing - use close-issue instead)

  ```yaml
  safe-outputs:
    update-issue:
      status: true                    # Optional: allow updating issue status (open/closed)
      target: "*"                     # Optional: target for updates (default: "triggering")
      title: true                     # Optional: allow updating issue title
      body: true                      # Optional: allow updating issue body
      max: 3                          # Optional: maximum number of issues to update (default: 1)
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

  **Note:** While `update-issue` technically supports changing status between 'open' and 'closed', use `close-issue` instead when you want to close an issue with a closing comment. Use `update-issue` primarily for changing the title, body, labels, assignees, or milestone without closing.
- `update-pull-request:` - Update PR title or body

  ```yaml
  safe-outputs:
    update-pull-request:
      title: true                     # Optional: enable title updates (default: true)
      body: true                      # Optional: enable body updates (default: true)
      operation: "replace"            # Optional: "replace" (default), "append", "prepend"
      update-branch: false            # Optional: update PR branch with latest base before updates (default: false)
      max: 1                          # Optional: max updates (default: 1)
      target: "*"                     # Optional: "triggering" (default), "*", or number
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

  Operation types: `replace` (default), `append`, `prepend`.
- `merge-pull-request:` - Merge pull requests under configured policy gates (experimental)

  ```yaml
  safe-outputs:
    merge-pull-request:
      required-labels: [approved]         # Optional: all listed labels must be present
      allowed-labels: [ready-to-merge]    # Optional: at least one PR label must match
      allowed-branches: ["feature/*"]     # Optional: glob patterns for source branch names
      max: 1                              # Optional: max merges (default: 1)
  ```

  **⚠️ Experimental**: Compilation emits a warning when this feature is used. The merge is blocked unless all configured gates pass.

- `close-pull-request:` - Safe pull request closing with filtering

  ```yaml
  safe-outputs:
    close-pull-request:
      required-labels: [test, automated]  # Optional: only close PRs with these labels
      required-title-prefix: "[bot]"      # Optional: only close PRs with this title prefix
      target: "triggering"                # Optional: "triggering" (default), "*" (any PR), or explicit PR number
      max: 10                             # Optional: maximum number of PRs to close (default: 1)
      target-repo: "owner/repo"           # Optional: cross-repository
      github-token: ${{ secrets.CUSTOM_TOKEN }}  # Optional: custom token
  ```

- `mark-pull-request-as-ready-for-review:` - Mark draft PRs as ready for review

  ```yaml
  safe-outputs:
    mark-pull-request-as-ready-for-review:
      max: 1                              # Optional: max operations (default: 1)
      target: "*"                         # Optional: "triggering" (default), "*", or number
      required-labels: [automated]        # Optional: only mark PRs with these labels
      required-title-prefix: "[bot]"      # Optional: only mark PRs with this prefix
      target-repo: "owner/repo"           # Optional: cross-repository
  ```

- `add-labels:` - Safe label addition to issues or PRs

  ```yaml
  safe-outputs:
    add-labels:
      allowed: [bug, enhancement, documentation]  # Optional: restrict to specific labels
      blocked: ["~*", "*[bot]"]                   # Optional: blocked label patterns (glob; takes precedence over allowed)
      max: 3                                      # Optional: maximum number of labels (default: 3)
      target: "*"                                 # Optional: "triggering" (default), "*" (any issue/PR), or number
      target-repo: "owner/repo"                   # Optional: cross-repository
  ```

- `remove-labels:` - Safe label removal from issues or PRs

  ```yaml
  safe-outputs:
    remove-labels:
      allowed: [automated, stale]  # Optional: restrict to specific labels
      blocked: ["~*", "*[bot]"]    # Optional: blocked label patterns (glob; takes precedence over allowed)
      max: 3                       # Optional: maximum number of operations (default: 3)
      target: "*"                  # Optional: "triggering" (default), "*" (any issue/PR), or number
      target-repo: "owner/repo"    # Optional: cross-repository
  ```

  When `allowed` is omitted, any labels can be removed.
- `add-reviewer:` - Add reviewers to pull requests

  ```yaml
  safe-outputs:
    add-reviewer:
      reviewers: [user1, copilot]     # Optional: restrict to specific reviewers
      team-reviewers: [platform-team] # Optional: allowed team slugs
      max: 3                          # Optional: max reviewers (default: 3)
      target: "*"                     # Optional: "triggering" (default), "*", or number
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

  At least one of `reviewers` or `team-reviewers` must be present in agent output. Use `reviewers: copilot` to assign Copilot PR reviewer bot. Requires PAT as `COPILOT_GITHUB_TOKEN`.
- `assign-milestone:` - Assign issues to milestones

  ```yaml
  safe-outputs:
    assign-milestone:
      allowed: [v1.0, v2.0]           # Optional: restrict to specific milestone titles
      auto-create: true               # Optional: auto-create milestones from the allowed list if missing (default: false)
      max: 1                          # Optional: max assignments (default: 1)
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

- `link-sub-issue:` - Safe sub-issue linking

  ```yaml
  safe-outputs:
    link-sub-issue:
      parent-required-labels: [epic]     # Optional: parent must have these labels
      parent-title-prefix: "[Epic]"      # Optional: parent must match this prefix
      sub-required-labels: [task]        # Optional: sub-issue must have these labels
      sub-title-prefix: "[Task]"         # Optional: sub-issue must match this prefix
      max: 1                             # Optional: maximum number of links (default: 1)
      target-repo: "owner/repo"          # Optional: cross-repository
  ```

  Links issues as sub-issues using GitHub's parent-child relationships. Agent output includes `parent_issue_number` and `sub_issue_number`. Use with `create-issue` temporary IDs or existing issue numbers.
- `create-project:` - Create a new GitHub Project board with optional fields and views

  ```yaml
  safe-outputs:
    create-project:
      max: 1                          # Optional: max projects (default: 1)
      # github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}  # Optional: override default PAT (NOT GITHUB_TOKEN)
      target-owner: "org-or-user"     # Optional: owner for created projects
      title-prefix: "[ai] "           # Optional: prefix for project titles
  ```

  Can optionally specify custom fields, project views, and an initial item to add. Requires PAT/App token with Projects permissions (`GH_AW_PROJECT_GITHUB_TOKEN`); `GITHUB_TOKEN` cannot access Projects v2 API. Not supported for cross-repository operations.
- `update-project:` - Add items to GitHub Projects, update custom fields, manage project structure

  ```yaml
  safe-outputs:
    update-project:
      max: 20                         # Optional: max project operations (default: 10)
      project: "https://github.com/orgs/myorg/projects/42"  # REQUIRED in agent output (full URL)
      # github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}  # Optional here if GH_AW_PROJECT_GITHUB_TOKEN is set; PAT with projects:write (NOT GITHUB_TOKEN) is still required
  ```

  **⚠️**: Agent must include full project URL (not just number) in every call. Requires PAT/App token with Projects access (same as `create-project:`). Not supported for cross-repository operations.

  **Three calling modes:**

  **Mode 1: Add/update existing issues or PRs**

  ```json
  {
    "type": "update_project",
    "project": "https://github.com/orgs/myorg/projects/42",
    "content_type": "issue",
    "content_number": 123,
    "fields": {"Status": "In Progress", "Priority": "High"}
  }
  ```

  - `content_type`: "issue" or "pull_request"
  - `content_number`: The issue or PR number to add/update
  - `fields`: Custom field values to set on the item (optional)

  **Mode 2: Create draft issues in the project**

  ```json
  {
    "type": "update_project",
    "project": "https://github.com/orgs/myorg/projects/42",
    "content_type": "draft_issue",
    "draft_title": "Follow-up: investigate performance",
    "draft_body": "Check memory usage under load",
    "temporary_id": "aw_abc123def456",
    "fields": {"Status": "Backlog"}
  }
  ```

  - `content_type`: "draft_issue"
  - `draft_title`: Title of the draft issue (required when creating new)
  - `draft_body`: Description in markdown (optional)
  - `temporary_id`: Unique ID for this draft (format: `aw_` + 3-8 alphanumeric chars) for referencing in future updates (optional)
  - `draft_issue_id`: Reference an existing draft by its temporary_id to update it (optional)
  - `fields`: Custom field values (optional)

  **Mode 3: Create custom fields or views** (with `operation` field)

  ```json
  {
    "type": "update_project",
    "project": "https://github.com/orgs/myorg/projects/42",
    "operation": "create_fields",
    "field_definitions": [
      {"name": "Priority", "data_type": "SINGLE_SELECT", "options": ["High", "Medium", "Low"]},
      {"name": "Due Date", "data_type": "DATE"}
    ]
  }
  ```

  - `operation`: "create_fields" or "create_view"
  - `field_definitions`: Array of field definitions (for create_fields)
  - `view`: View configuration object with `name`, `layout` (table/board/roadmap), optional `filter` and `visible_fields` (for create_view)

  Not supported for cross-repository operations.
- `create-project-status-update:` - Post status updates to GitHub Projects for progress tracking

  ```yaml
  safe-outputs:
    create-project-status-update:
      max: 1                          # Optional: max status updates (default: 1)
      project: "https://github.com/orgs/myorg/projects/42"  # REQUIRED in agent output (full URL)
      github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}  # REQUIRED: PAT with projects:write (NOT GITHUB_TOKEN)
  ```

  Requires same PAT/App token as `update-project`. Agent must include full project URL in every call.

  **Agent output fields:**
  - `project`: Full project URL (required) - MUST be explicitly included in output
  - `status`: ON_TRACK, AT_RISK, OFF_TRACK, COMPLETE, or INACTIVE (optional, defaults to ON_TRACK)
  - `start_date`: Project start date in YYYY-MM-DD format (optional)
  - `target_date`: Project end date in YYYY-MM-DD format (optional)
  - `body`: Status summary in markdown (required)

  Not supported for cross-repository operations.
- `push-to-pull-request-branch:` - Push changes to PR branch

  ```yaml
  safe-outputs:
    push-to-pull-request-branch:
      target: "*"                     # Optional: "triggering" (default), "*", or number
      title-prefix: "[bot] "          # Optional: require title prefix
      labels: [automated]             # Optional: require all labels
      if-no-changes: "warn"           # Optional: "warn" (default), "error", or "ignore"
      commit-title-suffix: "[auto]"   # Optional: suffix appended to commit title
      staged: true                    # Optional: preview mode (default: follows global staged)
      github-token-for-extra-empty-commit: ${{ secrets.MY_CI_PAT }}  # Optional: PAT or "app" to trigger CI on pushed commits
      allowed-files:                  # Optional: exclusive allowlist of glob patterns for eligible files
        - "src/**"
      excluded-files:                 # Optional: glob patterns to strip from the patch entirely
        - "**/*.lock"
      protected-files: blocked        # Optional: "blocked" (default), "fallback-to-issue", or "allowed"
  ```

  Not supported for cross-repository operations. To trigger CI on pushed commits, use `github-token-for-extra-empty-commit` or set the magic secret `GH_AW_CI_TRIGGER_TOKEN`.

  **File Restrictions**: Same as `create-pull-request`: `allowed-files` is an exclusive allowlist, `excluded-files` strips files before all checks, and `protected-files` controls handling of sensitive files. Object form supported: `protected-files: { policy: fallback-to-issue, exclude: [AGENTS.md] }`.

  **Compile-time warnings for `target: "*"`**: When `target: "*"` is set, the compiler emits warnings if:
  1. The checkout configuration does not include a wildcard fetch pattern — add `fetch: ["*"]` with `fetch-depth: 0` so the agent can access all PR branches at runtime
  2. No constraints are provided — add `title-prefix` or `labels` to restrict which PRs can receive pushes

  Example with all recommended settings:

  ```yaml
  checkout:
    fetch: ["*"]
    fetch-depth: 0
  safe-outputs:
    push-to-pull-request-branch:
      target: "*"
      title-prefix: "[bot] "   # restrict to PRs with this title prefix
      labels: [automated]      # restrict to PRs carrying all these labels
  ```

- `update-discussion:` - Update discussion title, body, or labels

  ```yaml
  safe-outputs:
    update-discussion:
      title: true                     # Optional: enable title updates
      body: true                      # Optional: enable body updates
      labels: true                    # Optional: enable label updates
      allowed-labels: [status, type]  # Optional: restrict to specific labels
      max: 1                          # Optional: max updates (default: 1)
      target: "*"                     # Optional: "triggering" (default), "*", or number
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

- `update-release:` - Update GitHub release descriptions

  ```yaml
  safe-outputs:
    update-release:
      max: 1                          # Optional: max releases (default: 1, max: 10)
      target-repo: "owner/repo"       # Optional: cross-repository
      github-token: ${{ secrets.GH_AW_UPDATE_RELEASE_TOKEN }}  # Optional: custom token
  ```

  Operation types: `replace`, `append`, `prepend`.
- `upload-asset:` - Publish files to orphaned git branch (recommended for images/charts/screenshots)

  ```yaml
  safe-outputs:
    upload-asset:
      branch: "assets/${{ github.workflow }}"  # Optional: branch name
      max-size: 10240                 # Optional: max file size in KB (default: 10MB)
      allowed-exts: [.png, .jpg, .pdf] # Optional: allowed file extensions
      max: 10                         # Optional: max assets (default: 10)
  ```

  Publishes files to an orphaned git branch for persistent storage and URL-addressable embedding. Default allowed extensions include common non-executable types. Maximum file size is 50MB (51200 KB). **Use this for images, charts, and screenshots that need embeddable URLs in issues/PRs/discussions.**
- `upload-artifact:` - Upload files as run-scoped GitHub Actions artifacts (recommended for temporary run artifacts)

  ```yaml
  safe-outputs:
    upload-artifact:
      max-uploads: 5                  # Optional: max upload_artifact tool calls (default: 1)
      default-retention-days: 7       # Optional: default retention period in days (default: 7)
      max-retention-days: 30          # Optional: maximum retention cap in days (default: 30)
      max-size-bytes: 104857600       # Optional: max bytes per upload (default: 100 MB)
      allowed-paths:                  # Optional: glob patterns restricting uploadable paths
        - "reports/**"
        - "*.json"
      filters:                        # Optional: default include/exclude glob filters
        include: ["*.json", "*.csv"]
        exclude: ["*secret*"]
      defaults:                       # Optional: default values injected when agent omits a field
        if-no-files: "ignore"         # "error" or "ignore" when no files match (default: "error")
      allow:                          # Optional: opt-in behaviors
        skip-archive: true            # Allow agent to upload files without zipping
  ```

  Uploads files as run-scoped GitHub Actions artifacts. Artifacts are temporary and tied to the workflow run, automatically cleaned up when they expire. Agents call `upload_artifact` with a `name`, `path`, and optional `retention_days`. **Use this for temporary downloadable artifacts**, while `upload-asset` is preferred for embedding images/charts in GitHub content.
- `dispatch-workflow:` - Trigger other workflows with inputs

  ```yaml
  safe-outputs:
    dispatch-workflow:
      workflows: [workflow-name]          # Required: list of workflow names to allow
      max: 3                              # Optional: max dispatches (default: 1, max: 3)
  ```

  Triggers other agentic workflows in the same repository using workflow_dispatch. Agent output includes `workflow_name` (without .md extension) and optional `inputs` (key-value pairs). Not supported for cross-repository operations.
- `dispatch_repository:` - Dispatch `repository_dispatch` events to external repositories (experimental)

  ```yaml
  safe-outputs:
    dispatch_repository:
      trigger_ci:                              # Tool name (normalized to MCP tool: trigger_ci)
        description: "Trigger CI in target repo"
        workflow: ci.yml                       # Required: target workflow name (for traceability)
        event_type: ci_trigger                 # Required: repository_dispatch event_type
        repository: org/target-repo           # Required: target repo (or use allowed_repositories)
        # allowed_repositories:               # Alternative: allow multiple target repos
        #   - org/repo1
        #   - org/repo2
        inputs:                               # Optional: input schema for agent
          environment:
            type: string
            description: "Deployment environment"
            required: true
        max: 1                                # Optional: max dispatches (templatable)
        github-token: ${{ secrets.MY_PAT }}   # Optional: override token
        staged: false                         # Optional: preview-only mode
  ```

  Accepts both `dispatch_repository` (underscore, preferred) and `dispatch-repository` (dash). Each key in the config defines a named MCP tool. Requires a token with `repo` scope since `GITHUB_TOKEN` cannot trigger `repository_dispatch` in external repositories. Use `github-token` or set a PAT as `GH_AW_SAFE_OUTPUTS_TOKEN`.

  **⚠️ Experimental**: Compilation emits a warning when this feature is used.
- `call-workflow:` - Call reusable workflows via workflow_call fan-out (orchestrator pattern)

  ```yaml
  safe-outputs:
    call-workflow:
      workflows: [worker-a, worker-b]     # Required: workflow names (without .md) with workflow_call trigger
      max: 1                              # Optional: max calls per run (default: 1, max: 50)
      github-token: ${{ secrets.TOKEN }}  # Optional: token passed to called workflows
  ```

  Array shorthand: `call-workflow: [worker-a, worker-b]`

  Unlike `dispatch-workflow` (which uses the GitHub Actions API at runtime), `call-workflow` generates static conditional `uses:` jobs at compile time. The agent selects which worker to activate; the compiler validates and wires up all fan-out jobs. Each listed workflow must exist in `.github/workflows/` and declare a `workflow_call` trigger. Use this for orchestrator/dispatcher patterns within the same repository.
- `create-code-scanning-alert:` - Generate SARIF security advisories

  ```yaml
  safe-outputs:
    create-code-scanning-alert:
      max: 50                         # Optional: max findings (default: unlimited)
  ```

  Severity levels: error, warning, info, note.
- `autofix-code-scanning-alert:` - Add autofixes to code scanning alerts

  ```yaml
  safe-outputs:
    autofix-code-scanning-alert:
      max: 10                         # Optional: max autofixes (default: 10)
  ```

  Provides automated fixes for code scanning alerts.
- `create-agent-session:` - Create GitHub Copilot coding agent sessions

  ```yaml
  safe-outputs:
    create-agent-session:
      base: main                      # Optional: base branch (defaults to current)
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

  Requires PAT as `COPILOT_GITHUB_TOKEN`.
- `assign-to-agent:` - Assign Copilot coding agent to issues

  ```yaml
  safe-outputs:
    assign-to-agent:
      name: "copilot"                 # Optional: agent name
      model: "claude-sonnet-4-5"      # Optional: model override
      custom-agent: "agent-id"        # Optional: custom agent ID
      custom-instructions: "..."      # Optional: additional instructions for the agent
      allowed: [copilot]              # Optional: restrict to specific agent names
      max: 1                          # Optional: max assignments (default: 1)
      target: "*"                     # Optional: "triggering" (default), "*", or number
      target-repo: "owner/repo"       # Optional: where the issue lives (cross-repository)
      pull-request-repo: "owner/repo" # Optional: where PR should be created (if different)
      allowed-pull-request-repos: [owner/repo1]  # Optional: additional repos for PR creation
      base-branch: "develop"          # Optional: target branch for PR (default: repo default)
      ignore-if-error: true           # Optional: continue workflow on assignment error (default: false)
  ```

  Requires PAT with elevated permissions as `GH_AW_AGENT_TOKEN`.
- `assign-to-user:` - Assign users to issues or pull requests

  ```yaml
  safe-outputs:
    assign-to-user:
      allowed: [user1, user2]         # Optional: restrict to specific users
      blocked: [copilot, "*[bot]"]    # Optional: deny specific users or glob patterns
      max: 3                          # Optional: max assignments (default: 3)
      target: "*"                     # Optional: "triggering" (default), "*", or number
      target-repo: "owner/repo"       # Optional: cross-repository
      unassign-first: true            # Optional: unassign all current assignees first (default: false)
  ```

- `unassign-from-user:` - Remove user assignments from issues or PRs

  ```yaml
  safe-outputs:
    unassign-from-user:
      allowed: [user1, user2]         # Optional: restrict to specific users
      blocked: [copilot, "*[bot]"]    # Optional: deny specific users or glob patterns
      max: 1                          # Optional: max unassignments (default: 1)
      target: "*"                     # Optional: "triggering" (default), "*", or number
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

- `hide-comment:` - Hide comments on issues, PRs, or discussions

  ```yaml
  safe-outputs:
    hide-comment:
      max: 5                          # Optional: max comments to hide (default: 5)
      allowed-reasons:                 # Optional: restrict hide reasons
        - spam
        - outdated
        - resolved
      target-repo: "owner/repo"       # Optional: cross-repository
  ```

  Allowed reasons: `spam`, `abuse`, `off_topic`, `outdated`, `resolved`.
- `set-issue-type:` - Set the type of an issue (requires organization-defined issue types)

  ```yaml
  safe-outputs:
    set-issue-type:
      allowed: [Bug, Feature, Enhancement]  # Optional: restrict to specific issue type names
      target: "triggering"                  # Optional: "triggering" (default), "*", or number
      max: 5                                # Optional: max operations (default: 5)
      target-repo: "owner/repo"             # Optional: cross-repository
  ```

  Set `allowed` to an empty string `""` to allow clearing the issue type. When `allowed` is omitted, any type name is accepted.
- `set-issue-field:` - Set a single issue field value by name/value (avoids the broader update-issue path)

  ```yaml
  safe-outputs:
    set-issue-field:
      allowed-fields: [Priority, Iteration]  # Optional: restrict which issue fields the agent may set (omit/empty = any field; ["*"] explicitly allows all)
      target: "triggering"                    # Optional: "triggering" (default), "*", or number
      max: 5                                  # Optional: max operations (default: 5)
      target-repo: "owner/repo"               # Optional: cross-repository
      allowed-repos: [owner/other]            # Optional: additional repos agent can target
  ```

  Agent calls `set_issue_field` with `value` plus either `field_name` (preferred) or `field_node_id`. `issue_number` is optional and defaults to the triggering issue.
- `noop:` - Log completion message for transparency (auto-enabled)

  ```yaml
  safe-outputs:
    noop:
      report-as-issue: false          # Optional: report noop as issue (default: true)
  ```

  The noop safe-output provides a fallback mechanism ensuring workflows never complete silently. When enabled (automatically by default), agents can emit human-visible messages even when no other actions are required (e.g., "Analysis complete - no issues found"). This ensures every workflow run produces visible output.
- `missing-tool:` - Report missing tools or functionality (auto-enabled)

  ```yaml
  safe-outputs:
    missing-tool:
      create-issue: true              # Optional: create issues for missing tools (default: true)
      title-prefix: "[missing tool]"  # Optional: prefix for issue titles
      labels: [tool-request]          # Optional: labels for created issues
  ```

  The missing-tool safe-output allows agents to report when they need tools or functionality not currently available. This is automatically enabled by default and helps track feature requests from agents. When `create-issue` is true, missing tool reports create or update GitHub issues for tracking.
- `missing-data:` - Report missing data required to complete tasks (auto-enabled)

  ```yaml
  safe-outputs:
    missing-data:
      create-issue: true              # Optional: create issues for missing data (default: true)
      title-prefix: "[missing data]"  # Optional: prefix for issue titles
      labels: [data-request]          # Optional: labels for created issues
  ```

  The missing-data safe-output allows agents to report when required data or information is unavailable. This is automatically enabled by default. When `create-issue` is true, missing data reports create or update GitHub issues for tracking.

- `report-incomplete:` - Signal that the task could not be completed due to an infrastructure or tool failure (auto-enabled)

  ```yaml
  safe-outputs:
    report-incomplete:
      create-issue: true              # Optional: create issues for incomplete tasks (default: true)
      title-prefix: "[incomplete]"    # Optional: prefix for issue titles
      labels: [agent-failure]         # Optional: labels for created issues
  ```

  The report-incomplete safe-output is automatically enabled by default and is distinct from `noop`. Use it when required tools or data are unavailable and the task cannot be meaningfully performed (e.g., MCP server crash, missing authentication, inaccessible repository). When an agent emits `report_incomplete`, gh-aw activates failure handling even when the agent process exits 0 — preventing empty outputs from being classified as successful. This ensures every unrecoverable failure is tracked.

- `jobs:` - Custom safe-output jobs registered as MCP tools for third-party integrations

  ```yaml
  safe-outputs:
    jobs:
      send-notification:
        description: "Send a notification to an external service"
        runs-on: ubuntu-latest
        output: "Notification sent successfully!"
        inputs:
          message:
            description: "The message to send"
            required: true
            type: string
        permissions:
          contents: read
        env:
          API_KEY: ${{ secrets.API_KEY }}
        steps:
          - name: Send notification
            run: |
              MESSAGE=$(cat "$GH_AW_AGENT_OUTPUT" | jq -r '.items[] | select(.type == "send_notification") | .message')
              curl -H "Authorization: $API_KEY" -d "$MESSAGE" https://api.example.com/notify
  ```

  Custom safe-output jobs define post-processing GitHub Actions jobs registered as MCP tools. Agents call the tool by its normalized name (dashes converted to underscores, e.g., `send_notification`). The job runs after the agent completes with access to `$GH_AW_AGENT_OUTPUT` (the path to agent output JSON). Use this to integrate with Slack, Discord, external APIs, databases, or any service requiring secrets. Import from shared files using the `imports:` field.

- `scripts:` - Inline JavaScript handlers running inside the safe-outputs job handler loop

  ```yaml
  safe-outputs:
    scripts:
      post-slack-message:
        description: "Post a message to Slack"
        inputs:
          channel:
            description: "Target Slack channel"
            type: string
            default: "#general"
        script: |
          // 'channel' is available from config inputs; 'item' contains runtime message values
          await fetch(process.env.SLACK_WEBHOOK_URL, {
            method: "POST",
            body: JSON.stringify({ text: item.message, channel })
          });
  ```

  Unlike `jobs:` (which create separate GitHub Actions jobs), scripts execute in-process alongside built-in handlers. Write only the handler body — the compiler generates the outer wrapper with config input destructuring and `async function handleX(item, resolvedTemporaryIds) { ... }`. Script names with dashes are normalized to underscores (e.g., `post-slack-message` → `post_slack_message`). The handler receives `item` (runtime message with input values) and `resolvedTemporaryIds` (map of temporary IDs).

- `actions:` - Custom GitHub Actions mounted as MCP tools for the AI agent (resolved at compile time)

  ```yaml
  safe-outputs:
    actions:
      my-action:
        uses: owner/repo/path@ref         # Required: GitHub Action reference (tag, SHA, or branch)
        description: "Custom description" # Optional: override action's description from action.yml
        env:
          API_KEY: ${{ secrets.API_KEY }} # Optional: environment variables for the injected step
  ```

  Actions are resolved at compile time — the compiler fetches `action.yml` and parses inputs automatically, exposing them as MCP tool parameters. The agent calls the action by its normalized name (dashes converted to underscores). Each action runs as an injected step in the safe-outputs job. Local actions (`./path/to/action`) are also supported.

**Global Safe Output Configuration:**
- `github-token:` - Custom GitHub token for all safe output jobs

  ```yaml
  safe-outputs:
    create-issue:
    add-comment:
    github-token: ${{ secrets.GH_AW_SAFE_OUTPUTS_TOKEN }}  # Use custom PAT instead of GITHUB_TOKEN
  ```

  Useful when you need additional permissions or want to perform actions across repositories.
- `allowed-domains:` - Allowed domains for URLs in safe output content (array)
  - URLs from unlisted domains are replaced with `(redacted)`
  - GitHub domains are always included by default
- `allowed-github-references:` - Allowed repositories for GitHub-style references (array)
  - Controls which GitHub references (`#123`, `owner/repo#456`) are allowed in workflow output
  - References to unlisted repositories are escaped with backticks to prevent timeline items
  - Configuration options:
    - `[]` - Escape all references (prevents all timeline items)
    - `["repo"]` - Allow only the target repository's references
    - `["repo", "owner/other-repo"]` - Allow specific repositories
    - Not specified (default) - All references allowed
  - Example:

    ```yaml
    safe-outputs:
      allowed-github-references: []  # Escape all references
      create-issue:
        target-repo: "my-org/main-repo"
    ```

    With `[]`, references like `#123` become `` `#123` `` and `other/repo#456` becomes `` `other/repo#456` ``, preventing timeline clutter while preserving information.
- `messages:` - Custom message templates for safe-output footer and notification messages (object)
  - Available placeholders: `{workflow_name}`, `{run_url}`, `{agentic_workflow_url}`, `{triggering_number}`, `{workflow_source}`, `{workflow_source_url}`, `{operation}`, `{event_type}`, `{status}`, `{effective_tokens}`, `{effective_tokens_formatted}`, `{effective_tokens_suffix}`
  - Message types:
    - `footer:` - Custom footer for AI-generated content
    - `footer-install:` - Installation instructions appended to footer
    - `footer-workflow-recompile:` - Footer for workflow recompile tracking issues (placeholder: `{repository}`)
    - `footer-workflow-recompile-comment:` - Footer for comments on workflow recompile issues (placeholder: `{repository}`)
    - `run-started:` - Workflow activation notification
    - `run-success:` - Successful completion message
    - `run-failure:` - Failure notification message
    - `detection-failure:` - Detection job failure message
    - `agent-failure-issue:` - Footer for agent failure tracking issues
    - `agent-failure-comment:` - Footer for comments on agent failure tracking issues
    - `staged-title:` - Staged mode preview title
    - `staged-description:` - Staged mode preview description
    - `append-only-comments:` - Create new comments instead of editing existing ones (boolean, default: false)
    - `pull-request-created:` - Custom message when a PR is created. Placeholders: `{item_number}`, `{item_url}`
    - `issue-created:` - Custom message when an issue is created. Placeholders: `{item_number}`, `{item_url}`
    - `commit-pushed:` - Custom message when a commit is pushed. Placeholders: `{commit_sha}`, `{short_sha}`, `{commit_url}`
    - `body-header:` - Custom header text prepended to every message body (issues, comments, PRs, discussions). Placeholders: `{workflow_name}`, `{run_url}`
  - Example:

    ```yaml
    safe-outputs:
      messages:
        append-only-comments: true
        footer: "> Generated by [{workflow_name}]({run_url})"
        run-started: "[{workflow_name}]({run_url}) started processing this {event_type}."
    ```

- `mentions:` - Configuration for @mention filtering in safe outputs (boolean or object)
  - Boolean format: `false` - Always escape mentions; `true` - Always allow (error in strict mode)
  - Object format for fine-grained control:

    ```yaml
    safe-outputs:
      mentions:
        allow-team-members: true    # Allow repository collaborators (default: true)
        allow-context: true          # Allow mentions from event context (default: true)
        allowed: [copilot, user1]    # Always allow specific users/bots
        max: 50                      # Maximum mentions per message (default: 50)
    ```

  - Team members include collaborators with any permission level (excluding bots unless explicitly listed)
  - Context mentions include issue/PR authors, assignees, and commenters
- `runs-on:` - Runner specification for all safe-outputs jobs (string)
  - Defaults to `ubuntu-slim` (1-vCPU runner)
  - Examples: `ubuntu-latest`, `windows-latest`, `self-hosted`
  - Applies to activation, create-issue, add-comment, and other safe-output jobs
- `footer:` - Global footer control for all safe outputs (boolean, default: `true`)
  - When `false`, omits visible AI-generated footer content from all created/updated entities (issues, PRs, discussions, releases) while still including XML markers for searchability
  - Individual safe-output types can override this setting
- `staged:` - Preview mode for all safe outputs (boolean)
  - When `true`, emits step summary messages instead of making GitHub API calls; useful for testing without side effects
- `env:` - Environment variables passed to all safe output jobs (object)
  - Values typically reference secrets: `MY_VAR: ${{ secrets.MY_SECRET }}`
- `steps:` - Custom steps injected into all safe-output jobs, running after repository checkout and before safe-output code (array)
  - Useful for installing dependencies or performing setup needed by safe-output logic
  - Example:

    ```yaml
    safe-outputs:
      steps:
        - name: Install custom dependencies
          run: npm install my-package
      create-issue:
    ```

- `max-bot-mentions:` - Maximum bot trigger references (e.g. `@copilot`, `@github-actions`) allowed in output before all excess are escaped with backticks (integer or expression, default: 10)
  - Set to `0` to escape all bot trigger phrases
  - Example: `max-bot-mentions: 3`
- `activation-comments:` - Disable all activation and fallback comments (boolean or expression, default: `true`)
  - When `false`, disables run-started, run-success, run-failure, and PR/issue creation link comments
  - Supports templatable boolean: `false`, `true`, or GitHub Actions expressions like `${{ inputs.activation-comments }}`

**Templatable Integer Fields**: The `max`, `expires`, and `max-bot-mentions` fields (and most other numeric/boolean fields) accept GitHub Actions expression strings in addition to literal values, enabling runtime-configured limits:

```yaml
safe-outputs:
  max-bot-mentions: ${{ inputs.max-mentions }}
  create-issue:
    max: ${{ inputs.max-issues }}
    expires: ${{ inputs.expires-days }}
```

Fields that influence permission computation (`add-comment.discussions`, `create-pull-request.fallback-as-issue`) remain literal booleans.

- `max-patch-size:` - Maximum allowed git patch size in kilobytes (integer, default: 1024 KB = 1 MB)
  - Patches exceeding this size are rejected to prevent accidental large changes
- `max-patch-files:` - Maximum allowed number of unique files in a create-pull-request patch (integer, default: 100)
  - Counts unique file paths deduplicated across multi-commit patches; reflects how many distinct files the agent is pushing per iteration
  - Increase this limit for long-running branches that touch many files
- `group-reports:` - Group workflow failure reports as sub-issues (boolean, default: `false`)
  - When `true`, creates a parent `[aw] Failed runs` issue that tracks all workflow failures as sub-issues; useful for larger repositories
- `report-failure-as-issue:` - Control whether workflow failures are reported as GitHub issues (boolean, default: `true`)
  - When `false`, suppresses automatic failure issue creation for this workflow
  - Use to silence noisy failure reports for workflows where failures are expected or handled externally
- `failure-issue-repo:` - Repository to create failure tracking issues in (string, format: `"owner/repo"`)
  - Defaults to the current repository when not specified
  - Use when the current repository has issues disabled: `failure-issue-repo: "myorg/infra-alerts"`
- `id-token:` - Override the id-token permission for the safe-outputs job (string: `"write"` or `"none"`)
  - `"write"`: force-enable `id-token: write` permission (required for OIDC authentication with cloud providers)
  - `"none"`: suppress automatic detection and prevent adding `id-token: write` even when vault/OIDC actions are detected in steps
  - Default: auto-detects known OIDC/vault actions (e.g., `aws-actions/configure-aws-credentials`, `azure/login`, `hashicorp/vault-action`) and adds `id-token: write` automatically
- `concurrency-group:` - Concurrency group for the safe-outputs job (string)
  - When set, the safe-outputs job uses this concurrency group with `cancel-in-progress: false`
  - Supports GitHub Actions expressions, e.g., `"safe-outputs-${{ github.repository }}"`
- `needs:` - Additional custom workflow jobs the safe-outputs job depends on (array)
  - Example: `needs: [secrets_fetcher]`
  - Use when the safe-outputs job requires outputs from a custom job defined in `jobs:`
- `environment:` - Override the GitHub deployment environment for the safe-outputs job (string)
  - Defaults to the top-level `environment:` field when not specified
  - Use when the main job and safe-outputs job need different deployment environments for protection rules
- `github-app:` - GitHub App credentials for minting installation access tokens (object)
  - When configured, generates a token from the app and uses it for all safe output operations (alternative to `github-token`)
  - Fields:
    - `client-id:` - GitHub App client ID (required, e.g., `${{ vars.APP_ID }}`). Use `app-id:` for legacy compatibility.
    - `private-key:` - GitHub App private key (required, e.g., `${{ secrets.APP_PRIVATE_KEY }}`)
    - `owner:` - Optional App installation owner (defaults to current repository owner)
    - `repositories:` - Optional list of repositories to grant access to
  - Example:

    ```yaml
    safe-outputs:
      github-app:
        client-id: ${{ vars.APP_ID }}
        private-key: ${{ secrets.APP_PRIVATE_KEY }}
      create-issue:
    ```

- `threat-detection:` - Threat detection configuration (auto-enabled for all safe-outputs workflows)
  - Automatically enabled by default; customizable via explicit configuration
  - Fields:
    - `enabled:` - Enable/disable threat detection (boolean, default: `true`)
    - `prompt:` - Additional instructions appended to threat detection analysis (string)
    - `engine:` - AI engine for threat detection (engine config or `false` to disable AI detection)
    - `steps:` - Extra job steps to run after detection (array)
  - Example to disable AI-based detection (use custom steps only):

    ```yaml
    safe-outputs:
      threat-detection:
        engine: false
        steps:
          - name: Custom check
            run: echo "Custom threat check"
    ```


## Output Variables

The safe-outputs job emits named step outputs for the first successful result of each type:

| Safe Output | Step Output Variables |
|---|---|
| `create-issue` | `created_issue_number`, `created_issue_url` |
| `create-pull-request` | `created_pr_number`, `created_pr_url` |
| `add-comment` | `comment_id`, `comment_url` |
| `push-to-pull-request-branch` | `push_commit_sha`, `push_commit_url` |

