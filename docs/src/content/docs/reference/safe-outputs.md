---
title: Safe Outputs
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 800
---

The [`safe-outputs:`](/gh-aw/reference/glossary/#safe-outputs) element of your workflow's frontmatter declares that your agentic workflow should conclude with optional automated actions based on the agentic workflow's output. This enables your workflow to write content that is then automatically processed to create GitHub issues, comments, pull requests, or add labels—all without giving the agentic portion of the workflow any write permissions.

## Why Safe Outputs?

**Safe outputs are a security feature.** Your AI agent runs with minimal permissions (read-only access by default). When the agent wants to make changes to your repository—like creating an issue, adding a comment, or opening a pull request—it cannot do so directly. Instead, it "requests" that action by writing structured output to a file.

A separate, permission-controlled job then reviews and executes these requests. This architecture provides:

- **Principle of least privilege**: The AI never has write permissions during execution
- **Defense against prompt injection**: Malicious inputs cannot trick the AI into making unauthorized changes
- **Auditability**: All requested actions are logged and can be reviewed before execution
- **Controlled blast radius**: Each safe output type has strict limits (e.g., maximum number of issues to create)

**How It Works:**
1. The agentic part of your workflow runs with minimal read-only permissions. It is given additional prompting to write its output to special known files
2. The compiler automatically generates additional jobs that read this output and perform the requested actions
3. Only these generated jobs receive the necessary write permissions—scoped precisely to what each safe output type requires

For example:

```yaml wrap
safe-outputs:
  create-issue:
```

This declares that the workflow should create at most one new issue. The AI agent can request issue creation, but a separate job with `issues: write` permission actually creates it.

## Available Safe Output Types

| Output Type | Key | Description | Max | Cross-Repo |
|-------------|-----|-------------|-----|-----------|
| [**Create Issue**](#issue-creation-create-issue) | `create-issue:` | Create GitHub issues | 1 | ✅ |
| [**Close Issue**](#close-issue-close-issue) | `close-issue:` | Close issues with comment | 1 | ✅ |
| [**Add Comment**](#comment-creation-add-comment) | `add-comment:` | Post comments on issues, PRs, or discussions | 1 | ✅ |
| [**Hide Comment**](#hide-comment-hide-comment) | `hide-comment:` | Hide comments on issues, PRs, or discussions | 5 | ✅ |
| [**Update Issue**](#issue-updates-update-issue) | `update-issue:` | Update issue status, title, or body | 1 | ✅ |
| [**Update PR**](#pull-request-updates-update-pull-request) | `update-pull-request:` | Update PR title or body | 1 | ✅ |
| [**Link Sub-Issue**](#link-sub-issue-link-sub-issue) | `link-sub-issue:` | Link issues as sub-issues | 1 | ✅ |
| [**Update Project**](#project-board-updates-update-project) | `update-project:` | Manage GitHub Projects boards and campaign labels | 10 | ❌ |
| [**Add Labels**](#add-labels-add-labels) | `add-labels:` | Add labels to issues or PRs | 3 | ✅ |
| [**Add Reviewer**](#add-reviewer-add-reviewer) | `add-reviewer:` | Add reviewers to pull requests | 3 | ✅ |
| [**Assign Milestone**](#assign-milestone-assign-milestone) | `assign-milestone:` | Assign issues to milestones | 1 | ✅ |
| [**Create PR**](#pull-request-creation-create-pull-request) | `create-pull-request:` | Create pull requests with code changes | 1 | ✅ |
| [**Close PR**](#close-pull-request-close-pull-request) | `close-pull-request:` | Close pull requests without merging | 10 | ✅ |
| [**PR Review Comments**](#pr-review-comments-create-pull-request-review-comment) | `create-pull-request-review-comment:` | Create review comments on code lines | 10 | ✅ |
| [**Create Discussion**](#discussion-creation-create-discussion) | `create-discussion:` | Create GitHub discussions | 1 | ✅ |
| [**Close Discussion**](#close-discussion-close-discussion) | `close-discussion:` | Close discussions with comment and resolution | 1 | ✅ |
| [**Create Agent Task**](#agent-task-creation-create-agent-task) | `create-agent-task:` | Create Copilot agent tasks | 1 | ✅ |
| [**Assign to Agent**](#assign-to-agent-assign-to-agent) | `assign-to-agent:` | Assign Copilot agents to issues | 1 | ✅ |
| [**Assign to User**](#assign-to-user-assign-to-user) | `assign-to-user:` | Assign users to issues | 1 | ✅ |
| [**Push to PR Branch**](#push-to-pr-branch-push-to-pull-request-branch) | `push-to-pull-request-branch:` | Push changes to PR branch | 1 | ❌ |
| [**Update Release**](#release-updates-update-release) | `update-release:` | Update GitHub release descriptions | 1 | ✅ |
| [**Upload Assets**](#asset-uploads-upload-asset) | `upload-asset:` | Upload files to orphaned git branch | 10 | ❌ |
| [**Code Scanning Alerts**](#code-scanning-alerts-create-code-scanning-alert) | `create-code-scanning-alert:` | Generate SARIF security advisories | unlimited | ❌ |
| [**No-Op**](#no-op-logging-noop) | `noop:` | Log completion message for transparency (auto-enabled) | 1 | ❌ |
| [**Missing Tool**](#missing-tool-reporting-missing-tool) | `missing-tool:` | Report missing tools (auto-enabled) | unlimited | ❌ |

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
    expires: 7                       # auto-close after 7 days
    target-repo: "owner/repo"        # cross-repository
```

#### Auto-Expiration

The `expires` field automatically closes issues after a specified time period. Supports both integer (days) and relative time formats:

- **Integer**: `expires: 7` (7 days)
- **Days**: `expires: 7d` or `7D` (7 days)
- **Weeks**: `expires: 2w` or `2W` (14 days)
- **Months**: `expires: 1m` or `1M` (30 days, approximate)
- **Years**: `expires: 1y` or `1Y` (365 days, approximate)

When enabled, the compiler automatically generates an `agentics-maintenance.yml` workflow that runs daily to close expired items. Issues are closed as "completed" with an explanatory comment and workflow attribution.

#### Temporary IDs for Issue References

When creating multiple issues, use temporary IDs to reference parent issues before they're created. The agent provides a `temporary_id` field with format `aw_` followed by 12 hex characters.

**Agent Output Format:**
```json
[
  {
    "type": "create_issue",
    "title": "Parent Issue",
    "body": "This is the parent issue",
    "temporary_id": "aw_abc123def456"
  },
  {
    "type": "create_issue",
    "title": "Sub Issue",
    "body": "References #aw_abc123def456",
    "parent": "aw_abc123def456"
  }
]
```

References like `#aw_abc123def456` in issue bodies are automatically replaced with the actual issue number (e.g., `#42`) after the parent issue is created. The `parent` field creates a sub-issue relationship.

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
    max: 3                       # max comments (default: 1)
    target: "*"                  # "triggering" (default), "*", or number
    discussion: true             # target discussions
    target-repo: "owner/repo"    # cross-repository
    hide-older-comments: true    # hide previous comments from same workflow
    allowed-reasons: [outdated]  # restrict hiding reasons (optional)
```

When combined with `create-issue`, `create-discussion`, or `create-pull-request`, comments automatically include a "Related Items" section.

#### Hide Older Comments

The `hide-older-comments` field automatically minimizes all previous comments from the same agentic workflow before creating a new comment. This is useful for workflows that provide status updates, where you want to keep the conversation clean by hiding outdated information.

**Configuration:**
```yaml wrap
safe-outputs:
  add-comment:
    hide-older-comments: true
    allowed-reasons: [OUTDATED, RESOLVED]  # optional: restrict reasons
```

**How it works:**
- Comments from the same workflow are identified by the workflow ID (automatically from `GITHUB_WORKFLOW` environment variable)
- All matching older comments are minimized/hidden with reason "outdated" (by default)
- The new comment is then created normally
- Works for both issue/PR comments and discussion comments

**Allowed Reasons:**
Use `allowed-reasons` to restrict which reasons can be used when hiding comments. Valid reasons are:
- `spam` - Mark as spam
- `abuse` - Mark as abusive
- `off_topic` - Mark as off-topic
- `outdated` - Mark as outdated (default)
- `resolved` - Mark as resolved

If `allowed-reasons` is not specified, all reasons are allowed. If specified, only the listed reasons can be used. If the default reason (outdated) is not in the allowed list, hiding will be skipped with a warning.

**Requirements:**
- Workflow ID is automatically obtained from the `GITHUB_WORKFLOW` environment variable
- Only comments with matching workflow ID will be hidden
- Requires write permissions (automatically granted to the safe-output job)

**Example workflow:**
```yaml wrap
---
safe-outputs:
  add-comment:
    hide-older-comments: true
    allowed-reasons: [outdated, resolved]
---

Current status: {{ statusMessage }}
```

### Hide Comment (`hide-comment:`)

Hides comments on issues, pull requests, or discussions. Comments are collapsed in the GitHub UI with a specified reason. This safe output is useful for content moderation workflows.

```yaml wrap
safe-outputs:
  hide-comment:
    max: 5                    # max comments to hide (default: 5)
    target-repo: "owner/repo" # cross-repository
```

**Requirements:**
- Agent must provide GraphQL node IDs (strings like `IC_kwDOABCD123456`) for comments
- REST API numeric comment IDs cannot be used (no conversion available)
- Agent can optionally specify a reason (spam, abuse, off_topic, outdated, resolved)

**Agent Output Format:**
```json
{
  "type": "hide_comment",
  "comment_id": "IC_kwDOABCD123456",
  "reason": "spam"
}
```

**Permissions Required:** `contents: read`, `issues: write`, `pull-requests: write`, `discussions: write`

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

Use `reviewers: copilot` to assign the Copilot PR reviewer bot. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` or `GH_AW_GITHUB_TOKEN` (legacy).

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

### Pull Request Updates (`update-pull-request:`)

Updates PR title or body. Both fields are enabled by default. The `operation` field controls how body updates are applied: `append` (default), `prepend`, or `replace`.

```yaml wrap
safe-outputs:
  update-pull-request:
    title: true               # enable title updates (default: true)
    body: true                # enable body updates (default: true)
    max: 1                    # max updates (default: 1)
    target: "*"               # "triggering" (default), "*", or number
    target-repo: "owner/repo" # cross-repository
```

**Operation Types**:
- `append` (default): Adds content to the end with separator and attribution
- `prepend`: Adds content to the start with separator and attribution
- `replace`: Completely replaces existing body

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
```

Agent output includes `parent_issue_number` and `sub_issue_number`. Validation ensures both issues exist and meet label/prefix requirements before linking.

### Project Board Updates (`update-project:`)

Manages GitHub Projects boards. Generated job runs with `projects: write` permissions, links the board to the repository, and maintains campaign metadata.

**Important**: GitHub Projects v2 requires a PAT or GitHub App token. The default `GITHUB_TOKEN` cannot access the Projects v2 GraphQL API. You must configure [`GH_AW_PROJECT_GITHUB_TOKEN`](/gh-aw/reference/tokens/#gh_aw_project_github_token-github-projects-v2) or provide a custom token via `safe-outputs.update-project.github-token`.

By default, `update-project` is **update-only**: if the project board does not exist, the job fails with instructions to create the board manually.

```yaml wrap
safe-outputs:
  update-project:
    max: 20                         # max project operations (default: 10)
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }} # required: PAT with Projects access
```

Agent output **must include a full GitHub project URL** in the `project` field (e.g., `https://github.com/orgs/myorg/projects/42` or `https://github.com/users/username/projects/5`). Project names or numbers alone are not accepted. Can also supply `content_number`, `content_type`, `fields`, and `campaign_id`. When `campaign_id` is provided, `update-project` treats it as the campaign tracker identifier and applies the `campaign:<id>` label (for example, `campaign_id: security-sprint` results in `campaign:security-sprint`). See [Agentic Campaign Workflows](/gh-aw/guides/campaigns/) for the end-to-end campaign model.

The job adds the issue or PR to the board, updates custom fields, and exposes `project-id`, `project-number`, `project-url`, `campaign-id`, and `item-id` outputs. Cross-repository targeting not supported.

To opt in to creating missing project boards, include `create_if_missing: true` in the `update_project` output. Your token must have sufficient permissions:
- **User-owned Projects**: Classic PAT with `project` + `repo` scopes (fine-grained PATs don't work)
- **Organization-owned Projects**: Classic PAT with `project` + `read:org` scopes, or fine-grained PAT with explicit Organization access and Projects: Read+Write, or GitHub App with Projects permissions

See [GitHub Projects v2 token requirements](/gh-aw/reference/tokens/#gh_aw_project_github_token-github-projects-v2) for detailed setup instructions.

### Pull Request Creation (`create-pull-request:`)

Creates pull requests with code changes. Falls back to creating an issue if PR creation fails (e.g., organization settings block it).

```yaml wrap
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "         # prefix for titles
    labels: [automation]          # labels to attach
    reviewers: [user1, copilot]   # reviewers (use 'copilot' for bot)
    draft: true                   # create as draft (default: true)
    expires: 14                   # auto-close after 14 days (same-repo only)
    if-no-changes: "warn"         # "warn" (default), "error", or "ignore"
    target-repo: "owner/repo"     # cross-repository
```

:::note
PR creation may fail if "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings → Actions → General → Workflow permissions. When fallback occurs, an issue is created with branch link and error details.
:::

#### Auto-Expiration (Same-Repository Only)

The `expires` field automatically closes pull requests after a specified time period. **Only works for same-repository PRs** (when `target-repo` is not set). Supports the same time formats as issues: integers for days, or relative time strings (`7d`, `2w`, `1m`, `1y`).

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
    max: 3                    # max comments (default: 10)
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

### Asset Uploads (`upload-asset:`)

Uploads generated files (screenshots, charts, reports, diagrams) to an orphaned git branch for persistent, version-controlled storage. Assets are uploaded without requiring elevated permissions during agent execution—a separate job with `contents: write` handles the actual commit and push.

**How it works:**
1. Agent generates files during workflow execution (screenshots, data visualizations, etc.)
2. Agent calls the `upload_asset` tool to register each file for upload
3. Files are uploaded to GitHub Actions artifacts
4. After agent completes, a separate job downloads the artifacts and commits them to the specified branch
5. Assets are accessible via predictable GitHub URLs

```yaml wrap
safe-outputs:
  upload-asset:
    branch: "assets/my-workflow"     # branch name (default: `"assets/${{ github.workflow }}"`)
    max-size: 5120                   # max file size in KB (default: 10240 = 10MB)
    allowed-exts: [.png, .jpg, .svg] # allowed extensions (default: [.png, .jpg, .jpeg])
    max: 20                          # max assets (default: 10)
```

**Branch Name Requirements:**

When creating a new branch, it must start with the `assets/` prefix for security. This restriction prevents accidental overwrites of important branches. If the branch already exists, any name is allowed.

- ✅ `assets/screenshots` - Valid for new branches
- ✅ `assets/my-workflow` - Valid for new branches  
- ✅ `assets/daily-reports` - Valid for new branches
- ✅ `existing-custom-branch` - Valid only if branch already exists
- ❌ `custom/branch-name` - Invalid for new branches (missing `assets/` prefix)

To use a custom branch name without the `assets/` prefix, create the branch manually first:

```bash
git checkout --orphan my-custom-branch
git rm -rf .
git commit --allow-empty -m "Initialize assets branch"
git push origin my-custom-branch
```

**Agent Output Format:**

The agent calls the `upload_asset` tool with the file path. The tool validates the file, uploads it as an artifact, and records it for later processing:

```json
{
  "type": "upload_asset",
  "path": "/tmp/screenshot.png"
}
```

The `upload_asset` tool automatically:
- Calculates the SHA-256 hash for integrity verification
- Records the file size and validates against `max-size` limit
- Validates the file extension against `allowed-exts` list
- Generates the target filename and future URL
- Uploads the file to GitHub Actions artifacts

**Accessing Uploaded Assets:**

Assets are stored in an orphaned branch with no commit history. Each asset gets a predictable URL (replace `{owner}`, `{repo}`, `{branch}`, and `{filename}` with actual values):

```
https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{filename}
```

For example, if your workflow uploads `screenshot.png` to branch `assets/docs-tester` in repository `octocat/hello-world`:

```
https://raw.githubusercontent.com/octocat/hello-world/assets/docs-tester/screenshot.png
```

These URLs can be used in:
- Issue comments and descriptions
- Pull request bodies
- Discussion posts
- Documentation and README files
- Any markdown content

**Security Features:**

- **File Path Validation**: Only files within the workspace or `/tmp` directory can be uploaded
- **Extension Allowlist**: Only specified file extensions are permitted (defaults to image formats)
- **Size Limits**: Maximum file size prevents excessive storage usage
- **SHA Verification**: Files are verified using SHA-256 hashes before and after upload
- **Branch Isolation**: Uses orphaned branches (no commit history) to isolate assets from code
- **Minimal Permissions**: Agent runs with read-only access; only the upload job has write permissions

**Common Use Cases:**

1. **Visual Testing**: Upload browser screenshots showing UI issues or test results
2. **Data Visualization**: Store generated charts, graphs, and data plots
3. **Documentation**: Generate and store architecture diagrams or API documentation
4. **Reports**: Save PDF or HTML reports for analysis results
5. **Test Artifacts**: Preserve test output, logs, or debug information

**Example Workflows:**

Multi-device screenshot testing:
```yaml wrap
---
name: Visual Testing
on: schedule
tools:
  playwright:
safe-outputs:
  upload-asset:
    branch: "assets/screenshots"
    allowed-exts: [.png]
    max: 50
  create-issue:
---

Test the documentation site on mobile, tablet, and desktop. Take screenshots
of any layout issues and upload them. Create an issue with the screenshots
embedded using their raw.githubusercontent.com URLs.
```

Data visualization workflow:
```yaml wrap
---
name: Weekly Analytics
on: schedule
tools:
  bash:
safe-outputs:
  upload-asset:
    branch: "assets/charts" 
    allowed-exts: [.png, .svg]
    max-size: 2048
  add-comment:
---

Generate charts showing repository metrics (PRs, issues, commits) for the
past week. Save charts to /tmp and upload them. Add a comment to issue #123
with the charts embedded.
```

**Job Outputs:**

The upload assets job provides outputs that can be used by subsequent jobs:

- `published_count`: Number of assets successfully uploaded
- `branch_name`: The branch name where assets were uploaded (normalized)

**Permissions Required:** `contents: write` (automatically granted to the upload job only)

**Limitations:**

- Cross-repository uploads not supported (assets must be in the same repository)
- Maximum file size is 50MB (configurable up to 51200 KB)
- Maximum 100 assets per workflow run (configurable)
- Branch names are normalized to valid git branch names (lowercase, special chars replaced with dashes)

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
    expires: 3                # auto-close after 3 days
    max: 3                    # max discussions (default: 1)
    target-repo: "owner/repo" # cross-repository
```

#### Auto-Expiration

The `expires` field automatically closes discussions after a specified time period. Supports both integer (days) and relative time formats (`7d`, `2w`, `1m`, `1y`). Discussions are closed as "OUTDATED" with an explanatory comment.

When `expires` is used in any workflow, the compiler automatically generates an `agentics-maintenance.yml` workflow that runs daily to process expired items.

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

Creates GitHub Copilot agent tasks. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` (recommended) or `GH_AW_GITHUB_TOKEN` (legacy). The default `GITHUB_TOKEN` lacks agent task permissions.

```yaml wrap
safe-outputs:
  create-agent-task:
    base: main                # base branch (defaults to current)
    target-repo: "owner/repo" # cross-repository
```

### Assign to Agent (`assign-to-agent:`)

Assigns the GitHub Copilot coding agent to issues. The generated job automatically receives the necessary workflow permissions, you only need to provide a token with agent assignment scope.

```yaml wrap
safe-outputs:
  assign-to-agent:
    name: "copilot"
    target-repo: "owner/repo" # for cross-repository only
```

**Token Requirements:**

The default `GITHUB_TOKEN` lacks permission to assign agents. The `replaceActorsForAssignable` mutation requires elevated permissions. Create a fine-grained personal access token with these permissions and store it as the `GH_AW_AGENT_TOKEN` secret:

- **Read** access to metadata (granted by default)
- **Write** access to actions, contents, issues, and pull requests

Without this token, agent assignment will fail with a clear error message

```yaml wrap
safe-outputs:
  assign-to-agent:
```

Alternatively, use a GitHub App installation token or override with `github-token`:

```yaml wrap
safe-outputs:
  app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  assign-to-agent:
```

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

**Repository Settings:**

Ensure Copilot is enabled for your repository. Check organization settings if bot assignments are restricted.

Reference: [GitHub Copilot agent documentation](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/)

### Assign to User (`assign-to-user:`)

Assigns GitHub users to issues. Specify `allowed` to restrict which users can be assigned.

```yaml wrap
safe-outputs:
  assign-to-user:
    allowed: [user1, user2]    # restrict to specific users
    max: 3                     # max assignments (default: 1)
    target: "*"                # "triggering" (default), "*", or number
    target-repo: "owner/repo"  # cross-repository
```

**Target**: `"triggering"` (requires issue event), `"*"` (any issue), or number (specific issue).

**Agent Output Format:**
```json
{
  "type": "assign_to_user",
  "issue_number": 123,
  "assignees": ["octocat", "mona"]
}
```

Single user assignment is also supported:
```json
{
  "type": "assign_to_user",
  "issue_number": 123,
  "assignee": "octocat"
}
```

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

Use `assignees: copilot` or `reviewers: copilot` to assign to the Copilot bot. Requires a PAT stored as `COPILOT_GITHUB_TOKEN` (recommended) or `GH_AW_GITHUB_TOKEN` (legacy). The default `GITHUB_TOKEN` lacks bot assignment permissions.

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

## Agentic Campaign Workflows

Combine `create-issue` with `update-project` to launch coordinated initiatives. The project job returns a campaign identifier, applies `campaign:<id>` labels, and keeps boards synchronized. See [Agentic Campaign Workflows](/gh-aw/guides/campaigns/).

## Custom Messages (`messages:`)

Customize notification messages and footers for safe output operations using template variables. Messages support full Markdown formatting including links, emphasis, and emoji.

### Basic Configuration

```yaml wrap
safe-outputs:
  messages:
    footer: "> 🤖 Generated by [{workflow_name}]({run_url})"
    run-started: "🚀 [{workflow_name}]({run_url}) is processing this {event_type}..."
    run-success: "✅ [{workflow_name}]({run_url}) completed successfully"
    run-failure: "❌ [{workflow_name}]({run_url}) encountered {status}"
  create-issue:
```

### Real-World Example

The `archie.md` workflow demonstrates custom messages with personality:

```yaml wrap
safe-outputs:
  add-comment:
    max: 1
  messages:
    footer: "> 📊 *Diagram rendered by [{workflow_name}]({run_url})*"
    run-started: "📐 Archie here! [{workflow_name}]({run_url}) is sketching the architecture on this {event_type}..."
    run-success: "🎨 Blueprint complete! [{workflow_name}]({run_url}) has visualized the connections. The architecture speaks for itself! ✅"
    run-failure: "📐 Drafting interrupted! [{workflow_name}]({run_url}) {status}. The diagram remains incomplete..."
```

This example shows:
- **Emoji usage**: Adding 📐, 🎨, 📊, ✅ for visual appeal
- **Markdown links**: `[{workflow_name}]({run_url})` creates clickable workflow links
- **Italic text**: `*Diagram rendered by...*` for emphasis
- **Placeholder interpolation**: `{event_type}` and `{status}` for dynamic content

### Available Templates

| Template | Description | Use Case |
|----------|-------------|----------|
| `footer` | Appended to AI-generated content | Attribution on issues, PRs, comments |
| `footer-install` | Installation instructions | Add install command to footer |
| `staged-title` | Preview title | Staged mode preview header |
| `staged-description` | Preview description | Staged mode preview body |
| `run-started` | Activation comment | Notify when workflow starts |
| `run-success` | Success comment | Notify on successful completion |
| `run-failure` | Failure comment | Notify when workflow fails |

### Template Variables

| Variable | Description | Available In |
|----------|-------------|--------------|
| `{workflow_name}` | Workflow name from frontmatter | All templates |
| `{run_url}` | GitHub Actions run URL | All templates |
| `{triggering_number}` | Issue, PR, or discussion number | All templates |
| `{workflow_source}` | Repository path (owner/repo/path@ref) | `footer`, `footer-install` |
| `{workflow_source_url}` | GitHub URL to workflow source | `footer`, `footer-install` |
| `{event_type}` | Event type description (issue, pull request, discussion, etc.) | `run-started` |
| `{status}` | Workflow status (failed, cancelled, timed out) | `run-failure` |
| `{operation}` | Safe output operation name | `staged-title`, `staged-description` |

### Markdown Formatting

Messages support standard GitHub Markdown:

```yaml wrap
safe-outputs:
  messages:
    # Bold and italic
    footer: "> **Generated** by *[{workflow_name}]({run_url})*"
    
    # Blockquotes
    run-started: "> 🤖 Processing [{workflow_name}]({run_url})..."
    
    # Links with placeholders
    run-success: "✅ [View run details]({run_url})"
```

### URL Generation

Template variables generate valid URLs automatically:

- `{run_url}` → `https://github.com/owner/repo/actions/runs/123456789`
- `{workflow_source_url}` → `https://github.com/owner/repo/blob/main/.github/workflows/workflow.md`

Combine with Markdown link syntax for clickable links:
```yaml
footer: "> [View workflow source]({workflow_source_url})"
```

Custom messages can be imported from shared workflows. Local messages override imported ones.

## Related Documentation

- [Threat Detection Guide](/gh-aw/guides/threat-detection/) - Complete threat detection documentation and examples
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/setup/cli/) - CLI commands for workflow management
