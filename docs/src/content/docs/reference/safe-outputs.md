---
title: Safe Output Processing
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 5
---

One of the primary security features of GitHub Agentic Workflows is "safe output processing", enabling the creation of GitHub issues, comments, pull requests, and other outputs without giving the agentic portion of the workflow write permissions.

The `safe-outputs:` element declares automated actions based on workflow output. The agentic part runs with read-only permissions and writes to special files, then compiler-generated jobs with write permissions process these outputs to create issues, comments, PRs, or add labels.

**Cross-Repository Support:**
Many safe output types support the `target-repo` configuration in format `"owner/repository"` to create outputs in different repositories (requires appropriate token permissions).

**Basic Example:**
```yaml
safe-outputs:
  create-issue:  # Creates at most one issue
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

**Configuration:**
```yaml
safe-outputs:
  create-issue:
    title-prefix: "[ai] "            # Optional: prefix for issue titles
    labels: [automation, agentic]    # Optional: labels to attach
    max: 5                           # Optional: maximum count (default: 1)
    target-repo: "owner/target-repo" # Optional: cross-repository support
```

**Usage Example:**
```aw wrap
# Code Analysis Agent
Analyze the latest commit. Create issues with findings, each with a title starting with "AI Code Analysis" and detailed description.
```

### Issue Comment Creation (`add-comment:`)

Posts comments on issues or pull requests. Defaults to the triggering issue/PR.

**Configuration:**
```yaml
safe-outputs:
  add-comment:
    max: 3                          # Optional: maximum comments (default: 1)
    target: "*"                     # Optional: "triggering" (default), "*" (any issue), or specific number
    target-repo: "owner/target-repo" # Optional: cross-repository support
```

**Usage Example:**
```aw wrap
---
on:
  issues:
    types: [opened, edited]
engine: claude
safe-outputs:
  add-comment:
    max: 3
---
# Issue/PR Analysis Agent
Analyze the issue and create comments with specific insights about different aspects.
```

### Add Issue Label (`add-labels:`)

Adds labels to issues or pull requests based on analysis. Defaults to the triggering issue/PR.

**Configuration:**
```yaml
safe-outputs:
  add-labels:
    allowed: [triage, bug, enhancement] # Optional: allowed labels
    max: 3                              # Optional: maximum count (default: 3)
    target: "*"                         # Optional: "triggering" (default), "*" (any issue), or specific number
    target-repo: "owner/target-repo"    # Optional: cross-repository support
```

**Usage Example:**
```aw wrap
# Issue Labeling Agent
Analyze the issue content and add appropriate labels.
```

**Safety:** Only `issues.addLabels` API is used (no removal). If `allowed` is set, all labels must be in the list or the job fails.

### Issue Updates (`update-issue:`)

Updates GitHub issues based on analysis. Only explicitly enabled fields can be updated. Defaults to the triggering issue.

**Configuration:**
```yaml
safe-outputs:
  update-issue:
    status:                          # Optional: enables status updates (open/closed)
    title:                           # Optional: enables title updates
    body:                            # Optional: enables body updates
    max: 3                           # Optional: maximum updates (default: 1)
    target: "*"                      # Optional: "triggering" (default), "*" (any issue), or specific number
    target-repo: "owner/target-repo" # Optional: cross-repository support
```

**Usage Example:**
```aw wrap
---
on:
  issues:
    types: [opened, edited]
engine: claude
safe-outputs:
  update-issue:
    status: true
    title: true
    body: true
---
# Issue Update Agent
Analyze and update the issue's status, title, or body as needed.
```

**Safety:** Status validated as "open" or "closed", empty values rejected, uses only `issues.update` API.

### Pull Request Creation (`create-pull-request:`)

Creates pull requests with code changes. Automatically falls back to creating an issue if PR creation fails (e.g., organization settings blocking Actions from creating PRs).

> [!NOTE]
> PR creation requires enabling "Allow GitHub Actions to create and approve pull requests" in Organization Settings → Actions → General → Workflow permissions.

**Configuration:**
```yaml
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "            # Optional: prefix for PR titles
    labels: [automation, agentic]    # Optional: labels to attach
    draft: true                      # Optional: create as draft (default: true)
    if-no-changes: "warn"            # Optional: "warn" (default), "error", or "ignore"
    target-repo: "owner/target-repo" # Optional: cross-repository support
```

**Fallback Behavior:**
When PR creation fails, the system creates an issue instead with the same title, description, labels, branch link, and error details. Outputs include `fallback_used: "true"`.

**Outputs:**
- **On success:** `pull_request_number`, `pull_request_url`, `branch_name`
- **On fallback:** `issue_number`, `issue_url`, `branch_name`, `fallback_used`

**Usage Example:**
```aw wrap
---
on:
  push:
    branches: [main]
engine: claude
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "
    labels: [automation]
    draft: true
---
# Code Improvement Agent
Analyze the latest commit and suggest improvements. Make file changes and create a pull request with descriptive title and details.
```

**Troubleshooting:** If fallback consistently occurs, check Organization/Repository Settings → Actions → General → Workflow permissions for "Allow GitHub Actions to create and approve pull requests".

### Pull Request Review Comment Creation (`create-pull-request-review-comment:`)

Creates line-specific review comments on pull requests. Defaults to the triggering PR.

**Configuration:**
```yaml
safe-outputs:
  create-pull-request-review-comment:
    max: 3                          # Optional: maximum comments (default: 1)
    side: "RIGHT"                   # Optional: "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"                     # Optional: "triggering" (default), "*" (any PR), or specific number
    target-repo: "owner/target-repo" # Optional: cross-repository support
```

**Usage Example:**
```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
engine: claude
safe-outputs:
  create-pull-request-review-comment:
    max: 3
---
# Code Review Agent
Analyze PR changes and create review comments. Specify file path, line number, and feedback for each comment. Supports single-line and multi-line comments.
```

**Output Fields:**
- `path`: File path (required)
- `line`: Line number (required)
- `start_line`: Starting line for multi-line comments (optional)
- `side`: "LEFT" or "RIGHT" (optional)
- `pull_request_number`: Required when using `target: "*"`
- `body`: Comment content (required)

### Code Scanning Alert Creation (`create-code-scanning-alert:`)

Creates repository security advisories in SARIF format and uploads to GitHub Code Scanning.

**Configuration:**
```yaml
safe-outputs:
  create-code-scanning-alert:
    max: 50  # Optional: maximum findings (default: unlimited)
```

**Usage Example:**
```aw wrap
# Security Analysis Agent
Analyze for vulnerabilities. Create security advisories specifying file path, line number, severity (error/warning/info/note), and description.
```

**Output Fields:**
- `file`: File path (required)
- `line`: Line number (required)
- `severity`: "error", "warning", "info", or "note" (required)
- `message`: Security issue description (required)
- `column`: Column number (optional, default: 1)
- `ruleIdSuffix`: Custom SARIF rule ID suffix (optional, alphanumeric/hyphens/underscores only)

**Features:** Generates SARIF reports, uploads as artifacts, integrates with GitHub Code Scanning dashboard, works in any workflow context.

### Push to Pull Request Branch (`push-to-pull-request-branch:`)

Pushes changes to a pull request branch. Defaults to the triggering PR.

**Configuration:**
```yaml
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                      # Optional: "triggering" (default), "*" (any PR), or specific number
    title-prefix: "[bot] "           # Optional: required title prefix for validation
    labels: [automated, enhancement] # Optional: required labels for validation
    if-no-changes: "warn"            # Optional: "warn" (default), "error", or "ignore"
```

**Usage Example:**
```aw wrap
---
on:
  pull_request:
    types: [opened, synchronize]
engine: claude
safe-outputs:
  push-to-pull-request-branch:
    if-no-changes: "warn"
---
# Code Update Agent
Analyze the PR and make improvements. Make file changes and push to the feature branch with a descriptive commit message.
```

**Validation:** When `title-prefix` or `labels` are specified, the workflow validates the PR before pushing. Validation occurs before changes are applied.

**Safety:** Changes applied via git patches, only specified branch modified, one push per execution.

### Missing Tool Reporting (`missing-tool:`)

**Enabled by default** when `safe-outputs:` is configured. Reports missing tools or permission errors.

**Configuration:**
```yaml
safe-outputs:
  create-issue:           # Enables missing-tool by default
  missing-tool: false     # Explicitly disable if needed
```

```yaml
safe-outputs:
  missing-tool:
    max: 10  # Optional: maximum reports (default: unlimited)
```

**Automatic Detection:** The workflow engine scans logs for permission errors and creates reports for tools that lacked permissions, failed API calls, or blocked operations.

**Safety:** No write permissions required, structured reports with tool name/reason/alternatives, data captured in artifacts.

### New Discussion Creation (`create-discussion:`)

Creates GitHub discussions based on workflow output.

**Configuration:**
```yaml
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "            # Optional: prefix for titles
    category: "DIC_kwDOGFsHUM4BsUn3"  # Optional: category ID, name, or slug (defaults to first available)
    max: 3                           # Optional: maximum count (default: 1)
    target-repo: "owner/target-repo" # Optional: cross-repository support
```

**Usage Example:**
```aw wrap
# Research Discussion Agent
Research latest AI developments. Create discussions with findings, each with title starting "AI Research Update" and detailed summary.
```

**Note:** `category` accepts ID, name, or slug. Matches in order: ID → name → slug. Uses first available category if unspecified.

## Cross-Repository Operations

Many safe output types support `target-repo` configuration to create outputs in different repositories using proper authentication.

**Authentication:** Standard `GITHUB_TOKEN` only works for the workflow's repository. Use a PAT or GitHub App token via the `github-token` field or `GH_AW_GITHUB_TOKEN` environment variable.

**Example:**
```yaml
---
on:
  issues:
    types: [opened]
engine: claude
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
    title-prefix: "[Cross-Repo] "
  add-comment:
    target-repo: "org/notifications-repo"
    target: "123"
---
# Multi-Repository Issue Processor
Create tracking issue in tracking-repo and comment on issue #123 in notifications-repo.
```

**Security:** Use specific repository names (no wildcards), grant minimum permissions, scope PATs to specific repositories when possible.

**Error Handling:** Failed operations show clear error messages with repository and permission details. In staged mode, errors appear as preview issues.

## Automatically Added Tools

When `create-pull-request` or `push-to-pull-request-branch` are configured, file editing tools (Edit, MultiEdit, Write, NotebookEdit) and Git commands (`git checkout:*`, `git branch:*`, `git switch:*`, `git add:*`, `git rm:*`, `git commit:*`, `git merge:*`) are automatically enabled.

## Security and Sanitization

All agent output is automatically sanitized: XML characters escaped, only HTTPS URIs allowed (others redacted), URIs checked against `allowed-domains` (defaults to GitHub domains: `github.com`, `github.io`, `githubusercontent.com`, `githubassets.com`, `github.dev`, `codespaces.new`), content truncated at 0.5MB/65,000 lines, control characters removed.

**Configuration:**
```yaml
safe-outputs:
  allowed-domains: [github.com, api.github.com, trusted-domain.com]  # Additional trusted domains
```

## Global Configuration Options

### Custom GitHub Token (`github-token:`)

Token precedence: `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN`. Override with custom PAT for enhanced permissions, cross-repository operations, or bypassing restrictions.

**Global:**
```yaml
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}
  create-issue:
```

**Per-output:**
```yaml
safe-outputs:
  create-pull-request:
    github-token: ${{ secrets.CUSTOM_PAT }}  # Only for PRs
```

### Maximum Patch Size (`max-patch-size:`)

Configures maximum git patch size (1-10,240 KB, default: 1024 KB). Job fails with clear error if exceeded.

```yaml
safe-outputs:
  max-patch-size: 512  # 512 KB limit
  create-pull-request:
```

**Use Cases:** Prevent repository bloat, avoid API limits/timeouts, ensure manageable reviews, control resource usage.

## Custom Runner Image

Specify `runs-on` for custom runner image for all safe output jobs.

```yaml
safe-outputs:
  runs-on: ubuntu-22.04
  create-issue:
```

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
