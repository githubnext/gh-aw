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

Key permissions include `contents` (code access), `issues` (issue management), `pull-requests` (PR management), `discussions`, `actions` (workflow control), `checks`, `deployments`, `packages`, `pages`, and `statuses`. Each has read and write levels. See [GitHub's permissions reference](https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs) for the complete list.

## Configuration

### Basic Configuration

Specify individual permission levels:

```yaml wrap
permissions:
  contents: read
  issues: write
```

### Shorthand Options

- **`read-all`**: Read access to all scopes (useful for inspection workflows)
- **`write-all`**: Write access to all scopes (avoid in agentic workflows; use specific permissions with safe outputs)
- **`{}`**: No permissions (for computation-only workflows)

## Common Patterns

Most workflows follow a similar pattern: read-only permissions for the AI job, with write operations handled through safe outputs:

```yaml wrap
# IssueOps: Read code, write to issues
permissions:
  contents: read
  issues: write
safe-outputs:
  add-comment:
    max: 5

# PR Review: Read code, write to PRs
permissions:
  contents: read
  pull-requests: write
safe-outputs:
  create-pr-review-comment:
    max: 10

# Scheduled: Analysis with issue creation
permissions:
  contents: read
  issues: write
safe-outputs:
  create-issue:
    max: 3

# Manual: Admin tasks with approval gate
permissions: read-all
manual-approval: production
```

## Safe Outputs

Write operations use safe outputs instead of direct API access. This provides content sanitization, rate limiting, audit trails, and security isolation by separating write permissions from AI execution. See [Safe Outputs](/gh-aw/reference/safe-outputs/) for details.

## Permission Validation

Run `gh aw compile workflow.md` to validate permissions. Common errors include undefined permissions, write permissions without safe outputs, and insufficient permissions for declared tools. Use `--strict` mode to refuse write permissions and require explicit network configuration.

## Related Documentation

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure write operations with content sanitization
- [Security Guide](/gh-aw/guides/security/) - Security best practices and permission strategies
- [Tools](/gh-aw/reference/tools/) - GitHub API tools and their permission requirements
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration reference
