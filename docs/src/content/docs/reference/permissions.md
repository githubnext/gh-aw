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
  actions: read
safe-outputs:
  create-issue:
  add-comment:
```

## Permission Model

Agentic workflows follow a principle of least privilege: the main job runs read-only, and all write operations happen in separate [safe outputs](/gh-aw/reference/safe-outputs/) jobs with sanitized content.

This separation provides an audit trail, limits blast radius if an agent misbehaves, supports compliance approval gates, and defends against prompt injection. Safe outputs add one extra job but provide critical safety guarantees.

## Permission Scopes

Key permissions include `contents` (code access), `issues` (issue management), `pull-requests` (PR management), `discussions`, `actions` (workflow control), `checks`, `deployments`, `packages`, `pages`, and `statuses`. Each has read and write levels. See [GitHub's permissions reference](https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs) for the complete list.

### Special Permission: `id-token`

The `id-token` permission controls access to GitHub's OIDC token service for [OpenID Connect (OIDC) authentication](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect) with cloud providers (AWS, GCP, Azure).

The only valid values are `write` and `none`. `id-token: read` is not a valid permission and will be rejected at compile time.

Unlike other write permissions, `id-token: write` does not grant any ability to modify repository content. It only allows the workflow to request a short-lived OIDC token from GitHub's token service for authentication with external cloud providers.

```yaml wrap
# Example: Deploy to AWS using OIDC authentication
permissions:
  id-token: write      # Allowed for OIDC authentication
  contents: read       # Read repository code
```

This permission is safe to use and does not require safe-outputs, even in strict mode.

### GitHub App-Only Permissions

Certain permission scopes cannot be granted to `GITHUB_TOKEN` and are forwarded instead as inputs to [`actions/create-github-app-token`](https://github.com/actions/create-github-app-token) when a GitHub App is configured. These scopes are omitted from the compiled workflow's `permissions:` block.

**Repository-level:** `administration`, `environments`, `git-signing`, `vulnerability-alerts`, `workflows`, `repository-hooks`, `single-file`, `codespaces`, `repository-custom-properties`

**Organization-level:** `organization-projects`, `members`, `organization-administration`, `team-discussions`, `organization-hooks`, `organization-members`, `organization-packages`, `organization-self-hosted-runners`, `organization-custom-org-roles`, `organization-custom-properties`, `organization-custom-repository-roles`, `organization-announcement-banners`, `organization-events`, `organization-plan`, `organization-user-blocking`, `organization-personal-access-token-requests`, `organization-personal-access-tokens`, `organization-copilot`, `organization-codespaces`

**User-level:** `email-addresses`, `codespaces-lifecycle-admin`, `codespaces-metadata`

These scopes must always be declared as `read`. Declaring `write` is a compile error; write operations through a GitHub App must go through [safe outputs](/gh-aw/reference/safe-outputs/), which provide a separate sanitized job for write operations.

Declaring any of these scopes without a configured `github-app` causes a compile error. The GitHub App can be configured in `tools.github.github-app`, `safe-outputs.github-app`, or the top-level `github-app:` field — see [Tools](/gh-aw/reference/tools/) for configuration details.

```aw wrap
permissions:
  contents: read
  workflows: read          # GitHub App-only scope
  members: read            # GitHub App-only scope
tools:
  github:
    github-app:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
```

> [!NOTE]
> Shorthand permissions (`read-all`, `write-all`, `all: read`) do not trigger the "GitHub App required" validation.

## Configuration

Specify individual permission levels:

```yaml wrap
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
```

### Shorthand Options

- **`read-all`**: Read access to all scopes (useful for inspection workflows)
- **`{}`**: No permissions (for computation-only workflows)

> [!CAUTION]
> Avoid using `write-all` or direct write permissions in agentic workflows. Use [safe outputs](/gh-aw/reference/safe-outputs/) instead for secure write operations.

## Common Patterns

All workflows should use read-only permissions with safe outputs for write operations:

```yaml wrap
# IssueOps: Read code, comment via safe outputs
permissions:
  contents: read
  actions: read
safe-outputs:
  add-comment:
    max: 5

# PR Review: Read code, review via safe outputs
permissions:
  contents: read
  actions: read
safe-outputs:
  create-pr-review-comment:
    max: 10

# Scheduled: Analysis with issue creation via safe outputs
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
    max: 3

# Manual: Admin tasks with approval gate
permissions: read-all
manual-approval: production
```

## Permission Validation

Run `gh aw compile workflow.md` to validate permissions. Common errors include undefined permissions, direct write permissions in the main job (use safe outputs instead), and insufficient permissions for declared tools. Use `--strict` mode to enforce read-only permissions and require explicit network configuration.

### Write Permission Policy

Write permissions are blocked by default to enforce the security-first design. Workflows with write permissions will fail compilation with an error:

```
Write permissions are not allowed.

Found write permissions:
  - contents: write

To fix this issue, change write permissions to read:
permissions:
  contents: read
```

**Exceptions:**
- `id-token: write` is allowed for OIDC authentication with cloud providers and does not grant repository write access.
- GitHub App-only scopes (see above) always refuse `write` at compile time regardless of this policy; use [safe outputs](/gh-aw/reference/safe-outputs/) for write operations that require a GitHub App.

#### Migrating Existing Workflows

To migrate workflows with write permissions, use the automated codemod (recommended):
```bash
# Check what would be changed (dry-run)
gh aw fix workflow.md

# Apply the fix
gh aw fix workflow.md --write
```

This automatically converts all write permissions to read permissions.

> [!TIP]
> For workflows that need to make changes to your repository, use [safe outputs](/gh-aw/reference/safe-outputs/) instead of write permissions.

This validation applies only to the top-level `permissions:` configuration. Custom jobs (`jobs:`) and safe outputs jobs (`safe-outputs.job:`) can have their own permission requirements.

### Tool-Specific Requirements

Some tools require specific permissions to function:

- **`agentic-workflows`**: Requires `actions: read` to access workflow logs and run data. Additionally, the `logs` and `audit` tools require the workflow actor to have **write, maintain, or admin** repository role.
- **GitHub Model Context Protocol (MCP) toolsets**: See [Tools](/gh-aw/reference/tools/) for GitHub API permission requirements

The compiler validates these requirements and provides clear error messages when permissions are missing.

## Related Documentation

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Secure write operations with content sanitization
- [Security Guide](/gh-aw/introduction/architecture/) - Security best practices and permission strategies
- [Tools](/gh-aw/reference/tools/) - GitHub API tools and their permission requirements
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration reference
