---
title: Safe Output Processing
description: Learn about safe output processing features that enable creating GitHub issues, comments, and pull requests without giving workflows write permissions.
sidebar:
  order: 5
---

Safe output processing enables creating GitHub issues, comments, pull requests, and other outputs without giving the agentic portion of the workflow write permissions. The `safe-outputs:` element declares automated actions based on workflow output.

The agentic part runs with read-only permissions and writes to special files. The compiler generates additional jobs that read this output and perform the actions with appropriate write permissions.

**Cross-Repository Support:**
Most safe output types support the `target-repo` configuration for operations in other repositories. Use format `"owner/repository"`:

```yaml
safe-outputs:
  create-issue:
    target-repo: "owner/target-repository"  # Optional: defaults to workflow repository
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
  create-issue:
    title-prefix: "[ai] "            # Optional: prefix for issue titles
    labels: [automation, agentic]    # Optional: labels to attach to issues
    max: 5                           # Optional: maximum (default: 1)
    target-repo: "owner/target-repo" # Optional: different repository
```

The workflow prompts the agent to describe issues and write details to a special file.

### Issue Comment Creation (`add-comment:`)

Posts comments on issues or pull requests. Defaults to the triggering issue/PR.

```yaml
safe-outputs:
  add-comment:
    max: 3                          # Optional: maximum (default: 1)
    target: "*"                     # Optional: "triggering" (default), "*" (any issue with issue_number), or explicit number
    target-repo: "owner/target-repo" # Optional: different repository
```

### Add Issue Label (`add-labels:`)

Adds labels to issues or pull requests based on agent analysis. Defaults to the triggering issue/PR.

```yaml
safe-outputs:
  add-labels:
    allowed: [triage, bug, enhancement] # Optional: restrict to specific labels
    max: 3                              # Optional: maximum (default: 3)
    target: "*"                         # Optional: "triggering" (default), "*" (any issue with issue_number), or explicit number
    target-repo: "owner/target-repo"    # Optional: different repository
```

The agent writes labels to a special file, one per line. If `allowed` is specified, all labels must be in that list.

### Issue Updates (`update-issue:`)

Updates GitHub issues status, title, or body. Defaults to the triggering issue. Only explicitly enabled fields can be updated.

```yaml
safe-outputs:
  update-issue:
    status:                             # Optional: enable status updates (open/closed)
    title:                              # Optional: enable title updates
    body:                               # Optional: enable body updates
    max: 3                              # Optional: maximum (default: 1)
    target: "*"                         # Optional: "triggering" (default), "*" (any issue with issue_number), or explicit number
    target-repo: "owner/target-repo"    # Optional: different repository
```

### Pull Request Creation (`create-pull-request:`)

Creates pull requests with code changes. Falls back to creating an issue if PR creation is blocked by organization settings.

> [!NOTE]
> Pull request creation may fail in organizations where "Allow GitHub Actions to create and approve pull requests" is disabled (Settings → Actions → General → Workflow permissions). Organization administrators can enable this setting.

```yaml
safe-outputs:
  create-pull-request:
    title-prefix: "[ai] "            # Optional: prefix for PR titles
    labels: [automation, agentic]    # Optional: labels to attach to PRs
    draft: true                      # Optional: create as draft (default: true)
    if-no-changes: "warn"            # Optional: "warn" (default), "error", or "ignore"
    target-repo: "owner/target-repo" # Optional: different repository
```

**Fallback Behavior:**
When PR creation fails, the system creates an issue with the PR title, description, labels, and branch link.

**Outputs:**
- Successful PR: `pull_request_number`, `pull_request_url`, `branch_name`
- Fallback: `issue_number`, `issue_url`, `branch_name`, `fallback_used: "true"`

**Troubleshooting:**
Check Organization Settings → Actions → General → Workflow permissions and enable "Allow GitHub Actions to create and approve pull requests".

### Pull Request Review Comment Creation (`create-pull-request-review-comment:`)

Creates line-specific review comments on pull requests. Supports single-line and multi-line comments.

```yaml
safe-outputs:
  create-pull-request-review-comment:
    max: 3                          # Optional: maximum (default: 1)
    side: "RIGHT"                   # Optional: "LEFT" or "RIGHT" (default: "RIGHT")
    target: "*"                     # Optional: "triggering" (default), "*" (any PR with pull_request_number), or explicit number
    target-repo: "owner/target-repo" # Optional: different repository
```

The agent writes comment details including `path`, `line`, optional `start_line`, and `body`.

### Code Scanning Alert Creation (`create-code-scanning-alert:`)

Creates repository security advisories in SARIF format and uploads to GitHub Code Scanning.

```yaml
safe-outputs:
  create-code-scanning-alert:
    max: 50                         # Optional: maximum findings (default: unlimited)
```

The agent writes security findings with `file`, `line`, optional `column`, `severity` (error/warning/info/note), `message`, and optional `ruleIdSuffix`.

### Push to Pull Request Branch (`push-to-pull-request-branch:`)

Pushes code changes directly to a PR branch. Supports validation to restrict which PRs can receive changes.

```yaml
safe-outputs:
  push-to-pull-request-branch:
    target: "*"                          # Optional: "triggering" (default), "*" (any PR with pull_request_number), or explicit number
    title-prefix: "[bot] "               # Optional: required title prefix for validation
    labels: [automated, enhancement]     # Optional: required labels for validation
    if-no-changes: "warn"                # Optional: "warn" (default), "error", or "ignore"
```

Changes are applied via git patches. Validation (if configured) occurs before applying changes.

### Missing Tool Reporting (`missing-tool:`)

**Enabled by default** when `safe-outputs:` is configured. Reports missing tools or insufficient permissions.

```yaml
safe-outputs:
  missing-tool:
    max: 10                             # Optional: maximum reports (default: unlimited)
  # Or explicitly disable:
  missing-tool: false
```

The system automatically detects permission errors and reports tools that lacked required permissions.

### New Discussion Creation (`create-discussion:`)

Creates GitHub discussions based on workflow output.

```yaml
safe-outputs:
  create-discussion:
    title-prefix: "[ai] "            # Optional: prefix for discussion titles
    category: "General"              # Optional: category ID, name, or slug (defaults to first available)
    max: 3                           # Optional: maximum (default: 1)
    target-repo: "owner/target-repo" # Optional: different repository
```

## Cross-Repository Operations

Cross-repository operations require authentication beyond the standard `GITHUB_TOKEN`. Use a Personal Access Token (PAT) or GitHub App token via the `github-token` field:

```yaml
safe-outputs:
  github-token: ${{ secrets.CROSS_REPO_PAT }}
  create-issue:
    target-repo: "org/tracking-repo"
```

**Security:** Use specific repository names (no wildcards) and grant minimum required permissions. Failed operations show clear error messages with permission requirements.

## Automatically Added Tools

When `create-pull-request` or `push-to-pull-request-branch` are configured, file editing tools (Edit, MultiEdit, Write, NotebookEdit) and git commands (checkout, branch, switch, add, rm, commit, merge) are automatically allowed.

## Security and Sanitization

Agent output is automatically sanitized: XML characters are escaped, only HTTPS URIs allowed (HTTP/FTP/javascript/etc. are redacted), domains are allowlisted (GitHub domains by default), content truncated at 0.5MB/65k lines, control characters stripped.

```yaml
safe-outputs:
  allowed-domains:                    # Optional: additional allowed domains (GitHub domains always included)
    - api.github.com
    - trusted-domain.com
```

## Global Configuration Options

### Custom GitHub Token (`github-token:`)

Default token precedence: `GH_AW_GITHUB_TOKEN` → `GITHUB_TOKEN`. Override globally or per-output:

```yaml
# Global override
safe-outputs:
  github-token: ${{ secrets.CUSTOM_PAT }}
  create-issue:

# Per-output override
safe-outputs:
  create-issue:
  create-pull-request:
    github-token: ${{ secrets.CUSTOM_PAT }}
```

Use custom tokens for cross-repository operations, enhanced permissions, or bypassing org restrictions on PR creation.

### Maximum Patch Size (`max-patch-size:`)

Configures maximum git patch size for `create-pull-request` or `push-to-pull-request-branch`:

```yaml
safe-outputs:
  max-patch-size: 512                   # Optional: KB (range: 1-10240, default: 1024)
  create-pull-request:
```

Jobs fail with clear errors when patches exceed the limit. Prevents repository bloat, API limits, and oversized code reviews.

## Custom Runner Image

```yaml
safe-outputs:
  runs-on: ubuntu-22.04                # Optional: custom runner for all safe output jobs
  create-issue:
```

## Related Documentation

- [Frontmatter Options](/gh-aw/reference/frontmatter/) - All configuration options for workflows
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Directory layout and organization
- [Command Triggers](/gh-aw/reference/command-triggers/) - Special /my-bot triggers and context text
- [CLI Commands](/gh-aw/tools/cli/) - CLI commands for workflow management
