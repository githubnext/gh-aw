---
title: Permissions
description: Configure GitHub Actions permissions for agentic workflows
sidebar:
  order: 500
---

The `permissions:` section controls what GitHub API operations your workflow can perform. GitHub Agentic Workflows uses read-only permissions by default for security, with write operations handled through [safe outputs](/gh-aw/reference/safe-outputs/).

```yaml wrap
permissions:
  contents: read
  issues: write
  pull-requests: write
```

## Permission Model

### Security-First Design

Agentic workflows follow a principle of least privilege:

- **Read-only by default**: Workflows run with minimal permissions
- **Write through safe outputs**: Write operations happen in separate jobs with sanitized content
- **Explicit permissions**: All permissions must be declared in frontmatter

This model prevents AI agents from accidentally or maliciously modifying repository content during execution.

### Permission Scopes

GitHub Actions permissions control access to different GitHub resources:

| Permission | Read Access | Write Access |
|------------|-------------|--------------|
| `contents` | Read repository code | Push code, create releases |
| `issues` | Read issues | Create/edit issues, add comments |
| `pull-requests` | Read pull requests | Create/edit PRs, add reviews |
| `discussions` | Read discussions | Create/edit discussions |
| `actions` | Read workflow runs | Cancel runs, approve deployments |
| `checks` | Read check runs | Create status checks |
| `deployments` | Read deployments | Create deployments |
| `packages` | Read packages | Publish packages |
| `pages` | Read Pages settings | Deploy to GitHub Pages |
| `statuses` | Read commit statuses | Create commit statuses |

See [GitHub's permissions reference](https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs) for the complete list.

## Configuration

### Basic Configuration

Specify individual permission levels:

```yaml wrap
permissions:
  contents: read
  issues: write
```

### Read-All Permissions

Grant read access to all scopes:

```yaml wrap
permissions: read-all
```

Equivalent to setting all permissions to `read`. This is useful for workflows that need to inspect various repository data without making changes.

### Write-All Permissions (Not Recommended)

:::caution
Avoid `write-all` in agentic workflows. Use specific permissions with safe outputs instead.
:::

```yaml wrap
permissions: write-all
```

This grants write access to all scopes and should only be used when absolutely necessary, such as for administrative automation tasks with strict access controls.

### No Permissions

Disable all permissions:

```yaml wrap
permissions: {}
```

Useful for workflows that only perform computation without accessing GitHub APIs.

## Common Patterns

### IssueOps Workflow

Read repository content, write to issues:

```yaml wrap
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
safe-outputs:
  add-comment:
    max: 5
```

The main AI job runs with `contents: read`. Comment creation happens in a separate safe output job with `issues: write`, ensuring AI-generated content is sanitized before posting.

### PR Review Workflow

Read pull requests, add review comments:

```yaml wrap
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  pull-requests: write
safe-outputs:
  create-pr-review-comment:
    max: 10
```

### Scheduled Analysis

Read-only analysis that creates issues:

```yaml wrap
on:
  schedule:
    - cron: "0 9 * * 1"
permissions:
  contents: read
  issues: write
safe-outputs:
  create-issue:
    max: 3
```

### Manual Workflow

Maximum permissions for administrative tasks:

```yaml wrap
on:
  workflow_dispatch:
permissions: read-all
manual-approval: production
```

Uses manual approval gate for human oversight before execution.

## Safe Outputs

Write operations should use safe outputs rather than direct API access:

```yaml wrap
permissions:
  contents: read  # AI job runs read-only
safe-outputs:
  add-comment:
    max: 5        # Separate job with issues: write
  create-issue:
    max: 3        # Separate job with issues: write
```

**Benefits:**
- Content sanitization (removes unsafe content, @mentions)
- Rate limiting (max outputs per run)
- Audit trail (outputs shown in step summary)
- Security isolation (write permissions separated from AI execution)

See [Safe Outputs](/gh-aw/reference/safe-outputs/) for complete documentation.

## Permission Validation

The compiler validates permissions during compilation:

```bash
gh aw compile workflow.md
```

**Common validation errors:**
- Undefined permissions (use explicit permission levels)
- Write permissions without safe outputs (security risk)
- Insufficient permissions for declared tools

Use `--strict` mode for additional permission validation:

```bash
gh aw compile --strict workflow.md
```

Strict mode refuses write permissions and requires explicit network configuration for all operations.

## Related Documentation

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure write operations with content sanitization
- [Security Guide](/gh-aw/guides/security/) - Security best practices and permission strategies
- [Tools](/gh-aw/reference/tools/) - GitHub API tools and their permission requirements
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration reference
