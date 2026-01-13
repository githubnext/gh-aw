# GitHub MCP Server Token Requirements

This document provides a comprehensive comparison of GitHub MCP Server tools against GitHub Actions `GITHUB_TOKEN` default permissions. It helps you understand which tools require a Personal Access Token (PAT) versus those that work with the default token.

**Last Updated**: 2026-01-13

## Understanding GitHub Actions Token Permissions

### GITHUB_TOKEN Default Permissions

As of 2023, GitHub changed the default permissions for `GITHUB_TOKEN` to **read-only** for new repositories and organizations. The default read-only permissions include:

- **contents**: `read` - Repository contents (code, files)
- **metadata**: `read` - Repository metadata (always read, cannot be changed)
- **issues**: `read` - Issues and comments
- **pull-requests**: `read` - Pull requests and reviews
- **actions**: `read` - Workflow runs and artifacts
- **checks**: `read` - Check runs and check suites
- **discussions**: `read` - Discussions (if enabled)

### What GITHUB_TOKEN Cannot Do

Without explicit permission grants in your workflow YAML:

- ❌ Cannot write to repository (push commits, create branches, create/update files)
- ❌ Cannot create, update, or comment on issues
- ❌ Cannot create, update, or merge pull requests
- ❌ Cannot create or update discussions
- ❌ Cannot manage repository settings
- ❌ Cannot access security events (code scanning, secret scanning, Dependabot)
- ❌ Cannot access organization-level data
- ❌ Cannot access user profile data beyond the triggering user
- ❌ Cannot access gists
- ❌ Cannot access notifications
- ❌ Cannot access GitHub Projects (Projects V2)

### Special Cases

1. **Pull Requests from Forks**: The token is **always read-only** regardless of your repository settings, preventing malicious PRs from modifying the repository.

2. **Organization Scope**: `GITHUB_TOKEN` is scoped to the repository where the workflow runs. It cannot access organization-level APIs like team membership.

3. **Projects V2**: GitHub Projects V2 requires the `project` OAuth scope, which is **not available** to `GITHUB_TOKEN` at all. You must use a PAT.

## Personal Access Token Types

When tools require a PAT (marked with ❌ in the tables below), you need to choose between two types of GitHub Personal Access Tokens:

### Classic Personal Access Tokens

**Characteristics**:
- Account-wide scope - grants access to all repositories and organizations the user has access to
- Broad permissions with limited granularity (e.g., `repo`, `admin:org`, `gist`)
- No mandatory expiration (though organizations can enforce policies)
- Required for some legacy use cases and specific scenarios

**When to use**:
- **User-owned Projects V2** - Classic PATs with `project` scope are **required** (fine-grained PATs do not work)
- External collaborator access in certain configurations
- Workflows requiring access across many repositories
- Legacy API endpoints that don't yet support fine-grained tokens

**Security considerations**:
- Higher risk if compromised - grants access to all user resources
- Difficult to audit and track usage
- Less control over permission scope

### Fine-Grained Personal Access Tokens (Recommended)

**Characteristics**:
- Repository-specific - you specify exactly which repositories the token can access
- Granular permissions - over 50 permission types with read/write/none options
- Mandatory expiration (up to 366 days, organization-configurable)
- Organization administrators can require approval before use
- Better audit trail and visibility

**When to use**:
- **Organization-owned Projects V2** - Fine-grained PATs work with proper organization permissions
- Most MCP server operations requiring PAT access
- CI/CD pipelines and automation requiring specific repository access
- Any scenario where you can apply least-privilege principles

**Security considerations**:
- Lower risk if compromised - limited to specified repositories and permissions
- Better organizational control and approval workflows
- Enforced expiration reduces long-term exposure
- Clearer permission model for auditing

### Choosing the Right Token Type

| Scenario | Recommended Token Type | Required Scopes/Permissions |
|----------|------------------------|----------------------------|
| User-owned Projects V2 | Classic PAT | `project`, `repo` (if private repos) |
| Organization Projects V2 | Fine-grained PAT | Org permissions: Projects: Read & Write; Repo permissions: as needed |
| Organization Projects V2 (alternative) | Classic PAT | `project`, `read:org`, `repo` (if private repos) |
| Code scanning, Dependabot | Fine-grained or Classic PAT | `security_events` (classic) or Security events permissions (fine-grained) |
| Organization/Team operations | Fine-grained or Classic PAT | `read:org` (classic) or Organization permissions: Members: Read (fine-grained) |
| Gist operations | Fine-grained or Classic PAT | `gist` (classic) or appropriate gist permissions (fine-grained) |
| Notifications | Fine-grained or Classic PAT | `notifications` (classic) or Notifications permissions (fine-grained) |

**Best practice**: Always prefer fine-grained PATs unless you specifically need classic PAT functionality (like user-owned Projects V2). Use the principle of least privilege by:
- Selecting only the repositories that need access
- Granting only the minimum permissions required
- Setting appropriate expiration dates
- Requiring organization approval when applicable

**Documentation**: See [GitHub's PAT documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens) and the [GitHub Tokens reference](/gh-aw/reference/tokens/) for detailed setup instructions.

## Tool Requirements by Toolset

The following tables show which tools in each GitHub MCP Server toolset work with default `GITHUB_TOKEN` permissions versus those requiring additional configuration or a PAT.

### Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Works with default `GITHUB_TOKEN` (read-only) |
| ⚠️ | Requires explicit permission grant in workflow YAML |
| ❌ | Requires PAT - not available to `GITHUB_TOKEN` |

---

## Actions Toolset

All Actions tools require `repo` scope (full repository access).

| Tool | Default Token | Notes |
|------|---------------|-------|
| `cancel_workflow_run` | ⚠️ | Requires `actions: write` permission |
| `delete_workflow_run_logs` | ⚠️ | Requires `actions: write` permission |
| `download_workflow_run_artifact` | ✅ | Works with `actions: read` (default) |
| `get_job_logs` | ✅ | Works with `actions: read` (default) |
| `get_workflow_run` | ✅ | Works with `actions: read` (default) |
| `get_workflow_run_logs` | ✅ | Works with `actions: read` (default) |
| `get_workflow_run_usage` | ✅ | Works with `actions: read` (default) |
| `list_workflow_jobs` | ✅ | Works with `actions: read` (default) |
| `list_workflow_run_artifacts` | ✅ | Works with `actions: read` (default) |
| `list_workflow_runs` | ✅ | Works with `actions: read` (default) |
| `list_workflows` | ✅ | Works with `actions: read` (default) |
| `rerun_failed_jobs` | ⚠️ | Requires `actions: write` permission |
| `rerun_workflow_run` | ⚠️ | Requires `actions: write` permission |
| `run_workflow` | ⚠️ | Requires `actions: write` permission |

**Summary**: Read operations work with defaults. Write operations (cancel, delete, rerun) require explicit `actions: write` in your workflow YAML.

---

## Code Security Toolset

All code security tools require `security_events` scope.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_code_scanning_alert` | ❌ | Requires PAT with `security_events` scope |
| `list_code_scanning_alerts` | ❌ | Requires PAT with `security_events` scope |

**Summary**: Code scanning alerts require PAT - `GITHUB_TOKEN` cannot access security events without GitHub Advanced Security and explicit permission grants.

**Configuration Required**: Even with explicit permission grants, code security tools may require repository-level GitHub Advanced Security features to be enabled.

---

## Context Toolset

Tools that provide context about the current user and GitHub environment.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_me` | ✅ | Works with default token (authenticated user info) |
| `get_team_members` | ❌ | Requires PAT with `read:org` scope |
| `get_teams` | ❌ | Requires PAT with `read:org` scope |

**Summary**: User profile works with defaults. Team/organization operations require PAT with organization read permissions.

---

## Dependabot Toolset

All Dependabot tools require `security_events` scope.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_dependabot_alert` | ❌ | Requires PAT with `security_events` scope |
| `list_dependabot_alerts` | ❌ | Requires PAT with `security_events` scope |

**Summary**: Dependabot alerts require PAT - `GITHUB_TOKEN` cannot access security events.

---

## Discussions Toolset

All discussion tools require `repo` scope.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_discussion` | ✅ | Works with default token (discussions: read) |
| `get_discussion_comments` | ✅ | Works with default token (discussions: read) |
| `list_discussion_categories` | ✅ | Works with default token |
| `list_discussions` | ✅ | Works with default token (discussions: read) |

**Summary**: Read operations work with defaults. Write operations (create, update discussions) would require explicit `discussions: write` permission grant.

---

## Gists Toolset

Gist operations require `gist` scope.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `create_gist` | ❌ | Requires PAT with `gist` scope |
| `get_gist` | ✅ | Public gists work; private gists require PAT |
| `list_gists` | ✅ | Lists public gists; private gists require PAT |
| `update_gist` | ❌ | Requires PAT with `gist` scope |

**Summary**: Read operations on public gists work. Creating or updating gists requires PAT with `gist` scope.

---

## Git Toolset

Low-level Git API operations.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_repository_tree` | ✅ | Works with `contents: read` (default) |

**Summary**: Repository tree reading works with defaults.

---

## Issues Toolset

Issue management tools.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `add_issue_comment` | ⚠️ | Requires `issues: write` permission grant |
| `assign_copilot_to_issue` | ⚠️ | Requires `issues: write` and `contents: write` |
| `get_label` | ✅ | Works with default token |
| `issue_read` | ✅ | Works with `issues: read` (default) |
| `issue_write` (create/update) | ⚠️ | Requires `issues: write` permission grant |
| `list_issue_types` | ❌ | Requires PAT with `read:org` scope |
| `list_issues` | ✅ | Works with `issues: read` (default) |
| `search_issues` | ✅ | Works with default token |
| `sub_issue_write` | ⚠️ | Requires `issues: write` permission grant |

**Summary**: Read operations work with defaults. Write operations (create, update, comment) require explicit `issues: write`. Organization-level features require PAT.

---

## Labels Toolset

Label management tools.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_label` | ✅ | Works with default token |
| `label_write` | ⚠️ | Requires `issues: write` or `pull-requests: write` |
| `list_label` | ✅ | Works with default token |

**Summary**: Read operations work with defaults. Label creation/updates require write permissions.

---

## Notifications Toolset

Notification management requires `notifications` scope.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `dismiss_notification` | ❌ | Requires PAT with `notifications` scope |
| `get_notification_details` | ❌ | Requires PAT with `notifications` scope |
| `list_notifications` | ❌ | Requires PAT with `notifications` scope |
| `manage_notification_subscription` | ❌ | Requires PAT with `notifications` scope |
| `manage_repository_notification_subscription` | ❌ | Requires PAT with `notifications` scope |
| `mark_all_notifications_read` | ❌ | Requires PAT with `notifications` scope |

**Summary**: All notification tools require PAT with `notifications` scope. `GITHUB_TOKEN` cannot access notifications.

---

## Organizations Toolset

Organization-level operations require organization permissions.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `search_orgs` | ❌ | Requires PAT with `read:org` scope |

**Summary**: Organization operations require PAT with organization read permissions.

---

## Projects Toolset

**⚠️ CRITICAL**: GitHub Projects V2 tools **always require a PAT** with `project` or `read:project` scope. The `project` scope is not available to `GITHUB_TOKEN` under any configuration.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `add_project_item` | ❌ | Requires PAT with `project` scope |
| `delete_project_item` | ❌ | Requires PAT with `project` scope |
| `get_project` | ❌ | Requires PAT with `read:project` or `project` scope |
| `get_project_field` | ❌ | Requires PAT with `read:project` or `project` scope |
| `get_project_item` | ❌ | Requires PAT with `read:project` or `project` scope |
| `list_project_fields` | ❌ | Requires PAT with `read:project` or `project` scope |
| `list_project_items` | ❌ | Requires PAT with `read:project` or `project` scope |
| `list_projects` | ❌ | Requires PAT with `read:project` or `project` scope |
| `update_project_item` | ❌ | Requires PAT with `project` scope |

**Summary**: **ALL** Projects tools require PAT. Use `read:project` for read-only operations or `project` for write operations.

**Documentation**: See [ProjectOps example](/gh-aw/examples/issue-pr-events/projectops/) for PAT configuration.

---

## Pull Requests Toolset

Pull request management tools.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `add_comment_to_pending_review` | ⚠️ | Requires `pull-requests: write` permission |
| `create_pull_request` | ⚠️ | Requires `pull-requests: write` and `contents: write` |
| `list_pull_requests` | ✅ | Works with `pull-requests: read` (default) |
| `merge_pull_request` | ⚠️ | Requires `pull-requests: write` and `contents: write` |
| `pull_request_read` | ✅ | Works with `pull-requests: read` (default) |
| `pull_request_review_write` | ⚠️ | Requires `pull-requests: write` permission |
| `request_copilot_review` | ⚠️ | Requires `pull-requests: write` permission |
| `search_pull_requests` | ✅ | Works with default token |
| `update_pull_request` | ⚠️ | Requires `pull-requests: write` permission |
| `update_pull_request_branch` | ⚠️ | Requires `pull-requests: write` and `contents: write` |

**Summary**: Read operations work with defaults. Write operations (create, update, merge, review) require explicit `pull-requests: write` and often `contents: write`.

---

## Repositories Toolset

Repository operations and code management.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `create_branch` | ⚠️ | Requires `contents: write` permission |
| `create_or_update_file` | ⚠️ | Requires `contents: write` permission |
| `create_repository` | ❌ | Requires PAT with `repo` or `public_repo` scope |
| `delete_file` | ⚠️ | Requires `contents: write` permission |
| `fork_repository` | ❌ | Requires PAT with `repo` scope |
| `get_commit` | ✅ | Works with `contents: read` (default) |
| `get_file_contents` | ✅ | Works with `contents: read` (default) |
| `get_latest_release` | ✅ | Works with default token |
| `get_release_by_tag` | ✅ | Works with default token |
| `get_tag` | ✅ | Works with default token |
| `list_branches` | ✅ | Works with default token |
| `list_commits` | ✅ | Works with `contents: read` (default) |
| `list_releases` | ✅ | Works with default token |
| `list_tags` | ✅ | Works with default token |
| `push_files` | ⚠️ | Requires `contents: write` permission |
| `search_code` | ✅ | Works with default token |
| `search_repositories` | ✅ | Works with default token |

**Summary**: Read operations work with defaults. Write operations (create, update, delete files/branches) require `contents: write`. Repository creation and forking require PAT.

---

## Secret Protection Toolset

Secret scanning tools require `security_events` scope.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_secret_scanning_alert` | ❌ | Requires PAT with `security_events` scope |
| `list_secret_scanning_alerts` | ❌ | Requires PAT with `security_events` scope |

**Summary**: Secret scanning requires PAT with `security_events` scope. May also require GitHub Advanced Security features.

---

## Security Advisories Toolset

Security advisory management.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_global_security_advisory` | ✅ | Works with default token (public data) |
| `list_global_security_advisories` | ✅ | Works with default token (public data) |
| `list_repository_security_advisories` | ❌ | Requires PAT with `security_events` scope |

**Summary**: Global advisories (public data) work with defaults. Repository-specific advisories require PAT.

---

## Stargazers Toolset

Star management tools.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `list_starred_repositories` | ✅ | Works with default token |

**Summary**: Listing stars works with defaults.

---

## Users Toolset

User profile and search tools.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `get_user` | ✅ | Works with default token (public profiles) |
| `search_users` | ✅ | Works with default token |

**Summary**: User profile operations work with defaults for public information.

---

## Search Toolset

Advanced search across GitHub.

| Tool | Default Token | Notes |
|------|---------------|-------|
| `search_code` | ✅ | Works with default token |
| `search_issues` | ✅ | Works with default token |
| `search_orgs` | ❌ | Requires PAT with `read:org` scope |
| `search_pull_requests` | ✅ | Works with default token |
| `search_repositories` | ✅ | Works with default token |
| `search_users` | ✅ | Works with default token |

**Summary**: Most search operations work with defaults. Organization search requires PAT with org permissions.

---

## Quick Reference: Tools Requiring PAT

The following tools **always require a Personal Access Token** and will not work with `GITHUB_TOKEN`:

### Organization/Team Operations
- `get_team_members` (requires `read:org`)
- `get_teams` (requires `read:org`)
- `list_issue_types` (requires `read:org`)
- `search_orgs` (requires `read:org`)
- `list_org_repository_security_advisories` (requires `read:org`)

### GitHub Projects V2
- **ALL** project tools (requires `project` or `read:project`)
- Projects V2 scope is not available to `GITHUB_TOKEN` at all

### Security Features
- All code security tools (requires `security_events`)
- All Dependabot tools (requires `security_events`)
- All secret protection tools (requires `security_events`)
- `list_repository_security_advisories` (requires `security_events`)

### Gist Operations
- `create_gist` (requires `gist`)
- `update_gist` (requires `gist`)

### Notifications
- **ALL** notification tools (requires `notifications`)

### Repository Creation
- `create_repository` (requires `repo` or `public_repo`)
- `fork_repository` (requires `repo`)

---

## Recommendations

### For Read-Only Workflows

Use default `GITHUB_TOKEN` with default toolsets:

```yaml
tools:
  github:
    toolsets: [default]  # context, repos, issues, pull_requests
```

This provides read access to:
- Repository contents and code
- Issues and comments
- Pull requests and reviews
- Workflow runs and artifacts

### For Write Operations

Use safe outputs instead of direct write permissions:

```yaml
permissions:
  contents: read
  issues: read
  pull-requests: read
safe-outputs:
  create-issue:
  add-comment:
  create-pr-review-comment:
```

This separates AI execution from write operations for security.

### For Advanced Features

Use a PAT when you need:

1. **GitHub Projects** (any Projects V2 operation)
2. **Security features** (code scanning, Dependabot, secret scanning)
3. **Organization operations** (teams, organization search)
4. **Notifications** (any notification management)
5. **Gist management** (creating or updating gists)

Configure PAT in your workflow:

```yaml
tools:
  github:
    mode: remote
    github-token: "${{ secrets.CUSTOM_GITHUB_PAT }}"
    toolsets: [default, projects, code_security]
```

### For Trial/Campaign Workflows

Consider read-only mode to prevent accidental writes:

```yaml
tools:
  github:
    mode: remote
    read-only: true
    toolsets: [default]
```

---

## See Also

- [GitHub MCP Server Documentation](/gh-aw/skills/github-mcp-server/)
- [Permissions Reference](/gh-aw/reference/permissions/)
- [Safe Outputs](/gh-aw/reference/safe-outputs/)
- [Tools Configuration](/gh-aw/reference/tools/)
- [ProjectOps Example](/gh-aw/examples/issue-pr-events/projectops/)
- [GitHub Token Documentation](https://docs.github.com/en/actions/security-guides/automatic-token-authentication)
