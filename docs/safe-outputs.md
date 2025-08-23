# 🔒 Safe Output Processing

This guide covers safe output processing configuration for agentic workflows, enabling automatic creation of GitHub issues, comments, pull requests, and label addition based on the agent's output without giving the agentic portion of the workflow write permissions.

## Overview (`output:`)

Configure automatic safe output processing from agentic workflow results. This enables simple, declarative, automatic creation of GitHub issues, comments, pull requests, and label addition based on the agent's output, without giving the agentic portion of the workflow write permissions.

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

## Security and Sanitization

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

## Issue Creation (`output.issue`)

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

## Issue Comment Creation (`output.issue_comment`)

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

## Pull Request Creation (`output.pull-request`)

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

## Label Addition (`output.labels`)

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

## Related Documentation

- [Frontmatter Options](frontmatter.md) - All configuration options for workflows
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Alias Triggers](alias-triggers.md) - Special @mention triggers and context text
- [Commands](commands.md) - CLI commands for workflow management
