---
title: Safe Outputs
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 800
---

One of the primary security features of GitHub Agentic Workflows is "safe output processing", enabling the creation of GitHub issues, comments, pull requests, and other outputs without giving the agentic portion of the workflow write permissions.

## Overview

The `safe-outputs:` frontmatter element enables workflows to create GitHub issues, comments, pull requests, and other outputs without granting write permissions to the agentic portion. The agentic workflow runs with read-only permissions and writes to special files, then automatically generated jobs process this output with appropriate write permissions.

Example:
```yaml
safe-outputs:
  create-issue:  # Creates at most one new issue
```

## Available Safe Output Types

| Output Type | Configuration Key | Description | Default Max | Cross-Repository |
|-------------|------------------|-------------|-------------|------------------|
| **Create Issue** | `create-issue:` | Create GitHub issues based on workflow output | 1 | ✅ |
| **Add Comments** | `add-comment:` | Post comments on issues, pull requests, or discussions | 1 | ✅ |
| **Update Issues** | `update-issue:` | Update issue status, title, or body | 1 | ✅ |
| **Add Issue Label** | `add-labels:` | Add labels to issues or pull requests | 3 | ✅ |
| **Create Pull Request** | `create-pull-request:` | Create pull requests with code changes | 1 | ✅ |
| **Pull Request Review Comments** | `create-pull-request-review-comment:` | Create review comments on specific lines of code | 1 | ✅ |
| **Create Discussions** | `create-discussion:` | Create GitHub discussions based on workflow output | 1 | ✅ |
| **Push to Pull Request Branch** | `push-to-pull-request-branch:` | Push changes directly to a branch | 1 | ❌ |
| **Create Code Scanning Alerts** | `create-code-scanning-alert:` | Generate SARIF repository security advisories and upload to GitHub Code Scanning | unlimited | ❌ |
| **Missing Tool Reporting** | `missing-tool:` | Report missing tools or functionality (enabled by default when safe-outputs is configured) | unlimited | ❌ |

Custom safe output types can be defined through [Custom Safe Output Jobs](/gh-aw/guides/custom-safe-outputs/).

### New Issue Creation (`create-issue:`)

Creates GitHub issues based on workflow output.

**Configuration:**
```yaml
safe-outputs:
  create-issue:
    title-prefix: "[ai] "            # Optional: prefix for issue titles
    labels: [automation, agentic]    # Optional: labels to attach to issues
    max: 5                           # Optional: maximum number of issues (default: 1)
    target-repo: "owner/target-repo" # Optional: cross-repository creation (requires github-token)
```

**Example workflow:**
```yaml
# Code Analysis Agent
Analyze the latest commit and provide insights.
Create new issues with your findings. For each issue, provide a title starting with "AI Code Analysis" and detailed description of the analysis findings.
```

### Comment Creation (`add-comment:`)

Posts comments on issues, pull requests, or discussions. Defaults to the triggering issue/PR.

**Configuration:**
```yaml
safe-outputs:
  add-comment:
    max: 3                          # Optional: maximum number of comments (default: 1)
    target: "*"                     # "triggering" (default), "*" (any issue/PR with number in output), or explicit number
    discussion: true                # Optional: target discussions instead of issues/PRs
    target-repo: "owner/target-repo" # Optional: cross-repository (requires github-token)
```

**Example workflow:**
```aw wrap
---
on:
  issues:
    types: [opened, edited]
safe-outputs:
  add-comment:
    max: 3
---
# Issue/PR Analysis Agent
Analyze the issue or pull request and provide feedback.
Create issue comments on the triggering issue or PR with your analysis findings.
```

### Add Issue Label (`add-labels:`)

Adds labels to issues or pull requests. Defaults to the triggering issue/PR.

**Configuration:**
```yaml
safe-outputs:
  add-labels:
    allowed: [triage, bug, enhancement] # Optional: restrict to specific labels
    max: 3                              # Optional: maximum number of labels (default: 3)
    target: "*"                         # "triggering" (default), "*" (any issue with number in output), or explicit number
    target-repo: "owner/target-repo"    # Optional: cross-repository (requires github-token)
```

**Safety:** Empty lines and lines starting with `-` are ignored. If `allowed` is specified, all labels must be in the allowed list or the job fails. Only uses GitHub's `issues.addLabels` API (no removal).

### Issue Updates (`update-issue:`)

Updates issue status, title, or body. Defaults to the triggering issue.

**Configuration:**
```yaml
safe-outputs:
  update-issue:
    status:                             # Optional: enable status updates (open/closed)
    title:                              # Optional: enable title updates
    body:                               # Optional: enable body updates
    target: "*"                         # "triggering" (default), "*" (any issue with number in output), or explicit number
    max: 3                              # Optional: maximum number of updates (default: 1)
    target-repo: "owner/target-repo"    # Optional: cross-repository (requires github-token)
```

**Example workflow:**
```aw wrap
---
on:
  issues:
    types: [opened, edited]
safe-outputs:
  update-issue:
    status: true
    title: true
    body: true
---
# Issue Update Agent
Analyze the issue and update its status, title, or body as needed.
```

**Safety:** Only explicitly enabled fields can be updated. Status values validated (must be "open" or "closed"). Empty/invalid values rejected.

### Pull Request Creation (`create-pull-request:`)

Creates a pull request with code changes. Falls back to creating an issue if PR creation is blocked by organization settings.

> [!NOTE]
> PR creation may fail when "Allow GitHub Actions to create and approve pull requests" is disabled in Organization Settings → Actions → General → Workflow permissions.

**Configuration:**
```yaml
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "            # Optional: prefix for PR titles
    labels: [automation, agentic]    # Optional: labels to attach
    draft: true                      # Optional: create as draft (default: true)
    if-no-changes: "warn"            # "warn" (default), "error", or "ignore"
    target-repo: "owner/target-repo" # Optional: cross-repository (requires github-token)
```

**Fallback Behavior:** When PR creation fails, automatically creates an issue with the same title, description, labels, branch link, and error details. Outputs include `fallback_used: "true"`.

**Outputs:**
- Success: `pull_request_number`, `pull_request_url`, `branch_name`
- Fallback: `issue_number`, `issue_url`, `branch_name`, `fallback_used`

**Example workflow:**
```aw wrap
---
on:
  push:
    branches: [main]
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "
    labels: [automation, code-improvement]
---
# Code Improvement Agent
Analyze the latest commit and suggest improvements.
1. Make file changes in the working directory
2. Create a pull request with a descriptive title and detailed description
```

### Pull Request Review Comment Creation (`create-pull-request-review-comment:`)

Creates line-specific review comments on pull requests.

**Configuration:**
```yaml
safe-outputs:
  create-pull-request-review-comment:
    max: 3                          # Optional: maximum comments (default: 1)
    side: "RIGHT"                   # Optional: "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"                     # "triggering" (default), "*" (any PR with number in output), or explicit number
    target-repo: "owner/target-repo" # Optional: cross-repository (requires github-token)
```

**Example workflow:**
```aw wrap
---
on:
  pull_request:
    types: [opened, edited, synchronize]
safe-outputs:
  create-pull-request-review-comment:
    max: 3
---
# Code Review Agent
Analyze the pull request changes and provide line-specific feedback.
For each comment, specify file path, line number (and optional start_line for multi-line), and feedback.
```

**Output structure:** `path`, `line`, `start_line` (optional), `side` (optional), `pull_request_number` (optional for `target: "*"`), `body`

### Code Scanning Alert Creation (`create-code-scanning-alert:`)

Creates SARIF security advisories and uploads them to GitHub Code Scanning.

**Configuration:**
```yaml
safe-outputs:
  create-code-scanning-alert:
    max: 50                         # Optional: maximum findings (default: unlimited)
```

**Example workflow:**
```aw wrap
# Security Analysis Agent
Analyze the codebase for security vulnerabilities.
For each finding, specify file path, line number, severity (error/warning/info/note), and detailed description.
```

**Output structure:** `file`, `line`, `column` (optional, defaults to 1), `severity`, `message`, `ruleIdSuffix` (optional). Rule IDs default to `{workflow-filename}-security-finding-{index}` format.

### Push to Pull Request Branch (`push-to-pull-request-branch:`)

Pushes changes directly to a pull request's branch.

**Configuration:**
```yaml
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                          # "triggering" (default), "*" (any PR with number in output), or explicit number
    title-prefix: "[bot] "               # Optional: required PR title prefix for validation
    labels: [automated, enhancement]     # Optional: required PR labels for validation (all must be present)
    if-no-changes: "warn"                # "warn" (default), "error", or "ignore"
```

**Example workflow:**
```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
safe-outputs:
  push-to-pull-request-branch:
    if-no-changes: "warn"
---
# Code Update Agent
Analyze the pull request and make necessary code improvements.
1. Make file changes directly in the working directory
2. Push changes with a descriptive commit message
```

**Safety:** PR validation (title-prefix/labels) occurs before applying changes. Changes applied via git patches. One push per workflow execution.

### Missing Tool Reporting (`missing-tool:`)

**Enabled by default** when `safe-outputs:` is configured. Reports missing tools, capabilities, or permission errors.

**Configuration:**
```yaml
safe-outputs:
  create-issue:              # Enables missing-tool by default
  missing-tool: false        # Explicitly disable
  # OR
  missing-tool:
    max: 10                  # Optional: limit reports (default: unlimited)
```

**Automatic Detection:** Scans logs for permission errors, failed API calls, and blocked operations. No manual reporting required.

### New Discussion Creation (`create-discussion:`)

Creates GitHub discussions based on workflow output.

**Configuration:**
```yaml
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "            # Optional: prefix for discussion titles
    category: "DIC_kwDOGFsHUM4BsUn3"  # Optional: category ID, name, or slug (defaults to first available)
    max: 3                           # Optional: maximum discussions (default: 1)
    target-repo: "owner/target-repo" # Optional: cross-repository (requires github-token)
```

**Example workflow:**
```yaml
# Research Discussion Agent
Research the latest developments in AI and create discussions to share findings.
```

## Cross-Repository Operations

Many safe outputs support `target-repo` for operations in other repositories. Requires a Personal Access Token (PAT) or GitHub App token via `github-token` field (standard `GITHUB_TOKEN` only works in the workflow's repository).

**Example:**
```yaml
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
  add-comment:
    target-repo: "org/notifications-repo"
    target: "123"
```

**Security:** Use explicit repository names (no wildcards). Grant tokens minimum necessary permissions. Errors show clear permission requirements.

## Automatically Added Tools

When `create-pull-request` or `push-to-pull-request-branch` are configured, automatically adds:
- **Claude tools**: Edit, MultiEdit, Write, NotebookEdit
- **Git commands**: `git checkout:*`, `git branch:*`, `git switch:*`, `git add:*`, `git rm:*`, `git commit:*`, `git merge:*`

## Security and Sanitization

All agent output is automatically sanitized:
- **XML escaping**: Special characters escaped
- **URI filtering**: Only HTTPS allowed; non-HTTPS replaced with "(redacted)"
- **Domain allowlisting**: Unlisted domains replaced with "(redacted)". Defaults to GitHub domains (`github.com`, `github.io`, `githubusercontent.com`, `githubassets.com`, `github.dev`, `codespaces.new`)
- **Size limits**: Truncated at 0.5MB or 65,000 lines
- **Control characters**: Stripped

**Configuration:**
```yaml
safe-outputs:
  allowed-domains: [github.com, api.github.com, trusted-domain.com]
```

## Global Configuration Options

### Custom GitHub Token (`github-token:`)

Default token precedence: `GH_AW_GITHUB_TOKEN` (highest priority) → `GITHUB_TOKEN` (fallback). Override globally or per-output:

```yaml
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}  # Global override
  create-issue:
  add-comment:
  create-pull-request:
    github-token: ${{ secrets.CUSTOM_PAT }}  # Per-output override
```

Use cases: Trial mode, enhanced permissions, cross-repository operations, bypassing token restrictions.

### Maximum Patch Size (`max-patch-size:`)

Limits git patch size for `create-pull-request` or `push-to-pull-request-branch` (range: 1-10,240 KB, default: 1024 KB):

```yaml
safe-outputs:
  max-patch-size: 512                   # KB (default: 1024)
  create-pull-request:
```

Job fails with clear error if patch exceeds limit. Use cases: prevent bloat, avoid API limits, ensure manageable reviews.

## Custom Runner Image

Specify custom GitHub Actions runner for all safe output jobs:

```yaml
safe-outputs:
  runs-on: ubuntu-22.04
  create-issue:
```

## Related Documentation

- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
