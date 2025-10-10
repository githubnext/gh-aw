---
title: Safe Output Processing
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 5
---

One of the primary security features of GitHub Agentic Workflows is "safe output processing", enabling the creation of GitHub issues, comments, pull requests, and other outputs without giving the agentic portion of the workflow write permissions.

The `safe-outputs:` element declares that your workflow should conclude with automated actions based on the workflow's output. The agentic part runs with read-only permissions and writes to special files, then generated jobs with appropriate write permissions process these outputs to perform the requested actions.

Many safe output types support cross-repository operations via the `target-repo: "owner/repository"` configuration, allowing workflows to create outputs in other repositories (requires appropriate token permissions).

Example:
```yaml
safe-outputs:
  create-issue:                      # Creates at most one new issue
    target-repo: "owner/other-repo"  # Optional: cross-repository support
```

## Available Safe Output Types

| Output Type | Configuration Key | Description | Default Max | Cross-Repository |
|-------------|------------------|-------------|-------------|------------------|
| **Create Issue** | `create-issue:` | Create GitHub issues based on workflow output | 1 | ✅ |
| **Add Issue Comments** | `add-comment:` | Post comments on issues or pull requests | 1 | ✅ |
| **Update Issues** | `update-issue:` | Update issue status, title, or body | 1 | ✅ |
| **Add Issue Label** | `add-labels:` | Add labels to issues or pull requests | 3 | ✅ |
| **Create Pull Request** | `create-pull-request:` | Create pull requests with code changes | 1 | ✅ |
| **Pull Request Review Comments** | `create-pull-request-review-comment:` | Create review comments on specific lines of code | 1 | ✅ |
| **Create Discussions** | `create-discussion:` | Create GitHub discussions based on workflow output | 1 | ✅ |
| **Push to Pull Request Branch** | `push-to-pull-request-branch:` | Push changes directly to a branch | 1 | ❌ |
| **Create Code Scanning Alerts** | `create-code-scanning-alert:` | Generate SARIF repository security advisories and upload to GitHub Code Scanning | unlimited | ❌ |
| **Missing Tool Reporting** | `missing-tool:` | Report missing tools or functionality (enabled by default when safe-outputs is configured) | unlimited | ❌ |

### New Issue Creation (`create-issue:`)

Creates GitHub issues based on workflow output.

```yaml
safe-outputs:
  create-issue:                      # Basic: creates 1 issue max
    title-prefix: "[ai] "            # Optional: prefix for issue titles
    labels: [automation, agentic]    # Optional: labels to attach
    max: 5                           # Optional: max issues (default: 1)
    target-repo: "owner/target-repo" # Optional: cross-repository
```

The workflow agent describes issues to create with titles and detailed descriptions. The compiler adds prompting to write issue details to special files.

### Issue Comment Creation (`add-comment:`)

Posts comments on issues or pull requests. By default, comments target the triggering issue/PR.

```yaml
safe-outputs:
  add-comment:                     # Basic: 1 comment on triggering issue/PR
    max: 3                         # Optional: max comments (default: 1)
    target: "*"                    # Optional: "triggering" (default), "*" (any issue, requires issue_number), or explicit number
    target-repo: "owner/other-repo" # Optional: cross-repository
```

The workflow agent describes comments to post. The compiler adds prompting to write comment content to special files.

### Add Issue Label (`add-labels:`)

Adds labels to issues or pull requests based on workflow analysis. By default, labels target the triggering issue/PR.

```yaml
safe-outputs:
  add-labels:                        # Basic: adds up to 3 labels to triggering issue/PR
    allowed: [triage, bug, feature]  # Optional: allowed labels (if omitted, any labels allowed)
    max: 3                           # Optional: max labels (default: 3)
    target: "*"                      # Optional: "triggering" (default), "*" (any issue, requires issue_number), or explicit number
    target-repo: "owner/other-repo"  # Optional: cross-repository
```

The workflow agent analyzes content and determines appropriate labels. Labels are written to a special file, one per line. If `allowed` is specified, all requested labels must be in the list or the job fails.

### Issue Updates (`update-issue:`)

Updates GitHub issue fields based on workflow analysis. By default, updates target the triggering issue. Only explicitly enabled fields can be updated.

```yaml
safe-outputs:
  update-issue:                      # Basic: updates 1 issue
    status:                          # Optional: enable status updates (open/closed)
    title:                           # Optional: enable title updates
    body:                            # Optional: enable body updates
    target: "*"                      # Optional: "triggering" (default), "*" (requires issue_number), or explicit number
    max: 3                           # Optional: max issues (default: 1)
    target-repo: "owner/other-repo"  # Optional: cross-repository
```

Status values are validated (must be "open" or "closed"). Empty or invalid field values are rejected.

### Pull Request Creation (`create-pull-request:`)

Creates a pull request containing code changes. Falls back to creating an issue if PR creation fails (e.g., when "Allow GitHub Actions to create and approve pull requests" is disabled in organization settings).

> [!NOTE]
> Check Settings → Actions → General → Workflow permissions to enable PR creation for agentic workflows.

```yaml
safe-outputs:
  create-pull-request:               # Creates exactly one PR
    title-prefix: "[ai] "            # Optional: prefix for PR titles
    labels: [automation, agentic]    # Optional: labels to attach
    draft: true                      # Optional: create as draft (default: true)
    if-no-changes: "warn"            # Optional: "warn" (default), "error", or "ignore"
    target-repo: "owner/other-repo"  # Optional: cross-repository
```

**Fallback Behavior:** When PR creation fails, creates an issue with the same title/description/labels, adds branch information, includes the error for debugging, and sets `fallback_used: "true"` output.

**Outputs:** On success: `pull_request_number`, `pull_request_url`, `branch_name`. On fallback: `issue_number`, `issue_url`, `branch_name`, `fallback_used`.

### Pull Request Review Comment Creation (`create-pull-request-review-comment:`)

Creates line-specific review comments on pull request code changes.

```yaml
safe-outputs:
  create-pull-request-review-comment: # Basic: 1 comment on triggering PR
    max: 3                            # Optional: max comments (default: 1)
    side: "RIGHT"                     # Optional: "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"                       # Optional: "triggering" (default), "*" (requires pull_request_number), or explicit number
    target-repo: "owner/other-repo"   # Optional: cross-repository
```

The workflow agent specifies file paths, line numbers (with optional start_line for multi-line comments), side of diff, and comment body. Supports single-line and multi-line code comments.

### Code Scanning Alert Creation (`create-code-scanning-alert:`)

Creates repository security advisories in SARIF format and uploads to GitHub Code Scanning.

```yaml
safe-outputs:
  create-code-scanning-alert:        # Basic: unlimited findings
    max: 50                          # Optional: max findings (default: unlimited)
```

The workflow agent specifies security findings with file paths, line numbers, optional column numbers, severity levels (error/warning/info/note), detailed descriptions, and optional ruleIdSuffix. Generates SARIF reports, uploads as artifacts, and integrates with GitHub Code Scanning dashboard.

### Push to Pull Request Branch (`push-to-pull-request-branch:`)

Pushes changes to a pull request branch. Applies changes via git patches.

```yaml
safe-outputs:
  push-to-pull-request-branch:       # Basic: push to triggering PR
    target: "*"                      # Optional: "triggering" (default), "*" (requires pull_request_number), or explicit number
    title-prefix: "[bot] "           # Optional: required PR title prefix for validation
    labels: [automated, enhancement] # Optional: required labels for validation (all must be present)
    if-no-changes: "warn"            # Optional: "warn" (default), "error", or "ignore"
```

When `title-prefix` or `labels` are specified, validates the PR before pushing. Validation failure stops the workflow with a clear error. Push operations are limited to one per workflow execution.

### Missing Tool Reporting (`missing-tool:`)

**Enabled by default** when any safe-output is configured. Reports missing tools, functionality, or permission errors.

```yaml
safe-outputs:
  create-issue:                      # missing-tool enabled by default
  missing-tool: false                # Explicitly disable if needed

# Or with configuration:
safe-outputs:
  missing-tool:
    max: 10                          # Optional: max reports (default: unlimited)
```

The workflow engine automatically scans logs for permission errors and creates missing-tool entries for failed tools, insufficient API authorization, and blocked operations. Reports include tool name, reason, and optional alternatives.

### New Discussion Creation (`create-discussion:`)

Creates GitHub discussions based on workflow output.

```yaml
safe-outputs:
  create-discussion:                 # Basic: creates 1 discussion
    title-prefix: "[ai] "            # Optional: prefix for titles
    category: "General"              # Optional: category ID, name, or slug (defaults to first available)
    max: 3                           # Optional: max discussions (default: 1)
    target-repo: "owner/other-repo"  # Optional: cross-repository
```

The workflow agent describes discussions with titles and detailed content. The `category` field accepts IDs (e.g., `"DIC_kwDOGFsHUM4BsUn3"`), names (e.g., `"General"`), or slugs (e.g., `"general"`).

## Cross-Repository Operations

Most safe output types support `target-repo: "owner/repository"` for cross-repository operations. Requires a Personal Access Token (PAT) or GitHub App token configured via `github-token` field or `GH_AW_GITHUB_TOKEN` environment variable (standard `GITHUB_TOKEN` only works in the current repository).

```yaml
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
  add-comment:
    target-repo: "org/notifications-repo"
    target: "123"
```

Use specific repository names (no wildcards), grant minimum required permissions, and scope PATs to specific repositories when possible. Failed operations show clear error messages with permission requirements.

## Automatically Added Tools

When `create-pull-request` or `push-to-pull-request-branch` are configured, file editing tools (Edit, MultiEdit, Write, NotebookEdit) and Git commands (`git checkout:*`, `git branch:*`, `git switch:*`, `git add:*`, `git rm:*`, `git commit:*`, `git merge:*`) are automatically added.

## Security and Sanitization

All workflow output is automatically sanitized: XML characters escaped, only HTTPS URIs allowed (non-HTTPS replaced with "(redacted)"), HTTPS URIs checked against domain allowlist, content truncated if exceeding 0.5MB or 65,000 lines, and control characters removed.

Default allowed domains (when `allowed-domains` not specified): `github.com`, `github.io`, `githubusercontent.com`, `githubassets.com`, `github.dev`, `codespaces.new`.

```yaml
safe-outputs:
  allowed-domains: [github.com, api.github.com, trusted.com]  # Additional domains beyond defaults
```

## Global Configuration Options

### Custom GitHub Token (`github-token:`)

Token precedence: `GH_AW_GITHUB_TOKEN` (highest priority) > `GITHUB_TOKEN` (fallback). Override with custom token for enhanced permissions, cross-repository operations, or bypassing restrictions.

```yaml
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}  # Global: applies to all safe outputs
  create-issue:

  # Or per-output:
  create-pull-request:
    github-token: ${{ secrets.PR_PAT }}    # Specific: only for this output
```

### Maximum Patch Size (`max-patch-size:`)

Configures maximum git patch size (1-10,240 KB range, default 1024 KB). Job fails with clear error if exceeded.

```yaml
safe-outputs:
  max-patch-size: 512                # 512 KB limit
  create-pull-request:
```

Prevents repository bloat, API limits/timeouts, and ensures manageable review sizes.

## Custom Runner Image

```yaml
safe-outputs:
  runs-on: ubuntu-22.04              # Custom runner for all safe output jobs
  create-issue:
```

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
